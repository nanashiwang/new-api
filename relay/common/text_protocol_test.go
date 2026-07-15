package common

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestResolveTextProtocolPlanPrefersNativeProtocol(t *testing.T) {
	plan, err := ResolveTextProtocolPlan(
		types.RelayFormatOpenAIResponses,
		NewTextProtocolSet(types.RelayFormatOpenAI, types.RelayFormatOpenAIResponses),
	)

	require.NoError(t, err)
	require.False(t, plan.RequiresConversion())
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), plan.UpstreamFormat)
}

func TestResolveTextProtocolPlanUsesRegisteredConversion(t *testing.T) {
	plan, err := ResolveTextProtocolPlan(
		types.RelayFormatClaude,
		NewTextProtocolSet(types.RelayFormatOpenAI),
	)

	require.NoError(t, err)
	require.True(t, plan.RequiresConversion())
	require.Equal(t, TextProtocolConverterClaudeToOpenAIChat, plan.Converter)
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAI), plan.UpstreamFormat)
}

func TestResolveTextProtocolPlanRejectsUnsupportedRoute(t *testing.T) {
	_, err := ResolveTextProtocolPlan(
		types.RelayFormatGemini,
		NewTextProtocolSet(types.RelayFormatClaude),
	)

	require.ErrorContains(t, err, "no safe text protocol route")
}

func TestTextProtocolPlanRejectsPassThroughConversion(t *testing.T) {
	plan := TextProtocolPlan{
		IncomingFormat: types.RelayFormatClaude,
		UpstreamFormat: types.RelayFormatOpenAI,
		Converter:      TextProtocolConverterClaudeToOpenAIChat,
	}

	require.ErrorContains(t, plan.ValidatePassThrough(true), "pass-through request body")
	require.NoError(t, plan.ValidatePassThrough(false))
}

func TestRelayInfoCommitsTextProtocolPlan(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatClaude,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatClaude},
		TextProtocolPlan: &TextProtocolPlan{
			IncomingFormat: types.RelayFormatClaude,
			UpstreamFormat: types.RelayFormatOpenAI,
			Converter:      TextProtocolConverterClaudeToOpenAIChat,
		},
	}

	info.CommitTextProtocolPlan()

	require.Equal(t, []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI}, info.RequestConversionChain)
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAI), info.GetFinalRequestRelayFormat())
}

func TestPrepareTextProtocolPlanClearsPreviousRetryState(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatClaude,
		FinalRequestRelayFormat: types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI},
		TextProtocolPlan: &TextProtocolPlan{
			IncomingFormat: types.RelayFormatClaude,
			UpstreamFormat: types.RelayFormatOpenAI,
			Converter:      TextProtocolConverterClaudeToOpenAIChat,
		},
		ChannelMeta: &ChannelMeta{},
	}

	require.NoError(t, info.PrepareTextProtocolPlan())
	require.Nil(t, info.TextProtocolPlan)
	require.Empty(t, info.FinalRequestRelayFormat)
	require.Equal(t, []types.RelayFormat{types.RelayFormatClaude}, info.RequestConversionChain)
}
