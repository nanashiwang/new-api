package relay

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type claudeThinkingHealTestAdaptor struct {
	requests []string
	response *http.Response
}

var _ channel.Adaptor = (*claudeThinkingHealTestAdaptor)(nil)

func (a *claudeThinkingHealTestAdaptor) Init(*relaycommon.RelayInfo) {}

func (a *claudeThinkingHealTestAdaptor) GetRequestURL(*relaycommon.RelayInfo) (string, error) {
	return "", nil
}

func (a *claudeThinkingHealTestAdaptor) SetupRequestHeader(*gin.Context, *http.Header, *relaycommon.RelayInfo) error {
	return nil
}

func (a *claudeThinkingHealTestAdaptor) ConvertOpenAIRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeneralOpenAIRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertOpenAIRequest call")
}

func (a *claudeThinkingHealTestAdaptor) ConvertRerankRequest(*gin.Context, int, dto.RerankRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertRerankRequest call")
}

func (a *claudeThinkingHealTestAdaptor) ConvertEmbeddingRequest(*gin.Context, *relaycommon.RelayInfo, dto.EmbeddingRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertEmbeddingRequest call")
}

func (a *claudeThinkingHealTestAdaptor) ConvertAudioRequest(*gin.Context, *relaycommon.RelayInfo, dto.AudioRequest) (io.Reader, error) {
	return nil, fmt.Errorf("unexpected ConvertAudioRequest call")
}

func (a *claudeThinkingHealTestAdaptor) ConvertImageRequest(*gin.Context, *relaycommon.RelayInfo, dto.ImageRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertImageRequest call")
}

func (a *claudeThinkingHealTestAdaptor) ConvertOpenAIResponsesRequest(*gin.Context, *relaycommon.RelayInfo, dto.OpenAIResponsesRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertOpenAIResponsesRequest call")
}

func (a *claudeThinkingHealTestAdaptor) DoRequest(_ *gin.Context, _ *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	body, err := io.ReadAll(requestBody)
	if err != nil {
		return nil, err
	}
	a.requests = append(a.requests, string(body))
	return a.response, nil
}

func (a *claudeThinkingHealTestAdaptor) DoResponse(*gin.Context, *http.Response, *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	return nil, types.NewOpenAIError(fmt.Errorf("unexpected DoResponse call"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
}

func (a *claudeThinkingHealTestAdaptor) GetModelList() []string { return nil }

func (a *claudeThinkingHealTestAdaptor) GetChannelName() string { return "test" }

func (a *claudeThinkingHealTestAdaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertClaudeRequest call")
}

func (a *claudeThinkingHealTestAdaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertGeminiRequest call")
}

func TestExtractHealableParam(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "anthropic temperature deprecated",
			body: `{"type":"error","error":{"type":"invalid_request_error","message":"` + "`temperature`" + ` is deprecated for this model."}}`,
			want: "temperature",
		},
		{
			name: "top_k not supported",
			body: `{"type":"error","error":{"type":"invalid_request_error","message":"` + "`top_k`" + ` is not supported with thinking."}}`,
			want: "top_k",
		},
		{
			name: "openai style unsupported parameter",
			body: `{"error":{"type":"invalid_request_error","message":"Unsupported parameter: top_p"}}`,
			want: "top_p",
		},
		{
			name: "semantic param never healed",
			body: `{"type":"error","error":{"type":"invalid_request_error","message":"` + "`max_tokens`" + ` is not supported."}}`,
			want: "",
		},
		{
			name: "unrelated 400",
			body: `{"type":"error","error":{"type":"invalid_request_error","message":"prompt is too long: 210000 tokens > 200000 maximum"}}`,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractHealableParam(tc.body); got != tc.want {
				t.Fatalf("extractHealableParam() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTryHealClaudeParamErrorRemovesEmptyThinkingAndRetriesOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	requestJSON := []byte(`{"model":"claude-test","messages":[{"role":"assistant","content":[{"type":"thinking","thinking":""},{"type":"thinking","thinking":"valid","signature":"keep"},{"type":"text","text":"answer"}]}]}`)
	originalResp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"invalid_request_error","message":"each thinking block must contain thinking"}}`)),
	}
	healedResp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}
	adaptor := &claudeThinkingHealTestAdaptor{response: healedResp}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:       constant.ChannelTypeAnthropic,
		ChannelId:         123,
		UpstreamModelName: "claude-test",
	}}

	got, healed := tryHealClaudeParamError(context, info, adaptor, requestJSON, originalResp)
	if !healed || got != healedResp {
		t.Fatalf("tryHealClaudeParamError() = (%v, %v), want healed response", got, healed)
	}
	if len(adaptor.requests) != 1 {
		t.Fatalf("retry requests = %d, want 1", len(adaptor.requests))
	}
	if strings.Contains(adaptor.requests[0], `"thinking":""`) {
		t.Fatalf("retry still contains empty thinking: %s", adaptor.requests[0])
	}
	if !strings.Contains(adaptor.requests[0], `"thinking":"valid"`) || !strings.Contains(adaptor.requests[0], `"signature":"keep"`) {
		t.Fatalf("retry lost valid thinking fields: %s", adaptor.requests[0])
	}
	if context.Writer.Header().Get("X-Request-Adjusted") != "empty_thinking" {
		t.Fatalf("X-Request-Adjusted = %q", context.Writer.Header().Get("X-Request-Adjusted"))
	}

	secondResp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"invalid_request_error","message":"each thinking block must contain thinking"}}`)),
	}
	if got, healed := tryHealClaudeParamError(context, info, adaptor, requestJSON, secondResp); healed || got != secondResp {
		t.Fatalf("second self-heal should be blocked, got (%v, %v)", got, healed)
	}
	if len(adaptor.requests) != 1 {
		t.Fatalf("second self-heal retried unexpectedly: %d requests", len(adaptor.requests))
	}
}
