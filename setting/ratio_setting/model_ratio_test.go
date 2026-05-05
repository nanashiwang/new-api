package ratio_setting

import "testing"

func TestGetCompletionRatioInfoGPT55UsesOfficialOutputMultiplier(t *testing.T) {
	info := GetCompletionRatioInfo("gpt-5.5")

	if info.Ratio != 6 {
		t.Fatalf("gpt-5.5 completion ratio = %v, want 6", info.Ratio)
	}
	if !info.Locked {
		t.Fatal("gpt-5.5 completion ratio should be locked to the official multiplier")
	}
}

func TestGetCompletionRatioGPT55DatedVariant(t *testing.T) {
	got := GetCompletionRatio("gpt-5.5-2026-04-24")

	if got != 6 {
		t.Fatalf("gpt-5.5 dated variant completion ratio = %v, want 6", got)
	}
}

func TestCodexAutoReviewUsesCodexRoutingModel(t *testing.T) {
	got := FormatMatchingModelName("codex-auto-review")

	if got != "gpt-5.3-codex" {
		t.Fatalf("FormatMatchingModelName(codex-auto-review) = %q, want gpt-5.3-codex", got)
	}
}

func TestCodexAutoReviewCompactUsesCodexRoutingModel(t *testing.T) {
	got := FormatMatchingModelName(WithCompactModelSuffix("codex-auto-review"))

	if got != "gpt-5.3-codex" {
		t.Fatalf("FormatMatchingModelName(codex-auto-review compact) = %q, want gpt-5.3-codex", got)
	}
}

func TestCodexAutoReviewCompletionRatioUsesGPT5Family(t *testing.T) {
	got := GetCompletionRatio("codex-auto-review")

	if got != 8 {
		t.Fatalf("codex-auto-review completion ratio = %v, want 8", got)
	}
}
