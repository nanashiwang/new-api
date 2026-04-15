package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	multiKeyCooldownKeyPrefix = "mk:cooldown"
	multiKeyStickyKeyPrefix   = "mk:sticky"
)

func buildMultiKeyCooldownKey(channelId int, keyIndex int) string {
	return fmt.Sprintf("%s:%d:%d", multiKeyCooldownKeyPrefix, channelId, keyIndex)
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

func ClearMultiKeyCooldown(channelId int, keyIndex int) error {
	if !common.RedisEnabled || channelId <= 0 || keyIndex < 0 {
		return nil
	}
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
