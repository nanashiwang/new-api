package relay

import (
	"io"
	"net/http"
	"strings"
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func TestClassifyUpstreamCompatibilityIssueStreamOptions(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Unsupported parameter: stream_options"}}`)),
	}

	issue := classifyUpstreamCompatibilityIssue(resp, relayconstant.RelayModeChatCompletions)
	if issue != upstreamCompatIssueStreamOptions {
		t.Fatalf("expected stream_options issue, got %q", issue)
	}
}

func TestClassifyUpstreamCompatibilityIssueResponsesAPI(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"unsupported endpoint /v1/responses, only /v1/chat/completions is supported"}}`)),
	}

	issue := classifyUpstreamCompatibilityIssue(resp, relayconstant.RelayModeResponses)
	if issue != upstreamCompatIssueResponsesAPI {
		t.Fatalf("expected responses api issue, got %q", issue)
	}
}

func TestClassifyUpstreamCompatibilityIssuePreviousResponseID(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Unsupported parameter: previous_response_id"}}`)),
	}

	issue := classifyUpstreamCompatibilityIssue(resp, relayconstant.RelayModeResponses)
	if issue != upstreamCompatIssuePreviousResponseID {
		t.Fatalf("expected previous_response_id issue, got %q", issue)
	}
}

func TestClassifyUpstreamCompatibilityIssuePreservesBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Unsupported parameter: stream_options"}}`)),
	}

	_ = classifyUpstreamCompatibilityIssue(resp, relayconstant.RelayModeChatCompletions)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body after classify failed: %v", err)
	}
	if !strings.Contains(string(body), "stream_options") {
		t.Fatalf("expected response body to remain readable, got %s", string(body))
	}
}
