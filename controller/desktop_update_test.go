package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUpdateDesktopUpdateSettingsRejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/desktop-update/settings", strings.NewReader(strings.Repeat("x", 64*1024+1)))

	UpdateDesktopUpdateSettings(context)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateDesktopUpdateSettingsRejectsTrailingJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	body := `{"enabled":false,"public_base_url":"","max_upload_mb":256,"retention_count":10}{}`
	context.Request = httptest.NewRequest(http.MethodPut, "/api/desktop-update/settings", strings.NewReader(body))

	UpdateDesktopUpdateSettings(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
