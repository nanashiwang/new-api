package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

func GetEnabledChannelIDsByTag(tag string, allowedChannels []int) ([]int, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, nil
	}

	if common.MemoryCacheEnabled {
		channelSyncLock.RLock()
		defer channelSyncLock.RUnlock()

		allowedSet := make(map[int]bool, len(allowedChannels))
		for _, id := range allowedChannels {
			allowedSet[id] = true
		}

		ids := make([]int, 0)
		for id, channel := range channelsIDM {
			if channel == nil || channel.Status != common.ChannelStatusEnabled {
				continue
			}
			if channel.GetTag() != tag {
				continue
			}
			if len(allowedSet) > 0 && !allowedSet[id] {
				continue
			}
			ids = append(ids, id)
		}
		return ids, nil
	}

	query := DB.Model(&Channel{}).
		Where("status = ?", common.ChannelStatusEnabled).
		Where("tag = ?", tag)
	if len(allowedChannels) > 0 {
		query = query.Where("id IN ?", allowedChannels)
	}

	var ids []int
	if err := query.Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
