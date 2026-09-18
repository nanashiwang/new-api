package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiStreamHealthSeparatesMetadataAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body        string
		failed, effective bool
	}{
		{"complete", `data: {"usageMetadata":{"totalTokenCount":2}}` + "\n" + `data: {"candidates":[{"content":{"parts":[{"text":"hello"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1,"totalTokenCount":2}}` + "\n", false, true},
		{"metadata_only", `data: {"usageMetadata":{"totalTokenCount":2}}` + "\n", true, false},
		{"truncated", `data: {"candidates":[{"content":{"parts":[{"text":"partial"}]}}],"usageMetadata":{"totalTokenCount":2}}` + "\n", true, true},
		{"in_band_error", `data: {"error":{"code":503,"message":"unavailable"}}` + "\n", true, false},
		{"blocked", `data: {"promptFeedback":{"blockReason":"SAFETY"}}` + "\n", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-test:streamGenerateContent", nil)
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-test"}}
			resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(tc.body))}
			_, err := geminiStreamHandler(c, info, resp, func(_ string, response *dto.GeminiChatResponse) bool {
				if len(response.Candidates) == 0 {
					require.True(t, info.GroupHealthFirstOutputTime.IsZero())
				}
				return true
			})
			require.Nil(t, err)
			require.Equal(t, tc.failed, info.StreamStatus.HasErrors())
			require.Equal(t, tc.effective, !info.GroupHealthFirstOutputTime.IsZero())
		})
	}
}
