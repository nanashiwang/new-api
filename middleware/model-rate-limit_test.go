package middleware

import (
	"github.com/QuantumNous/new-api/common/limiter"
	"github.com/QuantumNous/new-api/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func TestModelSuccessRateLimitReleasesHTTPAndStreamFailures(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		srv := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
		previous := common.RDB
		common.RDB = client
		modelSuccessWindow = limiter.SuccessWindow{}
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("id", 919); c.Next() })
		handler := memoryRateLimitHandler(int64(60), 0, 1)
		if useRedis {
			handler = redisRateLimitHandler(60, 0, 1)
		}
		r.Use(handler)
		r.GET("/fail", func(c *gin.Context) { c.Status(502) })
		r.GET("/stream-fail", func(c *gin.Context) { c.Set(service.RequestOutcomeKey, false); c.String(200, "data: error\n\n") })
		r.GET("/ok", func(c *gin.Context) { c.Status(200) })
		for _, path := range []string{"/fail", "/fail", "/stream-fail", "/stream-fail", "/ok"} {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			require.NotEqual(t, 429, w.Code)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/ok", nil))
		require.Equal(t, 429, w.Code)
		common.RDB = previous
		require.NoError(t, client.Close())
	}
}

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
	router := gin.New()

	calledNextCount := 0
	router.Use(func(c *gin.Context) {
		c.Set("id", 1)
		common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
		c.Next()
	})
	router.Use(handler)
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		calledNextCount++
		c.Status(http.StatusOK)
	})

	for i := 0; i < 3; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)

		if calledNextCount != i+1 {
			t.Fatalf("request %d should pass when success limit is 0", i+1)
		}
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d returned status %d, want %d", i+1, recorder.Code, http.StatusOK)
		}
	}
}
