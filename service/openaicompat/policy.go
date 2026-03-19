package openaicompat

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	if !policy.IsChannelEnabled(channelID, channelType) {
		return false
	}
	return matchAnyRegex(policy.ModelPatterns, model)
}

func ShouldChatCompletionsUseResponsesByChannelSetting(setting dto.ChannelSettings, policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	switch setting.GetChatCompletionsToResponsesMode() {
	case dto.ChatCompletionsToResponsesModeEnabled:
		return true
	case dto.ChatCompletionsToResponsesModeDisabled:
		return false
	default:
		return ShouldChatCompletionsUseResponsesPolicy(policy, channelID, channelType, model)
	}
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return ShouldChatCompletionsUseResponsesPolicy(
		model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy,
		channelID,
		channelType,
		model,
	)
}

func ShouldChatCompletionsUseResponsesWithChannelSetting(setting dto.ChannelSettings, channelID int, channelType int, model string) bool {
	return ShouldChatCompletionsUseResponsesByChannelSetting(
		setting,
		model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy,
		channelID,
		channelType,
		model,
	)
}
