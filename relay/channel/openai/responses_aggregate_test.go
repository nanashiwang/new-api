package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 方案B 核心:codex 恒定强制上游流式,客户端要非流式时把上游 SSE 聚合成单个非流式响应。
// 本组测试锁定「聚合后返回给客户端的一定是非流式 JSON、且计费 usage 来自 completed 事件」。

func newResponsesSSEResponse(events ...string) *http.Response {
	body := strings.Join(append(events, `data: [DONE]`), "\n")
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			// 模拟上游 codex 被强制流式后返回的真实 SSE 头组合(与 SetEventStreamHeaders 一致),
			// 用于验证聚合写出前会把这些流式语义头清理干净。
			"Content-Type":      []string{"text/event-stream"},
			"Cache-Control":     []string{"no-cache"},
			"Connection":        []string{"keep-alive"},
			"Transfer-Encoding": []string{"chunked"},
			"X-Accel-Buffering": []string{"no"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

// requireNoStreamingHeaders 断言聚合后的客户端响应头已剥离一切 SSE/流式语义头,只余非流式 JSON 语义。
func requireNoStreamingHeaders(t *testing.T, h http.Header) {
	t.Helper()
	require.Equal(t, "application/json", h.Get("Content-Type"))
	require.Empty(t, h.Get("X-Accel-Buffering"))
	require.Empty(t, h.Get("Transfer-Encoding"))
	require.Empty(t, h.Get("Cache-Control"))
}

// Path A:原生 /v1/responses 非流式客户端 → 聚合成 Responses JSON。
func TestOaiResponsesAggregateHandler_ReturnsNonStreamResponsesJSON(t *testing.T) {

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := newResponsesSSEResponse(
		`data: {"type":"response.created","response":{"id":"resp_agg","model":"gpt-5.1-codex","created_at":1700000000}}`,
		`data: {"type":"response.output_text.delta","delta":"Hello"}`,
		`data: {"type":"response.output_text.delta","delta":", world"}`,
		`data: {"type":"response.completed","response":{"id":"resp_agg","object":"response","status":"completed","model":"gpt-5.1-codex","created_at":1700000000,"usage":{"input_tokens":11,"output_tokens":7,"total_tokens":18,"input_tokens_details":{"cached_tokens":9}},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hello, world"}]}]}}`,
	)

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.1-codex"},
	}

	usage, apiErr := OaiResponsesAggregateHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	// 计费必须来自 completed 事件(含缓存命中),不能为 0。
	require.Equal(t, 11, usage.PromptTokens)
	require.Equal(t, 7, usage.CompletionTokens)
	require.Equal(t, 18, usage.TotalTokens)
	require.Equal(t, 9, usage.PromptTokensDetails.CachedTokens)

	// 客户端拿到的必须是非流式 JSON:Content-Type=application/json,且流式头已被剥离。
	requireNoStreamingHeaders(t, recorder.Header())
	out := recorder.Body.String()
	require.NotContains(t, out, "data:")
	require.NotContains(t, out, "event:")

	// body 能被反序列化回一个完整 Responses 对象。
	var parsed dto.OpenAIResponsesResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &parsed))
	require.Equal(t, "resp_agg", parsed.ID)
	require.Equal(t, "completed", parsed.Status)
	require.Equal(t, "Hello, world", service.ExtractOutputTextFromResponses(&parsed))
}

// Path B:chat/completions 非流式客户端(经桥接)→ 聚合成 chat.completion JSON。
func TestOaiResponsesToChatAggregateHandler_ReturnsNonStreamChatJSON(t *testing.T) {

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := newResponsesSSEResponse(
		`data: {"type":"response.output_text.delta","delta":"Answer"}`,
		`data: {"type":"response.completed","response":{"id":"resp_chat","object":"response","status":"completed","model":"gpt-5.1-codex","created_at":1700000000,"usage":{"input_tokens":20,"output_tokens":4,"total_tokens":24},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Answer"}]}]}}`,
	)

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.1-codex"},
	}

	usage, apiErr := OaiResponsesToChatAggregateHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	require.Equal(t, 20, usage.PromptTokens)
	require.Equal(t, 4, usage.CompletionTokens)

	requireNoStreamingHeaders(t, recorder.Header())
	out := recorder.Body.String()
	require.NotContains(t, out, "data:")
	require.Contains(t, out, `"object":"chat.completion"`)
	require.Contains(t, out, "Answer")
}

// 上游流中途报错(response.failed)→ 聚合必须返回错误,且不向客户端写半截内容。
func TestOaiResponsesAggregateHandler_UpstreamFailedReturnsError(t *testing.T) {

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := newResponsesSSEResponse(
		`data: {"type":"response.output_text.delta","delta":"partial"}`,
		`data: {"type":"response.failed","response":{"id":"resp_fail","status":"failed","error":{"type":"server_error","message":"boom"}}}`,
	)

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.1-codex"},
	}

	usage, apiErr := OaiResponsesAggregateHandler(c, info, resp)
	require.NotNil(t, apiErr)
	require.Nil(t, usage)
	require.Empty(t, recorder.Body.String())
}

// 上游流未发 completed 就结束 → 聚合返回错误(避免计费 0 / 回半截 SSE)。
func TestOaiResponsesAggregateHandler_MissingCompletedReturnsError(t *testing.T) {

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := newResponsesSSEResponse(
		`data: {"type":"response.output_text.delta","delta":"only a delta, no completed"}`,
	)

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.1-codex"},
	}

	usage, apiErr := OaiResponsesAggregateHandler(c, info, resp)
	require.NotNil(t, apiErr)
	require.Nil(t, usage)
}
