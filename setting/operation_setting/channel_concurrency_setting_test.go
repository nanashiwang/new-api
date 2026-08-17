package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelConcurrencyRpmNormalization(t *testing.T) {
	setting := ChannelConcurrencySetting{
		DefaultRpmLimit:  -1,
		RpmWindowSeconds: 0,
	}

	require.Zero(t, setting.NormalizedDefaultRpmLimit())
	require.Equal(t, 60, setting.NormalizedRpmWindowSeconds())

	setting.DefaultRpmLimit = 120
	setting.RpmWindowSeconds = 30
	require.Equal(t, 120, setting.NormalizedDefaultRpmLimit())
	require.Equal(t, 30, setting.NormalizedRpmWindowSeconds())
}

func TestValidateChannelConcurrencyRpmOption(t *testing.T) {
	require.NoError(t, ValidateChannelConcurrencyOption("channel_concurrency_setting.default_rpm_limit", "0"))
	require.NoError(t, ValidateChannelConcurrencyOption("channel_concurrency_setting.default_rpm_limit", "120"))
	require.NoError(t, ValidateChannelConcurrencyOption("channel_concurrency_setting.rpm_window_seconds", "60"))
	require.Error(t, ValidateChannelConcurrencyOption("channel_concurrency_setting.default_rpm_limit", "-1"))
	require.Error(t, ValidateChannelConcurrencyOption("channel_concurrency_setting.rpm_window_seconds", "bad"))
}
