package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterTurnstileCheckRequiresTokenWithEmailVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originTurnstileCheckEnabled := common.TurnstileCheckEnabled
	originEmailVerificationEnabled := common.EmailVerificationEnabled
	common.TurnstileCheckEnabled = true
	common.EmailVerificationEnabled = true
	t.Cleanup(func() {
		common.TurnstileCheckEnabled = originTurnstileCheckEnabled
		common.EmailVerificationEnabled = originEmailVerificationEnabled
	})

	router := gin.New()
	router.POST("/register", RegisterTurnstileCheck(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"success":false,"message":"Turnstile token 为空"}`, recorder.Body.String())
}

func TestRegisterTurnstileCheckRequiresTokenWithoutEmailVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originTurnstileCheckEnabled := common.TurnstileCheckEnabled
	originEmailVerificationEnabled := common.EmailVerificationEnabled
	common.TurnstileCheckEnabled = true
	common.EmailVerificationEnabled = false
	t.Cleanup(func() {
		common.TurnstileCheckEnabled = originTurnstileCheckEnabled
		common.EmailVerificationEnabled = originEmailVerificationEnabled
	})

	router := gin.New()
	router.POST("/register", RegisterTurnstileCheck(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"success":false,"message":"Turnstile token 为空"}`, recorder.Body.String())
}
