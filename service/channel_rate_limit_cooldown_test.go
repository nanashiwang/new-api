package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func useLocalRateLimitCooldown(t *testing.T) {
	t.Helper()
	originalRedisEnabled := common.RedisEnabled
	originalRDB := common.RDB
	common.RedisEnabled = false
	common.RDB = nil
	resetChannelRateLimitCooldownForTest()
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		common.RDB = originalRDB
		resetChannelRateLimitCooldownForTest()
	})
}

func TestRecordChannelRateLimitCooldownUsesRetryAfter(t *testing.T) {
	useLocalRateLimitCooldown(t)
	channel := &model.Channel{Id: 992001}
	apiErr := types.WithOpenAIError(types.OpenAIError{Message: "rate limited", Type: "rate_limit_error"}, http.StatusTooManyRequests)
	apiErr.RetryAfter = 2500 * time.Millisecond

	applied := RecordChannelRateLimitCooldown(nil, channel, apiErr)
	require.Greater(t, applied, 2*time.Second)
	assert.LessOrEqual(t, applied, 2500*time.Millisecond)
	assert.Equal(t, applied, apiErr.RetryAfter)
	assert.True(t, IsChannelRateLimitCoolingDown(channel))
	assert.True(t, IsChannelUnavailableForRequest(channel))
}

func TestRecordChannelRateLimitCooldownDefaultsAndClamps(t *testing.T) {
	assert.Equal(t, time.Minute, normalizeRateLimitCooldown(0))
	assert.Equal(t, time.Second, normalizeRateLimitCooldown(100*time.Millisecond))
	assert.Equal(t, time.Hour, normalizeRateLimitCooldown(2*time.Hour))
}

func TestRecordChannelRateLimitCooldownIgnoresNonRateLimit(t *testing.T) {
	useLocalRateLimitCooldown(t)
	channel := &model.Channel{Id: 992002}
	apiErr := types.NewOpenAIError(errors.New("upstream failed"), types.ErrorCodeBadResponseStatusCode, http.StatusBadGateway)

	assert.Zero(t, RecordChannelRateLimitCooldown(nil, channel, apiErr))
	assert.False(t, IsChannelRateLimitCoolingDown(channel))
}

func TestMultiKeyRateLimitFallsBackToChannelCooldownWithoutRedis(t *testing.T) {
	useLocalRateLimitCooldown(t)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	common.SetContextKey(ctx, constant.ContextKeyChannelMultiKeyIndex, 1)
	channel := &model.Channel{Id: 992003, ChannelInfo: model.ChannelInfo{IsMultiKey: true}}
	apiErr := types.WithOpenAIError(types.OpenAIError{Message: "too many requests", Type: "rate_limit_error"}, http.StatusTooManyRequests)
	apiErr.RetryAfter = 3 * time.Second

	applied := RecordChannelRateLimitCooldown(ctx, channel, apiErr)
	require.Greater(t, applied, 2*time.Second)
	assert.True(t, IsChannelRateLimitCoolingDown(channel))
}

func TestMappedUpstream429IsRecognized(t *testing.T) {
	apiErr := types.NewOpenAIError(errors.New("temporarily unavailable"), types.ErrorCodeBadResponseStatusCode, http.StatusServiceUnavailable)
	apiErr.UpstreamStatusCode = http.StatusTooManyRequests
	assert.True(t, IsUpstreamRateLimitError(apiErr))
}
