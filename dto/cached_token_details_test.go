package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestCachedTokenDetailsPreservePresenceAndDetach(t *testing.T) {
	var usage Usage
	require.NoError(t, common.Unmarshal([]byte(`{"prompt_tokens_details":{"cached_tokens":3,"cached_tokens_details":{"image_tokens":0,"text_tokens":3}}}`), &usage))
	details := usage.PromptTokensDetails.Clone()
	require.NotNil(t, details.CachedTokensDetails.ImageTokens)
	require.Zero(t, *details.CachedTokensDetails.ImageTokens)
	require.Nil(t, details.CachedTokensDetails.AudioTokens)
	*details.CachedTokensDetails.TextTokens = 99
	require.Equal(t, 3, *usage.PromptTokensDetails.CachedTokensDetails.TextTokens)
	wire := NewOpenAIChatCompletionsUsage(&usage)
	*wire.PromptTokensDetails.CachedTokensDetails.TextTokens = 88
	require.Equal(t, 3, *usage.PromptTokensDetails.CachedTokensDetails.TextTokens)
	var next InputTokenDetails
	require.NoError(t, common.Unmarshal([]byte(`{"cached_tokens":2,"cached_tokens_details":{"audio_tokens":0,"text_tokens":2}}`), &next))
	usage.PromptTokensDetails.Add(next)
	require.Equal(t, 5, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 5, *usage.PromptTokensDetails.CachedTokensDetails.TextTokens)
	require.NotNil(t, usage.PromptTokensDetails.CachedTokensDetails.AudioTokens)
	require.Zero(t, *usage.PromptTokensDetails.CachedTokensDetails.AudioTokens)
	*next.CachedTokensDetails.TextTokens = 77
	require.Equal(t, 5, *usage.PromptTokensDetails.CachedTokensDetails.TextTokens)
}
