package operation_setting

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestChannelAffinityDefaultRulesSkipRetryOnFailure(t *testing.T) {
	rules := GetChannelAffinitySetting().Rules
	expected := map[string]bool{
		"codex trace":       true,
		"claude code trace": true,
	}

	for name, want := range expected {
		found := false
		for _, rule := range rules {
			if rule.Name != name {
				continue
			}
			found = true
			if rule.SkipRetryOnFailure != want {
				t.Fatalf("unexpected skip_retry_on_failure for %q, want %v, got %v", name, want, rule.SkipRetryOnFailure)
			}
			break
		}
		if !found {
			t.Fatalf("expected default rule %q to exist", name)
		}
	}
}

func TestChannelAffinityRuleMarshalIncludesFalseSkipRetryOnFailure(t *testing.T) {
	data, err := common.Marshal(ChannelAffinityRule{
		Name:               "demo",
		SkipRetryOnFailure: false,
	})
	if err != nil {
		t.Fatalf("marshal ChannelAffinityRule: %v", err)
	}

	if !strings.Contains(string(data), `"skip_retry_on_failure":false`) {
		t.Fatalf("expected marshaled rule to include false skip_retry_on_failure, got %s", string(data))
	}
}
