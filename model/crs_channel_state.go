package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const crsChannelStateKey = "crs_auto_state"

type CRSChannelAutoState struct {
	SiteID               int    `json:"site_id"`
	Platform             string `json:"platform"`
	ConsecutiveHealthy   int    `json:"consecutive_healthy"`
	ConsecutiveUnhealthy int    `json:"consecutive_unhealthy"`
	LastObservedAt       int64  `json:"last_observed_at"`
	DisabledAt           int64  `json:"disabled_at,omitempty"`
	LastProbeSuccessAt   int64  `json:"last_probe_success_at,omitempty"`
	Owned                bool   `json:"owned"`
}

type CRSChannelTransition string

const (
	CRSChannelTransitionNone     CRSChannelTransition = ""
	CRSChannelTransitionDisabled CRSChannelTransition = "disabled"
	CRSChannelTransitionEnabled  CRSChannelTransition = "enabled"
)

var errCRSChannelConcurrentUpdate = errors.New("crs_channel:concurrent_update")

func GetCRSChannelAutoState(channel *Channel) (CRSChannelAutoState, bool) {
	if channel == nil {
		return CRSChannelAutoState{}, false
	}
	raw, exists := channel.GetOtherInfo()[crsChannelStateKey]
	if !exists {
		return CRSChannelAutoState{}, false
	}
	data, err := common.Marshal(raw)
	if err != nil {
		return CRSChannelAutoState{}, false
	}
	state := CRSChannelAutoState{}
	if err := common.Unmarshal(data, &state); err != nil || state.SiteID <= 0 {
		return CRSChannelAutoState{}, false
	}
	return state, true
}

func IsCRSAutoDisabledChannel(channel *Channel) bool {
	state, ok := GetCRSChannelAutoState(channel)
	if !ok || !state.Owned || channel.Status != common.ChannelStatusAutoDisabled {
		return false
	}
	settings := channel.GetSetting()
	return settings.CRSAutoManage && settings.CRSSiteID == state.SiteID && settings.CRSPlatform == state.Platform
}

func ObserveCRSManagedChannel(channelID, siteID int, platform string, observedAt int64, healthy bool) (CRSChannelTransition, error) {
	if channelID <= 0 || siteID <= 0 || platform == "" || observedAt <= 0 {
		return CRSChannelTransitionNone, errors.New("crs_channel:invalid_observation")
	}
	transition := CRSChannelTransitionNone
	err := DB.Transaction(func(tx *gorm.DB) error {
		channel := &Channel{}
		if err := tx.First(channel, "id = ?", channelID).Error; err != nil {
			return err
		}
		settings := channel.GetSetting()
		if !settings.CRSAutoManage || settings.CRSSiteID != siteID || settings.CRSPlatform != platform || !channel.GetAutoBan() {
			return nil
		}
		originalOtherInfo := channel.OtherInfo
		originalSetting := channel.Setting
		originalAutoBan := channel.AutoBan
		info := channel.GetOtherInfo()
		state, exists := GetCRSChannelAutoState(channel)
		if !exists || state.SiteID != siteID || state.Platform != platform || (state.Owned && channel.Status != common.ChannelStatusAutoDisabled) {
			state = CRSChannelAutoState{SiteID: siteID, Platform: platform}
		}
		if state.LastObservedAt >= observedAt {
			return nil
		}
		state.LastObservedAt = observedAt
		if healthy {
			state.ConsecutiveHealthy++
			state.ConsecutiveUnhealthy = 0
		} else {
			state.ConsecutiveUnhealthy++
			state.ConsecutiveHealthy = 0
		}

		oldStatus := channel.Status
		newStatus := oldStatus
		if !healthy && state.ConsecutiveUnhealthy >= 2 && oldStatus == common.ChannelStatusEnabled {
			newStatus = common.ChannelStatusAutoDisabled
			state.Owned = true
			state.DisabledAt = observedAt
			state.LastProbeSuccessAt = 0
			info["status_reason"] = fmt.Sprintf("CRS 站点 #%d OpenAI 账号池连续不可用", siteID)
			info["status_time"] = observedAt
			transition = CRSChannelTransitionDisabled
		} else if healthy && state.ConsecutiveHealthy >= 2 && oldStatus == common.ChannelStatusAutoDisabled && state.Owned && state.LastProbeSuccessAt > state.DisabledAt && !HasPeriodQuotaMeta(channel) {
			newStatus = common.ChannelStatusEnabled
			state.Owned = false
			state.DisabledAt = 0
			delete(info, "status_reason")
			delete(info, "status_time")
			transition = CRSChannelTransitionEnabled
		}
		info[crsChannelStateKey] = state
		updatedOtherInfo, err := common.Marshal(info)
		if err != nil {
			return err
		}
		query := tx.Model(&Channel{}).Where("id = ? AND status = ? AND other_info = ?", channelID, oldStatus, originalOtherInfo)
		query = whereNullableString(query, "setting", originalSetting)
		query = whereNullableInt(query, "auto_ban", originalAutoBan)
		result := query.Updates(map[string]any{"status": newStatus, "other_info": string(updatedOtherInfo)})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errCRSChannelConcurrentUpdate
		}
		if newStatus != oldStatus {
			if err := tx.Model(&Ability{}).Where("channel_id = ?", channelID).Update("enabled", newStatus == common.ChannelStatusEnabled).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, errCRSChannelConcurrentUpdate) {
		return CRSChannelTransitionNone, nil
	}
	if err == nil && transition != CRSChannelTransitionNone {
		InitChannelCache()
	}
	return transition, err
}

func MarkCRSRecoveryProbeSuccess(channelID int, probedAt int64) error {
	if channelID <= 0 || probedAt <= 0 {
		return nil
	}
	channel := &Channel{}
	if err := DB.First(channel, "id = ?", channelID).Error; err != nil {
		return err
	}
	state, ok := GetCRSChannelAutoState(channel)
	if !ok || !state.Owned || channel.Status != common.ChannelStatusAutoDisabled || probedAt <= state.DisabledAt {
		return nil
	}
	originalOtherInfo := channel.OtherInfo
	info := channel.GetOtherInfo()
	state.LastProbeSuccessAt = probedAt
	info[crsChannelStateKey] = state
	raw, err := common.Marshal(info)
	if err != nil {
		return err
	}
	result := DB.Model(&Channel{}).
		Where("id = ? AND status = ? AND other_info = ?", channelID, common.ChannelStatusAutoDisabled, originalOtherInfo).
		Update("other_info", string(raw))
	return result.Error
}

func whereNullableString(query *gorm.DB, column string, value *string) *gorm.DB {
	if value == nil {
		return query.Where(column + " IS NULL")
	}
	return query.Where(column+" = ?", *value)
}

func whereNullableInt(query *gorm.DB, column string, value *int) *gorm.DB {
	if value == nil {
		return query.Where(column + " IS NULL")
	}
	return query.Where(column+" = ?", *value)
}

// ClaimCRSAutoDisabledChannel transfers recovery ownership to another disable mechanism.
func ClaimCRSAutoDisabledChannel(channel *Channel, reason string) (bool, error) {
	if !IsCRSAutoDisabledChannel(channel) {
		return false, nil
	}
	originalOtherInfo := channel.OtherInfo
	info := channel.GetOtherInfo()
	state, _ := GetCRSChannelAutoState(channel)
	state.Owned = false
	info[crsChannelStateKey] = state
	info["status_reason"] = reason
	info["status_time"] = common.GetTimestamp()
	raw, err := common.Marshal(info)
	if err != nil {
		return false, err
	}
	result := DB.Model(&Channel{}).
		Where("id = ? AND status = ? AND other_info = ?", channel.Id, common.ChannelStatusAutoDisabled, originalOtherInfo).
		Update("other_info", string(raw))
	return result.RowsAffected == 1, result.Error
}

func AutoDisableChannelForPeriodQuota(channelID int, scope, scopeKey string, periodEnd int64, reason string) error {
	for attempt := 0; attempt < 2; attempt++ {
		channel, err := GetChannelById(channelID, true)
		if err != nil {
			return err
		}
		if channel.Status != common.ChannelStatusEnabled {
			if !IsCRSAutoDisabledChannel(channel) {
				return nil
			}
			originalOtherInfo := channel.OtherInfo
			SetPeriodQuotaMeta(channel, scope, scopeKey, periodEnd)
			result := DB.Model(&Channel{}).
				Where("id = ? AND status = ? AND other_info = ?", channelID, common.ChannelStatusAutoDisabled, originalOtherInfo).
				Update("other_info", channel.OtherInfo)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				return nil
			}
			continue
		}
		originalOtherInfo := channel.OtherInfo
		info := channel.GetOtherInfo()
		info["status_reason"] = reason
		info["status_time"] = common.GetTimestamp()
		channel.SetOtherInfo(info)
		SetPeriodQuotaMeta(channel, scope, scopeKey, periodEnd)
		result := DB.Model(&Channel{}).
			Where("id = ? AND status = ? AND other_info = ?", channelID, common.ChannelStatusEnabled, originalOtherInfo).
			Updates(map[string]any{"status": common.ChannelStatusAutoDisabled, "other_info": channel.OtherInfo})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 {
			if err := DB.Model(&Ability{}).Where("channel_id = ?", channelID).Update("enabled", false).Error; err != nil {
				return err
			}
			InitChannelCache()
			return nil
		}
	}
	return nil
}

func RecoverPeriodQuotaChannel(channel *Channel, periodEnd int64) (bool, error) {
	if channel == nil || channel.Status != common.ChannelStatusAutoDisabled || !HasPeriodQuotaMeta(channel) || GetPeriodQuotaUntil(channel) != periodEnd {
		return false, nil
	}
	originalOtherInfo := channel.OtherInfo
	ClearPeriodQuotaMeta(channel)
	if IsCRSAutoDisabledChannel(channel) {
		result := DB.Model(&Channel{}).
			Where("id = ? AND status = ? AND other_info = ?", channel.Id, common.ChannelStatusAutoDisabled, originalOtherInfo).
			Update("other_info", channel.OtherInfo)
		return false, result.Error
	}
	result := DB.Model(&Channel{}).
		Where("id = ? AND status = ? AND other_info = ?", channel.Id, common.ChannelStatusAutoDisabled, originalOtherInfo).
		Updates(map[string]any{"status": common.ChannelStatusEnabled, "other_info": channel.OtherInfo})
	if result.Error != nil || result.RowsAffected != 1 {
		return false, result.Error
	}
	if err := DB.Model(&Ability{}).Where("channel_id = ?", channel.Id).Update("enabled", true).Error; err != nil {
		return false, err
	}
	InitChannelCache()
	return true, nil
}
