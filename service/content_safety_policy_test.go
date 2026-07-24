package service

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIsContentSafetyPolicyErrorUsesExactAllowlist(t *testing.T) {
	tests := []struct {
		name string
		code types.ErrorCode
		typ  string
		want bool
	}{
		{name: "cyber policy", code: "cyber_policy", typ: "invalid_request", want: true},
		{name: "content filter in type", code: "unknown_error", typ: "content_filter", want: true},
		{name: "policy violation", code: "policy_violation", typ: "invalid_request", want: true},
		{name: "context length", code: "context_length_exceeded", typ: "invalid_request_error", want: false},
		{name: "rate limit", code: "rate_limit_exceeded", typ: "rate_limit_error", want: false},
		{name: "server error", code: "server_error", typ: "server_error", want: false},
		{name: "substring is not enough", code: "not_cyber_policy_related", typ: "invalid_request", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := types.WithOpenAIError(types.OpenAIError{
				Message: "upstream rejected request", Type: test.typ, Code: string(test.code),
			}, http.StatusBadRequest)
			require.Equal(t, test.want, IsContentSafetyPolicyError(err))
		})
	}
}

func TestNormalizeContentSafetyPolicyErrorPreservesErrorAndSkipsRetry(t *testing.T) {
	original := types.WithOpenAIError(types.OpenAIError{
		Message: "request rejected by policy", Type: "invalid_request", Code: "cyber_policy",
	}, http.StatusBadRequest)
	original.Upstream = &types.UpstreamDiagnostics{RequestID: "upstream-request"}

	normalized := NormalizeContentSafetyPolicyError(original)
	require.True(t, types.IsSkipRetryError(normalized))
	require.Equal(t, types.ErrorCode("cyber_policy"), normalized.GetErrorCode())
	require.Equal(t, "request rejected by policy", normalized.ToOpenAIError().Message)
	require.Equal(t, "upstream-request", normalized.Upstream.RequestID)

	contextLimit := types.WithOpenAIError(types.OpenAIError{
		Message: "too long", Type: "invalid_request_error", Code: "context_length_exceeded",
	}, http.StatusBadRequest)
	require.Same(t, contextLimit, NormalizeContentSafetyPolicyError(contextLimit))
}

func TestHashContentSafetyRequestDoesNotChangeStoragePosition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	storage, err := common.CreateBodyStorage([]byte(`{"model":"gpt-5.6-sol","input":"sensitive user content"}`))
	require.NoError(t, err)
	t.Cleanup(func() { _ = storage.Close() })
	c.Set(common.KeyBodyStorage, storage)
	_, err = storage.Seek(7, io.SeekStart)
	require.NoError(t, err)

	hash, err := hashContentSafetyRequest(c)
	require.NoError(t, err)
	require.Len(t, hash, 64)
	position, err := storage.Seek(0, io.SeekCurrent)
	require.NoError(t, err)
	require.EqualValues(t, 7, position)
	require.NotContains(t, hash, "sensitive")
}

func TestNormalizeContentSafetyPolicyErrorHandlesNil(t *testing.T) {
	require.Nil(t, NormalizeContentSafetyPolicyError(nil))
	require.False(t, IsContentSafetyPolicyError(types.NewError(errors.New("network reset"), types.ErrorCodeDoRequestFailed)))
}
