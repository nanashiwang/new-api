package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

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
