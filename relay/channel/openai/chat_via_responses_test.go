package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesToChatStreamHandler_CountsWebSearchCalls(t *testing.T) {
	t.Parallel()

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
		RelayFormat:       types.RelayFormatOpenAI,
		UpstreamModelName: "gpt-4.1",
	}, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 1, usage.WebSearchRequests)
	require.Equal(t, 10, usage.PromptTokens)
	require.Equal(t, 5, usage.CompletionTokens)
}
