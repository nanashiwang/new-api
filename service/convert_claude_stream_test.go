package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestStreamResponseOpenAI2Claude_UsageOnlyFinalChunk(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeNone,
		},
	}

	responses := StreamResponseOpenAI2Claude(&dto.ChatCompletionsStreamResponse{
		Usage: &dto.Usage{
			PromptTokens:     100,
			CompletionTokens: 20,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:         30,
				CachedCreationTokens: 50,
			},
			ClaudeCacheCreation5mTokens: 10,
			ClaudeCacheCreation1hTokens: 20,
		},
	}, info)

	require.Len(t, responses, 2)
	require.Equal(t, "message_delta", responses[0].Type)
	require.Equal(t, "message_stop", responses[1].Type)
	require.NotNil(t, responses[0].Usage)
	require.NotNil(t, responses[0].Usage.CacheCreation)
	require.NotNil(t, responses[0].Delta)
	require.NotNil(t, responses[0].Delta.StopReason)
	require.Equal(t, "end_turn", *responses[0].Delta.StopReason)
	require.Equal(t, 30, responses[0].Usage.CacheCreation.Ephemeral5mInputTokens)
	require.Equal(t, 20, responses[0].Usage.CacheCreation.Ephemeral1hInputTokens)
	require.True(t, info.ClaudeConvertInfo.Done)
}

func TestStreamResponseOpenAI2Claude_DefersFinalCloseUntilUsageArrives(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeNone,
		},
	}
	finishReason := "stop"

	responses := StreamResponseOpenAI2Claude(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{FinishReason: &finishReason},
		},
	}, info)

	require.Empty(t, responses)
	require.Equal(t, finishReason, info.FinishReason)
	require.False(t, info.ClaudeConvertInfo.Done)
}
