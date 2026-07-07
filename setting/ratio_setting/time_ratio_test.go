package ratio_setting

import (
	"testing"
	"time"
)

func withTimeRatioRules(t *testing.T, rules string) {
	t.Helper()
	original := TimeRatioRules2JSONString()
	t.Cleanup(func() {
		if err := UpdateTimeRatioRulesByJSONString(original); err != nil {
			t.Fatalf("restore time ratio rules: %v", err)
		}
	})
	if err := UpdateTimeRatioRulesByJSONString(rules); err != nil {
		t.Fatalf("update time ratio rules: %v", err)
	}
}

func TestResolveTimeRatioMatchesPriorityAndPatterns(t *testing.T) {
	withTimeRatioRules(t, `[
		{"id":"low","enabled":true,"timezone":"UTC","start":"00:00","end":"23:59","ratio":0.8,"models":["test-*"],"groups":["default"],"priority":1},
		{"id":"high","enabled":true,"timezone":"UTC","start":"00:00","end":"23:59","ratio":1.2,"models":["test-model"],"groups":["default"],"priority":10}
	]`)

	got := ResolveTimeRatio("test-model", "default", "default", time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC))
	if got.RuleID != "high" {
		t.Fatalf("expected high priority rule, got %q", got.RuleID)
	}
	if got.EffectiveRatio() != 1.2 {
		t.Fatalf("expected ratio 1.2, got %v", got.EffectiveRatio())
	}
}

func TestResolveTimeRatioSupportsCrossMidnightPreviousDay(t *testing.T) {
	withTimeRatioRules(t, `[
		{"id":"monday-night","enabled":true,"timezone":"UTC","start":"23:00","end":"08:00","days":["mon"],"ratio":0.7,"models":["*"],"groups":["*"],"priority":1}
	]`)

	got := ResolveTimeRatio("any-model", "default", "default", time.Date(2026, 7, 7, 1, 0, 0, 0, time.UTC))
	if got.RuleID != "monday-night" {
		t.Fatalf("expected monday-night rule after midnight, got %q", got.RuleID)
	}
	if got.EffectiveRatio() != 0.7 {
		t.Fatalf("expected ratio 0.7, got %v", got.EffectiveRatio())
	}

	got = ResolveTimeRatio("any-model", "default", "default", time.Date(2026, 7, 7, 22, 0, 0, 0, time.UTC))
	if got.Matched() {
		t.Fatalf("did not expect rule at 22:00 Tuesday, got %q", got.RuleID)
	}
}

func TestParseTimeRatioRulesRejectsInvalidEnabledRule(t *testing.T) {
	_, err := ParseTimeRatioRules(`[{"id":"bad","enabled":true,"timezone":"UTC","start":"9:00","end":"18:00","ratio":1}]`)
	if err == nil {
		t.Fatal("expected invalid time format error")
	}

	_, err = ParseTimeRatioRules(`[{"id":"bad","enabled":true,"timezone":"UTC","start":"09:00","end":"18:00","ratio":0}]`)
	if err == nil {
		t.Fatal("expected invalid ratio error")
	}
}
