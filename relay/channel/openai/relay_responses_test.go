package openai

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

var responsesStreamTokenEncoderOnce sync.Once

func newResponsesStreamTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	responsesStreamTokenEncoderOnce.Do(service.InitTokenEncoders)
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

type errorAfterBodyReadCloser struct {
	reader *strings.Reader
	err    error
}

func (r *errorAfterBodyReadCloser) Read(p []byte) (int, error) {
	if r.reader.Len() > 0 {
		return r.reader.Read(p)
	}
	return 0, r.err
}

func (r *errorAfterBodyReadCloser) Close() error {
	return nil
}

func newResponsesStreamHTTPResponseWithReadError(body string, err error) *http.Response {
	resp := newResponsesStreamHTTPResponse("")
	resp.Body = &errorAfterBodyReadCloser{
		reader: strings.NewReader(body),
		err:    err,
	}
	return resp
}

func TestOaiResponsesStreamHandler_FailsWhenMissingCompletedWithoutOutputAfterEOF(t *testing.T) {
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
	require.Contains(t, responseBody, "event: response.failed")
	require.Contains(t, responseBody, `"id":"resp_1"`)
	require.Contains(t, responseBody, `"status":"failed"`)
	require.Contains(t, responseBody, `stream end: eof`)
	require.NotContains(t, responseBody, "event: response.completed")
}

func TestOaiResponsesStreamHandler_SynthesizesCompletedWhenMissingWithOutput(t *testing.T) {
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
		`data: {"type":"response.output_text.delta","delta":"partial output"}`,
	}, "\n")

	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.NotNil(t, info.StreamStatus)
	require.True(t, info.StreamStatus.HasErrors())

	responseBody := recorder.Body.String()
	require.Contains(t, responseBody, "partial output")
	require.Contains(t, responseBody, "event: response.completed")
	require.Contains(t, responseBody, `"input_tokens":0`)
	require.NotContains(t, responseBody, "event: response.failed")
}

func TestOaiResponsesStreamHandler_FailsWhenMissingCompletedWithoutOutputAfterScannerError(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	c, recorder := newResponsesStreamTestContext()
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	body := `data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000}}` + "\n"

	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponseWithReadError(body, errors.New("upstream read timeout")))
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 0, usage.TotalTokens)
	require.NotNil(t, info.StreamStatus)
	require.Equal(t, relaycommon.StreamEndReasonScannerErr, info.StreamStatus.EndReason)
	require.True(t, info.StreamStatus.HasErrors())

	responseBody := recorder.Body.String()
	require.Contains(t, responseBody, "event: response.failed")
	require.Contains(t, responseBody, `"status":"failed"`)
	require.Contains(t, responseBody, `stream end: scanner_error`)
	require.NotContains(t, responseBody, "event: response.completed")
}

func TestOaiResponsesStreamHandler_SynthesizesCompletedWhenOutputExistsAfterScannerError(t *testing.T) {
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
		`data: {"type":"response.output_text.delta","delta":"partial output"}`,
	}, "\n") + "\n"

	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponseWithReadError(body, errors.New("upstream read timeout")))
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.NotNil(t, info.StreamStatus)
	require.Equal(t, relaycommon.StreamEndReasonScannerErr, info.StreamStatus.EndReason)
	require.True(t, info.StreamStatus.HasErrors())

	responseBody := recorder.Body.String()
	require.Contains(t, responseBody, "partial output")
	require.Contains(t, responseBody, "event: response.completed")
	require.NotContains(t, responseBody, "event: response.failed")
}

func TestOaiResponsesStreamHandler_AutoContinuesBeforeSyntheticCompleted(t *testing.T) {
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
		`data: {"type":"response.output_text.delta","delta":"partial output"}`,
	}, "\n") + "\n"

	called := false
	usage, err := OaiResponsesStreamHandlerWithOptions(c, info, newResponsesStreamHTTPResponseWithReadError(body, errors.New("upstream read timeout")), &ResponsesStreamHandlerOptions{
		AutoContinue: func(streamCtx ResponsesStreamAutoContinueContext) (*dto.Usage, bool) {
			called = true
			require.Equal(t, "partial output", streamCtx.OutputText)
			return &dto.Usage{PromptTokens: 5, CompletionTokens: 6, TotalTokens: 11}, true
		},
	})
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.True(t, called)
	require.GreaterOrEqual(t, usage.CompletionTokens, 6)

	responseBody := recorder.Body.String()
	require.Contains(t, responseBody, "partial output")
	require.NotContains(t, responseBody, "event: response.completed")
	require.NotContains(t, responseBody, "event: response.failed")
}

func TestOaiResponsesStreamHandler_ContinuationOutputOnlyFiltersLifecycleEvents(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	c, recorder := newResponsesStreamTestContext()
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_2","model":"gpt-5.5","created_at":1700000001}}`,
		`data: {"type":"response.output_item.added","item":{"type":"message","id":"msg_1","role":"assistant"}}`,
		`data: {"type":"response.output_text.delta","delta":"continued"}`,
		`data: {"type":"response.completed","response":{"id":"resp_2","model":"gpt-5.5","created_at":1700000001,"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"continued"}]}],"usage":{"input_tokens":5,"output_tokens":1,"total_tokens":6}}}`,
	}, "\n")

	usage, err := OaiResponsesStreamHandlerWithOptions(c, info, newResponsesStreamHTTPResponse(body), &ResponsesStreamHandlerOptions{
		ContinuationOutputOnly: true,
	})
	require.Nil(t, err)
	require.Equal(t, 6, usage.TotalTokens)

	responseBody := recorder.Body.String()
	require.NotContains(t, responseBody, "event: response.created")
	require.NotContains(t, responseBody, "event: response.output_item.added")
	require.Contains(t, responseBody, "continued")
	require.Contains(t, responseBody, "event: response.completed")
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
		`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000,"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}],"usage":{"input_tokens":3,"output_tokens":4,"total_tokens":7}}}`,
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

func TestOaiResponsesStreamHandler_RecordsCompletedSummary(t *testing.T) {
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
		`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000,"status":"completed","output":[{"type":"reasoning"},{"type":"function_call","name":"shell","arguments":"secret_value"},{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]},{"type":"web_search_call","status":"completed"},{"type":"image_generation_call","status":"completed"}],"usage":{"input_tokens":3,"output_tokens":4,"total_tokens":7}}}`,
		`data: [DONE]`,
	}, "\n")

	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.Nil(t, err)
	require.Equal(t, 7, usage.TotalTokens)

	summary := info.ResponsesCompletedSummary
	require.NotNil(t, summary)
	require.Equal(t, "completed", summary.Status)
	require.Equal(t, 5, summary.OutputCount)
	require.Equal(t, []string{"reasoning", "function_call", "message", "web_search_call", "image_generation_call"}, summary.OutputTypes)
	require.Equal(t, 1, summary.MessageCount)
	require.Equal(t, 1, summary.FunctionCallCount)
	require.Equal(t, 1, summary.WebSearchCallCount)
	require.Equal(t, 1, summary.ImageGenerationCallCount)
	require.Equal(t, 1, summary.ActionableToolCallCount)
	require.Equal(t, 4, summary.MessageTextChars)
	require.True(t, summary.HasActionableToolCall)

	summarySnapshot := fmt.Sprintf("%#v", summary)
	require.NotContains(t, summarySnapshot, "secret_value")
	require.NotContains(t, summarySnapshot, "done")
	require.Contains(t, recorder.Body.String(), "event: response.completed")
}

func TestOaiResponsesStreamHandler_RecordsMessageOnlyCompletedSummary(t *testing.T) {
	t.Parallel()
	setResponsesStreamTestTimeout(t)

	c, _ := newResponsesStreamTestContext()
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000}}`,
		`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000,"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"top-secret"}]}],"usage":{"input_tokens":3,"output_tokens":4,"total_tokens":7}}}`,
		`data: [DONE]`,
	}, "\n")

	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.Nil(t, err)
	require.Equal(t, 7, usage.TotalTokens)

	summary := info.ResponsesCompletedSummary
	require.NotNil(t, summary)
	require.Equal(t, 1, summary.OutputCount)
	require.Equal(t, []string{"message"}, summary.OutputTypes)
	require.Equal(t, 1, summary.MessageCount)
	require.Zero(t, summary.FunctionCallCount)
	require.Zero(t, summary.ActionableToolCallCount)
	require.False(t, summary.HasActionableToolCall)
	require.Equal(t, len("top-secret"), summary.MessageTextChars)
	require.NotContains(t, fmt.Sprintf("%#v", summary), "top-secret")
}

func TestOaiResponsesStreamHandler_FailsWhenCompletedHasNoOutput(t *testing.T) {
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
		`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-5.5","created_at":1700000000,"usage":{"input_tokens":3,"output_tokens":37,"total_tokens":40}}}`,
		`data: [DONE]`,
	}, "\n")

	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 3, usage.PromptTokens)
	require.Equal(t, 37, usage.CompletionTokens)
	require.True(t, info.StreamStatus.HasErrors())

	responseBody := recorder.Body.String()
	require.Contains(t, responseBody, "event: response.failed")
	require.Contains(t, responseBody, `"status":"failed"`)
	require.Contains(t, responseBody, "responses stream completed without visible output")
	require.NotContains(t, responseBody, "event: response.completed")
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
