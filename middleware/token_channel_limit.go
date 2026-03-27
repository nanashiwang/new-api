package middleware

import (
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func GetAllowedTokenChannelIDs(c *gin.Context) []int {
	if !common.GetContextKeyBool(c, constant.ContextKeyTokenChannelLimitEnabled) {
		return nil
	}
	value, ok := common.GetContextKey(c, constant.ContextKeyTokenChannelLimit)
	if !ok {
		return []int{}
	}
	limitMap, ok := value.(map[int]bool)
	if !ok {
		return []int{}
	}
	ids := make([]int, 0, len(limitMap))
	for channelID, allowed := range limitMap {
		if !allowed || channelID <= 0 {
			continue
		}
		ids = append(ids, channelID)
	}
	sort.Ints(ids)
	return ids
}

func IsTokenChannelAllowed(c *gin.Context, channelID int) bool {
	if !common.GetContextKeyBool(c, constant.ContextKeyTokenChannelLimitEnabled) {
		return true
	}
	value, ok := common.GetContextKey(c, constant.ContextKeyTokenChannelLimit)
	if !ok {
		return false
	}
	limitMap, ok := value.(map[int]bool)
	if !ok {
		return false
	}
	return limitMap[channelID]
}
