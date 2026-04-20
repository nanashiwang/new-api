package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func decodeUsageWindowsForTest(t *testing.T, raw string) []model.CRSUsageWindow {
	t.Helper()

	windows := make([]model.CRSUsageWindow, 0)
	require.NoError(t, common.UnmarshalJsonStr(raw, &windows))
	return windows
}

func TestNormalizeCRSRemoteAccountSnapshotCapturesClaudeSignals(t *testing.T) {
	t.Parallel()

	account := map[string]any{
		"id":          "acct-official-1",
		"name":        "Official A",
		"platform":    "claude",
		"authType":    "oauth",
		"accountType": "shared",
		"isActive":    true,
		"schedulable": true,
		"status":      "active",
		"rateLimitStatus": map[string]any{
			"isRateLimited":    true,
			"minutesRemaining": 137,
			"rateLimitEndAt":   "2026-04-20T12:00:00Z",
		},
		"sessionWindow": map[string]any{
			"hasActiveWindow":     true,
			"sessionWindowStatus": "active",
			"progress":            64.5,
			"remainingTime":       "5h 12m",
			"windowEnd":           "2026-04-20T15:00:00Z",
		},
		"claudeUsage": map[string]any{
			"fiveHour": map[string]any{
				"progress":      64.5,
				"remainingTime": "5h 12m",
				"resetAt":       "2026-04-20T15:00:00Z",
			},
			"sevenDay": map[string]any{
				"progress":      20,
				"remainingText": "余 4 天",
				"resetAt":       "2026-04-27T00:00:00Z",
			},
			"sevenDayOpus": map[string]any{
				"progress":      90,
				"remainingTime": "余 1 天",
				"windowEnd":     "2026-04-21T00:00:00Z",
			},
		},
		"subscriptionInfo": map[string]any{
			"accountType": "max",
		},
	}

	snapshot, err := normalizeCRSRemoteAccountSnapshot(7, "claude", account, nil, 1713600000)
	require.NoError(t, err)
	require.Equal(t, 7, snapshot.SiteID)
	require.Equal(t, "acct-official-1", snapshot.RemoteAccountID)
	require.Equal(t, "claude", snapshot.Platform)
	require.Equal(t, "Official A", snapshot.Name)
	require.Equal(t, "oauth", snapshot.AuthType)
	require.True(t, snapshot.IsActive)
	require.True(t, snapshot.Schedulable)
	require.True(t, snapshot.RateLimited)
	require.Equal(t, 137, snapshot.RateLimitMinutesRemaining)
	require.Equal(t, "2026-04-20T12:00:00Z", snapshot.RateLimitResetAt)
	require.True(t, snapshot.SessionWindowActive)
	require.Equal(t, "active", snapshot.SessionWindowStatus)
	require.EqualValues(t, 64.5, snapshot.SessionWindowProgress)
	require.Equal(t, "5h 12m", snapshot.SessionWindowRemaining)
	require.Equal(t, "2026-04-20T15:00:00Z", snapshot.SessionWindowEndAt)
	require.Equal(t, "max", snapshot.SubscriptionPlan)

	windows := decodeUsageWindowsForTest(t, snapshot.UsageWindowsJSON)
	require.Len(t, windows, 3)
	require.Equal(t, model.CRSUsageWindow{
		Key:           "five_hour",
		Label:         "5h",
		Progress:      64.5,
		RemainingText: "5h 12m",
		ResetAt:       "2026-04-20T15:00:00Z",
		Tone:          "info",
		Source:        "claude_usage",
	}, windows[0])
	require.Equal(t, model.CRSUsageWindow{
		Key:           "seven_day",
		Label:         "7d",
		Progress:      20,
		RemainingText: "余 4 天",
		ResetAt:       "2026-04-27T00:00:00Z",
		Tone:          "success",
		Source:        "claude_usage",
	}, windows[1])
	require.Equal(t, model.CRSUsageWindow{
		Key:           "seven_day_opus",
		Label:         "Opus 周限",
		Progress:      90,
		RemainingText: "余 1 天",
		ResetAt:       "2026-04-21T00:00:00Z",
		Tone:          "danger",
		Source:        "claude_usage",
	}, windows[2])
}

func TestNormalizeCRSRemoteAccountSnapshotUsesCodexUsageWindowsBeforeSessionWindow(t *testing.T) {
	t.Parallel()

	account := map[string]any{
		"id":       "acct-codex-1",
		"name":     "Codex A",
		"platform": "openai",
		"codexUsage": map[string]any{
			"primary": map[string]any{
				"progress":      48,
				"remainingTime": "2h 36m",
				"resetAt":       "2026-04-20T18:00:00Z",
			},
			"secondary": map[string]any{
				"progress":      82,
				"remainingText": "余 2 天",
				"resetAt":       "2026-04-27T00:00:00Z",
			},
		},
		"sessionWindow": map[string]any{
			"hasActiveWindow":     true,
			"sessionWindowStatus": "active",
			"progress":            5,
			"remainingTime":       "should-not-win",
			"windowEnd":           "2026-04-20T19:00:00Z",
		},
	}

	snapshot, err := normalizeCRSRemoteAccountSnapshot(8, "openai", account, nil, 1713600000)
	require.NoError(t, err)

	windows := decodeUsageWindowsForTest(t, snapshot.UsageWindowsJSON)
	require.Len(t, windows, 2)
	require.Equal(t, model.CRSUsageWindow{
		Key:           "primary",
		Label:         "5h",
		Progress:      48,
		RemainingText: "2h 36m",
		ResetAt:       "2026-04-20T18:00:00Z",
		Tone:          "info",
		Source:        "codex_usage",
	}, windows[0])
	require.Equal(t, model.CRSUsageWindow{
		Key:           "secondary",
		Label:         "周限",
		Progress:      82,
		RemainingText: "余 2 天",
		ResetAt:       "2026-04-27T00:00:00Z",
		Tone:          "warning",
		Source:        "codex_usage",
	}, windows[1])
}

func TestNormalizeCRSRemoteAccountSnapshotUsesBalanceQuota(t *testing.T) {
	t.Parallel()

	account := map[string]any{
		"id":             "acct-console-1",
		"name":           "Console A",
		"platform":       "claude-console",
		"accountType":    "shared",
		"isActive":       true,
		"schedulable":    true,
		"status":         "active",
		"dailyQuota":     20,
		"quotaResetTime": "00:00",
	}
	balance := map[string]any{
		"data": map[string]any{
			"quota": map[string]any{
				"used":       8.5,
				"remaining":  11.5,
				"percentage": 42.5,
				"resetAt":    "2026-04-21T00:00:00Z",
			},
			"balance": map[string]any{
				"amount":   11.5,
				"currency": "USD",
			},
			"status": "success",
		},
	}

	snapshot, err := normalizeCRSRemoteAccountSnapshot(9, "claude-console", account, balance, 1713600000)
	require.NoError(t, err)
	require.Equal(t, "acct-console-1", snapshot.RemoteAccountID)
	require.Equal(t, "claude-console", snapshot.Platform)
	require.EqualValues(t, 8.5, snapshot.QuotaUsed)
	require.EqualValues(t, 11.5, snapshot.QuotaRemaining)
	require.EqualValues(t, 42.5, snapshot.QuotaPercentage)
	require.Equal(t, "2026-04-21T00:00:00Z", snapshot.QuotaResetAt)
	require.EqualValues(t, 11.5, snapshot.BalanceAmount)
	require.Equal(t, "USD", snapshot.BalanceCurrency)

	windows := decodeUsageWindowsForTest(t, snapshot.UsageWindowsJSON)
	require.Len(t, windows, 1)
	require.Equal(t, model.CRSUsageWindow{
		Key:           "quota",
		Label:         "额度",
		Progress:      42.5,
		RemainingText: "11.5",
		ResetAt:       "2026-04-21T00:00:00Z",
		Tone:          "success",
		Source:        "quota_balance",
	}, windows[0])
}

func TestNormalizeCRSRemoteAccountSnapshotFallsBackToSessionWindow(t *testing.T) {
	t.Parallel()

	account := map[string]any{
		"id":       "acct-session-1",
		"name":     "Session A",
		"platform": "claude",
		"sessionWindow": map[string]any{
			"hasActiveWindow":     true,
			"sessionWindowStatus": "active",
			"progress":            73,
			"remainingTime":       "1h 21m",
			"windowEnd":           "2026-04-20T20:00:00Z",
		},
	}

	snapshot, err := normalizeCRSRemoteAccountSnapshot(10, "claude", account, nil, 1713600000)
	require.NoError(t, err)

	windows := decodeUsageWindowsForTest(t, snapshot.UsageWindowsJSON)
	require.Len(t, windows, 1)
	require.Equal(t, model.CRSUsageWindow{
		Key:           "session_window",
		Label:         "5h",
		Progress:      73,
		RemainingText: "1h 21m",
		ResetAt:       "2026-04-20T20:00:00Z",
		Tone:          "info",
		Source:        "session_window",
	}, windows[0])
}
