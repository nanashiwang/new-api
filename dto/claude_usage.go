package dto

import "github.com/QuantumNous/new-api/common"

// Preserve omitted versus explicitly zero counters without changing the wire
// format or treating message_start's output estimate as final stream usage.
func (u *ClaudeUsage) UnmarshalJSON(data []byte) error {
	type wireUsage ClaudeUsage
	var decoded struct {
		wireUsage
		InputTokens  *int `json:"input_tokens"`
		OutputTokens *int `json:"output_tokens"`
	}
	if err := common.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*u = ClaudeUsage(decoded.wireUsage)
	if decoded.InputTokens != nil {
		u.InputTokens, u.inputTokensPresent = *decoded.InputTokens, true
	}
	if decoded.OutputTokens != nil {
		u.OutputTokens, u.outputTokensPresent = *decoded.OutputTokens, true
	}
	return nil
}

func (u *ClaudeUsage) HasInputTokens() bool {
	return u != nil && (u.inputTokensPresent || u.InputTokens != 0)
}

func (u *ClaudeUsage) HasOutputTokens() bool {
	return u != nil && (u.outputTokensPresent || u.OutputTokens != 0)
}
