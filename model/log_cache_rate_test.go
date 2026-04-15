package model

import (
	"math"
	"testing"
)

func seedCacheRateLogFixtures(t *testing.T) {
	t.Helper()

	logs := []Log{
		{
			UserId:           1,
			Username:         "alice",
			Type:             LogTypeConsume,
			ModelName:        "gpt-5.4",
			Quota:            100,
			PromptTokens:     100,
			CompletionTokens: 20,
			Group:            "default",
			ChannelId:        1,
			RequestId:        "req-openai-hit",
			CreatedAt:        100,
			Other:            `{"cache_tokens":80}`,
		},
		{
			UserId:           1,
			Username:         "alice",
			Type:             LogTypeConsume,
			ModelName:        "gpt-5.4",
			Quota:            50,
			PromptTokens:     50,
			CompletionTokens: 10,
			Group:            "default",
			ChannelId:        1,
			RequestId:        "req-openai-plain",
			CreatedAt:        101,
			Other:            `{}`,
		},
		{
			UserId:           1,
			Username:         "alice",
			Type:             LogTypeConsume,
			ModelName:        "claude-sonnet-4",
			Quota:            60,
			PromptTokens:     10,
			CompletionTokens: 5,
			Group:            "default",
			ChannelId:        1,
			RequestId:        "req-claude-write",
			CreatedAt:        102,
			Other:            `{"claude":true,"cache_creation_tokens":50}`,
		},
		{
			UserId:           1,
			Username:         "alice",
			Type:             LogTypeConsume,
			ModelName:        "claude-sonnet-4",
			Quota:            80,
			PromptTokens:     20,
			CompletionTokens: 8,
			Group:            "default",
			ChannelId:        1,
			RequestId:        "req-claude-hit",
			CreatedAt:        103,
			Other:            `{"claude":true,"cache_tokens":30,"cache_creation_tokens":10}`,
		},
	}

	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatalf("seed cache rate logs: %v", err)
	}
}

func assertFloatEquals(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("float mismatch: got=%f want=%f", got, want)
	}
}

func TestSumUsedQuota_ComputesCacheRatesWithMixedUsageSemantics(t *testing.T) {
	setupLogQueryTestDB(t)
	seedCacheRateLogFixtures(t)

	stat, err := SumUsedQuota(
		LogTypeConsume,
		0,
		0,
		"",
		"alice",
		"",
		0,
		"default",
		"",
		false,
	)
	if err != nil {
		t.Fatalf("SumUsedQuota: %v", err)
	}

	if stat.Quota != 290 {
		t.Fatalf("quota mismatch: got=%d want=%d", stat.Quota, 290)
	}

	assertFloatEquals(t, stat.CacheGlobalRate, 110.0/270.0)
	assertFloatEquals(t, stat.CacheHitRate, 110.0/160.0)
}

func TestSumUsedQuota_RequestIDFilterAffectsCacheRates(t *testing.T) {
	setupLogQueryTestDB(t)
	seedCacheRateLogFixtures(t)

	stat, err := SumUsedQuota(
		LogTypeConsume,
		0,
		0,
		"",
		"alice",
		"",
		0,
		"default",
		"req-claude-hit",
		false,
	)
	if err != nil {
		t.Fatalf("SumUsedQuota: %v", err)
	}

	if stat.Quota != 80 {
		t.Fatalf("quota mismatch: got=%d want=%d", stat.Quota, 80)
	}

	assertFloatEquals(t, stat.CacheGlobalRate, 0.5)
	assertFloatEquals(t, stat.CacheHitRate, 0.5)
}
