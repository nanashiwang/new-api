package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func TestGetRequestBodyLimitMB_ResponsesUsesLowerBusinessLimit(t *testing.T) {
	originMax := constant.MaxRequestBodyMB
	originResponses := constant.ResponsesRequestBodyLimitMB
	constant.MaxRequestBodyMB = 256
	constant.ResponsesRequestBodyLimitMB = 20
	t.Cleanup(func() {
		constant.MaxRequestBodyMB = originMax
		constant.ResponsesRequestBodyLimitMB = originResponses
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	if got := GetRequestBodyLimitMB(ctx); got != 20 {
		t.Fatalf("responses limit = %d, want 20", got)
	}
}

func TestGetRequestBodyLimitMB_NonResponsesUsesGlobalLimit(t *testing.T) {
	originMax := constant.MaxRequestBodyMB
	originResponses := constant.ResponsesRequestBodyLimitMB
	constant.MaxRequestBodyMB = 256
	constant.ResponsesRequestBodyLimitMB = 20
	t.Cleanup(func() {
		constant.MaxRequestBodyMB = originMax
		constant.ResponsesRequestBodyLimitMB = originResponses
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)

	if got := GetRequestBodyLimitMB(ctx); got != 256 {
		t.Fatalf("non-responses limit = %d, want 256", got)
	}
}

func TestFormatRequestBodyTooLargeMessageIncludesActionableHint(t *testing.T) {
	msg := FormatRequestBodyTooLargeMessage(30<<20, 20<<20)
	for _, want := range []string{"30.00 MiB", "20.00 MiB", "减少图片数量"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message %q missing %q", msg, want)
		}
	}
}
