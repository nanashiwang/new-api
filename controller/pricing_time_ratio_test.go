package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestBuildPricingTimeRatioMap(t *testing.T) {
	original := ratio_setting.TimeRatioRules2JSONString()
	t.Cleanup(func() {
		if err := ratio_setting.UpdateTimeRatioRulesByJSONString(original); err != nil {
			t.Fatalf("restore time ratio rules: %v", err)
		}
	})

	if err := ratio_setting.UpdateTimeRatioRulesByJSONString(`[
		{"id":"member-peak","enabled":true,"timezone":"UTC","start":"00:00","end":"23:59","ratio":1.5,"models":["peak-*"],"groups":["default"],"user_groups":["member"],"priority":10}
	]`); err != nil {
		t.Fatalf("set time ratio rules: %v", err)
	}

	got := buildPricingTimeRatioMap(
		[]model.Pricing{{ModelName: "peak-model"}, {ModelName: "plain-model"}},
		map[string]string{"default": "默认分组", "vip": "VIP"},
		"member",
		time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC),
	)

	info, ok := got["peak-model"]["default"]
	if !ok {
		t.Fatalf("expected peak-model default time ratio, got %#v", got)
	}
	if info.Ratio != 1.5 || !info.Matched || info.Timezone != "UTC" {
		t.Fatalf("unexpected preview info: %#v", info)
	}
	if _, ok := got["peak-model"]["vip"]; ok {
		t.Fatalf("unexpected vip ratio: %#v", got["peak-model"]["vip"])
	}
	if _, ok := got["plain-model"]; ok {
		t.Fatalf("plain model should not expose default 1x ratio: %#v", got["plain-model"])
	}
}
