package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

const (
	EmailVerificationRateLimitMark      = "EV"
	EmailVerificationMaxRequests        = 2 // 30秒内最多2次/IP
	EmailVerificationDuration           = 30
	EmailVerificationEmailMaxRequests   = 3 // 10分钟内最多3次/邮箱
	EmailVerificationEmailDuration      = 10 * 60
	EmailVerificationIPEmailMaxRequests = 2 // 60秒内最多2次/IP+邮箱
	EmailVerificationIPEmailDuration    = 60
)

type emailVerificationLimitRule struct {
	key         string
	maxRequests int
	duration    int64
}

func emailVerificationLimitRules(c *gin.Context) []emailVerificationLimitRule {
	clientIP := c.ClientIP()
	rules := []emailVerificationLimitRule{
		{
			key:         "emailVerification:" + EmailVerificationRateLimitMark + ":ip:" + clientIP,
			maxRequests: EmailVerificationMaxRequests,
			duration:    EmailVerificationDuration,
		},
	}

	email := strings.ToLower(strings.TrimSpace(c.Query("email")))
	if email == "" {
		return rules
	}

	emailHash := common.GenerateHMAC("email-verification:" + email)
	rules = append(rules,
		emailVerificationLimitRule{
			key:         "emailVerification:" + EmailVerificationRateLimitMark + ":email:" + emailHash,
			maxRequests: EmailVerificationEmailMaxRequests,
			duration:    EmailVerificationEmailDuration,
		},
		emailVerificationLimitRule{
			key:         "emailVerification:" + EmailVerificationRateLimitMark + ":ip-email:" + common.GenerateHMAC(clientIP+"|"+email),
			maxRequests: EmailVerificationIPEmailMaxRequests,
			duration:    EmailVerificationIPEmailDuration,
		},
	)
	return rules
}

func redisCheckEmailVerificationLimit(ctx context.Context, rule emailVerificationLimitRule) (bool, int64, error) {
	rdb := common.RDB
	count, err := rdb.Incr(ctx, rule.key).Result()
	if err != nil {
		return false, 0, err
	}

	// 第一次设置键时设置过期时间
	if count == 1 {
		_ = rdb.Expire(ctx, rule.key, time.Duration(rule.duration)*time.Second).Err()
	}

	// 检查是否超出限制
	if count <= int64(rule.maxRequests) {
		return true, 0, nil
	}

	// 获取剩余等待时间
	ttl, err := rdb.TTL(ctx, rule.key).Result()
	waitSeconds := rule.duration
	if err == nil && ttl > 0 {
		waitSeconds = int64(ttl.Seconds())
	}
	return false, waitSeconds, nil
}

func redisEmailVerificationRateLimiter(c *gin.Context) {
	ctx := context.Background()
	for _, rule := range emailVerificationLimitRules(c) {
		allowed, waitSeconds, err := redisCheckEmailVerificationLimit(ctx, rule)
		if err != nil {
			// Redis 异常时退回内存限流，避免验证码接口完全不可用。
			memoryEmailVerificationRateLimiter(c)
			return
		}
		if !allowed {
			abortEmailVerificationRateLimit(c, waitSeconds)
			return
		}
	}
	c.Next()
}

func abortEmailVerificationRateLimit(c *gin.Context, waitSeconds int64) {
	if waitSeconds <= 0 {
		waitSeconds = int64(EmailVerificationDuration)
	}
	c.JSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"message": fmt.Sprintf("发送过于频繁，请等待 %d 秒后再试", waitSeconds),
	})
	c.Abort()
}

func memoryEmailVerificationRateLimiter(c *gin.Context) {
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	for _, rule := range emailVerificationLimitRules(c) {
		if !inMemoryRateLimiter.Request(rule.key, rule.maxRequests, rule.duration) {
			abortEmailVerificationRateLimit(c, rule.duration)
			return
		}
	}

	c.Next()
}

func EmailVerificationRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.RedisEnabled {
			redisEmailVerificationRateLimiter(c)
		} else {
			inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
			memoryEmailVerificationRateLimiter(c)
		}
	}
}
