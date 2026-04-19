package model_setting

import (
	"net/http"
	"testing"
)

func TestClaudeSettingsWriteHeadersMergesCommaSeparatedValues(t *testing.T) {
	settings := &ClaudeSettings{
		HeadersSettings: map[string]map[string][]string{
			"claude-sonnet-4-5-20250929": {
				"anthropic-beta": {
					"context-1m-2025-08-07",
					"token-efficient-tools-2025-02-19",
				},
			},
		},
	}

	headers := http.Header{}
	headers.Set("anthropic-beta", "output-128k-2025-02-19, context-1m-2025-08-07")

	settings.WriteHeaders("claude-sonnet-4-5-20250929", &headers)

	got := headers.Values("anthropic-beta")
	if len(got) != 1 {
		t.Fatalf("expected a single merged anthropic-beta header, got %v", got)
	}

	want := "output-128k-2025-02-19,context-1m-2025-08-07,token-efficient-tools-2025-02-19"
	if got[0] != want {
		t.Fatalf("unexpected anthropic-beta header, want %q, got %q", want, got[0])
	}
}

func TestClaudeSettingsWriteHeaders_StripsDeprecatedContextBetaForClaude46(t *testing.T) {
	settings := &ClaudeSettings{
		HeadersSettings: map[string]map[string][]string{},
	}

	headers := http.Header{}
	headers.Set("anthropic-beta", "context-1m-2025-08-07,computer-use-2025-01-24")

	settings.WriteHeaders("claude-sonnet-4-6", &headers)

	got := headers.Get("anthropic-beta")
	want := "computer-use-2025-01-24"
	if got != want {
		t.Fatalf("unexpected anthropic-beta header, want %q, got %q", want, got)
	}
}

func TestClaudeSettingsWriteHeaders_PreservesContextBetaForEarlierClaudeModels(t *testing.T) {
	settings := &ClaudeSettings{
		HeadersSettings: map[string]map[string][]string{},
	}

	headers := http.Header{}
	headers.Set("anthropic-beta", "context-1m-2025-08-07,computer-use-2025-01-24")

	settings.WriteHeaders("claude-sonnet-4-5-20250929", &headers)

	got := headers.Get("anthropic-beta")
	want := "context-1m-2025-08-07,computer-use-2025-01-24"
	if got != want {
		t.Fatalf("unexpected anthropic-beta header, want %q, got %q", want, got)
	}
}
