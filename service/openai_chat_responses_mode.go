package service

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service/openaicompat"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	return openaicompat.ShouldChatCompletionsUseResponsesPolicy(policy, channelID, channelType, model)
}

func ShouldChatCompletionsUseResponsesByChannelSetting(setting dto.ChannelSettings, policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	return openaicompat.ShouldChatCompletionsUseResponsesByChannelSetting(setting, policy, channelID, channelType, model)
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return openaicompat.ShouldChatCompletionsUseResponsesGlobal(channelID, channelType, model)
}

func ShouldChatCompletionsUseResponsesWithChannelSetting(setting dto.ChannelSettings, channelID int, channelType int, model string) bool {
	return openaicompat.ShouldChatCompletionsUseResponsesWithChannelSetting(setting, channelID, channelType, model)
}
