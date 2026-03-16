package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestClaudeToOpenAIRequest_MapsThinkingBudgetToReasoningEffort(t *testing.T) {
	setResponsesBridgeTestOptions(t, map[string]string{
		claudeToOpenAIReasoningMapOption: `{"low":"minimal","medium":"low","high":"high","max":"xhigh"}`,
	})

	request, err := ClaudeToOpenAIRequest(nil, dto.ClaudeRequest{
		Model: "claude-3-7-sonnet",
		Thinking: &dto.Thinking{
			Type:         "enabled",
			BudgetTokens: common.GetPointer(1280),
		},
	}, &relaycommon.RelayInfo{
		ChannelType:     constant.ChannelTypeOpenAI,
		OriginModelName: "gpt-5",
	})
	require.NoError(t, err)
	require.Equal(t, "minimal", request.ReasoningEffort)
}

func TestClaudeToOpenAIRequest_UsesMaxBucketForLargeThinkingBudget(t *testing.T) {
	setResponsesBridgeTestOptions(t, map[string]string{
		claudeToOpenAIReasoningMapOption: `{"low":"minimal","medium":"low","high":"medium","max":"xhigh"}`,
	})

	request, err := ClaudeToOpenAIRequest(nil, dto.ClaudeRequest{
		Model: "claude-3-7-sonnet",
		Thinking: &dto.Thinking{
			Type:         "enabled",
			BudgetTokens: common.GetPointer(6000),
		},
	}, &relaycommon.RelayInfo{
		ChannelType:     constant.ChannelTypeOpenAI,
		OriginModelName: "gpt-5",
	})
	require.NoError(t, err)
	require.Equal(t, "xhigh", request.ReasoningEffort)
}

func TestClaudeToOpenAIRequest_PreservesOpenRouterReasoningPayload(t *testing.T) {
	request, err := ClaudeToOpenAIRequest(nil, dto.ClaudeRequest{
		Model: "claude-3-7-sonnet",
		Thinking: &dto.Thinking{
			Type:         "enabled",
			BudgetTokens: common.GetPointer(2048),
		},
	}, &relaycommon.RelayInfo{
		ChannelType:     constant.ChannelTypeOpenRouter,
		OriginModelName: "gpt-5",
	})
	require.NoError(t, err)
	require.NotEmpty(t, request.Reasoning)
	require.Empty(t, request.ReasoningEffort)
}

func TestClaudeToOpenAIRequest_AddsThinkingSuffixAlongsideReasoningEffort(t *testing.T) {
	setResponsesBridgeTestOptions(t, map[string]string{
		claudeToOpenAIReasoningMapOption: `{"low":"minimal","medium":"medium","high":"high","max":"xhigh"}`,
	})

	request, err := ClaudeToOpenAIRequest(nil, dto.ClaudeRequest{
		Model: "claude-3-7-sonnet",
		Thinking: &dto.Thinking{
			Type:         "enabled",
			BudgetTokens: common.GetPointer(2048),
		},
	}, &relaycommon.RelayInfo{
		ChannelType:     constant.ChannelTypeOpenAI,
		OriginModelName: "gpt-5-thinking",
	})
	require.NoError(t, err)
	require.Equal(t, "gpt-5-thinking", request.Model)
	require.Equal(t, "medium", request.ReasoningEffort)
}

func TestGetClaudeToOpenAIReasoningMap_FallsBackToDefaultsOnInvalidOption(t *testing.T) {
	setResponsesBridgeTestOptions(t, map[string]string{
		claudeToOpenAIReasoningMapOption: `not-json`,
	})

	mapping := getClaudeToOpenAIReasoningMap()
	require.Equal(t, defaultClaudeToOpenAIReasoningMap, mapping)
}
