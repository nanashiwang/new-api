package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestUpdateOptionMapInviteBindingSettingsPublishesOnlyValidSnapshot(t *testing.T) {
	originalSettings := common.GetInviteBindingSettings()
	common.OptionMapRWMutex.Lock()
	originalMap := common.OptionMap
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	originalValue, hadOriginalValue := common.OptionMap["InviteBindingSettings"]
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		_ = common.SetInviteBindingSettings(originalSettings)
		common.OptionMapRWMutex.Lock()
		if originalMap == nil {
			common.OptionMap = nil
		} else if hadOriginalValue {
			common.OptionMap["InviteBindingSettings"] = originalValue
		} else {
			delete(common.OptionMap, "InviteBindingSettings")
		}
		common.OptionMapRWMutex.Unlock()
	})

	valid := `{"threshold":1000,"rate_after_threshold":20}`
	if err := updateOptionMap("InviteBindingSettings", valid); err != nil {
		t.Fatalf("update valid settings: %v", err)
	}
	want := common.InviteBindingSettings{Threshold: 1000, RateAfterThreshold: 20}
	if got := common.GetInviteBindingSettings(); got != want {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}

	if err := updateOptionMap("InviteBindingSettings", `{"threshold":1000}`); err == nil {
		t.Fatal("expected incomplete settings to be rejected")
	}
	if got := common.GetInviteBindingSettings(); got != want {
		t.Fatalf("invalid update changed settings to %+v", got)
	}
	common.OptionMapRWMutex.RLock()
	gotValue := common.OptionMap["InviteBindingSettings"]
	common.OptionMapRWMutex.RUnlock()
	if gotValue != valid {
		t.Fatalf("invalid update changed option map to %q", gotValue)
	}
}
