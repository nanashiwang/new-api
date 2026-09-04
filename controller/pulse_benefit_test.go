package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGrantPulseBenefitBindsBodyUserToSignedServiceSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/internal/pulse/benefits/grant", bytes.NewBufferString(`{"grant_id":"grant-1","user_id":43,"amount":10,"source_ref":"grant-1","reward_type":"period"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("pulse_service_user_id", uint64(42))

	GrantPulseBenefit(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "user_id 与签名主体不一致")
}

func TestGrantPulseBenefitRequiresAuthenticatedServiceSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/internal/pulse/benefits/grant", bytes.NewBufferString(`{"grant_id":"grant-1","user_id":42,"amount":10,"source_ref":"grant-1","reward_type":"period"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	GrantPulseBenefit(ctx)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}
