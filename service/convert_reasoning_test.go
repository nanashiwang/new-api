package service

import (
	"encoding/json"
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
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
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
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
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
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenRouter},
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
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
		OriginModelName: "gpt-5-thinking",
	})
	require.NoError(t, err)
	require.Equal(t, "claude-3-7-sonnet-thinking", request.Model)
	require.Equal(t, "medium", request.ReasoningEffort)
}

func TestGetClaudeToOpenAIReasoningMap_FallsBackToDefaultsOnInvalidOption(t *testing.T) {
	setResponsesBridgeTestOptions(t, map[string]string{
		claudeToOpenAIReasoningMapOption: `not-json`,
	})

	mapping := getClaudeToOpenAIReasoningMap()
	require.Equal(t, defaultClaudeToOpenAIReasoningMap, mapping)
}

func TestClaudeToOpenAIRequest_MapsClaudeWebSearchToolToWebSearchOptions(t *testing.T) {
	userLocation, err := json.Marshal(map[string]any{
		"approximate": map[string]any{
			"timezone": "Asia/Shanghai",
			"country":  "CN",
			"city":     "Shanghai",
		},
	})
	require.NoError(t, err)

	request, err := ClaudeToOpenAIRequest(nil, dto.ClaudeRequest{
		Model: "claude-3-7-sonnet",
		Tools: []any{
			dto.ClaudeWebSearchTool{
				Type:    dto.ClaudeWebSearchTool20250305,
				Name:    "web_search",
				MaxUses: 10,
				UserLocation: &dto.ClaudeWebSearchUserLocation{
					Type:     "approximate",
					Timezone: "Asia/Shanghai",
					Country:  "CN",
					City:     "Shanghai",
				},
			},
		},
	}, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
	})
	require.NoError(t, err)
	require.NotNil(t, request.WebSearchOptions)
	require.Equal(t, "high", request.WebSearchOptions.SearchContextSize)
	require.JSONEq(t, string(userLocation), string(request.WebSearchOptions.UserLocation))
}

func TestClaudeToOpenAIRequest_ParsesClaudeWebSearchToolFromRawMap(t *testing.T) {
	request, err := ClaudeToOpenAIRequest(nil, dto.ClaudeRequest{
		Model: "claude-3-7-sonnet",
		Tools: []any{
			map[string]any{
				"type":     dto.ClaudeWebSearchTool20260209,
				"name":     "web_search",
				"max_uses": 1,
			},
		},
	}, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
	})
	require.NoError(t, err)
	require.NotNil(t, request.WebSearchOptions)
	require.Equal(t, "low", request.WebSearchOptions.SearchContextSize)
}

func TestBuildClaudeUsageFromOpenAIUsage_MapsWebSearchRequests(t *testing.T) {
	usage := buildClaudeUsageFromOpenAIUsage(&dto.Usage{
		PromptTokens:      12,
		CompletionTokens:  34,
		WebSearchRequests: 2,
	})
	require.NotNil(t, usage)
	require.NotNil(t, usage.ServerToolUse)
	require.Equal(t, 2, usage.ServerToolUse.WebSearchRequests)
}
