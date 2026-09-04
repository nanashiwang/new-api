package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRedisRateLimiter_UsesV2KeyAndRejectsWhenEvaluatorDenies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originEval := evalRedisRateLimit
	evalRedisRateLimit = func(_ context.Context, key string, _ time.Time, maxRequestNum int, duration int64, expiration time.Duration) (bool, error) {
		require.Equal(t, "rateLimit:v2:GW:203.0.113.8", key)
		require.Equal(t, 3, maxRequestNum)
		require.EqualValues(t, 60, duration)
		require.Equal(t, getRateLimitExpiration(duration), expiration)
		return false, nil
	}
	t.Cleanup(func() {
		evalRedisRateLimit = originEval
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/rate-limit", nil)
	ctx.Request.RemoteAddr = "203.0.113.8:12345"

	redisRateLimiter(ctx, 3, 60, "GW")
	require.Equal(t, http.StatusTooManyRequests, ctx.Writer.Status())
	require.True(t, ctx.IsAborted())
}

func TestUserRateLimitFactory_UsesV2UserKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originRedisEnabled := common.RedisEnabled
	originEval := evalRedisRateLimit
	common.RedisEnabled = true
	evalRedisRateLimit = func(_ context.Context, key string, _ time.Time, _ int, _ int64, _ time.Duration) (bool, error) {
		require.Equal(t, "rateLimit:v2:SR:user:7", key)
		return true, nil
	}
	t.Cleanup(func() {
		common.RedisEnabled = originRedisEnabled
		evalRedisRateLimit = originEval
	})

	handler := userRateLimitFactory(10, 60, "SR")

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/search", nil)
	ctx.Set("id", 7)

	handler(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.False(t, ctx.IsAborted())
}

func TestRedisRateLimiter_ReturnsInternalServerErrorOnEvaluatorFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originEval := evalRedisRateLimit
	evalRedisRateLimit = func(_ context.Context, _ string, _ time.Time, _ int, _ int64, _ time.Duration) (bool, error) {
		return false, errors.New("redis down")
	}
	t.Cleanup(func() {
		evalRedisRateLimit = originEval
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/rate-limit", nil)
	ctx.Request.RemoteAddr = "203.0.113.9:12345"

	redisRateLimiter(ctx, 3, 60, "GW")
	require.Equal(t, http.StatusInternalServerError, ctx.Writer.Status())
	require.True(t, ctx.IsAborted())
}

func TestPulseBenefitRateLimitUsesVerifiedServiceIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldRedisEnabled := common.RedisEnabled
	oldEnabled := common.PulseBenefitRateLimitEnable
	oldNum := common.PulseBenefitRateLimitNum
	oldDuration := common.PulseBenefitRateLimitDuration
	oldEval := evalRedisRateLimit
	common.RedisEnabled = true
	common.PulseBenefitRateLimitEnable = true
	common.PulseBenefitRateLimitNum = 600
	common.PulseBenefitRateLimitDuration = 60
	evalRedisRateLimit = func(_ context.Context, key string, _ time.Time, maxRequestNum int, duration int64, expiration time.Duration) (bool, error) {
		require.Equal(t, "rateLimit:v2:PB:service:pulse-settlement", key)
		require.Equal(t, 600, maxRequestNum)
		require.EqualValues(t, 60, duration)
		require.Equal(t, getRateLimitExpiration(duration), expiration)
		return true, nil
	}
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
		common.PulseBenefitRateLimitEnable = oldEnabled
		common.PulseBenefitRateLimitNum = oldNum
		common.PulseBenefitRateLimitDuration = oldDuration
		evalRedisRateLimit = oldEval
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "http://example.test/api/internal/pulse/benefits/grant", nil)
	ctx.Set("pulse_service_role", "pulse-settlement")
	ctx.Request.Header.Set("X-Pulse-Role", "attacker-controlled-value")

	PulseBenefitRateLimit()(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.False(t, ctx.IsAborted())
}

func TestGlobalAPIRateLimitSkipsPulseBenefitBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldRedisEnabled := common.RedisEnabled
	oldEnabled := common.GlobalApiRateLimitEnable
	oldEval := evalRedisRateLimit
	common.RedisEnabled = true
	common.GlobalApiRateLimitEnable = true
	evaluated := false
	evalRedisRateLimit = func(_ context.Context, _ string, _ time.Time, _ int, _ int64, _ time.Duration) (bool, error) {
		evaluated = true
		return false, nil
	}
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
		common.GlobalApiRateLimitEnable = oldEnabled
		evalRedisRateLimit = oldEval
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "http://example.test/api/internal/pulse/benefits/grant", nil)

	GlobalAPIRateLimit()(ctx)
	require.False(t, evaluated, "internal Pulse calls must use the service-identity limit")
	require.False(t, ctx.IsAborted())
}

func TestGlobalAPIRateLimitDoesNotSkipSimilarPublicPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldRedisEnabled := common.RedisEnabled
	oldEnabled := common.GlobalApiRateLimitEnable
	oldEval := evalRedisRateLimit
	common.RedisEnabled = true
	common.GlobalApiRateLimitEnable = true
	evalRedisRateLimit = func(_ context.Context, _ string, _ time.Time, _ int, _ int64, _ time.Duration) (bool, error) {
		return false, nil
	}
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
		common.GlobalApiRateLimitEnable = oldEnabled
		evalRedisRateLimit = oldEval
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "http://example.test/api/internal/pulse/benefits-public/grant", nil)

	GlobalAPIRateLimit()(ctx)
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.True(t, ctx.IsAborted())
}
