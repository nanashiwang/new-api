package openai

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestChatStreamHealthRequiresCompletedOutput(t *testing.T) {
	for _, tc := range []struct {
		name, body        string
		failed, effective bool
	}{
		{"metadata", `data: {"choices":[{"delta":{"role":"assistant","content":""}}]}` + "\n" + "data: [DONE]\n", false, false},
		{"completed", `data: {"choices":[{"delta":{"content":"hello"}}]}` + "\n" + `data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}` + "\n", false, true},
		{"truncated", `data: {"choices":[{"delta":{"content":"partial"}}]}` + "\n", true, true},
		{"explicit_error", `data: {"error":{"type":"server_error","message":"unavailable"}}` + "\n", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newResponsesStreamTestContext()
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-test"}, RelayMode: relayconstant.RelayModeChatCompletions, RelayFormat: types.RelayFormatOpenAI, IsStream: true}
			_, err := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(tc.body))
			failed := err != nil || info.StreamStatus.HasErrors() || !info.StreamStatus.IsNormalEnd()
			require.Equal(t, tc.failed, failed)
			require.Equal(t, tc.effective, !info.GroupHealthFirstOutputTime.IsZero())
		})
	}
}
