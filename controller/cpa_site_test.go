package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/require"
)

func TestCPAAccountStateDoesNotTreatUnknownAsAvailable(t *testing.T) {
	now := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		account service.CPAAccount
		want    string
	}{
		{name: "active", account: service.CPAAccount{Status: "active"}, want: "available"},
		{name: "pending", account: service.CPAAccount{Status: "pending"}, want: "limited"},
		{name: "refreshing", account: service.CPAAccount{Status: "refreshing"}, want: "limited"},
		{name: "future retry", account: service.CPAAccount{Status: "active", NextRetryAfter: "2026-07-28T09:00:00Z"}, want: "limited"},
		{name: "error", account: service.CPAAccount{Status: "error"}, want: "abnormal"},
		{name: "disabled flag", account: service.CPAAccount{Disabled: true}, want: "disabled"},
		{name: "disabled status", account: service.CPAAccount{Status: "disabled"}, want: "disabled"},
		{name: "unknown", account: service.CPAAccount{Status: "unknown"}, want: "unknown"},
		{name: "empty", account: service.CPAAccount{}, want: "unknown"},
		{name: "future status", account: service.CPAAccount{Status: "warming_up"}, want: "unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, cpaAccountState(test.account, now))
		})
	}
}

func TestCPASiteToVOSeparatesAccountStates(t *testing.T) {
	accounts := []service.CPAAccount{
		{Status: "active"},
		{Status: "pending"},
		{Status: "error"},
		{Status: "disabled"},
		{Status: "unknown"},
	}
	vo := cpaSiteToVO(&model.CPASite{Id: 1}, accounts)

	require.Equal(t, 5, vo.AccountCount)
	require.Equal(t, 1, vo.AvailableCount)
	require.Equal(t, 1, vo.LimitedCount)
	require.Equal(t, 1, vo.AbnormalCount)
	require.Equal(t, 1, vo.DisabledCount)
	require.Equal(t, 1, vo.UnknownCount)
	require.Equal(t, cpaSiteVO{}, cpaSiteToVO(nil, nil))
}

func TestCPASiteEndpointChanged(t *testing.T) {
	existing := &model.CPASite{Host: "CPA.Example.com:8317", Scheme: "https"}
	require.False(t, cpaSiteEndpointChanged(existing, &model.CPASite{Host: "cpa.example.com:8317/", Scheme: "HTTPS"}))
	require.True(t, cpaSiteEndpointChanged(existing, &model.CPASite{Host: "other.example.com:8317", Scheme: "https"}))
	require.True(t, cpaSiteEndpointChanged(existing, &model.CPASite{Host: "cpa.example.com:8317", Scheme: "http"}))
	require.True(t, cpaSiteEndpointChanged(nil, existing))
}
