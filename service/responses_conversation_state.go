package service

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/types"
)

var missingResponsesItemPattern = regexp.MustCompile(`(?i)item with id\s+['"\x60]([a-z0-9_-]+)['"\x60]\s+(?:was\s+)?not found`)

// IsResponsesConversationStateError identifies request-owned Responses state
// failures. Generic 404s remain upstream/channel errors.
func IsResponsesConversationStateError(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	if err.GetErrorCode() == types.ErrorCodeConversationStateNotFound {
		return true
	}
	if err.StatusCode != http.StatusBadRequest && err.StatusCode != http.StatusNotFound {
		return false
	}

	message := strings.ToLower(strings.Join(strings.Fields(err.Error()), " "))
	if message == "" {
		message = strings.ToLower(strings.Join(strings.Fields(err.ToOpenAIError().Message), " "))
	}
	if strings.Contains(message, "previous_response_id") &&
		(strings.Contains(message, "not found") || strings.Contains(message, "does not exist")) {
		return true
	}

	return missingResponsesItemPattern.MatchString(message) &&
		strings.Contains(message, "items are not persisted") &&
		strings.Contains(message, "store") &&
		strings.Contains(message, "false")
}

// NormalizeResponsesConversationStateError keeps the upstream payload and
// status while marking the failure as terminal for this request.
func NormalizeResponsesConversationStateError(err *types.NewAPIError) *types.NewAPIError {
	if !IsResponsesConversationStateError(err) {
		return err
	}
	if err.GetErrorCode() == types.ErrorCodeConversationStateNotFound && types.IsSkipRetryError(err) {
		return err
	}

	normalized := types.WithOpenAIError(
		err.ToOpenAIError(),
		err.StatusCode,
		types.ErrOptionWithSkipRetry(),
		types.ErrOptionWithInternalErrorCode(types.ErrorCodeConversationStateNotFound),
	)
	normalized.Upstream = err.Upstream
	return normalized
}

// ResponsesConversationStateMissingItemID returns only an explicitly rejected
// Responses item ID. previous_response_id errors intentionally return empty.
func ResponsesConversationStateMissingItemID(err *types.NewAPIError) string {
	if !IsResponsesConversationStateError(err) {
		return ""
	}
	matches := missingResponsesItemPattern.FindStringSubmatch(err.Error())
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}
