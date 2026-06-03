package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newResponsesStreamTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	return c, recorder
}

func newResponsesStreamHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func TestOaiResponsesStreamHandler_SynthesizesCompletedWhenMissing(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	c, recorder := newResponsesStreamTestContext()
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000}}`,
	}, "\n")

	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 0, usage.TotalTokens)
	require.NotNil(t, info.StreamStatus)
	require.True(t, info.StreamStatus.HasErrors())

	responseBody := recorder.Body.String()
	require.Contains(t, responseBody, "event: response.completed")
	require.Contains(t, responseBody, `"id":"resp_1"`)
	require.Contains(t, responseBody, `"status":"completed"`)
	require.Contains(t, responseBody, `"input_tokens":0`)
}

func TestOaiResponsesStreamHandler_DoesNotDuplicateCompleted(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	c, recorder := newResponsesStreamTestContext()
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000}}`,
		`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000,"usage":{"input_tokens":3,"output_tokens":4,"total_tokens":7}}}`,
		`data: [DONE]`,
	}, "\n")

	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 3, usage.PromptTokens)
	require.Equal(t, 4, usage.CompletionTokens)
	require.Equal(t, 7, usage.TotalTokens)
	require.False(t, info.StreamStatus.HasErrors())

	require.Equal(t, 1, strings.Count(recorder.Body.String(), "event: response.completed"))
}

func TestOaiResponsesStreamHandler_ChannelTestFailsWhenCompletedMissing(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	c, recorder := newResponsesStreamTestContext()
	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	body := `data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000}}`

	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.NotNil(t, usage)
	require.Error(t, err)
	require.Contains(t, err.Error(), "response.completed")
	require.NotContains(t, recorder.Body.String(), "event: response.completed")
}
