package channel

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

const (
	cpaUserIDHeader     = "X-CPA-User-ID"
	cpaInstanceIDHeader = "X-CPA-Instance-ID"
)

// applyCPAIdentity runs after adapters and all administrator/client header overrides.
// Only authenticated relay metadata may supply the downstream user's identity.
func applyCPAIdentity(header http.Header, info *relaycommon.RelayInfo) error {
	for key := range header {
		if strings.EqualFold(key, cpaUserIDHeader) || strings.EqualFold(key, cpaInstanceIDHeader) {
			delete(header, key)
		}
	}
	if info == nil || info.ChannelMeta == nil || !info.ChannelSetting.CPAUserIdentityEnabled {
		return nil
	}
	if err := info.ChannelSetting.ValidateCPAIdentity(); err != nil {
		return err
	}
	if info.UserId <= 0 {
		return errors.New("CPA identity requires an authenticated user")
	}
	if header == nil {
		return errors.New("CPA identity requires request headers")
	}
	header.Set(cpaUserIDHeader, strconv.Itoa(info.UserId))
	header.Set(cpaInstanceIDHeader, info.ChannelSetting.CPAInstanceID)
	return nil
}

// Clone the client instead of mutating shared redirect policy. Identity headers
// must never follow a redirect to another origin or a less secure scheme.
func cpaIdentityHTTPClient(client *http.Client, info *relaycommon.RelayInfo) *http.Client {
	if info == nil || info.ChannelMeta == nil || !info.ChannelSetting.CPAUserIdentityEnabled {
		return client
	}
	cloned := *client
	previous := client.CheckRedirect
	cloned.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 && (!strings.EqualFold(req.URL.Scheme, via[0].URL.Scheme) || !strings.EqualFold(req.URL.Host, via[0].URL.Host)) {
			return errors.New("CPA identity redirect to another origin is not allowed")
		}
		if previous != nil {
			return previous(req, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return &cloned
}
