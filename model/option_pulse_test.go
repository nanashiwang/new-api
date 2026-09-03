package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestPulseUsageLogGateRejectsDisablingConsumeLogs(t *testing.T) {
	original := common.PulseUsageLogRequired
	common.PulseUsageLogRequired = true
	t.Cleanup(func() { common.PulseUsageLogRequired = original })

	if err := validatePulseUsageLogOption("LogConsumeEnabled", "false"); err == nil || !strings.Contains(err.Error(), "不能关闭消费日志") {
		t.Fatalf("gate error = %v, want consume log rejection", err)
	}
	if err := validatePulseUsageLogOption("LogConsumeEnabled", "true"); err != nil {
		t.Fatalf("enabling consume logs rejected: %v", err)
	}
	if err := validatePulseUsageLogOption("OtherEnabled", "false"); err != nil {
		t.Fatalf("unrelated option rejected: %v", err)
	}
}

func TestPulseUsageLogGateAllowsLegacyToggleWhenDisabled(t *testing.T) {
	original := common.PulseUsageLogRequired
	common.PulseUsageLogRequired = false
	t.Cleanup(func() { common.PulseUsageLogRequired = original })

	if err := validatePulseUsageLogOption("LogConsumeEnabled", "false"); err != nil {
		t.Fatalf("legacy toggle rejected without gate: %v", err)
	}
}
