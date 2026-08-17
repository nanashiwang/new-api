package service

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const (
	ResponsesStreamMissingCompletedReason = "responses stream ended without response.completed"
	ResponsesStreamEmptyCompletedReason   = "responses stream completed without visible output"
)

func IsChannelUnavailableForRequest(channel *model.Channel) bool {
	if channel == nil {
		return true
	}
	return channel.IsTemporarilyUnavailable() || IsCRSShortCircuitOpenForChannel(channel) || IsChannelRateLimitCoolingDown(channel)
}

func ShouldFallbackAfterSetupError(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	return err.GetErrorCode() == types.ErrorCodeChannelNoAvailableKey
}

func ShouldUsePreDisableWait() bool {
	setting := operation_setting.GetMonitorSetting()
	return setting != nil && setting.PreDisableWaitEnabled && setting.PreDisableWaitMinutes > 0
}

func SchedulePreDisableWait(channelError types.ChannelError, reason string) (bool, error) {
	setting := operation_setting.GetMonitorSetting()
	if setting == nil || !setting.PreDisableWaitEnabled || setting.PreDisableWaitMinutes <= 0 {
		return false, nil
	}
	waitDuration := time.Duration(setting.PreDisableWaitMinutes * float64(time.Minute))
	if waitDuration <= 0 {
		return false, nil
	}
	until := time.Now().Add(waitDuration).Unix()
	return model.ScheduleChannelPreDisable(channelError.ChannelId, channelError.UsingKey, reason, until)
}

// ScheduleCurrentChannelPreDisableWait gives the selected channel a short
// cooldown by reusing the existing pending-disable flow.
func ScheduleCurrentChannelPreDisableWait(c *gin.Context, reason string) (bool, error) {
	if c == nil || !common.GetContextKeyBool(c, constant.ContextKeyChannelAutoBan) {
		return false, nil
	}
	channelId := common.GetContextKeyInt(c, constant.ContextKeyChannelId)
	if channelId <= 0 {
		return false, nil
	}
	if reason == "" {
		reason = "temporary upstream stream instability"
	}
	channelError := types.NewChannelError(
		channelId,
		common.GetContextKeyInt(c, constant.ContextKeyChannelType),
		common.GetContextKeyString(c, constant.ContextKeyChannelName),
		common.GetContextKeyBool(c, constant.ContextKeyChannelIsMultiKey),
		common.GetContextKeyString(c, constant.ContextKeyChannelKey),
		true,
	)
	return SchedulePreDisableWait(*channelError, reason)
}

func ScheduleCurrentChannelScopedPreDisableWait(c *gin.Context, reason string) (bool, []CRSShortCircuitTrip, error) {
	scheduled, err := ScheduleCurrentChannelPreDisableWait(c, reason)
	if c == nil || !common.GetContextKeyBool(c, constant.ContextKeyChannelAutoBan) {
		return scheduled, nil, err
	}
	channelId := common.GetContextKeyInt(c, constant.ContextKeyChannelId)
	if channelId <= 0 {
		return scheduled, nil, err
	}
	if reason == "" {
		reason = "temporary upstream stream instability"
	}
	trips := ApplyChannelScopedShortCircuit(currentChannelForScopedShortCircuit(c, channelId), reason)
	return scheduled || len(trips) > 0, trips, err
}

func currentChannelForScopedShortCircuit(c *gin.Context, channelId int) *model.Channel {
	if channel, err := model.CacheGetChannel(channelId); err == nil && channel != nil {
		return channel
	}
	autoBan := 0
	if common.GetContextKeyBool(c, constant.ContextKeyChannelAutoBan) {
		autoBan = 1
	}
	baseURL := common.GetContextKeyString(c, constant.ContextKeyChannelBaseUrl)
	return &model.Channel{
		Id:      channelId,
		Type:    common.GetContextKeyInt(c, constant.ContextKeyChannelType),
		Name:    common.GetContextKeyString(c, constant.ContextKeyChannelName),
		BaseURL: &baseURL,
		AutoBan: &autoBan,
	}
}
