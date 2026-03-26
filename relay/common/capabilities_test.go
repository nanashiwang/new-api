package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
)

func TestResolveChannelCapabilitiesOpenAIOfficialHost(t *testing.T) {
	caps := ResolveChannelCapabilities(constant.ChannelTypeOpenAI, "https://api.openai.com/v1", dto.ChannelOtherSettings{})

	if !caps.SupportsChatStreamOptions || !caps.SupportsResponsesAPI || !caps.SupportsResponsesStreamOptions {
		t.Fatalf("expected official openai host to support advanced capabilities, got %+v", caps)
	}
}

func TestResolveChannelCapabilitiesOpenAICompatibleHostDefaultsConservative(t *testing.T) {
	caps := ResolveChannelCapabilities(constant.ChannelTypeOpenAI, "https://nan.meta-api.vip/v1", dto.ChannelOtherSettings{})

	if caps.SupportsChatStreamOptions || caps.SupportsResponsesAPI || caps.SupportsResponsesStreamOptions {
		t.Fatalf("expected unknown openai-compatible host to default conservative, got %+v", caps)
	}
}

func TestResolveChannelCapabilitiesHonorsOverrides(t *testing.T) {
	caps := ResolveChannelCapabilities(constant.ChannelTypeOpenAI, "https://nan.meta-api.vip/v1", dto.ChannelOtherSettings{
		ChatStreamOptionsMode:      dto.CapabilityModeEnabled,
		ResponsesAPIMode:           dto.CapabilityModeEnabled,
		ResponsesStreamOptionsMode: dto.CapabilityModeEnabled,
	})

	if !caps.SupportsChatStreamOptions || !caps.SupportsResponsesAPI || !caps.SupportsResponsesStreamOptions {
		t.Fatalf("expected overrides to force-enable capabilities, got %+v", caps)
	}
}

func TestResolveChannelCapabilitiesResponsesStreamRequiresResponsesAPI(t *testing.T) {
	caps := ResolveChannelCapabilities(constant.ChannelTypeAzure, "https://example.openai.azure.com", dto.ChannelOtherSettings{
		ResponsesAPIMode:           dto.CapabilityModeDisabled,
		ResponsesStreamOptionsMode: dto.CapabilityModeEnabled,
	})

	if caps.SupportsResponsesAPI {
		t.Fatalf("expected responses api to be disabled, got %+v", caps)
	}
	if caps.SupportsResponsesStreamOptions {
		t.Fatalf("expected responses stream options to be disabled with responses api, got %+v", caps)
	}
}
