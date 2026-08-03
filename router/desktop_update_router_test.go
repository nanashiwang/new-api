package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func setDesktopUpdateRouterTestSettings(t *testing.T, raw string) {
	t.Helper()
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	old, existed := common.OptionMap[service.DesktopUpdateSettingsOptionKey]
	common.OptionMap[service.DesktopUpdateSettingsOptionKey] = raw
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if existed {
			common.OptionMap[service.DesktopUpdateSettingsOptionKey] = old
		} else {
			delete(common.OptionMap, service.DesktopUpdateSettingsOptionKey)
		}
		common.OptionMapRWMutex.Unlock()
	})
}

func TestDesktopUpdatePublicRoutesSupportHeadAndRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	t.Setenv("DESKTOP_UPDATE_DIR", root)
	setDesktopUpdateRouterTestSettings(t, `{"enabled":true,"public_base_url":"https://updates.example.com/desktop/update","max_upload_mb":256,"retention_count":10}`)

	content := "0123456789"
	if _, err := service.SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader(content), 1024); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveDesktopUpdateArtifact("1.2.3", "YuanHeng Desktop_1.2.3_x64-setup.exe", strings.NewReader("installer"), 1024); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":"1.2.3","platforms":{"windows-x86_64":{"signature":"signed","url":"https://example.com/bundle.exe"}}}`
	if _, err := service.PublishDesktopUpdateManifest(strings.NewReader(manifest), service.GetDesktopUpdateSettings().PublicBaseURL, 10); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	SetDesktopUpdateRouter(r)

	head := httptest.NewRecorder()
	r.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/desktop/update/latest.json", nil))
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("unexpected HEAD response: code=%d body=%q", head.Code, head.Body.String())
	}
	if head.Header().Get("Cache-Control") != "no-cache, no-store, must-revalidate" {
		t.Fatalf("unexpected manifest cache policy: %q", head.Header().Get("Cache-Control"))
	}

	catalogResponse := httptest.NewRecorder()
	r.ServeHTTP(catalogResponse, httptest.NewRequest(http.MethodGet, "/desktop/update/downloads.json", nil))
	if catalogResponse.Code != http.StatusOK {
		t.Fatalf("unexpected downloads response: code=%d body=%q", catalogResponse.Code, catalogResponse.Body.String())
	}
	if catalogResponse.Header().Get("Cache-Control") != "public, max-age=300, must-revalidate" {
		t.Fatalf("unexpected downloads cache policy: %q", catalogResponse.Header().Get("Cache-Control"))
	}
	var catalog service.DesktopDownloadCatalog
	if err := common.Unmarshal(catalogResponse.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.Version != "1.2.3" || len(catalog.Packages) != 1 || catalog.Packages[0].ID != "windows-x64" {
		t.Fatalf("unexpected downloads catalog: %+v", catalog)
	}
	if etag := catalogResponse.Header().Get("ETag"); etag == "" {
		t.Fatal("downloads catalog must include an ETag")
	} else {
		notModified := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/desktop/update/downloads.json", nil)
		request.Header.Set("If-None-Match", etag)
		r.ServeHTTP(notModified, request)
		if notModified.Code != http.StatusNotModified || notModified.Body.Len() != 0 {
			t.Fatalf("unexpected conditional response: code=%d body=%q", notModified.Code, notModified.Body.String())
		}
	}
	catalogHead := httptest.NewRecorder()
	r.ServeHTTP(catalogHead, httptest.NewRequest(http.MethodHead, "/desktop/update/downloads.json", nil))
	if catalogHead.Code != http.StatusOK || catalogHead.Body.Len() != 0 || catalogHead.Header().Get("Content-Length") == "" {
		t.Fatalf("unexpected downloads HEAD response: code=%d body=%q length=%q", catalogHead.Code, catalogHead.Body.String(), catalogHead.Header().Get("Content-Length"))
	}

	rangeResponse := httptest.NewRecorder()
	rangeRequest := httptest.NewRequest(http.MethodGet, "/desktop/update/releases/1.2.3/bundle.exe", nil)
	rangeRequest.Header.Set("Range", "bytes=2-5")
	r.ServeHTTP(rangeResponse, rangeRequest)
	if rangeResponse.Code != http.StatusPartialContent || rangeResponse.Body.String() != "2345" {
		t.Fatalf("unexpected range response: code=%d body=%q", rangeResponse.Code, rangeResponse.Body.String())
	}
	if rangeResponse.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("unexpected artifact cache policy: %q", rangeResponse.Header().Get("Cache-Control"))
	}

	if err := os.Remove(filepath.Join(root, "releases", "1.2.3", "bundle.exe")); err != nil {
		t.Fatal(err)
	}
	invalidManifest := httptest.NewRecorder()
	r.ServeHTTP(invalidManifest, httptest.NewRequest(http.MethodGet, "/desktop/update/latest.json", nil))
	if invalidManifest.Code != http.StatusInternalServerError {
		t.Fatalf("invalid active manifest must fail for updater fallback, got %d", invalidManifest.Code)
	}
}

func TestDesktopUpdatePublicRoutesAreHiddenWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DESKTOP_UPDATE_DIR", t.TempDir())
	setDesktopUpdateRouterTestSettings(t, `{"enabled":false,"public_base_url":"https://updates.example.com/desktop/update","max_upload_mb":256,"retention_count":10}`)

	r := gin.New()
	SetDesktopUpdateRouter(r)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/desktop/update/latest.json", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected disabled route to be hidden, got %d", response.Code)
	}
	if response.Header().Get("Cache-Control") != "no-cache, no-store, must-revalidate" {
		t.Fatalf("disabled manifest response must not be cached: %q", response.Header().Get("Cache-Control"))
	}

	downloads := httptest.NewRecorder()
	r.ServeHTTP(downloads, httptest.NewRequest(http.MethodGet, "/desktop/update/downloads.json", nil))
	if downloads.Code != http.StatusNotFound || downloads.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("disabled downloads response must be hidden and uncached: code=%d cache=%q", downloads.Code, downloads.Header().Get("Cache-Control"))
	}

	artifact := httptest.NewRecorder()
	r.ServeHTTP(artifact, httptest.NewRequest(http.MethodGet, "/desktop/update/releases/1.2.3/bundle.exe", nil))
	if artifact.Code != http.StatusNotFound || artifact.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("disabled artifact response must be hidden and uncached: code=%d cache=%q", artifact.Code, artifact.Header().Get("Cache-Control"))
	}
}

func TestDesktopUpdatePublisherAuthAndBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DESKTOP_UPDATE_DIR", t.TempDir())
	setDesktopUpdateRouterTestSettings(t, `{"enabled":true,"public_base_url":"https://updates.example.com/desktop/update","max_upload_mb":1,"retention_count":10}`)

	common.OptionMapRWMutex.Lock()
	oldToken, tokenExisted := common.OptionMap[service.DesktopUpdateTokenOptionKey]
	common.OptionMap[service.DesktopUpdateTokenOptionKey] = service.HashDesktopUpdatePublishToken("publish-token")
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if tokenExisted {
			common.OptionMap[service.DesktopUpdateTokenOptionKey] = oldToken
		} else {
			delete(common.OptionMap, service.DesktopUpdateTokenOptionKey)
		}
		common.OptionMapRWMutex.Unlock()
	})

	r := gin.New()
	SetDesktopUpdateRouter(r)

	unauthorized := httptest.NewRecorder()
	r.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPut, "/desktop/update/publish/1.2.3/bundle.exe", strings.NewReader("bundle")))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized publisher response, got %d", unauthorized.Code)
	}

	oversized := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/desktop/update/publish/1.2.3/bundle.exe", strings.NewReader(strings.Repeat("x", 1024*1024+1)))
	request.Header.Set("Authorization", "Bearer publish-token")
	r.ServeHTTP(oversized, request)
	if oversized.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized upload, got %d: %s", oversized.Code, oversized.Body.String())
	}

	invalidManifest := httptest.NewRecorder()
	manifestRequest := httptest.NewRequest(http.MethodPut, "/desktop/update/publish/latest.json", strings.NewReader(`{"version":`))
	manifestRequest.Header.Set("Authorization", "Bearer publish-token")
	r.ServeHTTP(invalidManifest, manifestRequest)
	if invalidManifest.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed manifest, got %d: %s", invalidManifest.Code, invalidManifest.Body.String())
	}
}
