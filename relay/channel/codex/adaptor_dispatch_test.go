package codex

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 方案B 的 codex 侧改动:普通 responses 恒定强制上游 stream=true;客户端要非流式时,
// DoResponse 依据上游实际返回类型把 SSE 聚合成非流式 JSON。本组测试锁定这些分支
//(此前 codex 包仅有 constants_test.go,这些路径零覆盖)。

func newCodexInfo(mode int, isStream bool) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayMode:   mode,
		IsStream:    isStream,
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.1-codex"},
	}
}

// 普通 responses:即使客户端 stream=false,也必须强制上游 stream=true(否则 codex 报
// "Stream must be set to true")。
func TestCodexConvertOpenAIResponsesRequest_ForcesUpstreamStream(t *testing.T) {
	a := &Adaptor{}
	info := newCodexInfo(relayconstant.RelayModeResponses, false)

	out, err := a.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model:  "gpt-5.1-codex",
		Stream: false,
	})
	require.NoError(t, err)
	converted, ok := out.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.True(t, converted.Stream, "普通 responses 必须强制上游 stream=true")
}

// compact 路径不属于本次改造范围:不得被强制流式(保持原样)。
func TestCodexConvertOpenAIResponsesRequest_CompactNotForced(t *testing.T) {
	a := &Adaptor{}
	info := newCodexInfo(relayconstant.RelayModeResponsesCompact, false)

	out, err := a.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model:  "gpt-5.1-codex",
		Stream: false,
	})
	require.NoError(t, err)
	converted, ok := out.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.False(t, converted.Stream, "compact 路径不得被强制流式")
}

// 核心分派:客户端非流式 + 上游返回 SSE(被强制流式)→ 聚合成单个非流式 JSON 返回。
func TestCodexDoResponse_NonStreamClientAggregatesUpstreamSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	a := &Adaptor{}
	info := newCodexInfo(relayconstant.RelayModeResponses, false) // 客户端要非流式

	body := strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"ok"}`,
		`data: {"type":"response.completed","response":{"id":"r1","object":"response","status":"completed","model":"gpt-5.1-codex","usage":{"input_tokens":5,"output_tokens":3,"total_tokens":8},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}}`,
		`data: [DONE]`,
	}, "\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	usage, apiErr := a.DoResponse(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	// 客户端拿到的是非流式 JSON,不是 SSE。
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	require.NotContains(t, recorder.Body.String(), "data:")
	require.Contains(t, recorder.Body.String(), `"status":"completed"`)
}

// 分派:客户端非流式 + 上游返回普通 JSON(理论兜底)→ 走原非流式 handler,直接透传。
func TestCodexDoResponse_NonStreamClientNonStreamUpstreamUsesPlainHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	a := &Adaptor{}
	info := newCodexInfo(relayconstant.RelayModeResponses, false)

	jsonBody := `{"id":"r2","object":"response","status":"completed","model":"gpt-5.1-codex","usage":{"input_tokens":6,"output_tokens":2,"total_tokens":8},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi"}]}]}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(jsonBody)),
	}

	usage, apiErr := a.DoResponse(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	require.Contains(t, recorder.Body.String(), `"id":"r2"`)
}
