package service

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
)

func TestIsUpstreamModelTemporaryUnavailableError_CapacityMessage(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "Selected model is at capacity.Please try a different model.",
		Type:    "api_error",
		Code:    nil,
	}, 529)
	if !IsUpstreamModelTemporaryUnavailableError(err) {
		t.Fatalf("expected capacity message to be treated as temporary upstream model unavailability")
	}
}

func TestIsRetryableSharedUpstreamPoolError_CapacityMessage(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "Selected model is at capacity.Please try a different model.",
		Type:    "api_error",
		Code:    nil,
	}, 529)
	if !IsRetryableSharedUpstreamPoolError(err) {
		t.Fatalf("expected capacity message to be treated as shared upstream pool error")
	}
}
