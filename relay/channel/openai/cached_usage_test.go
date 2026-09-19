package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestCachedModalityUsageNormalizationAndContinuation(t *testing.T) {
	var input dto.Usage
	require.NoError(t, common.Unmarshal([]byte(`{"input_tokens":10,"input_tokens_details":{"cached_tokens":5,"cached_tokens_details":{"text_tokens":5,"image_tokens":0}}}`), &input))
	normalizeOpenAIUsage(&input)
	require.Equal(t, 5, *input.PromptTokensDetails.CachedTokensDetails.TextTokens)
	response := &dto.OpenAIResponsesResponse{Usage: &input}
	usage := buildResponsesUsage(nil, nil, response)
	normalized := normalizeResponsesUsage(usage)
	require.NotNil(t, normalized.InputTokensDetails.CachedTokensDetails.ImageTokens)
	*normalized.InputTokensDetails.CachedTokensDetails.TextTokens = 1
	require.Equal(t, 5, *usage.PromptTokensDetails.CachedTokensDetails.TextTokens)
	mergeResponsesStreamUsage(usage, buildResponsesUsage(nil, nil, response))
	require.Equal(t, 10, *usage.PromptTokensDetails.CachedTokensDetails.TextTokens)
	require.Equal(t, 10, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 5, *input.InputTokensDetails.CachedTokensDetails.TextTokens)
}
