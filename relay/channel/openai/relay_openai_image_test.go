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
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImagesOutputModalityUsageAcrossFormats(t *testing.T) {
	const rawUsage = `{"input_tokens":15,"output_tokens":1352,"output_tokens_details":{"image_tokens":1120,"text_tokens":232}}`
	for _, format := range []string{"json", "sse", "json-to-sse"} {
		t.Run(format, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
			body := `{"data":[{"b64_json":"image"}],"usage":` + rawUsage + `}`
			contentType := "application/json"
			handler := OpenaiHandlerWithUsage
			if format == "sse" {
				body = "data: {\"type\":\"image_generation.completed\",\"usage\":" + rawUsage + "}\n\ndata: [DONE]\n\n"
				contentType, handler = "text/event-stream", OpenaiImageStreamHandler
			} else if format == "json-to-sse" {
				handler = OpenaiImageJSONAsStreamHandler
			}
			resp := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(strings.NewReader(body))}
			usage, apiErr := handler(c, &relaycommon.RelayInfo{}, resp)
			require.Nil(t, apiErr)
			require.Equal(t, 1120, usage.CompletionTokenDetails.ImageTokens)
			require.Equal(t, 232, usage.CompletionTokenDetails.TextTokens)
			require.Equal(t, 1367, usage.TotalTokens)
			normalizeOpenAIUsage(usage)
			require.Equal(t, 1120, usage.CompletionTokenDetails.ImageTokens, "normalization must be idempotent")
		})
	}
	var usage dto.Usage
	require.NoError(t, common.Unmarshal([]byte(`{"completion_tokens_details":{"image_tokens":7}}`), &usage))
	normalizeOpenAIUsage(&usage)
	require.Equal(t, 7, usage.CompletionTokenDetails.ImageTokens, "missing alias must preserve canonical usage")
	require.NoError(t, common.Unmarshal([]byte(`{"output_tokens_details":{"image_tokens":0}}`), &usage))
	normalizeOpenAIUsage(&usage)
	require.Zero(t, usage.CompletionTokenDetails.ImageTokens, "explicit zero is authoritative")
}

func TestOpenaiHandlerWithUsagePassesThroughImageEventStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := "data: {\"type\":\"image_generation.partial_image\"}\n\ndata: [DONE]\n\n"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream; charset=utf-8"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	usage, apiErr := OpenaiHandlerWithUsage(c, &relaycommon.RelayInfo{}, resp)
	if apiErr != nil {
		t.Fatalf("OpenaiHandlerWithUsage returned error: %v", apiErr)
	}
	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if usage.TotalTokens != 0 {
		t.Fatalf("expected empty usage fallback, got total_tokens=%d", usage.TotalTokens)
	}
	if got := recorder.Body.String(); got != body {
		t.Fatalf("response body mismatch:\ngot:  %q\nwant: %q", got, body)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want event stream", got)
	}

	var _ *dto.Usage = usage
}

func TestOpenaiImageStreamHandlerUsesSharedScanner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	body := "data: {\"type\":\"image_generation.partial_image\",\"usage\":{\"input_tokens\":2,\"output_tokens\":3}}\n\ndata: [DONE]\n\n"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream; charset=utf-8"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	info := &relaycommon.RelayInfo{}
	usage, apiErr := OpenaiImageStreamHandler(c, info, resp)
	if apiErr != nil {
		t.Fatalf("OpenaiImageStreamHandler returned error: %v", apiErr)
	}
	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if usage.PromptTokens != 2 || usage.CompletionTokens != 3 || usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage: %#v", usage)
	}

	got := recorder.Body.String()
	if !strings.Contains(got, "event: image_generation.partial_image") {
		t.Fatalf("expected image event in body, got %q", got)
	}
	if !strings.Contains(got, "data: {\"type\":\"image_generation.partial_image\"") {
		t.Fatalf("expected image data in body, got %q", got)
	}
	if !strings.Contains(got, "data: [DONE]") {
		t.Fatalf("expected DONE marker in body, got %q", got)
	}
	if info.StreamStatus == nil || info.StreamStatus.EndReason != relaycommon.StreamEndReasonDone {
		t.Fatalf("expected done stream status, got %#v", info.StreamStatus)
	}
}

func TestOpenaiImageJSONAsStreamHandlerWrapsJSONResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(`{"created":1710000000,"data":[{"url":"https://example.com/a.png"}],"usage":{"input_tokens":12,"output_tokens":34}}`)),
	}

	usage, apiErr := OpenaiImageJSONAsStreamHandler(c, &relaycommon.RelayInfo{}, resp)
	if apiErr != nil {
		t.Fatalf("OpenaiImageJSONAsStreamHandler returned error: %v", apiErr)
	}
	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if usage.PromptTokens != 12 || usage.CompletionTokens != 34 || usage.TotalTokens != 46 {
		t.Fatalf("unexpected usage: %#v", usage)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "event: image_generation.completed") {
		t.Fatalf("expected image event in body, got %q", body)
	}
	if !strings.Contains(body, "\"url\":\"https://example.com/a.png\"") {
		t.Fatalf("expected image url in body, got %q", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("expected DONE marker in body, got %q", body)
	}
}
