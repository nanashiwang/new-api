package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestEmailVerificationRateLimitMemory_LimitsSameEmailAcrossIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originRedisEnabled := common.RedisEnabled
	originLimiter := inMemoryRateLimiter
	common.RedisEnabled = false
	inMemoryRateLimiter = common.InMemoryRateLimiter{}
	t.Cleanup(func() {
		common.RedisEnabled = originRedisEnabled
		inMemoryRateLimiter = originLimiter
	})

	router := gin.New()
	router.GET("/verification", EmailVerificationRateLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	for i := 0; i < EmailVerificationEmailMaxRequests; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/verification?email=Spam%40Example.com", nil)
		req.RemoteAddr = fmt.Sprintf("203.0.113.%d:12345", i+1)
		router.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusOK, recorder.Code)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/verification?email=spam%40example.com", nil)
	req.RemoteAddr = "203.0.113.99:12345"
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
}
