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

func TestDialogRequestBodyLimitIsConsistentAcrossProtocols(t *testing.T) {
	oldGlobal, oldBusiness := constant.MaxRequestBodyMB, constant.ResponsesRequestBodyLimitMB
	t.Cleanup(func() {
		constant.MaxRequestBodyMB, constant.ResponsesRequestBodyLimitMB = oldGlobal, oldBusiness
	})
	for _, path := range []string{
		"/v1/chat/completions", "/v1/messages", "/v1/responses", "/v1/responses/compact",
		"/openai/v1/chat/completions", "/openai/v1/messages", "/openai/v1/responses", "/pg/chat/completions",
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, path, nil)
		for _, limits := range [][3]int{{256, 192, 192}, {128, 192, 128}, {256, 0, 256}, {256, 20, 20}} {
			constant.MaxRequestBodyMB, constant.ResponsesRequestBodyLimitMB = limits[0], limits[1]
			if got := GetRequestBodyLimitMB(c); got != limits[2] {
				t.Fatalf("%s global=%d business=%d: got %d want %d", path, limits[0], limits[1], got, limits[2])
			}
		}
	}
	for _, path := range []string{"/v1/messages-other", "/v1/responses_other", "/v1/chat/completionsFake", "/api/user/", "/v1/images/edits"} {
		if IsResponsesRequestBodyLimitedPath(path) {
			t.Fatalf("unrelated path matched business limit: %s", path)
		}
	}
}
