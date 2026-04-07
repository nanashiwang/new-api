package service

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// IsRetryableUpstreamQuotaError returns true only for upstream quota failures.
// Local quota failures use skipRetry and must not trigger failover.
func IsRetryableUpstreamQuotaError(err *types.NewAPIError) bool {
	if err == nil || types.IsSkipRetryError(err) {
		return false
	}
	return IsQuotaRelatedError(err)
}

func ShouldRetryChannelError(c *gin.Context, openaiErr *types.NewAPIError, retryTimes int) bool {
	if openaiErr == nil {
		return false
	}
	if ShouldSkipRetryAfterChannelAffinityFailure(c) {
		return false
	}
	if types.IsChannelError(openaiErr) {
		return true
	}
	if types.IsSkipRetryError(openaiErr) {
		return false
	}
	if retryTimes <= 0 {
		return false
	}
	if _, ok := c.Get("specific_channel_id"); ok {
		return false
	}
	code := openaiErr.StatusCode
	if code >= 200 && code < 300 {
		return false
	}
	if code < 100 || code > 599 {
		return true
	}
	if operation_setting.IsAlwaysSkipRetryCode(openaiErr.GetErrorCode()) {
		return false
	}
	return operation_setting.ShouldRetryByStatusCode(code)
}

func ApplyChannelFailureRetryExclusion(param *RetryParam, channel *model.Channel, err *types.NewAPIError) {
	if param == nil || channel == nil {
		return
	}

	ids := []int{channel.Id}
	if shouldExcludeRetryByTag(err) {
		tag := strings.TrimSpace(channel.GetTag())
		if tag != "" {
			if tagIDs := param.getCachedTagChannelIDs(tag, param.AllowedChannels); len(tagIDs) > 0 {
				ids = tagIDs
			}
		}
	}

	seen := make(map[int]struct{}, len(param.ExcludeChannels))
	for _, id := range param.ExcludeChannels {
		seen[id] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		param.ExcludeChannels = append(param.ExcludeChannels, id)
		seen[id] = struct{}{}
	}
}

func shouldExcludeRetryByTag(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	return IsRetryableUpstreamQuotaError(err) || IsRequestedModelUnavailableError(err)
}
