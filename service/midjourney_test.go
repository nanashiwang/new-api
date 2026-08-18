package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDoMidjourneyHttpRequestPreservesRateLimitMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"code":30,"description":"rate limited","result":""}`))
	}))
	defer server.Close()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/mj/submit/imagine", strings.NewReader(`{"prompt":"test"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	result, _, err := DoMidjourneyHttpRequest(ctx, time.Second, server.URL)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusTooManyRequests, result.StatusCode)
	require.Equal(t, http.StatusTooManyRequests, result.UpstreamStatusCode)
	require.Equal(t, 7*time.Second, result.RetryAfter)
}
