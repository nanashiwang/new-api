package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSlowTTFTSettingUsesSafeDefaults(t *testing.T) {
	setting := NormalizeSlowTTFTSetting(SlowTTFTSetting{
		ThresholdMS:             -1,
		BaselineMultiplier:      0,
		GlobalMinUsers:          1,
		MaxEntries:              1,
		ContextBucketBoundaries: []int{100, 50},
	})

	require.Equal(t, defaultSlowTTFTThresholdMS, setting.ThresholdMS)
	require.Equal(t, defaultSlowTTFTBaselineMultiplier, setting.BaselineMultiplier)
	require.Equal(t, defaultSlowTTFTGlobalMinUsers, setting.GlobalMinUsers)
	require.Equal(t, defaultSlowTTFTMaxEntries, setting.MaxEntries)
	require.Equal(t, defaultSlowTTFTContextBuckets, setting.ContextBucketBoundaries)
	require.True(t, setting.ObserveOnly)
}

func TestValidateSlowTTFTOption(t *testing.T) {
	require.NoError(t, ValidateSlowTTFTOption("slow_ttft_setting.global_min_users", "3"))
	require.NoError(t, ValidateSlowTTFTOption("slow_ttft_setting.context_bucket_boundaries", "[50000,100000]"))
	require.Error(t, ValidateSlowTTFTOption("slow_ttft_setting.global_min_users", "1"))
	require.Error(t, ValidateSlowTTFTOption("slow_ttft_setting.global_slow_rate", "1.1"))
	require.Error(t, ValidateSlowTTFTOption("slow_ttft_setting.global_slow_rate", "NaN"))
	require.Error(t, ValidateSlowTTFTOption("slow_ttft_setting.context_bucket_boundaries", "[100000,50000]"))
	require.Error(t, ValidateSlowTTFTOption("slow_ttft_setting.unknown", "1"))
}
