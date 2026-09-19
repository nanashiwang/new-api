package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestGroupHealthCacheUsageNormalization(t *testing.T) {
	for _, tc := range []struct {
		name   string
		usage  *dto.Usage
		format types.RelayFormat
		mode   int
		want   *relaycommon.GroupHealthCacheUsage
	}{
		{name: "openai total includes cache", usage: &dto.Usage{PromptTokens: 100, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 80}}, want: &relaycommon.GroupHealthCacheUsage{ReadTokens: 80, InputTokens: 100}},
		{name: "zero cache is a real sample", usage: &dto.Usage{PromptTokens: 110}, want: &relaycommon.GroupHealthCacheUsage{InputTokens: 110}},
		{name: "anthropic adds read and creation once", format: types.RelayFormatClaude, usage: &dto.Usage{PromptTokens: 10, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 30, CachedCreationTokens: 20}, ClaudeCacheCreation5mTokens: 5, ClaudeCacheCreation1hTokens: 15}, want: &relaycommon.GroupHealthCacheUsage{ReadTokens: 30, InputTokens: 60}},
		{name: "explicit anthropic semantic with split creation", usage: &dto.Usage{UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 30}, ClaudeCacheCreation5mTokens: 5, ClaudeCacheCreation1hTokens: 15}, want: &relaycommon.GroupHealthCacheUsage{ReadTokens: 30, InputTokens: 50}},
		{name: "responses input details", usage: &dto.Usage{InputTokens: 100, InputTokensDetails: &dto.InputTokenDetails{CachedTokens: 40}}, want: &relaycommon.GroupHealthCacheUsage{ReadTokens: 40, InputTokens: 100}},
		{name: "legacy cached field", usage: &dto.Usage{PromptTokens: 100, PromptCacheHitTokens: 40}, want: &relaycommon.GroupHealthCacheUsage{ReadTokens: 40, InputTokens: 100}},
		{name: "missing usage"},
		{name: "zero input", usage: &dto.Usage{CompletionTokens: 100}},
		{name: "estimated input", usage: &dto.Usage{PromptTokens: 100, InputTokensEstimated: true}},
		{name: "invalid negative cache", usage: &dto.Usage{PromptTokens: 100, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: -1}}},
		{name: "cache exceeds input", usage: &dto.Usage{PromptTokens: 10, PromptCacheHitTokens: 20}},
		{name: "image billing units", mode: relayconstant.RelayModeImagesGenerations, usage: &dto.Usage{PromptTokens: 1}},
		{name: "audio billing units", mode: relayconstant.RelayModeAudioSpeech, usage: &dto.Usage{PromptTokens: 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{RelayFormat: tc.format, RelayMode: tc.mode, GroupHealthCacheUsage: &relaycommon.GroupHealthCacheUsage{InputTokens: 999}}
			ObserveGroupHealthCacheUsage(info, tc.usage)
			require.Equal(t, tc.want, info.GroupHealthCacheUsage)
		})
	}
}
