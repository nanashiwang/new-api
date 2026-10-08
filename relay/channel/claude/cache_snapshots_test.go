package claude

import (
	"fmt"
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestClaudeCacheSnapshotsPreserveMissingAndRespectZero(t *testing.T) {
	state := &ClaudeResponseInfo{Usage: &dto.Usage{}}
	merge := func(raw string) {
		var u dto.ClaudeUsage
		require.NoError(t, common.UnmarshalJsonStr(raw, &u))
		mergeClaudeUsage(state, &u, true)
		require.NoError(t, state.UsageError)
	}
	merge(`{"cache_read_input_tokens":100,"cache_creation_input_tokens":200,"cache_creation":{"ephemeral_5m_input_tokens":50,"ephemeral_1h_input_tokens":100}}`)
	merge(`{"output_tokens":42}`)
	require.Equal(t, 200, state.Usage.PromptTokensDetails.CachedCreationTokens)
	require.Equal(t, 100, state.Usage.ClaudeCacheCreation1hTokens)
	merge(`{"cache_creation_input_tokens":200}`)
	require.Equal(t, 100, state.Usage.ClaudeCacheCreation1hTokens)
	merge(`{"cache_creation":{"ephemeral_5m_input_tokens":0}}`)
	require.Zero(t, state.Usage.ClaudeCacheCreation5mTokens)
	require.Equal(t, 100, state.Usage.ClaudeCacheCreation1hTokens, "omitted TTL counter retains the prior snapshot")
	merge(`{"cache_read_input_tokens":0,"cache_creation_input_tokens":0}`)
	require.Zero(t, state.Usage.PromptTokensDetails.CachedTokens)
	require.Zero(t, state.Usage.PromptTokensDetails.CachedCreationTokens)
	require.Zero(t, state.Usage.ClaudeCacheCreation1hTokens)
	merge(`{"cache_creation_input_tokens":300}`)
	require.Equal(t, 300, state.Usage.PromptTokensDetails.CachedCreationTokens)
	require.Zero(t, state.Usage.ClaudeCacheCreation1hTokens)
	require.Equal(t, "anthropic", state.Usage.UsageSemantic)

	// Derived totals must update with a later breakdown-only snapshot.
	state = &ClaudeResponseInfo{Usage: &dto.Usage{}}
	merge(`{"cache_creation":{"ephemeral_5m_input_tokens":50,"ephemeral_1h_input_tokens":100}}`)
	merge(`{"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":200}}`)
	require.Equal(t, 200, state.Usage.PromptTokensDetails.CachedCreationTokens)
	require.Zero(t, state.Usage.ClaudeCacheCreation5mTokens)
	merge(`{"cache_creation_input_tokens":50,"cache_creation":{"ephemeral_5m_input_tokens":20}}`)
	require.Equal(t, 50, state.Usage.PromptTokensDetails.CachedCreationTokens)
	require.Equal(t, 20, state.Usage.ClaudeCacheCreation5mTokens)
	require.Zero(t, state.Usage.ClaudeCacheCreation1hTokens, "changed total cannot retain an unreported stale TTL")
}

func TestClaudeCacheInvalidUsageIsSticky(t *testing.T) {
	for _, raw := range []string{
		`{"cache_creation_input_tokens":0,"cache_creation":{"ephemeral_1h_input_tokens":1}}`,
		`{"cache_read_input_tokens":-1}`,
		`{"cache_creation":{"ephemeral_5m_input_tokens":-1}}`,
		fmt.Sprintf(`{"input_tokens":%d,"output_tokens":1}`, math.MaxInt),
	} {
		state := &ClaudeResponseInfo{Usage: &dto.Usage{}}
		var u dto.ClaudeUsage
		require.NoError(t, common.UnmarshalJsonStr(raw, &u))
		mergeClaudeUsage(state, &u, true)
		require.Error(t, state.UsageError)
		mergeClaudeUsage(state, &dto.ClaudeUsage{InputTokens: 1}, true)
		require.Error(t, state.UsageError, "later valid frame must not erase an earlier billing conflict")
	}
}
