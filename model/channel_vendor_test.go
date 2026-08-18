package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestChannelMatchesVendor_MiMoBoundaryRules(t *testing.T) {
	tests := []struct {
		name   string
		models string
		match  bool
	}{
		{name: "canonical prefix", models: "mimo-v2.5", match: true},
		{name: "xiaomi namespace", models: "xiaomi/mimo-v2.5-pro", match: true},
		{name: "mixed model set", models: "gpt-4o, vendor.xiaomi-mimo ,claude-3", match: true},
		{name: "mimosa is unrelated", models: "mimosa", match: false},
		{name: "embedded mimo is unrelated", models: "notmimo-model", match: false},
		{name: "empty model set", models: "", match: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &Channel{Models: test.models, Type: 1}
			require.Equal(t, test.match, ChannelMatchesVendor(channel, ChannelVendorMiMo))
		})
	}
}

func TestCountAndFilterChannelsByVendor_DeduplicatesMultiModelChannels(t *testing.T) {
	channels := []*Channel{
		{Id: 1, Type: 1, Models: "mimo-v2.5,mimo-v2.5-pro"},
		{Id: 2, Type: 1, Models: "mimosa,gpt-4o"},
		{Id: 3, Type: 2, Models: "xiaomi/mimo-v2.5"},
		nil,
	}

	counts := CountChannelVendors(channels)
	require.EqualValues(t, 3, counts[ChannelVendorAll])
	require.EqualValues(t, 2, counts[ChannelVendorMiMo])

	filtered := FilterChannelsByVendor(channels, "小米 MiMo")
	require.Len(t, filtered, 2)
	require.Equal(t, []int{1, 3}, []int{filtered[0].Id, filtered[1].Id})

	typeCounts := CountChannelTypes(filtered)
	require.EqualValues(t, 1, typeCounts[1])
	require.EqualValues(t, 1, typeCounts[2])
}

func TestDisplayTypesSeparateMiMoFromOpenAI(t *testing.T) {
	channels := []*Channel{
		{Id: 1, Type: constant.ChannelTypeOpenAI, Models: "mimo-v2.5"},
		{Id: 2, Type: constant.ChannelTypeOpenAI, Models: "gpt-4o"},
		{Id: 3, Type: constant.ChannelTypeAnthropic, Models: "claude-opus-4-6"},
		{Id: 4, Type: constant.ChannelTypeAnthropic, Models: "xiaomi/mimo-v2.5"},
	}

	counts := CountChannelDisplayTypes(channels)
	require.EqualValues(t, 1, counts[constant.ChannelTypeOpenAI])
	require.EqualValues(t, 1, counts[constant.ChannelTypeAnthropic])

	filtered := FilterChannelsByDisplayType(channels, constant.ChannelTypeOpenAI)
	require.Len(t, filtered, 1)
	require.Equal(t, 2, filtered[0].Id)

	filtered = FilterChannelsByDisplayType(channels, constant.ChannelTypeAnthropic)
	require.Len(t, filtered, 1)
	require.Equal(t, 3, filtered[0].Id)

	require.Len(t, FilterChannelsByDisplayType(channels, -1), 4)
}
