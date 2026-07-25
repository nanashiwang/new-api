package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIsResponsesConversationStateErrorUsesNarrowSignatures(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		want       bool
	}{
		{
			name:       "non-persisted item",
			statusCode: http.StatusNotFound,
			message:    "Item with id 'rs_missing' not found. Items are not persisted when `store` is set to false.",
			want:       true,
		},
		{
			name:       "missing previous response",
			statusCode: http.StatusBadRequest,
			message:    "previous_response_id does not exist",
			want:       true,
		},
		{
			name:       "generic 404",
			statusCode: http.StatusNotFound,
			message:    "route not found",
			want:       false,
		},
		{
			name:       "generic item not found",
			statusCode: http.StatusNotFound,
			message:    "Item with id 'file_missing' not found",
			want:       false,
		},
		{
			name:       "same words on server failure",
			statusCode: http.StatusInternalServerError,
			message:    "Item with id 'rs_missing' not found. Items are not persisted when store is set to false.",
			want:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := types.WithOpenAIError(types.OpenAIError{
				Message: test.message,
				Type:    "invalid_request_error",
				Code:    "invalid_request",
			}, test.statusCode)
			require.Equal(t, test.want, IsResponsesConversationStateError(err))
		})
	}
}

func TestNormalizeResponsesConversationStateErrorPreservesUpstreamPayload(t *testing.T) {
	original := types.WithOpenAIError(types.OpenAIError{
		Message: "Item with id 'rs_missing' not found. Items are not persisted when store is set to false.",
		Type:    "invalid_request_error",
		Code:    "invalid_request",
		Param:   "input",
	}, http.StatusNotFound)
	original.Upstream = &types.UpstreamDiagnostics{RequestID: "upstream-request"}

	normalized := NormalizeResponsesConversationStateError(original)
	require.NotSame(t, original, normalized)
	require.True(t, types.IsSkipRetryError(normalized))
	require.Equal(t, types.ErrorCodeConversationStateNotFound, normalized.GetErrorCode())
	require.Equal(t, http.StatusNotFound, normalized.StatusCode)
	require.Equal(t, original.ToOpenAIError(), normalized.ToOpenAIError())
	require.Equal(t, "upstream-request", normalized.Upstream.RequestID)
	require.Equal(t, "rs_missing", ResponsesConversationStateMissingItemID(normalized))
}

func TestResponsesConversationStateErrorNeverRetriesOrDisablesChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	err := NormalizeResponsesConversationStateError(types.WithOpenAIError(types.OpenAIError{
		Message: "previous_response_id 'resp_missing' not found",
		Type:    "invalid_request_error",
		Code:    "invalid_request",
	}, http.StatusNotFound))

	require.False(t, ShouldRetryChannelError(c, err, 3))
	originalAutoDisable := common.AutomaticDisableChannelEnabled
	common.AutomaticDisableChannelEnabled = true
	t.Cleanup(func() { common.AutomaticDisableChannelEnabled = originalAutoDisable })
	require.False(t, ShouldDisableChannel(0, err))
}

func TestNormalizeResponsesConversationStateErrorLeavesGeneric404Untouched(t *testing.T) {
	original := types.WithOpenAIError(types.OpenAIError{
		Message: "route not found",
		Type:    "upstream_error",
		Code:    "not_found",
	}, http.StatusNotFound)

	require.Same(t, original, NormalizeResponsesConversationStateError(original))
	require.False(t, types.IsSkipRetryError(original))
	require.Empty(t, ResponsesConversationStateMissingItemID(original))
}
