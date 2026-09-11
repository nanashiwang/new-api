package channel

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func cpaInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{UserId: 41, ChannelMeta: &relaycommon.ChannelMeta{ChannelSetting: dto.ChannelSettings{CPAUserIdentityEnabled: true, CPAInstanceID: "newapi-main"}}}
}

func TestCPAIdentityOverridesSpoofedValuesAndClearsDisabledChannels(t *testing.T) {
	headers := http.Header{"x-cpa-user-id": {"spoof"}, "X-Cpa-User-Id": {"other"}, "x-cpa-instance-id": {"attacker"}}
	info := cpaInfo()
	require.NoError(t, applyCPAIdentity(headers, info))
	require.Equal(t, []string{"41"}, headers.Values(cpaUserIDHeader))
	require.Equal(t, "newapi-main", headers.Get(cpaInstanceIDHeader))
	require.Len(t, headers, 2)
	info.ChannelSetting.CPAUserIdentityEnabled = false
	require.NoError(t, applyCPAIdentity(headers, info))
	require.Empty(t, headers)
}

func TestCPAIdentityRejectsMissingUserOrInvalidInstance(t *testing.T) {
	for _, user := range []int{0, -1} {
		info := cpaInfo()
		info.UserId = user
		require.Error(t, applyCPAIdentity(http.Header{}, info))
	}
	info := cpaInfo()
	info.ChannelSetting.CPAInstanceID = ""
	require.Error(t, applyCPAIdentity(http.Header{}, info))
	info.ChannelSetting.CPAInstanceID = "bad\r\nHeader: injected"
	require.Error(t, applyCPAIdentity(http.Header{}, info))
}

type cpaAdaptor struct {
	Adaptor
	url string
}

func (a cpaAdaptor) GetRequestURL(*relaycommon.RelayInfo) (string, error) { return a.url, nil }
func (a cpaAdaptor) SetupRequestHeader(_ *gin.Context, h *http.Header, _ *relaycommon.RelayInfo) error {
	h.Set(cpaUserIDHeader, "adapter-spoof")
	return nil
}

func cpaContext() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"user":"spoof"}`))
	c.Request.Header.Set(cpaUserIDHeader, "client-spoof")
	c.Request.Header.Set(cpaInstanceIDHeader, "client-instance")
	return c
}

func TestCPAIdentityHTTPFormStreamAndRetry(t *testing.T) {
	service.InitHttpClient()
	received := make(chan http.Header, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Clone()
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()
	for _, mode := range []string{"http", "form", "stream"} {
		t.Run(mode, func(t *testing.T) {
			info := cpaInfo()
			info.IsStream = mode == "stream"
			info.HeadersOverride = map[string]any{"*": "", cpaUserIDHeader: "override-spoof", cpaInstanceIDHeader: "override-instance"}
			for attempt := 0; attempt < 2; attempt++ {
				info.TokenId = 100 + attempt
				var resp *http.Response
				var err error
				if mode == "form" {
					resp, err = DoFormRequest(cpaAdaptor{url: server.URL}, cpaContext(), info, strings.NewReader("x=y"))
				} else {
					resp, err = DoApiRequest(cpaAdaptor{url: server.URL}, cpaContext(), info, strings.NewReader("{}"))
				}
				require.NoError(t, err)
				_ = resp.Body.Close()
				header := <-received
				require.Equal(t, "41", header.Get(cpaUserIDHeader))
				require.Equal(t, "newapi-main", header.Get(cpaInstanceIDHeader))
			}
		})
	}
}

func TestCPAIdentityWebsocketHandshake(t *testing.T) {
	received := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Clone()
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err == nil {
			_ = conn.Close()
		}
	}))
	defer server.Close()
	info := cpaInfo()
	info.HeadersOverride = map[string]any{cpaUserIDHeader: "override-spoof"}
	conn, err := DoWssRequest(cpaAdaptor{url: "ws" + strings.TrimPrefix(server.URL, "http")}, cpaContext(), info, strings.NewReader("{}"))
	require.NoError(t, err)
	_ = conn.Close()
	require.Equal(t, "41", (<-received).Get(cpaUserIDHeader))
}

func TestCPAIdentityBlocksCrossOriginRedirectWithoutMutatingClient(t *testing.T) {
	leaked := make(chan struct{}, 1)
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked <- struct{}{} }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	base := &http.Client{}
	client := cpaIdentityHTTPClient(base, cpaInfo())
	require.Nil(t, base.CheckRedirect)
	req, _ := http.NewRequest(http.MethodGet, source.URL, nil)
	require.NoError(t, applyCPAIdentity(req.Header, cpaInfo()))
	resp, err := client.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.ErrorContains(t, err, "another origin")
	require.Empty(t, leaked)
}
