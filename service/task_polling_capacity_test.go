package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/require"
)

func TestAcquireTaskPollingChannelCapacityUsesInjectedAdmission(t *testing.T) {
	original := AcquireTaskPollingChannelCapacityFunc
	t.Cleanup(func() { AcquireTaskPollingChannelCapacityFunc = original })

	called := false
	released := false
	AcquireTaskPollingChannelCapacityFunc = func(channel *model.Channel, requestID string) (func(), bool) {
		called = true
		require.Equal(t, 993101, channel.Id)
		require.Contains(t, requestID, "task-poll:video:993101:")
		return func() { released = true }, true
	}

	release, acquired := acquireTaskPollingChannelCapacity(context.Background(), &model.Channel{Id: 993101}, "video")
	require.True(t, acquired)
	require.True(t, called)
	require.NotNil(t, release)
	release()
	require.True(t, released)
}

func TestRecordTaskPollingRateLimitStartsCooldown(t *testing.T) {
	resetChannelRateLimitCooldownForTest()
	t.Cleanup(resetChannelRateLimitCooldownForTest)

	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     make(http.Header),
	}
	resp.Header.Set("Retry-After", "7")
	channel := &model.Channel{Id: 993102}

	recordTaskPollingRateLimit(context.Background(), channel, resp)
	require.GreaterOrEqual(t, ChannelRateLimitCooldownRemaining(channel.Id), 6*time.Second)
}
