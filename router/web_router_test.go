package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestImagePlaygroundIndexRoutesDoNotRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerImagePlaygroundIndexRoutes(r, []byte("<html>image playground</html>"))

	for _, path := range []string{
		"/image-playground?apiKey=sk-test",
		"/image-playground/?apiKey=sk-test",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("%s: expected status 200, got %d", path, w.Code)
		}
		if location := w.Header().Get("Location"); location != "" {
			t.Fatalf("%s: expected no redirect location, got %q", path, location)
		}
		if body := w.Body.String(); body != "<html>image playground</html>" {
			t.Fatalf("%s: unexpected body %q", path, body)
		}
	}
}
