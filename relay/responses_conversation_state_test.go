package relay

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newMissingResponsesItemError(itemID string) *types.NewAPIError {
	return service.NormalizeResponsesConversationStateError(types.WithOpenAIError(types.OpenAIError{
		Message: "Item with id '" + itemID + "' not found. Items are not persisted when store is set to false.",
		Type:    "invalid_request_error",
		Code:    "invalid_request",
	}, http.StatusNotFound))
}

func newResponsesRecoveryContext(t *testing.T) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	return c
}

func TestRemoveMissingReasoningItemReferencePreservesReplayableInput(t *testing.T) {
	input := []byte(`[
		{"type":"message","role":"user","content":"first question"},
		{"type":"item_reference","id":"rs_missing"},
		{"type":"message","role":"assistant","content":"answer"},
		{"type":"message","role":"user","content":[{"type":"input_text","text":"continue"}]}
	]`)

	recovered, ok := removeMissingReasoningItemReference(input, "rs_missing")
	require.True(t, ok)
	require.NotContains(t, string(recovered), `"id":"rs_missing"`)
	require.Contains(t, string(recovered), "first question")
	require.Contains(t, string(recovered), "answer")
	require.Contains(t, string(recovered), "continue")
}

func TestRemoveMissingReasoningItemReferenceRejectsUnsafeShapes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		missing string
	}{
		{
			name:    "id only appears in message text",
			input:   `[{"type":"message","role":"user","content":"rs_missing"}]`,
			missing: "rs_missing",
		},
		{
			name:    "tool call reference",
			input:   `[{"type":"item_reference","id":"fc_missing"},{"type":"message","role":"user","content":"continue"}]`,
			missing: "fc_missing",
		},
		{
			name:    "reference has extra semantics",
			input:   `[{"type":"item_reference","id":"rs_missing","status":"completed"},{"type":"message","role":"user","content":"continue"}]`,
			missing: "rs_missing",
		},
		{
			name:    "only input item",
			input:   `[{"type":"item_reference","id":"rs_missing"}]`,
			missing: "rs_missing",
		},
		{
			name:    "no replayable user message",
			input:   `[{"type":"item_reference","id":"rs_missing"},{"type":"function_call_output","call_id":"call_1","output":"ok"}]`,
			missing: "rs_missing",
		},
		{
			name:    "duplicate matching references",
			input:   `[{"type":"item_reference","id":"rs_missing"},{"type":"item_reference","id":"rs_missing"},{"role":"user","content":"continue"}]`,
			missing: "rs_missing",
		},
		{
			name:    "malformed input",
			input:   `[{"type":"item_reference"`,
			missing: "rs_missing",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recovered, ok := removeMissingReasoningItemReference([]byte(test.input), test.missing)
			require.False(t, ok)
			require.Equal(t, test.input, string(recovered))
		})
	}
}

func TestResponsesConversationStateRecoveryRunsAtMostOnce(t *testing.T) {
	c := newResponsesRecoveryContext(t)
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeResponses}
	originalInput := []byte(`[{"type":"item_reference","id":"rs_missing"},{"role":"user","content":"continue"}]`)
	request := &dto.OpenAIResponsesRequest{Input: append([]byte(nil), originalInput...)}
	originalRequest := &dto.OpenAIResponsesRequest{Input: append([]byte(nil), originalInput...)}
	recovery := responsesConversationStateRecovery{}
	err := newMissingResponsesItemError("rs_missing")

	require.True(t, recovery.prepare(c, info, request, originalRequest, false, err))
	require.NotContains(t, string(request.Input), "rs_missing")
	require.Equal(t, request.Input, originalRequest.Input)

	request.Input = append(request.Input[:0:0], originalInput...)
	originalRequest.Input = append(originalRequest.Input[:0:0], originalInput...)
	require.False(t, recovery.prepare(c, info, request, originalRequest, false, err))
	require.Equal(t, string(originalInput), string(request.Input))
}

func TestResponsesHelperRetriesMissingReasoningReferenceOnSameUpstreamOnce(t *testing.T) {
	service.InitHttpClient()
	var requestsMu sync.Mutex
	requests := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestsMu.Lock()
		requests = append(requests, string(body))
		requestsMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Item with id 'rs_missing' not found. Items are not persisted when store is set to false.","type":"invalid_request_error","code":"invalid_request"}}`))
	}))
	t.Cleanup(server.Close)

	c := newResponsesRecoveryContext(t)
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	common.SetContextKey(c, constant.ContextKeyChannelId, 42)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
	common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "gpt-5")
	request := &dto.OpenAIResponsesRequest{
		Model: "gpt-5",
		Input: []byte(`[{"type":"item_reference","id":"rs_missing"},{"role":"user","content":"continue"}]`),
	}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAI,
		RequestURLPath:  "/v1/responses",
		OriginModelName: "gpt-5",
		Request:         request,
	}

	newAPIError := ResponsesHelper(c, info)
	require.NotNil(t, newAPIError)
	require.True(t, service.IsResponsesConversationStateError(newAPIError))
	require.True(t, types.IsSkipRetryError(newAPIError))
	require.Equal(t, http.StatusNotFound, newAPIError.StatusCode)
	requestsMu.Lock()
	defer requestsMu.Unlock()
	require.Len(t, requests, 2)
	require.Contains(t, requests[0], `"id":"rs_missing"`)
	require.NotContains(t, requests[1], `"id":"rs_missing"`)
}

func TestResponsesConversationStateRecoveryRespectsCancellationAndPassThrough(t *testing.T) {
	input := []byte(`[{"type":"item_reference","id":"rs_missing"},{"role":"user","content":"continue"}]`)
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeResponses}
	err := newMissingResponsesItemError("rs_missing")

	t.Run("client canceled", func(t *testing.T) {
		c := newResponsesRecoveryContext(t)
		ctx, cancel := context.WithCancel(c.Request.Context())
		cancel()
		c.Request = c.Request.WithContext(ctx)
		request := &dto.OpenAIResponsesRequest{Input: append([]byte(nil), input...)}
		originalRequest := &dto.OpenAIResponsesRequest{Input: append([]byte(nil), input...)}
		recovery := responsesConversationStateRecovery{}
		require.False(t, recovery.prepare(c, info, request, originalRequest, false, err))
		require.Equal(t, string(input), string(request.Input))
	})

	t.Run("pass through", func(t *testing.T) {
		c := newResponsesRecoveryContext(t)
		request := &dto.OpenAIResponsesRequest{Input: append([]byte(nil), input...)}
		originalRequest := &dto.OpenAIResponsesRequest{Input: append([]byte(nil), input...)}
		recovery := responsesConversationStateRecovery{}
		require.False(t, recovery.prepare(c, info, request, originalRequest, true, err))
		require.Equal(t, string(input), string(request.Input))
	})
}
