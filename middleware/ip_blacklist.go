package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func IPBlacklist() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if strings.TrimSpace(clientIP) == "" {
			c.Next()
			return
		}

		match, err := model.FindIPBlacklistMatch(clientIP)
		if err != nil {
			c.Next()
			return
		}
		if match == nil {
			c.Next()
			return
		}

		message := "当前 IP 已被拉黑"
		if isRelayPath(c.Request.URL.Path) {
			abortWithOpenAiMessage(c, http.StatusForbidden, message, types.ErrorCodeAccessDenied)
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": message,
			})
			c.Abort()
			return
		}
		c.String(http.StatusForbidden, message)
		c.Abort()
	}
}

func isRelayPath(path string) bool {
	switch {
	case strings.HasPrefix(path, "/v1"):
		return true
	case strings.HasPrefix(path, "/v1beta"):
		return true
	case strings.HasPrefix(path, "/mj"):
		return true
	case strings.Contains(path, "/mj"):
		return true
	case strings.HasPrefix(path, "/suno"):
		return true
	case strings.HasPrefix(path, "/kling/v1"):
		return true
	case strings.HasPrefix(path, "/jimeng"):
		return true
	case strings.HasPrefix(path, "/pg"):
		return true
	default:
		return false
	}
}
