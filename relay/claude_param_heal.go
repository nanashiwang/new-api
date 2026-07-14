package relay

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/channel"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/sjson"
)

// 错误驱动的参数自愈：上游 400 明确指认某个采样参数（deprecated / not supported）时，
// 剥除该参数并在同渠道原样重发一次。上游参数规则随模型版本演化，客户端永远滞后，
// 枚举式适配（硬编码模型前缀）只能覆盖已知规则；这里对上游的明确反馈做自动响应，
// 未来任何参数废弃都无需改代码。

const claudeParamHealDoneKey = "claude_param_heal_done"

// 只允许剥除采样类参数：剥掉它们不改变请求语义，最多损失一点采样偏好。
// max_tokens/messages/tools 等语义参数绝不自动剥。
var healableParams = map[string]struct{}{
	"temperature": {},
	"top_p":       {},
	"top_k":       {},
}

// 匹配 Anthropic/OpenAI 风格的参数错误指认，如：
//   `temperature` is deprecated for this model.
//   `top_k` is not supported ...
//   Unsupported parameter: `top_p` ...
var healableParamErrorRegexp = regexp.MustCompile(
	"`([a-zA-Z_][a-zA-Z0-9_]*)`(?:[a-zA-Z ]*)? is (?:deprecated|not supported|unsupported)|[Uu]nsupported parameter[: ]+`?([a-zA-Z_][a-zA-Z0-9_]*)`?",
)

func extractHealableParam(bodyText string) string {
	m := healableParamErrorRegexp.FindStringSubmatch(bodyText)
	if m == nil {
		return ""
	}
	param := m[1]
	if param == "" {
		param = m[2]
	}
	if _, ok := healableParams[param]; !ok {
		return ""
	}
	return param
}

// tryHealClaudeParamError 在上游返回 400 时尝试剥参重发一次。
// 返回值：(生效的响应, 是否重发过)。任何环节不满足条件都原样返回首次响应。
func tryHealClaudeParamError(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	adaptor channel.Adaptor,
	requestJSON []byte,
	httpResp *http.Response,
) (*http.Response, bool) {
	if c == nil || info == nil || adaptor == nil || httpResp == nil || len(requestJSON) == 0 {
		return httpResp, false
	}
	if info.ChannelType != constant.ChannelTypeAnthropic {
		return httpResp, false
	}
	if httpResp.StatusCode != http.StatusBadRequest {
		return httpResp, false
	}
	// 每请求最多自愈一次，防止循环
	if c.GetBool(claudeParamHealDoneKey) {
		return httpResp, false
	}

	bodyText, ok := peekResponseBody(httpResp)
	if !ok || !strings.Contains(bodyText, "invalid_request_error") {
		return httpResp, false
	}
	param := extractHealableParam(bodyText)
	if param == "" {
		return httpResp, false
	}
	// 请求体里确实带了这个参数才有剥除的意义
	strippedJSON, err := sjson.DeleteBytes(requestJSON, param)
	if err != nil || bytes.Equal(strippedJSON, requestJSON) {
		return httpResp, false
	}

	c.Set(claudeParamHealDoneKey, true)
	logger.LogWarn(c, fmt.Sprintf(
		"param self-heal: upstream 400 rejected `%s`, stripped and retrying once (channel #%d, model %s)",
		param, info.ChannelId, info.UpstreamModelName,
	))

	retryResp, err := adaptor.DoRequest(c, info, bytes.NewBuffer(strippedJSON))
	if err != nil {
		logger.LogWarn(c, fmt.Sprintf("param self-heal retry failed, falling back to original 400: %v", err))
		return httpResp, false
	}
	healedResp, ok := retryResp.(*http.Response)
	if !ok {
		return httpResp, false
	}
	// 可观测性：让调用方知道参数被服务端调整过
	c.Header("X-Param-Adjusted", param)
	return healedResp, true
}
