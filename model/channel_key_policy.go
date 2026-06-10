package model

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	multiKeyCooldownKeyPrefix  = "mk:cooldown"
	multiKeyCooldownFailPrefix = "mk:cooldown:fails"
	multiKeyStickyKeyPrefix    = "mk:sticky"

	// multiKeyCooldownMaxSeconds 指数退避冷却的封顶时长（2 小时）。
	// 连续额度失败：60s → 120s → … → 7200s，避免余额耗尽的死 Key 每分钟
	// 拿真实用户请求当探针。
	multiKeyCooldownMaxSeconds = 7200
	// 失败计数器的滑动窗口取封顶的 2 倍：只要 Key 仍在被探测且持续失败，
	// 计数就不会过期；恢复正常（或长时间无流量）后计数自然消失，下次
	// 失败重新从基础时长起步。
	multiKeyCooldownFailCounterTTL = 2 * multiKeyCooldownMaxSeconds * time.Second
)

func buildMultiKeyCooldownKey(channelId int, keyIndex int) string {
	return fmt.Sprintf("%s:%d:%d", multiKeyCooldownKeyPrefix, channelId, keyIndex)
}

func buildMultiKeyCooldownFailKey(channelId int, keyIndex int) string {
	return fmt.Sprintf("%s:%d:%d", multiKeyCooldownFailPrefix, channelId, keyIndex)
}

func buildMultiKeyStickyKey(tokenId int, channelId int, modelName string) string {
	model := strings.TrimSpace(modelName)
	if model == "" {
		model = "default"
	}
	// 模型名做哈希，避免 Redis key 过长。
	modelHash := common.GenerateHMAC(model)
	return fmt.Sprintf("%s:%d:%d:%s", multiKeyStickyKeyPrefix, tokenId, channelId, modelHash)
}

func IsMultiKeyInCooldown(channelId int, keyIndex int) bool {
	if !common.RedisEnabled || channelId <= 0 || keyIndex < 0 {
		return false
	}
	_, err := common.RedisGet(buildMultiKeyCooldownKey(channelId, keyIndex))
	return err == nil
}

func SetMultiKeyCooldown(channelId int, keyIndex int, reason string, seconds int) error {
	if !common.RedisEnabled || channelId <= 0 || keyIndex < 0 {
		return nil
	}
	if seconds <= 0 {
		seconds = common.MultiKeyCooldownSeconds
	}
	if seconds <= 0 {
		return nil
	}
	if reason == "" {
		reason = "quota_related_error"
	}
	return common.RedisSet(
		buildMultiKeyCooldownKey(channelId, keyIndex),
		reason,
		time.Duration(seconds)*time.Second,
	)
}

// SetMultiKeyCooldownWithBackoff 按连续失败次数做指数退避冷却：
// 第 n 次失败的冷却时长 = MultiKeyCooldownSeconds * 2^(n-1)，封顶 multiKeyCooldownMaxSeconds。
// 返回实际应用的冷却秒数（Redis 不可用等跳过场景返回 0）。
func SetMultiKeyCooldownWithBackoff(channelId int, keyIndex int, reason string) (int, error) {
	if !common.RedisEnabled || channelId <= 0 || keyIndex < 0 {
		return 0, nil
	}
	base := common.MultiKeyCooldownSeconds
	if base <= 0 {
		return 0, nil
	}

	// 失败计数 +1，并刷新滑动窗口（事务保证两步原子）。
	ctx := context.Background()
	txn := common.RDB.TxPipeline()
	incrCmd := txn.Incr(ctx, buildMultiKeyCooldownFailKey(channelId, keyIndex))
	txn.Expire(ctx, buildMultiKeyCooldownFailKey(channelId, keyIndex), multiKeyCooldownFailCounterTTL)
	if _, err := txn.Exec(ctx); err != nil {
		// 计数失败时退回基础时长，保证冷却本身不丢。
		return base, SetMultiKeyCooldown(channelId, keyIndex, reason, base)
	}

	fails := incrCmd.Val()
	seconds := base
	for i := int64(1); i < fails && seconds < multiKeyCooldownMaxSeconds; i++ {
		seconds *= 2
	}
	if seconds > multiKeyCooldownMaxSeconds {
		seconds = multiKeyCooldownMaxSeconds
	}
	return seconds, SetMultiKeyCooldown(channelId, keyIndex, reason, seconds)
}

// ResetMultiKeyCooldownBackoff 清除连续失败计数：Key 成功完成一次调用后，
// 下次失败的冷却时长回归基础值，避免历史失败抬高健康 Key 的冷却起点。
func ResetMultiKeyCooldownBackoff(channelId int, keyIndex int) error {
	if !common.RedisEnabled || channelId <= 0 || keyIndex < 0 {
		return nil
	}
	return common.RedisDel(buildMultiKeyCooldownFailKey(channelId, keyIndex))
}

func ClearMultiKeyCooldown(channelId int, keyIndex int) error {
	if !common.RedisEnabled || channelId <= 0 || keyIndex < 0 {
		return nil
	}
	// 手动解除冷却同时清掉失败计数，让下次失败从基础时长重新起步。
	_ = ResetMultiKeyCooldownBackoff(channelId, keyIndex)
	return common.RedisDel(buildMultiKeyCooldownKey(channelId, keyIndex))
}

func ClearAllMultiKeyCooldown(channelId int, keyCount int) error {
	if !common.RedisEnabled || channelId <= 0 || keyCount <= 0 {
		return nil
	}
	for i := 0; i < keyCount; i++ {
		if err := ClearMultiKeyCooldown(channelId, i); err != nil {
			return err
		}
	}
	return nil
}

func GetMultiKeyStickyIndex(tokenId int, channelId int, modelName string) (int, bool) {
	if !common.RedisEnabled || tokenId <= 0 || channelId <= 0 {
		return 0, false
	}
	val, err := common.RedisGet(buildMultiKeyStickyKey(tokenId, channelId, modelName))
	if err != nil {
		return 0, false
	}
	idx, convErr := strconv.Atoi(strings.TrimSpace(val))
	if convErr != nil || idx < 0 {
		return 0, false
	}
	return idx, true
}

func SetMultiKeyStickyIndex(tokenId int, channelId int, modelName string, keyIndex int, seconds int) error {
	if !common.RedisEnabled || tokenId <= 0 || channelId <= 0 || keyIndex < 0 {
		return nil
	}
	if seconds <= 0 {
		seconds = common.MultiKeyStickySeconds
	}
	if seconds <= 0 {
		return nil
	}
	return common.RedisSet(
		buildMultiKeyStickyKey(tokenId, channelId, modelName),
		strconv.Itoa(keyIndex),
		time.Duration(seconds)*time.Second,
	)
}
