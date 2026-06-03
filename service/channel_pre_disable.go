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

const ResponsesStreamMissingCompletedReason = "responses stream ended without response.completed"

func IsChannelUnavailableForRequest(channel *model.Channel) bool {
	if channel == nil {
		return true
	}
	return channel.IsTemporarilyUnavailable()
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
