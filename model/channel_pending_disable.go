package model

import (
	"fmt"
	"sort"

	"github.com/QuantumNous/new-api/common"
)

const (
	channelPendingDisableUntilKey  = "pre_disable_wait_until"
	channelPendingDisableReasonKey = "pre_disable_wait_reason"
)

func (channel *Channel) GetPendingDisableUntil() int64 {
	if channel == nil {
		return 0
	}
	info := channel.GetOtherInfo()
	if info == nil {
		return 0
	}
	switch v := info[channelPendingDisableUntilKey].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func (channel *Channel) GetPendingDisableReason() string {
	if channel == nil {
		return ""
	}
	info := channel.GetOtherInfo()
	if info == nil {
		return ""
	}
	return common.Interface2String(info[channelPendingDisableReasonKey])
}

func (channel *Channel) HasPendingDisable() bool {
	return channel.GetPendingDisableUntil() > 0
}

func (channel *Channel) SetPendingDisable(until int64, reason string) {
	if channel == nil {
		return
	}
	info := channel.GetOtherInfo()
	info[channelPendingDisableUntilKey] = until
	info[channelPendingDisableReasonKey] = reason
	channel.SetOtherInfo(info)
}

func (channel *Channel) ClearPendingDisable() {
	if channel == nil {
		return
	}
	info := channel.GetOtherInfo()
	delete(info, channelPendingDisableUntilKey)
	delete(info, channelPendingDisableReasonKey)
	channel.SetOtherInfo(info)
}

func (channel *Channel) IsPendingDisableUnavailable() bool {
	return channel != nil && channel.HasPendingDisable()
}

func (channel *Channel) GetKeyIndex(usingKey string) int {
	if channel == nil || usingKey == "" {
		return -1
	}
	keys := channel.GetKeys()
	for i, key := range keys {
		if key == usingKey {
			return i
		}
	}
	return -1
}

func (channel *Channel) GetKeyByIndex(index int) (string, error) {
	if channel == nil {
		return "", fmt.Errorf("channel is nil")
	}
	keys := channel.GetKeys()
	if index < 0 || index >= len(keys) {
		return "", fmt.Errorf("key index out of range")
	}
	return keys[index], nil
}

func (channel *Channel) HasPendingDisableKey(index int) bool {
	if channel == nil || index < 0 {
		return false
	}
	if channel.ChannelInfo.MultiKeyPendingDisableUntil == nil {
		return false
	}
	return channel.ChannelInfo.MultiKeyPendingDisableUntil[index] > 0
}

func (channel *Channel) SetPendingDisableKey(index int, until int64, reason string) {
	if channel == nil || index < 0 {
		return
	}
	if channel.ChannelInfo.MultiKeyPendingDisableUntil == nil {
		channel.ChannelInfo.MultiKeyPendingDisableUntil = make(map[int]int64)
	}
	if channel.ChannelInfo.MultiKeyPendingDisableReason == nil {
		channel.ChannelInfo.MultiKeyPendingDisableReason = make(map[int]string)
	}
	channel.ChannelInfo.MultiKeyPendingDisableUntil[index] = until
	channel.ChannelInfo.MultiKeyPendingDisableReason[index] = reason
}

func (channel *Channel) ClearPendingDisableKey(index int) {
	if channel == nil || index < 0 {
		return
	}
	if channel.ChannelInfo.MultiKeyPendingDisableUntil != nil {
		delete(channel.ChannelInfo.MultiKeyPendingDisableUntil, index)
	}
	if channel.ChannelInfo.MultiKeyPendingDisableReason != nil {
		delete(channel.ChannelInfo.MultiKeyPendingDisableReason, index)
	}
}

func (channel *Channel) GetPendingDisableKeyIndices() []int {
	if channel == nil || channel.ChannelInfo.MultiKeyPendingDisableUntil == nil {
		return nil
	}
	indices := make([]int, 0, len(channel.ChannelInfo.MultiKeyPendingDisableUntil))
	for index, until := range channel.ChannelInfo.MultiKeyPendingDisableUntil {
		if until > 0 {
			indices = append(indices, index)
		}
	}
	sort.Ints(indices)
	return indices
}

func (channel *Channel) IsTemporarilyUnavailable() bool {
	if channel == nil {
		return true
	}
	if channel.Status != common.ChannelStatusEnabled {
		return true
	}
	if channel.HasPendingDisable() {
		return true
	}
	return false
}

func mutateCachedChannel(channelId int, mutate func(channel *Channel)) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	channel, ok := channelsIDM[channelId]
	if !ok || channel == nil {
		return
	}
	mutate(channel)
}

func ScheduleChannelPreDisable(channelId int, usingKey string, reason string, until int64) (bool, error) {
	channel, err := GetChannelById(channelId, true)
	if err != nil {
		return false, err
	}
	if channel.ChannelInfo.IsMultiKey {
		keyIndex := channel.GetKeyIndex(usingKey)
		if keyIndex < 0 {
			return false, fmt.Errorf("pending disable key not found for channel %d", channelId)
		}
		if channel.HasPendingDisableKey(keyIndex) {
			return false, nil
		}
		channel.SetPendingDisableKey(keyIndex, until, reason)
		if err := channel.SaveWithoutKey(); err != nil {
			return false, err
		}
		mutateCachedChannel(channelId, func(channel *Channel) {
			channel.SetPendingDisableKey(keyIndex, until, reason)
		})
		return true, nil
	}

	if channel.HasPendingDisable() {
		return false, nil
	}
	channel.SetPendingDisable(until, reason)
	if err := channel.SaveWithoutKey(); err != nil {
		return false, err
	}
	mutateCachedChannel(channelId, func(channel *Channel) {
		channel.SetPendingDisable(until, reason)
	})
	return true, nil
}

func ClearChannelPreDisable(channelId int, usingKey string) error {
	channel, err := GetChannelById(channelId, true)
	if err != nil {
		return err
	}
	if channel.ChannelInfo.IsMultiKey {
		if usingKey == "" {
			if len(channel.ChannelInfo.MultiKeyPendingDisableUntil) == 0 && len(channel.ChannelInfo.MultiKeyPendingDisableReason) == 0 {
				return nil
			}
			channel.ChannelInfo.MultiKeyPendingDisableUntil = make(map[int]int64)
			channel.ChannelInfo.MultiKeyPendingDisableReason = make(map[int]string)
			if err := channel.SaveWithoutKey(); err != nil {
				return err
			}
			mutateCachedChannel(channelId, func(channel *Channel) {
				channel.ChannelInfo.MultiKeyPendingDisableUntil = make(map[int]int64)
				channel.ChannelInfo.MultiKeyPendingDisableReason = make(map[int]string)
			})
			return nil
		}
		keyIndex := channel.GetKeyIndex(usingKey)
		if keyIndex >= 0 {
			channel.ClearPendingDisableKey(keyIndex)
			if err := channel.SaveWithoutKey(); err != nil {
				return err
			}
			mutateCachedChannel(channelId, func(channel *Channel) {
				channel.ClearPendingDisableKey(keyIndex)
			})
			return nil
		}
	}

	if !channel.HasPendingDisable() {
		return nil
	}
	channel.ClearPendingDisable()
	if err := channel.SaveWithoutKey(); err != nil {
		return err
	}
	mutateCachedChannel(channelId, func(channel *Channel) {
		channel.ClearPendingDisable()
	})
	return nil
}
