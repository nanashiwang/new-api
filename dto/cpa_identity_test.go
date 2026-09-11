package dto

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestCPAIdentitySettingsValidationAndRoundTrip(t *testing.T) {
	for _, id := range []string{"", "bad id", "bad\nvalue", strings.Repeat("a", 65), "-bad"} {
		require.Error(t, (ChannelSettings{CPAUserIdentityEnabled: true, CPAInstanceID: id}).Validate())
	}
	settings := ChannelSettings{CPAUserIdentityEnabled: true, CPAInstanceID: "newapi.main-1"}
	require.NoError(t, settings.Validate())
	raw, err := common.Marshal(settings)
	require.NoError(t, err)
	var restored ChannelSettings
	require.NoError(t, common.Unmarshal(raw, &restored))
	require.Equal(t, settings, restored)
	require.NoError(t, (ChannelSettings{}).Validate())
}
