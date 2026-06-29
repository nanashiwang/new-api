package common

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

const defaultAnonymousRequestBodyLimitKB = 512
const defaultMaxRequestBodyMB = 128

func GetAnonymousRequestBodyLimitBytes() int64 {
	limitKB := constant.AnonymousRequestBodyLimitKB
	if limitKB < 0 {
		limitKB = defaultAnonymousRequestBodyLimitKB
	}
	return int64(limitKB) << 10
}

func GetRequestBodyLimitMB(c *gin.Context) int {
	maxMB := constant.MaxRequestBodyMB
	if maxMB <= 0 {
		maxMB = defaultMaxRequestBodyMB
	}
	if responsesLimitMB := GetResponsesRequestBodyLimitMB(c); responsesLimitMB > 0 && responsesLimitMB < maxMB {
		return responsesLimitMB
	}
	return maxMB
}

func GetRequestBodyLimitBytes(c *gin.Context) int64 {
	return int64(GetRequestBodyLimitMB(c)) << 20
}

func GetResponsesRequestBodyLimitMB(c *gin.Context) int {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return 0
	}
	if !IsResponsesRequestBodyLimitedPath(c.Request.URL.Path) {
		return 0
	}
	return constant.ResponsesRequestBodyLimitMB
}

func IsResponsesRequestBodyLimitedPath(path string) bool {
	path = strings.TrimSpace(path)
	for _, prefix := range []string{
		"/v1/responses",
		"/v1/chat/completions",
		"/pg/chat/completions",
		"/openai/v1/responses",
		"/openai/v1/chat/completions",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func FormatRequestBodyTooLargeMessage(actualBytes, limitBytes int64) string {
	limitText := formatBytes(limitBytes)
	if limitText == "" {
		limitText = "当前服务限制"
	}
	if actualBytes > 0 {
		return fmt.Sprintf("请求体过大：当前约 %s，限制 %s。请减少图片数量/尺寸，避免把大图片以内联 base64 反复放入上下文，或缩短历史上下文后重试。", formatBytes(actualBytes), limitText)
	}
	return fmt.Sprintf("请求体过大：限制 %s。请减少图片数量/尺寸，避免把大图片以内联 base64 反复放入上下文，或缩短历史上下文后重试。", limitText)
}

func formatBytes(n int64) string {
	if n <= 0 {
		return ""
	}
	const unit = int64(1024)
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div := unit
	exp := 0
	for n >= div*unit && exp < len("KMGTPE")-1 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
