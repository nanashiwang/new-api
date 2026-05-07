package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestApplyUserRegisterAuditCapturesSourceIPAndUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", nil)
	req.RemoteAddr = "198.51.100.10:12345"
	req.Header.Set("User-Agent", strings.Repeat("a", registerUserAgentMaxLength+20))
	ctx.Request = req

	user := &model.User{}
	applyUserRegisterAudit(ctx, user, model.UserRegisterSourcePassword)

	require.Equal(t, model.UserRegisterSourcePassword, user.RegisterSource)
	require.Equal(t, "198.51.100.10", user.RegisterIP)
	require.Len(t, []rune(user.RegisterUserAgent), registerUserAgentMaxLength)
}

func TestApplyUserRegisterAuditDefaultsUnknownSource(t *testing.T) {
	user := &model.User{}
	applyUserRegisterAudit(nil, user, "")

	require.Equal(t, model.UserRegisterSourceUnknown, user.RegisterSource)
	require.Empty(t, user.RegisterIP)
	require.Empty(t, user.RegisterUserAgent)
}
