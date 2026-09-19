package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestClaudeUsageCounterPresenceAndWireCompatibility(t *testing.T) {
	var usage ClaudeUsage
	require.NoError(t, common.UnmarshalJsonStr(`{"input_tokens":0,"output_tokens":0,"cache_read_input_tokens":30,"cache_creation":{"ephemeral_1h_input_tokens":15}}`, &usage))
	require.True(t, usage.HasInputTokens())
	require.True(t, usage.HasOutputTokens())
	require.Equal(t, 30, usage.CacheReadInputTokens)
	require.Equal(t, 15, usage.GetCacheCreation1hTokens())
	wire, err := common.Marshal(usage)
	require.NoError(t, err)
	require.NotContains(t, string(wire), "Present")
	require.Contains(t, string(wire), `"output_tokens":0`)
	require.NoError(t, common.UnmarshalJsonStr(`{"output_tokens":5}`, &usage))
	require.False(t, usage.HasInputTokens(), "reusing a DTO must not retain presence from the previous frame")
	require.True(t, usage.HasOutputTokens())
	require.Zero(t, usage.CacheReadInputTokens)
	require.NoError(t, common.UnmarshalJsonStr(`{"input_tokens":null,"output_tokens":null}`, &usage))
	require.False(t, usage.HasInputTokens())
	require.False(t, usage.HasOutputTokens())
	require.Error(t, common.UnmarshalJsonStr(`{"output_tokens":"not-a-number"}`, &usage))
}
