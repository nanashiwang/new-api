package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionsRequestToResponsesRequest_MapsAssistantContentAndFields(t *testing.T) {
	t.Parallel()

	assistant := dto.Message{Role: "assistant"}
	assistant.SetMediaContent([]dto.MediaContent{{Type: dto.ContentTypeText, Text: "done"}})

	req := &dto.GeneralOpenAIRequest{
		Model:                "gpt-5",
		Messages:             []dto.Message{{Role: "system", Content: "be helpful"}, assistant, {Role: "user", Content: "next"}},
		Stream:               common.GetPointer(true),
		StreamOptions:        &dto.StreamOptions{IncludeUsage: true, IncludeObfuscation: true},
		ServiceTier:          "flex",
		SafetyIdentifier:     "user-1",
		PromptCacheKey:       "cache-key",
		PromptCacheRetention: []byte(`{"type":"ephemeral"}`),
		TopLogProbs:          common.GetPointer(5),
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, req.StreamOptions, out.StreamOptions)
	require.Equal(t, "flex", out.ServiceTier)
	require.Equal(t, "user-1", out.SafetyIdentifier)
	require.NotNil(t, out.TopLogProbs)
	require.Equal(t, 5, *out.TopLogProbs)

	var promptCacheKey string
	err = common.Unmarshal(out.PromptCacheKey, &promptCacheKey)
	require.NoError(t, err)
	require.Equal(t, "cache-key", promptCacheKey)
	require.JSONEq(t, `{"type":"ephemeral"}`, string(out.PromptCacheRetention))

	var inputItems []map[string]any
	err = common.Unmarshal(out.Input, &inputItems)
	require.NoError(t, err)
	require.Len(t, inputItems, 2)

	assistantContent, ok := inputItems[0]["content"].([]any)
	require.True(t, ok)
	require.Len(t, assistantContent, 1)

	assistantPart, ok := assistantContent[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "output_text", assistantPart["type"])
	require.Equal(t, "done", assistantPart["text"])
}
