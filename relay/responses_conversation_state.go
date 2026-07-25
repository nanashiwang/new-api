package relay

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type responsesConversationStateRecovery struct {
	attempted bool
}

// prepare removes one explicitly rejected, reference-only reasoning item. It
// never rewrites pass-through payloads or requests without replayable user text.
func (r *responsesConversationStateRecovery) prepare(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	request *dto.OpenAIResponsesRequest,
	originalRequest *dto.OpenAIResponsesRequest,
	passThrough bool,
	err *types.NewAPIError,
) bool {
	if r == nil || r.attempted || passThrough || c == nil || c.Request == nil ||
		info == nil || info.RelayMode != relayconstant.RelayModeResponses ||
		request == nil || originalRequest == nil || c.Request.Context().Err() != nil {
		return false
	}

	missingItemID := service.ResponsesConversationStateMissingItemID(err)
	if !strings.HasPrefix(strings.ToLower(missingItemID), "rs_") {
		return false
	}
	recoveredInput, ok := removeMissingReasoningItemReference(request.Input, missingItemID)
	if !ok {
		return false
	}

	r.attempted = true
	request.Input = recoveredInput
	originalRequest.Input = append(originalRequest.Input[:0:0], recoveredInput...)
	return true
}

func removeMissingReasoningItemReference(input json.RawMessage, missingItemID string) (json.RawMessage, bool) {
	if len(input) == 0 || !strings.HasPrefix(strings.ToLower(missingItemID), "rs_") {
		return input, false
	}

	var items []json.RawMessage
	if err := common.Unmarshal(input, &items); err != nil || len(items) < 2 {
		return input, false
	}

	matchIndex := -1
	hasReplayableUserMessage := false
	for index, item := range items {
		var fields map[string]json.RawMessage
		if err := common.Unmarshal(item, &fields); err != nil {
			continue
		}

		itemType := rawJSONString(fields["type"])
		if itemType == "item_reference" {
			if rawJSONString(fields["id"]) != missingItemID {
				continue
			}
			if len(fields) != 2 || matchIndex >= 0 {
				return input, false
			}
			matchIndex = index
			continue
		}

		if (itemType == "" || itemType == "message") &&
			rawJSONString(fields["role"]) == "user" && rawJSONHasContent(fields["content"]) {
			hasReplayableUserMessage = true
		}
	}

	if matchIndex < 0 || !hasReplayableUserMessage {
		return input, false
	}
	remaining := append(append([]json.RawMessage{}, items[:matchIndex]...), items[matchIndex+1:]...)
	recovered, err := common.Marshal(remaining)
	if err != nil {
		return input, false
	}
	return recovered, true
}

func rawJSONString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := common.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func rawJSONHasContent(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return false
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return false
	}
}
