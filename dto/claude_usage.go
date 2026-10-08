package dto

import "github.com/QuantumNous/new-api/common"

// Preserve omitted versus explicitly zero counters without changing the wire
// format or treating message_start's output estimate as final stream usage.
func (u *ClaudeUsage) UnmarshalJSON(data []byte) error {
	type wireUsage ClaudeUsage
	var decoded struct {
		wireUsage
		InputTokens         *int `json:"input_tokens"`
		OutputTokens        *int `json:"output_tokens"`
		CacheReadTokens     *int `json:"cache_read_input_tokens"`
		CacheCreationTokens *int `json:"cache_creation_input_tokens"`
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
	if decoded.CacheReadTokens != nil {
		u.CacheReadInputTokens, u.cacheReadPresent = *decoded.CacheReadTokens, true
	}
	if decoded.CacheCreationTokens != nil {
		u.CacheCreationInputTokens, u.cacheCreationPresent = *decoded.CacheCreationTokens, true
	}
	return nil
}

func (u *ClaudeUsage) HasCacheReadTokens() bool {
	return u != nil && (u.cacheReadPresent || u.CacheReadInputTokens != 0)
}

func (u *ClaudeUsage) HasCacheCreationTokens() bool {
	return u != nil && (u.cacheCreationPresent || u.CacheCreationInputTokens != 0)
}

func (u *ClaudeUsage) HasInputTokens() bool {
	return u != nil && (u.inputTokensPresent || u.InputTokens != 0)
}

func (u *ClaudeUsage) HasOutputTokens() bool {
	return u != nil && (u.outputTokensPresent || u.OutputTokens != 0)
}

func (u *ClaudeCacheCreationUsage) UnmarshalJSON(data []byte) error {
	var wire struct {
		FiveMinute *int `json:"ephemeral_5m_input_tokens"`
		OneHour    *int `json:"ephemeral_1h_input_tokens"`
	}
	if err := common.Unmarshal(data, &wire); err != nil {
		return err
	}
	*u = ClaudeCacheCreationUsage{}
	if wire.FiveMinute != nil {
		u.Ephemeral5mInputTokens, u.fiveMinutePresent = *wire.FiveMinute, true
	}
	if wire.OneHour != nil {
		u.Ephemeral1hInputTokens, u.oneHourPresent = *wire.OneHour, true
	}
	return nil
}

func (u *ClaudeCacheCreationUsage) HasFiveMinute() bool {
	return u != nil && (u.fiveMinutePresent || u.Ephemeral5mInputTokens != 0)
}

func (u *ClaudeCacheCreationUsage) HasOneHour() bool {
	return u != nil && (u.oneHourPresent || u.Ephemeral1hInputTokens != 0)
}
