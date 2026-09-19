package router

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiCountTokensRejectedBeforeAuthentication(t *testing.T) {
	r := gin.New()
	SetRelayRouter(r)
	for _, path := range []string{"/v1beta/models/gemini-test:countTokens", "/v1beta/models/gemini-test:countTokens?key=invalid"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", path, nil))
		require.Equal(t, 404, w.Code)
		require.Contains(t, w.Body.String(), "invalid_request_error")
	}
	// A supported method still reaches the existing auth middleware.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/v1beta/models/gemini-test:generateContent", nil))
	require.Equal(t, 401, w.Code)
}
