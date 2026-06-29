package service

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
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

func IsTemporaryUpstreamError(err *types.NewAPIError) bool {
	if err == nil || types.IsSkipRetryError(err) {
		return false
	}
	if types.IsChannelError(err) {
		return true
	}
	if IsQuotaRelatedError(err) || IsChannelModelMismatchError(err) {
		return false
	}
	if IsRetryableSpecialBadRequestError(err) {
		return true
	}

	code := err.StatusCode
	if code < 100 || code > 599 {
		return true
	}
	if code >= 200 && code < 300 {
		return false
	}
	if operation_setting.IsAlwaysSkipRetryCode(err.GetErrorCode()) {
		return false
	}
	return operation_setting.ShouldRetryByStatusCode(code)
}

func ShouldRetryChannelError(c *gin.Context, openaiErr *types.NewAPIError, retryTimes int) bool {
	if openaiErr == nil {
		return false
	}
	if ShouldSkipRetryAfterChannelAffinityFailure(c, openaiErr) {
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
	if IsUpstreamRequestTooLargeError(openaiErr) && isUpstreamRequestTooLargeBeyondKnownLimit(c, openaiErr) {
		return false
	}
	if IsRetryableSpecialBadRequestError(openaiErr) {
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
		ids = retryExclusionIDsByTag(param, channel)
	}
	if shouldExcludeRetryByBaseURL(err) {
		ids = append(ids, retryExclusionIDsByBaseURL(param, channel)...)
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

func ApplyChannelTagRetryExclusion(param *RetryParam, channel *model.Channel) {
	if param == nil || channel == nil {
		return
	}
	ids := retryExclusionIDsByTag(param, channel)
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

func retryExclusionIDsByTag(param *RetryParam, channel *model.Channel) []int {
	if param == nil || channel == nil {
		return nil
	}
	ids := []int{channel.Id}
	tag := strings.TrimSpace(channel.GetTag())
	if tag != "" {
		if tagIDs := param.getCachedTagChannelIDs(tag, param.AllowedChannels); len(tagIDs) > 0 {
			ids = tagIDs
		}
	}
	return ids
}

func retryExclusionIDsByBaseURL(param *RetryParam, channel *model.Channel) []int {
	if param == nil || channel == nil {
		return nil
	}
	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		return nil
	}
	return param.getCachedBaseURLChannelIDs(baseURL, param.AllowedChannels)
}

func shouldExcludeRetryByTag(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	return IsRetryableSharedUpstreamPoolError(err) ||
		IsChannelModelMismatchError(err) ||
		IsUpstreamRequestTooLargeError(err)
}

func shouldExcludeRetryByBaseURL(err *types.NewAPIError) bool {
	return IsUpstreamRequestTooLargeError(err)
}

func isUpstreamRequestTooLargeBeyondKnownLimit(c *gin.Context, err *types.NewAPIError) bool {
	limitBytes := common.GetRequestBodyLimitBytes(c)
	if limitBytes <= 0 {
		return false
	}
	requestLength := int64(0)
	if err != nil && err.Upstream != nil {
		requestLength = err.Upstream.RequestLength
	}
	if requestLength <= 0 && c != nil && c.Request != nil {
		requestLength = c.Request.ContentLength
	}
	return requestLength > limitBytes
}

func IsRetryableSharedUpstreamPoolError(err *types.NewAPIError) bool {
	if err == nil || types.IsSkipRetryError(err) {
		return false
	}
	return IsQuotaRelatedError(err) ||
		IsUpstreamModelTemporaryUnavailableError(err) ||
		IsUpstreamRateLimitError(err) ||
		IsCRSMemoryPressureError(err)
}

func IsUpstreamRateLimitError(err *types.NewAPIError) bool {
	if err == nil || types.IsSkipRetryError(err) {
		return false
	}
	if err.StatusCode == http.StatusTooManyRequests {
		return true
	}
	lowerMessage := normalizeUpstreamErrorMessage(err)
	return strings.Contains(lowerMessage, "rate limit") ||
		strings.Contains(lowerMessage, "too many requests")
}

func IsUpstreamRequestTooLargeError(err *types.NewAPIError) bool {
	if err == nil || types.IsSkipRetryError(err) {
		return false
	}
	if err.StatusCode == http.StatusRequestEntityTooLarge {
		return true
	}
	lowerMessage := normalizeUpstreamErrorMessage(err)
	return strings.Contains(lowerMessage, "request entity too large") ||
		strings.Contains(lowerMessage, "payload too large") ||
		strings.Contains(lowerMessage, "content too large")
}
