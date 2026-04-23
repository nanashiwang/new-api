package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var timeFormat = "2006-01-02T15:04:05.000Z"

var inMemoryRateLimiter common.InMemoryRateLimiter

var redisSlidingWindowRateLimitScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local max_count = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])
local member = ARGV[5]

redis.call("ZREMRANGEBYSCORE", key, "-inf", now - window)

local current = redis.call("ZCARD", key)
if current >= max_count then
	redis.call("EXPIRE", key, ttl)
	return 0
end

redis.call("ZADD", key, now, member)
redis.call("EXPIRE", key, ttl)
return 1
`)

var evalRedisRateLimit = runRedisRateLimit

var defNext = func(c *gin.Context) {
	c.Next()
}

// abortWithRateLimit writes standard rate-limit headers (Retry-After,
// X-RateLimit-*) and a JSON error body before aborting with HTTP 429. This
// allows well-behaved clients (OpenAI SDK, LangChain, etc.) to back off
// instead of immediately retrying and amplifying load.
func abortWithRateLimit(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	retryAfter := strconv.FormatInt(duration, 10)
	c.Header("Retry-After", retryAfter)
	c.Header("X-RateLimit-Limit", strconv.Itoa(maxRequestNum))
	c.Header("X-RateLimit-Window", retryAfter)
	c.Header("X-RateLimit-Scope", mark)
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error": gin.H{
			"code":        "rate_limited",
			"message":     fmt.Sprintf("Rate limit exceeded (%s). Retry after %ss.", mark, retryAfter),
			"scope":       mark,
			"retry_after": duration,
		},
	})
}

func redisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	ctx := context.Background()
	expiration := getRateLimitExpiration(duration)
	allowed, err := evalRedisRateLimit(ctx, buildRateLimitRedisKey(mark, c.ClientIP()), time.Now(), maxRequestNum, duration, expiration)
	if err != nil {
		fmt.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}
	if !allowed {
		abortWithRateLimit(c, maxRequestNum, duration, mark)
	}
}

func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	key := mark + c.ClientIP()
	if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
		abortWithRateLimit(c, maxRequestNum, duration, mark)
		return
	}
}

func rateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			redisRateLimiter(c, maxRequestNum, duration, mark)
		}
	} else {
		// It's safe to call multi times.
		inMemoryRateLimiter.Init(getRateLimitExpiration(duration))
		return func(c *gin.Context) {
			memoryRateLimiter(c, maxRequestNum, duration, mark)
		}
	}
}

func GlobalWebRateLimit() func(c *gin.Context) {
	if common.GlobalWebRateLimitEnable {
		return rateLimitFactory(common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration, "GW")
	}
	return defNext
}

func GlobalAPIRateLimit() func(c *gin.Context) {
	if common.GlobalApiRateLimitEnable {
		return rateLimitFactory(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration, "GA")
	}
	return defNext
}

func PublicTokenUsageRateLimit() func(c *gin.Context) {
	if common.PublicTokenUsageRateLimitEnable {
		return rateLimitFactory(
			common.PublicTokenUsageRateLimitNum,
			common.PublicTokenUsageRateLimitDuration,
			"PU",
		)
	}
	return defNext
}

func CriticalRateLimit() func(c *gin.Context) {
	if common.CriticalRateLimitEnable {
		return rateLimitFactory(common.CriticalRateLimitNum, common.CriticalRateLimitDuration, "CT")
	}
	return defNext
}

func RegisterRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.RegisterRateLimitNum, common.RegisterRateLimitDuration, "RG")
}

func DownloadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.DownloadRateLimitNum, common.DownloadRateLimitDuration, "DW")
}

func UploadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.UploadRateLimitNum, common.UploadRateLimitDuration, "UP")
}

// userRateLimitFactory creates a rate limiter keyed by authenticated user ID
// instead of client IP, making it resistant to proxy rotation attacks.
// Must be used AFTER authentication middleware (UserAuth).
func userRateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			userId := c.GetInt("id")
			if userId == 0 {
				c.Status(http.StatusUnauthorized)
				c.Abort()
				return
			}
			key := buildRateLimitRedisKey(mark, fmt.Sprintf("user:%d", userId))
			userRedisRateLimiter(c, maxRequestNum, duration, mark, key)
		}
	}
	// It's safe to call multi times.
	inMemoryRateLimiter.Init(getRateLimitExpiration(duration))
	return func(c *gin.Context) {
		userId := c.GetInt("id")
		if userId == 0 {
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}
		key := fmt.Sprintf("%s:user:%d", mark, userId)
		if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
			abortWithRateLimit(c, maxRequestNum, duration, mark)
			return
		}
	}
}

// userRedisRateLimiter is like redisRateLimiter but accepts a pre-built key
// (to support user-ID-based keys).
func userRedisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string, key string) {
	ctx := context.Background()
	expiration := getRateLimitExpiration(duration)
	allowed, err := evalRedisRateLimit(ctx, key, time.Now(), maxRequestNum, duration, expiration)
	if err != nil {
		fmt.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}
	if !allowed {
		abortWithRateLimit(c, maxRequestNum, duration, mark)
	}
}

func buildRateLimitRedisKey(mark string, subject string) string {
	return fmt.Sprintf("rateLimit:v2:%s:%s", mark, subject)
}

func runRedisRateLimit(ctx context.Context, key string, now time.Time, maxRequestNum int, duration int64, expiration time.Duration) (bool, error) {
	if maxRequestNum <= 0 {
		return true, nil
	}

	windowMilliseconds := duration * int64(time.Second/time.Millisecond)
	if windowMilliseconds <= 0 {
		windowMilliseconds = int64(time.Second / time.Millisecond)
	}
	ttlSeconds := int64(expiration / time.Second)
	if ttlSeconds <= 0 {
		ttlSeconds = 1
	}

	member := strconv.FormatInt(now.UnixNano(), 10)
	result, err := redisSlidingWindowRateLimitScript.Run(
		ctx,
		common.RDB,
		[]string{key},
		now.UnixMilli(),
		windowMilliseconds,
		maxRequestNum,
		ttlSeconds,
		member,
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func getRateLimitExpiration(duration int64) time.Duration {
	if duration <= 0 {
		return common.RateLimitKeyExpirationDuration
	}
	windowExpiration := time.Duration(duration) * time.Second
	if windowExpiration < common.RateLimitKeyExpirationDuration {
		return common.RateLimitKeyExpirationDuration
	}
	return windowExpiration
}

// SearchRateLimit returns a per-user rate limiter for search endpoints.
// 10 requests per 60 seconds per user (by user ID, not IP).
func SearchRateLimit() func(c *gin.Context) {
	return userRateLimitFactory(common.SearchRateLimitNum, common.SearchRateLimitDuration, "SR")
}

func ProfitBoardQueryRateLimit() gin.HandlerFunc {
	return rateLimitFactory(10, 60, "PB")
}
