package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func TestUpdateOptionRejectsInvalidResponsesRequestBodyLimitMB(t *testing.T) {
	originMax := constant.MaxRequestBodyMB
	constant.MaxRequestBodyMB = 256
	t.Cleanup(func() {
		constant.MaxRequestBodyMB = originMax
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/option/",
		strings.NewReader(`{"key":"ResponsesRequestBodyLimitMB","value":"300"}`),
	)

	UpdateOption(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"success":false`) {
		t.Fatalf("expected failure response, got %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "0-256") {
		t.Fatalf("expected limit range in response, got %s", recorder.Body.String())
	}
}
