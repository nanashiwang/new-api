package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setResponsesStreamTestTimeout(t *testing.T) {
	t.Helper()
	original := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = original
	})
}

func TestOaiResponsesToChatStreamHandler_CountsWebSearchCalls(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-4.1","created_at":1700000000}}`,
		`data: {"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_1","status":"completed"}}`,
		`data: {"type":"response.output_text.delta","delta":"Found it."}`,
		`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-4.1","created_at":1700000000,"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15},"output":[{"type":"web_search_call","id":"ws_1"},{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Found it."}]}]}}`,
		`data: [DONE]`,
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	usage, err := OaiResponsesToChatStreamHandler(c, &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-4.1",
		},
	}, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 1, usage.WebSearchRequests)
	require.Equal(t, 10, usage.PromptTokens)
	require.Equal(t, 5, usage.CompletionTokens)
}

func TestOaiResponsesToChatStreamHandler_ClaudeWebSearchEmitsAnthropicBlocks(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-4.1","created_at":1700000000}}`,
		`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-4.1","created_at":1700000000,"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15},"output":[{"type":"web_search_call","id":"ws_1","action":{"query":"latest OpenAI news","sources":[{"url":"https://example.com/openai","title":"OpenAI source","snippet":"Alpha summary"}]}},{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Alpha summary with context","annotations":[{"type":"url_citation","url":"https://example.com/openai","title":"OpenAI source","start_index":0,"end_index":5}]}]}]}}`,
		`data: [DONE]`,
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	usage, err := OaiResponsesToChatStreamHandler(c, &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-4.1",
		},
	}, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 1, usage.WebSearchRequests)

	responseBody := recorder.Body.String()
	require.Contains(t, responseBody, "event: message_start")
	require.Contains(t, responseBody, `"type":"server_tool_use"`)
	require.Contains(t, responseBody, `"type":"web_search_tool_result"`)
	require.Contains(t, responseBody, `"type":"web_search_result_location"`)
	require.Contains(t, responseBody, `"web_search_requests":1`)
}

func TestOaiResponsesToChatStreamHandler_ReturnsIncompleteError(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-5","created_at":1700000000}}`,
		`data: {"type":"response.incomplete","response":{"id":"resp_1","model":"gpt-5","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}}`,
		`data: [DONE]`,
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	usage, err := OaiResponsesToChatStreamHandler(c, &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5",
		},
	}, resp)
	require.Nil(t, usage)
	require.Error(t, err)
	require.Contains(t, err.Error(), "responses stream incomplete")
	require.Contains(t, err.Error(), "max_output_tokens")
}

func TestOaiResponsesToChatStreamHandler_HandlesTopLevelErrorEvent(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-5","created_at":1700000000}}`,
		`data: {"type":"error","error":{"message":"upstream boom","type":"server_error","code":"server_error"}}`,
		`data: [DONE]`,
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	usage, err := OaiResponsesToChatStreamHandler(c, &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5",
		},
	}, resp)
	require.Nil(t, usage)
	require.Error(t, err)
	require.Contains(t, err.Error(), "upstream boom")
}
