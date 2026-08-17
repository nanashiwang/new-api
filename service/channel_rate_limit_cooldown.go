package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const (
	channelRateLimitCooldownPrefix = "channel:rate_limit_cooldown"
	defaultRateLimitCooldown       = time.Minute
	minRateLimitCooldown           = time.Second
	maxRateLimitCooldown           = time.Hour
	channelRateLimitCooldownReason = "upstream_rate_limit"
)

var (
	channelRateLimitCooldownMu    sync.Mutex
	channelRateLimitLocalCooldown = map[int]time.Time{}
)

// RecordChannelRateLimitCooldown 根据上游 429 冷却当前渠道或当前多 Key 账号。
func RecordChannelRateLimitCooldown(c *gin.Context, channel *model.Channel, apiErr *types.NewAPIError) time.Duration {
	if channel == nil || channel.Id <= 0 || !IsUpstreamRateLimitError(apiErr) {
		return 0
	}
	cooldown := normalizeRateLimitCooldown(apiErr.RetryAfter)
	apiErr.RetryAfter = cooldown

	if channel.ChannelInfo.IsMultiKey && c != nil {
		if keyIndex, ok := common.GetContextKeyType[int](c, constant.ContextKeyChannelMultiKeyIndex); ok && keyIndex >= 0 {
			seconds := retryAfterSeconds(cooldown)
			if applied, err := model.SetMultiKeyCooldownAtLeast(channel.Id, keyIndex, channelRateLimitCooldownReason, seconds); err == nil && applied > 0 {
				appliedCooldown := time.Duration(applied) * time.Second
				apiErr.RetryAfter = appliedCooldown
				return appliedCooldown
			}
		}
	}

	appliedCooldown := extendChannelRateLimitLocalCooldown(channel.Id, cooldown, time.Now())
	apiErr.RetryAfter = appliedCooldown
	return appliedCooldown
}

func IsChannelRateLimitCoolingDown(channel *model.Channel) bool {
	return channel != nil && ChannelRateLimitCooldownRemaining(channel.Id) > 0
}

// ChannelRateLimitCooldownRemaining 返回渠道级 429 冷却的剩余时间。
func ChannelRateLimitCooldownRemaining(channelID int) time.Duration {
	if channelID <= 0 {
		return 0
	}
	now := time.Now()
	remaining := channelRateLimitLocalRemaining(channelID, now)
	if common.RedisEnabled && common.RDB != nil {
		if ttl, err := common.RDB.PTTL(context.Background(), channelRateLimitCooldownKey(channelID)).Result(); err == nil && ttl > remaining {
			remaining = ttl
		}
	}
	return remaining
}

func normalizeRateLimitCooldown(retryAfter time.Duration) time.Duration {
	if retryAfter <= 0 {
		return defaultRateLimitCooldown
	}
	if retryAfter < minRateLimitCooldown {
		return minRateLimitCooldown
	}
	if retryAfter > maxRateLimitCooldown {
		return maxRateLimitCooldown
	}
	return retryAfter
}

func extendChannelRateLimitLocalCooldown(channelID int, cooldown time.Duration, now time.Time) time.Duration {
	if channelID <= 0 || cooldown <= 0 {
		return 0
	}
	until := now.Add(cooldown)
	channelRateLimitCooldownMu.Lock()
	if existing := channelRateLimitLocalCooldown[channelID]; existing.After(until) {
		until = existing
	} else {
		channelRateLimitLocalCooldown[channelID] = until
	}
	channelRateLimitCooldownMu.Unlock()

	if common.RedisEnabled && common.RDB != nil {
		_, _ = common.RDB.Eval(context.Background(), `
local current_ttl = redis.call('PTTL', KEYS[1])
local requested_ttl = tonumber(ARGV[2])
if current_ttl < requested_ttl then
    redis.call('PSETEX', KEYS[1], requested_ttl, ARGV[1])
    return requested_ttl
end
return current_ttl
`, []string{channelRateLimitCooldownKey(channelID)}, channelRateLimitCooldownReason, cooldown.Milliseconds()).Result()
	}
	remaining := time.Until(until)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func channelRateLimitLocalRemaining(channelID int, now time.Time) time.Duration {
	channelRateLimitCooldownMu.Lock()
	defer channelRateLimitCooldownMu.Unlock()
	until, ok := channelRateLimitLocalCooldown[channelID]
	if !ok {
		return 0
	}
	if !now.Before(until) {
		delete(channelRateLimitLocalCooldown, channelID)
		return 0
	}
	return until.Sub(now)
}

func channelRateLimitCooldownKey(channelID int) string {
	return fmt.Sprintf("%s:%d", channelRateLimitCooldownPrefix, channelID)
}

func retryAfterSeconds(retryAfter time.Duration) int {
	if retryAfter <= 0 {
		return 1
	}
	return int((retryAfter + time.Second - 1) / time.Second)
}

func resetChannelRateLimitCooldownForTest() {
	channelRateLimitCooldownMu.Lock()
	channelRateLimitLocalCooldown = map[int]time.Time{}
	channelRateLimitCooldownMu.Unlock()
}
