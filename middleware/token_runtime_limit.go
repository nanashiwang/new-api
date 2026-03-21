package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/limiter"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

var tokenConcurrencyCounters sync.Map
var tokenWindowRateLimiter common.InMemoryRateLimiter

type TokenRuntimeLimitError struct {
	StatusCode int
	Message    string
}

func (e *TokenRuntimeLimitError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func init() {
	go cleanupIdleConcurrencyCounters()
}

func cleanupIdleConcurrencyCounters() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		tokenConcurrencyCounters.Range(func(key, value any) bool {
			counter := value.(*tokenConcurrencyCounter)
			if atomic.LoadInt64(&counter.value) == 0 {
				tokenConcurrencyCounters.Delete(key)
			}
			return true
		})
	}
}

type tokenConcurrencyCounter struct {
	value int64
}

func getTokenConcurrencyCounter(key string) *tokenConcurrencyCounter {
	if counter, ok := tokenConcurrencyCounters.Load(key); ok {
		return counter.(*tokenConcurrencyCounter)
	}
	counter := &tokenConcurrencyCounter{}
	actual, _ := tokenConcurrencyCounters.LoadOrStore(key, counter)
	return actual.(*tokenConcurrencyCounter)
}

func TokenRuntimeLimit() func(c *gin.Context) {
	return func(c *gin.Context) {
		releaseRuntimeLimit, err := AcquireTokenRuntimeLimit(c)
		if err != nil {
			if runtimeErr, ok := err.(*TokenRuntimeLimitError); ok {
				abortWithOpenAiMessage(c, runtimeErr.StatusCode, runtimeErr.Message)
				return
			}
			abortWithOpenAiMessage(c, http.StatusInternalServerError, err.Error())
			return
		}
		defer releaseRuntimeLimit()
		c.Next()
	}
}

func AcquireTokenRuntimeLimit(c *gin.Context) (func(), error) {
	tokenID := common.GetContextKeyInt(c, constant.ContextKeyTokenId)
	if tokenID <= 0 {
		return func() {}, nil
	}

	releaseConcurrency := func() {}
	maxConcurrency := common.GetContextKeyInt(c, constant.ContextKeyTokenMaxConcurrency)
	if maxConcurrency > 0 {
		releaser, allowed, err := acquireTokenConcurrency(tokenID, maxConcurrency)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, &TokenRuntimeLimitError{
				StatusCode: http.StatusTooManyRequests,
				Message:    fmt.Sprintf("当前令牌并发已达到上限 %d", maxConcurrency),
			}
		}
		releaseConcurrency = releaser
	}

	windowLimit := common.GetContextKeyInt(c, constant.ContextKeyTokenWindowRequestLimit)
	windowSecondsValue, _ := common.GetContextKey(c, constant.ContextKeyTokenWindowSeconds)
	windowSeconds, _ := windowSecondsValue.(int64)
	if windowLimit > 0 && windowSeconds > 0 {
		allowed, err := checkTokenWindowLimit(tokenID, windowLimit, windowSeconds)
		if err != nil {
			releaseConcurrency()
			return nil, err
		}
		if !allowed {
			releaseConcurrency()
			return nil, &TokenRuntimeLimitError{
				StatusCode: http.StatusTooManyRequests,
				Message:    fmt.Sprintf("当前令牌在 %d 秒内的请求数已达到上限 %d", windowSeconds, windowLimit),
			}
		}
	}

	return releaseConcurrency, nil
}

func acquireTokenConcurrency(tokenID int, maxConcurrency int) (func(), bool, error) {
	if common.RedisEnabled {
		ctx := context.Background()
		key := fmt.Sprintf("token:concurrency:%d", tokenID)
		value, err := common.RDB.Incr(ctx, key).Result()
		if err != nil {
			return nil, false, err
		}
		if _, err = common.RDB.Expire(ctx, key, 30*time.Minute).Result(); err != nil {
			return nil, false, err
		}
		if value > int64(maxConcurrency) {
			_, _ = common.RDB.Decr(ctx, key).Result()
			return nil, false, nil
		}
		return func() {
			_, _ = common.RDB.Decr(ctx, key).Result()
		}, true, nil
	}

	key := fmt.Sprintf("token:concurrency:%d", tokenID)
	counter := getTokenConcurrencyCounter(key)
	next := atomic.AddInt64(&counter.value, 1)
	if next > int64(maxConcurrency) {
		atomic.AddInt64(&counter.value, -1)
		return nil, false, nil
	}
	return func() {
		atomic.AddInt64(&counter.value, -1)
	}, true, nil
}

func checkTokenWindowLimit(tokenID int, maxRequestNum int, duration int64) (bool, error) {
	expiration := getTokenWindowLimitExpiration(duration)
	if common.RedisEnabled {
		ctx := context.Background()
		key := fmt.Sprintf("token:window:%d", tokenID)
		return limiter.AllowStrictSlidingWindow(
			ctx,
			common.RDB,
			key,
			time.Duration(duration)*time.Second,
			int64(maxRequestNum),
			buildTokenWindowRequestNonce(),
			expiration,
		)
	}

	tokenWindowRateLimiter.Init(expiration)
	key := fmt.Sprintf("token:window:%d", tokenID)
	return tokenWindowRateLimiter.Request(key, maxRequestNum, duration), nil
}

func buildTokenWindowRequestNonce() string {
	return common.GetUUID()
}

// QueryTokenConcurrency returns the current concurrency count for a token.
func QueryTokenConcurrency(tokenID int) (int64, error) {
	key := fmt.Sprintf("token:concurrency:%d", tokenID)
	if common.RedisEnabled {
		ctx := context.Background()
		val, err := common.RDB.Get(ctx, key).Int64()
		if err != nil {
			if err.Error() == "redis: nil" {
				return 0, nil
			}
			return 0, err
		}
		return val, nil
	}

	if counter, ok := tokenConcurrencyCounters.Load(key); ok {
		return atomic.LoadInt64(&counter.(*tokenConcurrencyCounter).value), nil
	}
	return 0, nil
}

// QueryTokenWindowStatus returns the current window usage for a token.
// Returns: count of requests in window, window end timestamp (ms), server now (ms), error.
func QueryTokenWindowStatus(tokenID int, windowSeconds int64) (count int64, windowEndMs int64, serverNowMs int64, err error) {
	if windowSeconds <= 0 {
		return 0, 0, time.Now().UnixMilli(), nil
	}

	key := fmt.Sprintf("token:window:%d", tokenID)
	window := time.Duration(windowSeconds) * time.Second

	if common.RedisEnabled {
		ctx := context.Background()
		c, oldestMs, nowMs, err := limiter.QuerySlidingWindowStatus(ctx, common.RDB, key, window)
		if err != nil {
			return 0, 0, 0, err
		}
		var wEnd int64
		if oldestMs > 0 {
			wEnd = oldestMs + windowSeconds*1000
		}
		return c, wEnd, nowMs, nil
	}

	// In-memory fallback
	nowMs := time.Now().UnixMilli()
	inMemCount, oldestTs := tokenWindowRateLimiter.QueryStatus(key, windowSeconds)
	var wEnd int64
	if oldestTs > 0 {
		wEnd = oldestTs*1000 + windowSeconds*1000
	}
	return int64(inMemCount), wEnd, nowMs, nil
}

func getTokenWindowLimitExpiration(duration int64) time.Duration {
	if duration <= 0 {
		return common.RateLimitKeyExpirationDuration
	}
	windowExpiration := time.Duration(duration) * time.Second
	if windowExpiration < common.RateLimitKeyExpirationDuration {
		return common.RateLimitKeyExpirationDuration
	}
	return windowExpiration
}
