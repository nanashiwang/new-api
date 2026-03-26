package relay

import (
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

type responsesBridgeMode string

const (
	responsesBridgeModeDisabled responsesBridgeMode = "disabled"
	responsesBridgeModeAuto     responsesBridgeMode = "auto"
	responsesBridgeModeForced   responsesBridgeMode = "forced"
)

func resolveResponsesBridgeMode(info *relaycommon.RelayInfo) responsesBridgeMode {
	if info == nil {
		return responsesBridgeModeDisabled
	}
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		return responsesBridgeModeDisabled
	}

	switch info.ChannelSetting.GetChatCompletionsToResponsesMode() {
	case dto.ChatCompletionsToResponsesModeEnabled:
		return responsesBridgeModeForced
	case dto.ChatCompletionsToResponsesModeDisabled:
		return responsesBridgeModeDisabled
	}

	if !service.ShouldChatCompletionsUseResponsesWithChannelSetting(info.ChannelSetting, info.ChannelId, info.ChannelType, info.OriginModelName) {
		return responsesBridgeModeDisabled
	}
	if !info.SupportsResponsesAPI {
		return responsesBridgeModeDisabled
	}

	return responsesBridgeModeAuto
}

func shouldUseResponsesBridge(info *relaycommon.RelayInfo) bool {
	if info == nil || info.RelayMode != relayconstant.RelayModeChatCompletions {
		return false
	}
	return resolveResponsesBridgeMode(info) != responsesBridgeModeDisabled
}

func shouldUseResponsesBridgeForClaude(info *relaycommon.RelayInfo) bool {
	return resolveResponsesBridgeMode(info) != responsesBridgeModeDisabled
}
