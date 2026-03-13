package model

import "testing"

func TestAggregatePublicTokenDistributionRows_OpenAIStyle(t *testing.T) {
	rows := []tokenUsageLogRow{
		{
			PromptTokens:     2006,
			CompletionTokens: 120,
			Other:            `{"cache_tokens":1920}`,
		},
	}

	got := aggregatePublicTokenDistributionRows(rows)

	if got.InputTokens != 86 {
		t.Fatalf("input tokens mismatch: got=%d want=%d", got.InputTokens, 86)
	}
	if got.CacheReadTokens != 1920 {
		t.Fatalf("cache read tokens mismatch: got=%d want=%d", got.CacheReadTokens, 1920)
	}
	if got.CacheCreationTokens != 0 {
		t.Fatalf("cache creation tokens mismatch: got=%d want=%d", got.CacheCreationTokens, 0)
	}
	if got.CompletionTokens != 120 {
		t.Fatalf("completion tokens mismatch: got=%d want=%d", got.CompletionTokens, 120)
	}
	if got.TotalTokens != 2126 {
		t.Fatalf("total tokens mismatch: got=%d want=%d", got.TotalTokens, 2126)
	}
	if got.CacheCreationSupported {
		t.Fatal("cache creation should not be marked as supported for OpenAI-only cached reads")
	}
}

func TestAggregatePublicTokenDistributionRows_ClaudeStyle(t *testing.T) {
	rows := []tokenUsageLogRow{
		{
			PromptTokens:     120,
			CompletionTokens: 80,
			Other:            `{"claude":true,"cache_tokens":300,"cache_creation_tokens":200}`,
		},
	}

	got := aggregatePublicTokenDistributionRows(rows)

	if got.InputTokens != 120 {
		t.Fatalf("input tokens mismatch: got=%d want=%d", got.InputTokens, 120)
	}
	if got.CacheReadTokens != 300 {
		t.Fatalf("cache read tokens mismatch: got=%d want=%d", got.CacheReadTokens, 300)
	}
	if got.CacheCreationTokens != 200 {
		t.Fatalf("cache creation tokens mismatch: got=%d want=%d", got.CacheCreationTokens, 200)
	}
	if got.CompletionTokens != 80 {
		t.Fatalf("completion tokens mismatch: got=%d want=%d", got.CompletionTokens, 80)
	}
	if got.TotalTokens != 700 {
		t.Fatalf("total tokens mismatch: got=%d want=%d", got.TotalTokens, 700)
	}
	if !got.CacheCreationSupported {
		t.Fatal("cache creation should be marked as supported for Claude usage")
	}
}

func TestAggregatePublicTokenDistributionRows_SplitCacheCreation(t *testing.T) {
	rows := []tokenUsageLogRow{
		{
			PromptTokens:     100,
			CompletionTokens: 50,
			Other:            `{"usage_semantic":"anthropic","cache_creation_tokens_5m":60,"cache_creation_tokens_1h":40}`,
		},
	}

	got := aggregatePublicTokenDistributionRows(rows)

	if got.InputTokens != 100 {
		t.Fatalf("input tokens mismatch: got=%d want=%d", got.InputTokens, 100)
	}
	if got.CacheCreationTokens != 100 {
		t.Fatalf("cache creation tokens mismatch: got=%d want=%d", got.CacheCreationTokens, 100)
	}
	if got.TotalTokens != 250 {
		t.Fatalf("total tokens mismatch: got=%d want=%d", got.TotalTokens, 250)
	}
	if !got.CacheCreationSupported {
		t.Fatal("split cache creation tokens should mark support as true")
	}
}

func TestAggregatePublicTokenDistributionRows_InvalidOther(t *testing.T) {
	rows := []tokenUsageLogRow{
		{
			PromptTokens:     90,
			CompletionTokens: 10,
			Other:            `{"cache_tokens":`,
		},
	}

	got := aggregatePublicTokenDistributionRows(rows)

	if got.InputTokens != 90 {
		t.Fatalf("input tokens mismatch: got=%d want=%d", got.InputTokens, 90)
	}
	if got.CompletionTokens != 10 {
		t.Fatalf("completion tokens mismatch: got=%d want=%d", got.CompletionTokens, 10)
	}
	if got.CacheReadTokens != 0 || got.CacheCreationTokens != 0 {
		t.Fatalf("invalid other should not produce cache stats: got=%+v", got)
	}
	if got.TotalTokens != 100 {
		t.Fatalf("total tokens mismatch: got=%d want=%d", got.TotalTokens, 100)
	}
}

func TestGetPromptCacheSummaryFromOther_OpenAIStyle(t *testing.T) {
	cacheReadTokens, cacheWriteTokens := getPromptCacheSummaryFromOther(
		`{"cache_tokens":1920}`,
	)

	if cacheReadTokens != 1920 {
		t.Fatalf("cache read tokens mismatch: got=%d want=%d", cacheReadTokens, 1920)
	}
	if cacheWriteTokens != 0 {
		t.Fatalf("cache write tokens mismatch: got=%d want=%d", cacheWriteTokens, 0)
	}
}

func TestGetPromptCacheSummaryFromOther_AnthropicStyle(t *testing.T) {
	cacheReadTokens, cacheWriteTokens := getPromptCacheSummaryFromOther(
		`{"cache_tokens":300,"cache_creation_tokens_5m":60,"cache_creation_tokens_1h":40}`,
	)

	if cacheReadTokens != 300 {
		t.Fatalf("cache read tokens mismatch: got=%d want=%d", cacheReadTokens, 300)
	}
	if cacheWriteTokens != 100 {
		t.Fatalf("cache write tokens mismatch: got=%d want=%d", cacheWriteTokens, 100)
	}
}

func TestGetPromptCacheSummaryFromOther_InvalidOther(t *testing.T) {
	cacheReadTokens, cacheWriteTokens := getPromptCacheSummaryFromOther(
		`{"cache_tokens":`,
	)

	if cacheReadTokens != 0 || cacheWriteTokens != 0 {
		t.Fatalf("invalid other should not produce cache stats: read=%d write=%d", cacheReadTokens, cacheWriteTokens)
	}
}
