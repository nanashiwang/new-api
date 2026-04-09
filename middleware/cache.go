package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Cache() func(c *gin.Context) {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		switch {
		case strings.HasPrefix(path, "/api"),
			strings.HasPrefix(path, "/v1"),
			strings.HasPrefix(path, "/mj"),
			strings.HasPrefix(path, "/pg"):
			c.Next()
			return
		case path == "/":
			c.Header("Cache-Control", "no-cache")
		case strings.HasPrefix(path, "/assets/"):
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		default:
			c.Header("Cache-Control", "public, max-age=604800")
		}
		c.Header("Cache-Version", "b688f2fb5be447c25e5aa3bd063087a83db32a288bf6a4f35f2d8db310e40b14")
		c.Next()
	}
}
