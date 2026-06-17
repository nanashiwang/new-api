package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
)

func TestInitHttpClientUsesImageHeaderTimeout(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	originalIdleConnTimeout := common.RelayIdleConnTimeout
	originalHeaderTimeout := common.RelayResponseHeaderTimeout
	originalImageHeaderTimeout := common.RelayImageResponseHeaderTimeout
	defer func() {
		common.RelayTimeout = originalRelayTimeout
		common.RelayIdleConnTimeout = originalIdleConnTimeout
		common.RelayResponseHeaderTimeout = originalHeaderTimeout
		common.RelayImageResponseHeaderTimeout = originalImageHeaderTimeout
		InitHttpClient()
	}()

	common.RelayTimeout = 0
	common.RelayIdleConnTimeout = 45
	common.RelayResponseHeaderTimeout = 60
	common.RelayImageResponseHeaderTimeout = 300
	InitHttpClient()

	if got := GetResponseHeaderTimeout(); got != 60*time.Second {
		t.Fatalf("default response header timeout = %s, want 60s", got)
	}
	if got := GetImageResponseHeaderTimeout(); got != 300*time.Second {
		t.Fatalf("image response header timeout = %s, want 300s", got)
	}
	if got := GetHttpClient().Transport.(*http.Transport).IdleConnTimeout; got != 45*time.Second {
		t.Fatalf("idle conn timeout = %s, want 45s", got)
	}
}
