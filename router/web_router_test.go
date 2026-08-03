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

func TestPrerenderedIndexRoutesDoNotRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerPrerenderedIndexRoutes(r, []byte("<html>index</html>"))

	for _, path := range []string{
		"/register?aff=R3bk",
		"/register/?aff=R3bk",
		"/login?next=/console",
		"/pricing",
		"/download",
		"/download/",
		"/about",
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
	}
}

func TestShouldReturnRelayNotFoundForStaticAndApiPaths(t *testing.T) {
	tests := []struct {
		requestURI string
		want       bool
	}{
		{requestURI: "/api/status", want: true},
		{requestURI: "/v1/models", want: true},
		{requestURI: "/assets/index.js", want: true},
		{requestURI: "/image-playground/assets/missing.js", want: true},
		{requestURI: "/image-playground/assets/missing.js?v=stale", want: true},
		{requestURI: "/image-playground/assets/", want: true},
		{requestURI: "/image-playground/?appMode=gallery", want: false},
		{requestURI: "/console/image-playground", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.requestURI, func(t *testing.T) {
			if got := shouldReturnRelayNotFound(tt.requestURI); got != tt.want {
				t.Fatalf("shouldReturnRelayNotFound(%q) = %v, want %v", tt.requestURI, got, tt.want)
			}
		})
	}
}

func TestRedirectMalformedFullwidthQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		target   string
		location string
	}{
		{
			name:     "register invite link",
			target:   "/register/%EF%BC%9Faff=Xr62",
			location: "/register?aff=Xr62",
		},
		{
			name:     "merge existing query",
			target:   "/register/%EF%BC%9Faff=Xr62&source=invite?lang=zh",
			location: "/register?aff=Xr62&source=invite&lang=zh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.target, nil)

			if !redirectMalformedFullwidthQuery(c) {
				t.Fatal("expected malformed query redirect")
			}
			if w.Code != http.StatusFound {
				t.Fatalf("expected status 302, got %d", w.Code)
			}
			if location := w.Header().Get("Location"); location != tt.location {
				t.Fatalf("expected location %q, got %q", tt.location, location)
			}
		})
	}
}

func TestRedirectMalformedFullwidthQueryIgnoresNormalPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/register?aff=Xr62", nil)

	if redirectMalformedFullwidthQuery(c) {
		t.Fatal("expected normal query path to be ignored")
	}
}
