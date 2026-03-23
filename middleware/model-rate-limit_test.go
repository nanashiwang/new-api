package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func TestModelRequestRateLimitMemory_AllowsZeroSuccessLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	common.RedisEnabled = false
	inMemoryRateLimiter = common.InMemoryRateLimiter{}

	originalEnabled := setting.ModelRequestRateLimitEnabled
	originalDuration := setting.ModelRequestRateLimitDurationMinutes
	originalCount := setting.ModelRequestRateLimitCount
	originalSuccessCount := setting.ModelRequestRateLimitSuccessCount
	originalGroup := setting.ModelRequestRateLimitGroup

	setting.ModelRequestRateLimitEnabled = true
	setting.ModelRequestRateLimitDurationMinutes = 1
	setting.ModelRequestRateLimitCount = 0
	setting.ModelRequestRateLimitSuccessCount = 0
	setting.ModelRequestRateLimitGroup = map[string][2]int{}

	t.Cleanup(func() {
		setting.ModelRequestRateLimitEnabled = originalEnabled
		setting.ModelRequestRateLimitDurationMinutes = originalDuration
		setting.ModelRequestRateLimitCount = originalCount
		setting.ModelRequestRateLimitSuccessCount = originalSuccessCount
		setting.ModelRequestRateLimitGroup = originalGroup
	})

	handler := ModelRequestRateLimit()

	for i := 0; i < 3; i++ {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		ctx.Set("id", 1)
		common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")

		calledNext := false
		ctx.HandlerFunc = func(c *gin.Context) {
			calledNext = true
			c.Status(http.StatusOK)
		}

		handler(ctx)

		if !calledNext {
			t.Fatalf("request %d should pass when success limit is 0", i+1)
		}
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d returned status %d, want %d", i+1, recorder.Code, http.StatusOK)
		}
	}
}
