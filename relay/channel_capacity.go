package relay

import (
	"errors"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// acquireAdditionalChannelRequestRpm 为同一 relay 尝试中的额外上游 HTTP 请求补记 RPM。
// 外层已持有并发 Lease，因此这里只申请 RPM，避免同渠道 max_concurrency=1 时自锁。
func acquireAdditionalChannelRequestRpm(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	if !middleware.ChannelConcurrencyEnabled() || info == nil || info.ChannelId <= 0 {
		return nil
	}
	channel, err := model.CacheGetChannel(info.ChannelId)
	if err != nil || channel == nil {
		apiErr := types.NewErrorWithStatusCode(
			errors.New("failed to resolve channel for additional request capacity"),
			types.ErrorCodeGetChannelFailed,
			http.StatusServiceUnavailable,
			types.ErrOptionWithSkipRetry(),
		)
		apiErr.RetryAfter = time.Second
		return apiErr
	}
	admission := middleware.AcquireChannelRpmForChannel(channel, c.GetString(common.RequestIdKey))
	if admission.Acquired {
		if admission.Release != nil {
			admission.Release()
		}
		return nil
	}
	statusCode := http.StatusTooManyRequests
	if admission.Reason == middleware.ChannelAdmissionBackendUnavailable {
		statusCode = http.StatusServiceUnavailable
	}
	retryAfter := admission.RetryAfter
	if retryAfter <= 0 {
		retryAfter = time.Second
	}
	apiErr := types.NewErrorWithStatusCode(
		errors.New("additional upstream request reached channel rpm limit"),
		types.ErrorCodeGetChannelFailed,
		statusCode,
		types.ErrOptionWithSkipRetry(),
	)
	apiErr.RetryAfter = retryAfter
	return apiErr
}
