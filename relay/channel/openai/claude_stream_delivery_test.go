package openai

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func claudeDeliveryInfo() *relaycommon.RelayInfo {
	info := chatDeliveryInfo()
	info.RelayFormat = types.RelayFormatClaude
	info.ClaudeConvertInfo = &relaycommon.ClaudeConvertInfo{}
	return info
}

func TestClaudeStreamFirstContentDoesNotWaitForNextChunk(t *testing.T) {
	newResponsesStreamTestContext()
	for _, delta := range []string{`{"content":"first"}`, `{"reasoning_content":"first"}`, `{"tool_calls":[{"index":9,"id":"call","function":{"name":"first","arguments":"{}"}}]}`} {
		t.Run(delta, func(t *testing.T) {
			release := make(chan struct{})
			finished := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reader, writer := io.Pipe()
				defer reader.Close()
				go func() {
					defer writer.Close()
					_, _ = io.WriteString(writer, "data: "+`{"choices":[{"delta":`+delta+`}]}`+"\n\n")
					select {
					case <-release:
						_, _ = io.WriteString(writer, "data: "+`{"choices":[{"delta":{},"finish_reason":"stop"}]}`+"\n\ndata: [DONE]\n\n")
					case <-r.Context().Done():
					}
				}()
				c, _ := gin.CreateTestContext(w)
				c.Request = r
				_, _ = OaiStreamHandler(c, claudeDeliveryInfo(), &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader})
				close(finished)
			}))
			t.Cleanup(func() {
				close(release)
				server.CloseClientConnections()
				server.Close()
			})
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Get(server.URL)
			require.NoError(t, err, "content must arrive while the upstream is paused")
			defer resp.Body.Close()
			reader := bufio.NewReader(resp.Body)
			for {
				line, err := reader.ReadString('\n')
				require.NoError(t, err)
				if strings.Contains(line, `"first"`) {
					break
				}
			}
			resp.Body.Close()
			select {
			case <-finished:
			case <-time.After(3 * time.Second):
				t.Fatal("handler must exit after cancellation")
			}
		})
	}
}

func TestClaudeStreamTerminalUsesLatestUsageWithoutReplayingContent(t *testing.T) {
	c, recorder := newResponsesStreamTestContext()
	body := "data: " + `{"model":"kimi-k3","choices":[{"delta":{"reasoning_content":"plan","content":"answer"},"finish_reason":"length"}]}` + "\n\n" +
		"data: " + `{"choices":[],"usage":{"prompt_tokens":42,"completion_tokens":9,"total_tokens":51}}` + "\n\n" +
		"data: " + `{"choices":[],"metadata":"trailer"}` + "\n\ndata: [DONE]\n\n"
	usage, err := OaiStreamHandler(c, claudeDeliveryInfo(), newResponsesStreamHTTPResponse(body))
	require.Nil(t, err)
	require.Equal(t, 9, usage.CompletionTokens)
	out := recorder.Body.String()
	require.Equal(t, 1, strings.Count(out, "event: message_start"))
	require.Equal(t, 1, strings.Count(out, "event: message_stop"))
	require.Equal(t, 1, strings.Count(out, `"text":"answer"`))
	require.Equal(t, 1, strings.Count(out, `"thinking":"plan"`))
	var final dto.ClaudeResponse
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"type":"message_delta"`) {
			require.NoError(t, common.UnmarshalJsonStr(strings.TrimPrefix(line, "data: "), &final))
		}
	}
	require.NotNil(t, final.Usage)
	require.Equal(t, 9, final.Usage.OutputTokens)
	require.Equal(t, "max_tokens", *final.Delta.StopReason)
}

func TestChatAndClaudeInterruptedStreamsNeverSendSuccessTerminal(t *testing.T) {
	for _, format := range []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude} {
		for _, ending := range []string{"", "data: {bad json}\n\n", "data: " + `{"error":{"type":"server_error","message":"failed"}}` + "\n\n"} {
			c, recorder := newResponsesStreamTestContext()
			info := claudeDeliveryInfo()
			info.RelayFormat = format
			body := "data: " + `{"choices":[{"delta":{"content":"partial"}}]}` + "\n\n" + ending
			_, err := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
			require.NotNil(t, err, string(format)+ending)
			require.True(t, types.IsSkipRetryError(err))
			require.NotContains(t, recorder.Body.String(), "message_stop")
			require.NotContains(t, recorder.Body.String(), "[DONE]")
		}
	}
}
