package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func TestResolveResponsesBridgeModeForcedOverrideIgnoresCapabilityDefault(t *testing.T) {
	original := *model_setting.GetGlobalSettings()
	model_setting.GetGlobalSettings().PassThroughRequestEnabled = false
	model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy = model_setting.ChatCompletionsToResponsesPolicy{}
	defer func() {
		*model_setting.GetGlobalSettings() = original
	}()

	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: "gpt-5",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:            1,
			ChannelType:          1,
			SupportsResponsesAPI: false,
			ChannelSetting: dto.ChannelSettings{
				ChatCompletionsToResponsesMode: dto.ChatCompletionsToResponsesModeEnabled,
			},
		},
	}

	if got := resolveResponsesBridgeMode(info); got != responsesBridgeModeForced {
		t.Fatalf("expected forced bridge mode, got %q", got)
	}
	if !shouldUseResponsesBridge(info) {
		t.Fatal("expected chat/completions bridge to remain enabled for explicit mode")
	}
}

func TestResolveResponsesBridgeModeInheritRequiresCapability(t *testing.T) {
	original := *model_setting.GetGlobalSettings()
	model_setting.GetGlobalSettings().PassThroughRequestEnabled = false
	model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		AllChannels:   true,
		ModelPatterns: []string{"^gpt-5$"},
	}
	defer func() {
		*model_setting.GetGlobalSettings() = original
	}()

	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: "gpt-5",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:            1,
			ChannelType:          1,
			SupportsResponsesAPI: false,
			ChannelSetting:       dto.ChannelSettings{},
		},
	}

	if got := resolveResponsesBridgeMode(info); got != responsesBridgeModeDisabled {
		t.Fatalf("expected disabled bridge mode for inherit without capability, got %q", got)
	}
	if shouldUseResponsesBridge(info) {
		t.Fatal("expected chat/completions bridge to stay disabled in inherit mode")
	}
}

func TestResolveResponsesBridgeModeDisabledWins(t *testing.T) {
	original := *model_setting.GetGlobalSettings()
	model_setting.GetGlobalSettings().PassThroughRequestEnabled = false
	model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		AllChannels:   true,
		ModelPatterns: []string{"^gpt-5$"},
	}
	defer func() {
		*model_setting.GetGlobalSettings() = original
	}()

	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: "gpt-5",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:            1,
			ChannelType:          1,
			SupportsResponsesAPI: true,
			ChannelSetting: dto.ChannelSettings{
				ChatCompletionsToResponsesMode: dto.ChatCompletionsToResponsesModeDisabled,
			},
		},
	}

	if got := resolveResponsesBridgeMode(info); got != responsesBridgeModeDisabled {
		t.Fatalf("expected disabled bridge mode, got %q", got)
	}
	if shouldUseResponsesBridgeForClaude(info) {
		t.Fatal("expected claude bridge to stay disabled when channel setting disables it")
	}
}
