package model

import (
	"testing"

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
