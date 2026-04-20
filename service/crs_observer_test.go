package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
			"hasActiveWindow":   true,
			"sessionWindowStatus": "active",
			"progress":          64.5,
			"remainingTime":     "5h 12m",
			"windowEnd":         "2026-04-20T15:00:00Z",
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
}
