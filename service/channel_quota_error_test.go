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
