package service

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/types"
)

func TestIsQuotaRelatedErrorByCode(t *testing.T) {
	err := types.NewError(errors.New("insufficient"), types.ErrorCodeInsufficientUserQuota)
	if !IsQuotaRelatedError(err) {
		t.Fatalf("expected quota related error by error code")
	}
}

func TestIsQuotaRelatedErrorByOpenAIType(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "quota exceeded",
		Type:    "insufficient_quota",
		Code:    "insufficient_quota",
	}, 403)
	if !IsQuotaRelatedError(err) {
		t.Fatalf("expected quota related error by openai type/code")
	}
}

func TestIsQuotaRelatedErrorNegative(t *testing.T) {
	err := types.NewError(errors.New("upstream timeout"), types.ErrorCodeDoRequestFailed)
	if IsQuotaRelatedError(err) {
		t.Fatalf("did not expect timeout to be quota related")
	}
}

func TestIsChannelModelMismatchError_CodexUnsupportedModel(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "The 'gpt-5.4' model is not supported when using Codex with a ChatGPT account.",
		Type:    "bad_response_status_code",
		Code:    "bad_response_status_code",
	}, 400)
	if !IsChannelModelMismatchError(err) {
		t.Fatalf("expected codex unsupported model to be treated as channel mismatch")
	}
}

func TestIsChannelModelMismatchError_StreamRequired(t *testing.T) {
	err := types.NewOpenAIError(errors.New("bad response status code 400, message: Stream must be set to true"), types.ErrorCodeBadResponseStatusCode, 400)
	if !IsChannelModelMismatchError(err) {
		t.Fatalf("expected stream-required error to be treated as channel mismatch")
	}
}
