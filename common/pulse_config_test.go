package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPulseConfigOverridesRemainExplicitAndSnapshotsAreDetached(t *testing.T) {
	original := GetPulseConfigOverrides()
	t.Cleanup(func() { ReplacePulseConfig(original) })
	t.Setenv("PULSE_BENEFIT_ENABLED", "true")
	t.Setenv("PULSE_SERVICE_HMAC_SECRET", "environment-secret")
	t.Setenv("PULSE_ROLLBACK_HMAC_SECRET", "environment-rollback")
	overrides := map[string]string{"PulseBenefitEnabled": "false", "PulseServiceHMACSecret": ""}
	ReplacePulseConfig(overrides)
	overrides["PulseBenefitEnabled"] = "true"
	cfg := GetPulseConfig()
	require.Equal(t, "false", cfg["PulseBenefitEnabled"])
	require.Empty(t, cfg["PulseServiceHMACSecret"])
	require.Equal(t, "environment-rollback", cfg["PulseRollbackHMACSecret"])
	cfg["PulseRollbackHMACSecret"] = "mutated"
	require.Equal(t, "environment-rollback", GetPulseConfig()["PulseRollbackHMACSecret"])
	t.Setenv("PULSE_ROLLBACK_HMAC_SECRET", "rotated-environment-rollback")
	require.Equal(t, "rotated-environment-rollback", GetPulseConfig()["PulseRollbackHMACSecret"])
}
