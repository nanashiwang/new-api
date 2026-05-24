package dto

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func TestImageRequestIsStreamFromJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := &ImageRequest{}
	if err := common.Unmarshal([]byte(`{"model":"gpt-image-2","prompt":"cat","stream":true}`), req); err != nil {
		t.Fatalf("unmarshal image request: %v", err)
	}

	if !req.IsStream(nil) {
		t.Fatal("expected JSON stream=true to enable streaming")
	}
}

func TestImageRequestIsStreamFromMultipartForm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", strings.NewReader(""))
	ctx.Request.PostForm = map[string][]string{"stream": {"true"}}

	req := &ImageRequest{}
	if !req.IsStream(ctx) {
		t.Fatal("expected multipart stream=true to enable streaming")
	}
}
