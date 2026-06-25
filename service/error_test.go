package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newAPIError := &types.NewAPIError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(newAPIError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, newAPIError.StatusCode)
		})
	}
}

func TestRelayErrorHandlerAttachesUpstreamDiagnostics(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://crs.example.com/openai/v1/responses?api_key=secret", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.ContentLength = 1114684
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header: http.Header{
			"X-Request-Id": []string{"2hr1mdrh0ni"},
		},
		Body:    io.NopCloser(strings.NewReader(`{"error":{"message":"bad payload","type":"upstream_error","code":null}}`)),
		Request: req,
	}

	apiErr := RelayErrorHandler(context.Background(), resp, false)
	if apiErr == nil || apiErr.Upstream == nil {
		t.Fatalf("expected upstream diagnostics, got %#v", apiErr)
	}
	if apiErr.Upstream.RequestID != "2hr1mdrh0ni" {
		t.Fatalf("request id = %q", apiErr.Upstream.RequestID)
	}
	if apiErr.Upstream.RequestLength != 1114684 {
		t.Fatalf("request length = %d", apiErr.Upstream.RequestLength)
	}
	if strings.Contains(apiErr.Upstream.URL, "secret") {
		t.Fatalf("upstream url leaked query: %s", apiErr.Upstream.URL)
	}
	if !strings.Contains(apiErr.UpstreamDiagnosticsLogString(), "2hr1mdrh0ni") {
		t.Fatalf("diagnostics log missing request id: %s", apiErr.UpstreamDiagnosticsLogString())
	}
}
