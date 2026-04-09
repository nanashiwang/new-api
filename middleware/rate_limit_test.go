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
