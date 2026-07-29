package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUpdateOptionRejectsInvalidInviteBindingSettings(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing combined field", body: `{"key":"InviteBindingSettings","value":"{\"threshold\":1000}"}`},
		{name: "rate above one hundred", body: `{"key":"InviteBindingSettings","value":"{\"threshold\":1000,\"rate_after_threshold\":101}"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(tt.body))

			UpdateOption(ctx)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", recorder.Code)
			}
			if !strings.Contains(recorder.Body.String(), `"success":false`) {
				t.Fatalf("expected failure response, got %s", recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), "0% 到 100%") {
				t.Fatalf("expected invite binding validation message, got %s", recorder.Body.String())
			}
		})
	}
}
