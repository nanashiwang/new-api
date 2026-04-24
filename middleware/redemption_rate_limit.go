package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

const (
	redemptionSuccessMark = "RDS"
	redemptionFailureMark = "RDF"
	// redemptionRateLimitBufferSize 是 Record 路径上调用 InMemoryRateLimiter.Request 时
	// 允许队列溢出的缓冲。配置的 max + buffer 作为内存队列上限，既保证 Check 到达阈值之前
	// 总能写入，又防止 queue 无限增长。
	redemptionRateLimitBufferSize = 8
)

// CheckRedemptionRateLimit 在 controller.Redeem 入口处调用。
// 只查询计数、不写入；已达阈值即 abort 并返回 429。
// 返回 false 表示已 abort，调用方应立即 return。
func CheckRedemptionRateLimit(c *gin.Context) bool {
	if !setting.RedemptionRateLimitEnabled {
		return true
	}
	userId := c.GetInt("id")
	if userId == 0 {
		return true
	}
	duration := int64(setting.RedemptionRateLimitDurationSeconds)
	if duration <= 0 {
		return true
	}

	if max := setting.RedemptionRateLimitSuccessCount; max > 0 {
		if queryRedemptionCount(redemptionSuccessMark, userId, duration) >= max {
			abortWithRateLimit(c, max, duration, redemptionSuccessMark)
			return false
		}
	}
	if max := setting.RedemptionRateLimitFailureCount; max > 0 {
		if queryRedemptionCount(redemptionFailureMark, userId, duration) >= max {
			abortWithRateLimit(c, max, duration, redemptionFailureMark)
			return false
		}
	}
	return true
}

// RecordRedemptionAttempt 兑换结果落定后调用：success=true 记成功次数，false 记失败次数。
// 若对应方向未启用（max=0），不记录，避免内存/Redis 中累积无用数据。
func RecordRedemptionAttempt(c *gin.Context, success bool) {
	if !setting.RedemptionRateLimitEnabled {
		return
	}
	userId := c.GetInt("id")
	if userId == 0 {
		return
	}
	duration := int64(setting.RedemptionRateLimitDurationSeconds)
	if duration <= 0 {
		return
	}

	var max int
	mark := redemptionFailureMark
	if success {
		mark = redemptionSuccessMark
		max = setting.RedemptionRateLimitSuccessCount
	} else {
		max = setting.RedemptionRateLimitFailureCount
	}
	if max <= 0 {
		return
	}
	recordRedemptionHit(mark, userId, duration, max)
}

func queryRedemptionCount(mark string, userId int, duration int64) int {
	if common.RedisEnabled {
		ctx := context.Background()
		key := buildRateLimitRedisKey(mark, fmt.Sprintf("user:%d", userId))
		// 用左开区间排除恰好落在窗口起点的过期成员，保持与 Lua 脚本中
		// ZREMRANGEBYSCORE(-inf, now-window) 的剔除语义一致。
		windowStart := time.Now().UnixMilli() - duration*int64(time.Second/time.Millisecond)
		n, err := common.RDB.ZCount(ctx, key, fmt.Sprintf("(%d", windowStart), "+inf").Result()
		if err != nil {
			return 0
		}
		return int(n)
	}
	inMemoryRateLimiter.Init(getRateLimitExpiration(duration))
	key := fmt.Sprintf("%s:user:%d", mark, userId)
	count, _ := inMemoryRateLimiter.QueryStatus(key, duration)
	return count
}

func recordRedemptionHit(mark string, userId int, duration int64, maxCount int) {
	if common.RedisEnabled {
		ctx := context.Background()
		key := buildRateLimitRedisKey(mark, fmt.Sprintf("user:%d", userId))
		// Lua 脚本内部会先 ZREMRANGEBYSCORE 清过期；用 max+buffer 确保必写入。
		if _, err := evalRedisRateLimit(ctx, key, time.Now(), maxCount+redemptionRateLimitBufferSize, duration, getRateLimitExpiration(duration)); err != nil {
			common.SysError(fmt.Sprintf("failed to record redemption rate limit hit: %v", err))
		}
		return
	}
	inMemoryRateLimiter.Init(getRateLimitExpiration(duration))
	key := fmt.Sprintf("%s:user:%d", mark, userId)
	// 传 max+buffer：未到上限时 Request 直接 append；触达上限则触发队头过期剔除分支。
	// 这样 queue 长度天然有界，避免因超大 maxCount 退化为"只增不减"的内存泄漏。
	_ = inMemoryRateLimiter.Request(key, maxCount+redemptionRateLimitBufferSize, duration)
}
