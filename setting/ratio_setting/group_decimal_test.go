package ratio_setting

import (
	"github.com/QuantumNous/new-api/common"
	"testing"
)

func TestFourDecimalGroupRatios(t *testing.T) {
	old := GroupRatio2JSONString()
	oldRules := GroupGroupRatio2JSONString()
	oldTopup := common.TopupGroupRatio2JSONString()
	t.Cleanup(func() {
		_ = UpdateGroupRatioByJSONString(old)
		_ = UpdateGroupGroupRatioByJSONString(oldRules)
		_ = common.UpdateTopupGroupRatioByJSONString(oldTopup)
	})
	const payload = `{"decimal":0.0001,"fraction":0.04,"zero":0}`
	if err := CheckGroupRatio(payload); err != nil {
		t.Fatal(err)
	}
	if err := UpdateGroupRatioByJSONString(payload); err != nil {
		t.Fatal(err)
	}
	if err := common.UpdateTopupGroupRatioByJSONString(payload); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]float64{"decimal": 0.0001, "fraction": 0.04, "zero": 0} {
		if got := GetGroupRatio(name); got != want {
			t.Fatalf("group %s: %v != %v", name, got, want)
		}
		if got := common.GetTopupGroupRatio(name); got != want {
			t.Fatalf("topup %s: %v != %v", name, got, want)
		}
	}
	if err := UpdateGroupGroupRatioByJSONString(`{"vip":{"decimal":0.0001}}`); err != nil {
		t.Fatal(err)
	}
	if got, ok := GetGroupGroupRatio("vip", "decimal"); !ok || got != 0.0001 {
		t.Fatalf("special ratio: %v %v", got, ok)
	}
	if err := CheckGroupRatio(`{"invalid":-0.0001}`); err == nil {
		t.Fatal("negative ratio accepted")
	}
}
