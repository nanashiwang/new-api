package model

import (
	"net/url"
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

func GetEnabledChannelIDsByBaseURL(baseURL string, allowedChannels []int) ([]int, error) {
	normalizedBaseURL := NormalizeChannelBaseURL(baseURL)
	if normalizedBaseURL == "" {
		return nil, nil
	}

	allowedSet := make(map[int]bool, len(allowedChannels))
	for _, id := range allowedChannels {
		allowedSet[id] = true
	}

	filter := func(id int, channel *Channel) bool {
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			return false
		}
		if len(allowedSet) > 0 && !allowedSet[id] {
			return false
		}
		return NormalizeChannelBaseURL(channel.GetBaseURL()) == normalizedBaseURL
	}

	if common.MemoryCacheEnabled {
		channelSyncLock.RLock()
		defer channelSyncLock.RUnlock()

		ids := make([]int, 0)
		for id, channel := range channelsIDM {
			if filter(id, channel) {
				ids = append(ids, id)
			}
		}
		return ids, nil
	}

	query := DB.Model(&Channel{}).Where("status = ?", common.ChannelStatusEnabled)
	if len(allowedChannels) > 0 {
		query = query.Where("id IN ?", allowedChannels)
	}

	var channels []Channel
	if err := query.Find(&channels).Error; err != nil {
		return nil, err
	}
	ids := make([]int, 0)
	for i := range channels {
		if filter(channels[i].Id, &channels[i]) {
			ids = append(ids, channels[i].Id)
		}
	}
	return ids, nil
}

func NormalizeChannelBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.TrimRight(raw, "/")
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.ToLower(raw)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
