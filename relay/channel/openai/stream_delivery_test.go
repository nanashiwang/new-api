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
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func chatDeliveryInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kimi-k3"},
		RelayMode:   relayconstant.RelayModeChatCompletions, RelayFormat: types.RelayFormatOpenAI,
		IsStream: true, ShouldIncludeUsage: true,
	}
}

func TestChatStreamDeliversFirstEventBeforeNextUpstreamChunk(t *testing.T) {
	newResponsesStreamTestContext()
	for _, delta := range []string{`{"content":"first"}`, `{"reasoning_content":"first"}`, `{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"lookup","arguments":""}}]}`} {
		t.Run(delta, func(t *testing.T) {
			release := make(chan struct{})
			finished := make(chan struct{})
			first := `{"model":"kimi-k3","choices":[{"index":0,"delta":` + delta + `}]}`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reader, writer := io.Pipe()
				defer reader.Close()
				go func() {
					defer writer.Close()
					_, _ = io.WriteString(writer, "data: "+first+"\n\n")
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
					_, _ = io.WriteString(writer, "data: {\"model\":\"kimi-k3\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":1,\"total_tokens\":11}}\n\ndata: [DONE]\n\n")
				}()
				c, _ := gin.CreateTestContext(w)
				c.Request = r
				_, _ = OaiStreamHandler(c, chatDeliveryInfo(), &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader})
				close(finished)
			}))
			defer server.Close()
			defer close(release)
			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get(server.URL)
			require.NoError(t, err, "first event must be flushed while upstream waits for the client")
			defer resp.Body.Close()
			line, err := bufio.NewReader(resp.Body).ReadString('\n')
			require.NoError(t, err)
			require.Equal(t, "data: "+first+"\n", line)
			// Closing the response cancels the deliberately paused upstream; the
			// normal terminal/usage cases are exercised separately below.
			resp.Body.Close()
			select {
			case <-finished:
			case <-time.After(3 * time.Second):
				t.Fatal("stream handler did not stop after client disconnect")
			}
		})
	}
}

func TestChatStreamUsageAndFinishAreDeliveredExactlyOnce(t *testing.T) {
	for _, include := range []bool{false, true} {
		c, recorder := newResponsesStreamTestContext()
		info := chatDeliveryInfo()
		info.ShouldIncludeUsage = include
		body := "data: " + `{"model":"kimi-k3","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"lookup","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}` + "\n\n" +
			"data: " + `{"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}` + "\n\n" +
			"data: " + `{"choices":[],"metadata":"trailer"}` + "\n\ndata: [DONE]\n\n"
		usage, err := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
		require.Nil(t, err)
		require.Equal(t, 10, usage.PromptTokens)
		require.Equal(t, 2, usage.CompletionTokens)
		out := recorder.Body.String()
		require.Equal(t, 1, strings.Count(out, `"finish_reason":"tool_calls"`))
		require.Equal(t, 1, strings.Count(out, `"id":"call_1"`))
		require.Equal(t, 1, strings.Count(out, "data: [DONE]"))
		require.Equal(t, include, strings.Contains(out, `"usage"`))
		require.Contains(t, out, `"metadata":"trailer"`)
	}
}

func TestChatStructuredStreamKeepsReasoningOutOfJSON(t *testing.T) {
	for _, format := range []string{"json_object", "json_schema", "text"} {
		c, recorder := newResponsesStreamTestContext()
		info := chatDeliveryInfo()
		info.ChannelSetting.ThinkingToContent = true
		info.ThinkingContentInfo.IsFirstThinkingContent = true
		info.Request = &dto.GeneralOpenAIRequest{ResponseFormat: &dto.ResponseFormat{Type: format}}
		body := "data: " + `{"choices":[{"delta":{"reasoning_content":"plan"}}]}` + "\n\n" +
			"data: " + `{"choices":[{"delta":{"content":"{\"ok\":true}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":5,"total_tokens":6}}` + "\n\ndata: [DONE]\n\n"
		_, err := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
		require.Nil(t, err)
		var content strings.Builder
		for _, line := range strings.Split(recorder.Body.String(), "\n") {
			if !strings.HasPrefix(line, "data: {") {
				continue
			}
			var chunk dto.ChatCompletionsStreamResponse
			require.NoError(t, common.UnmarshalJsonStr(strings.TrimPrefix(line, "data: "), &chunk))
			for _, choice := range chunk.Choices {
				content.WriteString(choice.Delta.GetContentString())
			}
		}
		if format == "text" {
			require.Contains(t, content.String(), "<think>")
		} else {
			require.JSONEq(t, `{"ok":true}`, content.String())
			require.Contains(t, recorder.Body.String(), `"reasoning_content":"plan"`)
		}
	}
}

func TestChatStreamErrorDoesNotReplayDeliveredOutput(t *testing.T) {
	c, recorder := newResponsesStreamTestContext()
	info := chatDeliveryInfo()
	body := "data: " + `{"choices":[{"delta":{"content":"partial"}}]}` + "\n\n" +
		"data: " + `{"error":{"type":"server_error","code":"upstream_error","message":"unavailable"}}` + "\n\n"
	_, err := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.NotNil(t, err)
	require.True(t, types.IsSkipRetryError(err))
	require.Equal(t, 1, strings.Count(recorder.Body.String(), `"content":"partial"`))
	require.Equal(t, 1, strings.Count(recorder.Body.String(), `"code":"upstream_error"`))
	require.NotContains(t, recorder.Body.String(), "[DONE]")
}
