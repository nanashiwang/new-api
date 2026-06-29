package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCacheHeadersForImagePlaygroundStaticPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Cache())
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name      string
		path      string
		wantCache string
		wantExtra map[string]string
	}{
		{
			name:      "service worker is never long cached",
			path:      "/image-playground/sw.js",
			wantCache: "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0",
			wantExtra: map[string]string{
				"Pragma":  "no-cache",
				"Expires": "0",
			},
		},
		{
			name:      "image playground assets are immutable",
			path:      "/image-playground/assets/index.js",
			wantCache: "public, max-age=31536000, immutable",
		},
		{
			name:      "main app assets stay immutable",
			path:      "/assets/index.js",
			wantCache: "public, max-age=31536000, immutable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			router.ServeHTTP(recorder, req)

			if got := recorder.Header().Get("Cache-Control"); got != tt.wantCache {
				t.Fatalf("Cache-Control = %q, want %q", got, tt.wantCache)
			}
			for key, want := range tt.wantExtra {
				if got := recorder.Header().Get(key); got != want {
					t.Fatalf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}
