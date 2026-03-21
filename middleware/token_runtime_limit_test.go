package middleware

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
)

func TestGetTokenWindowLimitExpiration(t *testing.T) {
	tests := []struct {
		name     string
		duration int64
		want     time.Duration
	}{
		{
			name:     "non-positive uses default expiration",
			duration: 0,
			want:     common.RateLimitKeyExpirationDuration,
		},
		{
			name:     "short window keeps default expiration",
			duration: 60,
			want:     common.RateLimitKeyExpirationDuration,
		},
		{
			name:     "long window extends expiration",
			duration: 3600,
			want:     time.Hour,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getTokenWindowLimitExpiration(tc.duration)
			if got != tc.want {
				t.Fatalf("getTokenWindowLimitExpiration(%d)=%s, want=%s", tc.duration, got, tc.want)
			}
		})
	}
}

func TestBuildTokenWindowRequestNonce(t *testing.T) {
	first := buildTokenWindowRequestNonce()
	second := buildTokenWindowRequestNonce()

	if first == "" || second == "" {
		t.Fatal("nonce should not be empty")
	}
	if first == second {
		t.Fatal("nonce should be unique between requests")
	}
}
