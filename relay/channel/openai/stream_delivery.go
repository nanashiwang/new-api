package openai

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
)

// filterChatStreamUsage keeps billing usage separate from the client's usage
// preference. Never drop content, tool calls or finish_reason in a mixed chunk.
func filterChatStreamUsage(data string, includeUsage bool) (string, error) {
	if includeUsage {
		return data, nil
	}
	var event map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(data, &event); err != nil {
		return "", err
	}
	if _, ok := event["usage"]; !ok {
		return data, nil
	}
	var choices []json.RawMessage
	if err := common.Unmarshal(event["choices"], &choices); err == nil && len(choices) == 0 {
		return "", nil
	}
	delete(event, "usage")
	encoded, err := common.Marshal(event)
	return string(encoded), err
}
