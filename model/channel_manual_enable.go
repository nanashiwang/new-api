package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

func getMultiKeySizeForRecovery(channel *Channel) int {
	if channel == nil {
		return 0
	}
	size := channel.ChannelInfo.MultiKeySize
	if size <= 0 {
		size = len(channel.GetKeys())
	}
	return size
}

func clearChannelStatusMetadata(channel *Channel) {
	if channel == nil {
		return
	}
	info := channel.GetOtherInfo()
	delete(info, "status_reason")
	delete(info, "status_time")
	channel.SetOtherInfo(info)
}

func RecoverChannelForManualEnable(channelId int) error {
	channel, err := GetChannelById(channelId, true)
	if err != nil {
		return err
	}

	channel.Status = common.ChannelStatusEnabled
	clearChannelStatusMetadata(channel)
	channel.ClearPendingDisable()

	if channel.ChannelInfo.IsMultiKey {
		channel.ChannelInfo.MultiKeyStatusList = make(map[int]int)
		channel.ChannelInfo.MultiKeyDisabledReason = make(map[int]string)
		channel.ChannelInfo.MultiKeyDisabledTime = make(map[int]int64)
		channel.ChannelInfo.MultiKeyPendingDisableUntil = make(map[int]int64)
		channel.ChannelInfo.MultiKeyPendingDisableReason = make(map[int]string)
	}

	if err := channel.SaveWithoutKey(); err != nil {
		return err
	}
	if channel.ChannelInfo.IsMultiKey {
		if err := ClearAllMultiKeyCooldown(channel.Id, getMultiKeySizeForRecovery(channel)); err != nil {
			return err
		}
	}
	return UpdateAbilityStatus(channel.Id, true)
}

func RecoverChannelKeyForManualEnable(channelId int, keyIndex int) error {
	channel, err := GetChannelById(channelId, true)
	if err != nil {
		return err
	}
	if !channel.ChannelInfo.IsMultiKey {
		return RecoverChannelForManualEnable(channelId)
	}

	size := getMultiKeySizeForRecovery(channel)
	if keyIndex < 0 || keyIndex >= size {
		return fmt.Errorf("key index out of range")
	}

	channel.Status = common.ChannelStatusEnabled
	clearChannelStatusMetadata(channel)
	channel.ClearPendingDisable()

	if channel.ChannelInfo.MultiKeyStatusList != nil {
		delete(channel.ChannelInfo.MultiKeyStatusList, keyIndex)
	}
	if channel.ChannelInfo.MultiKeyDisabledTime != nil {
		delete(channel.ChannelInfo.MultiKeyDisabledTime, keyIndex)
	}
	if channel.ChannelInfo.MultiKeyDisabledReason != nil {
		delete(channel.ChannelInfo.MultiKeyDisabledReason, keyIndex)
	}
	if channel.ChannelInfo.MultiKeyPendingDisableUntil != nil {
		delete(channel.ChannelInfo.MultiKeyPendingDisableUntil, keyIndex)
	}
	if channel.ChannelInfo.MultiKeyPendingDisableReason != nil {
		delete(channel.ChannelInfo.MultiKeyPendingDisableReason, keyIndex)
	}

	if err := channel.SaveWithoutKey(); err != nil {
		return err
	}
	if err := ClearMultiKeyCooldown(channel.Id, keyIndex); err != nil {
		return err
	}
	return UpdateAbilityStatus(channel.Id, true)
}

func RecoverChannelsByTagForManualEnable(tag string) error {
	channels, err := GetChannelsByTag(tag, false, true)
	if err != nil {
		return err
	}
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		if err := RecoverChannelForManualEnable(channel.Id); err != nil {
			return err
		}
	}
	return nil
}
