package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
)

type upstreamCompatIssue string

const (
	upstreamCompatIssueNone              upstreamCompatIssue = ""
	upstreamCompatIssueStreamOptions     upstreamCompatIssue = "stream_options_unsupported"
	upstreamCompatIssueResponsesAPI      upstreamCompatIssue = "responses_api_unsupported"
	compatFallbackToNativeChatContextKey                     = "compat_fallback_to_native_chat"
)

func classifyUpstreamCompatibilityIssue(resp *http.Response, relayMode int) upstreamCompatIssue {
	if resp == nil {
		return upstreamCompatIssueNone
	}

	bodyText, ok := peekResponseBody(resp)
	if !ok {
		return upstreamCompatIssueNone
	}
	bodyText = strings.ToLower(bodyText)

	if isUnsupportedStreamOptionsResponse(resp.StatusCode, bodyText) {
		return upstreamCompatIssueStreamOptions
	}
	if isUnsupportedResponsesResponse(resp.StatusCode, bodyText, relayMode) {
		return upstreamCompatIssueResponsesAPI
	}
	return upstreamCompatIssueNone
}

func isUnsupportedStreamOptionsResponse(statusCode int, bodyText string) bool {
	if statusCode != http.StatusBadRequest {
		return false
	}
	if !strings.Contains(bodyText, "stream_options") {
		return false
	}
	return strings.Contains(bodyText, "unsupported parameter") ||
		strings.Contains(bodyText, "unknown parameter") ||
		strings.Contains(bodyText, "unrecognized") ||
		strings.Contains(bodyText, "not supported")
}

func isUnsupportedResponsesResponse(statusCode int, bodyText string, relayMode int) bool {
	if relayMode != relayconstant.RelayModeResponses && relayMode != relayconstant.RelayModeResponsesCompact {
		return false
	}

	if statusCode == http.StatusNotFound || statusCode == http.StatusMethodNotAllowed {
		return true
	}

	if statusCode != http.StatusBadRequest {
		return false
	}

	return strings.Contains(bodyText, "unsupported endpoint") ||
		strings.Contains(bodyText, "responses not supported") ||
		strings.Contains(bodyText, "only /v1/chat/completions") ||
		(strings.Contains(bodyText, "/v1/responses") &&
			(strings.Contains(bodyText, "not supported") || strings.Contains(bodyText, "unsupported")))
}

func peekResponseBody(resp *http.Response) (string, bool) {
	if resp == nil || resp.Body == nil {
		return "", false
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return string(bodyBytes), true
}

func markFallbackToNativeChat(c *gin.Context) {
	if c == nil {
		return
	}
	c.Set(compatFallbackToNativeChatContextKey, true)
}

func shouldFallbackToNativeChat(c *gin.Context) bool {
	if c == nil {
		return false
	}
	v, ok := c.Get(compatFallbackToNativeChatContextKey)
	if !ok {
		return false
	}
	enabled, ok := v.(bool)
	return ok && enabled
}

func logCompatFallback(c *gin.Context, info *relaycommon.RelayInfo, reason string) {
	if info == nil {
		return
	}
	logger.LogWarn(c.Request.Context(), fmt.Sprintf(
		"compat fallback: channel_id=%d base_url=%s relay_mode=%d reason=%s",
		info.ChannelId,
		common.MaskSensitiveInfo(info.ChannelBaseUrl),
		info.RelayMode,
		reason,
	))
}
