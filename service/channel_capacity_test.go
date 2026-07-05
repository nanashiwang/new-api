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

func TestIsUpstreamAccountPoolUnavailableError_CRSGroupMessage(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "No available accounts in group Pro",
		Type:    "upstream_error",
		Code:    nil,
	}, 402)
	if !IsUpstreamAccountPoolUnavailableError(err) {
		t.Fatalf("expected CRS account pool exhaustion to be detected")
	}
	if !IsRetryableSharedUpstreamPoolError(err) {
		t.Fatalf("expected CRS account pool exhaustion to be treated as shared upstream pool error")
	}
}
