package ollama

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOllamaFinalFramePayloadBeforeFinish(t *testing.T) {
	for _, payload := range []string{
		`"message":{"content":"final-text","thinking":"final-thought","tool_calls":[{"id":"original-id","function":{"name":"search","arguments":{"q":"test"}}},{"function":{"name":"clock","arguments":{}}}]}`,
		`"response":"final-text"`,
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		resp := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{` + payload + `,"done":true,"prompt_eval_count":5,"eval_count":7}`))}
		usage, apiErr := ollamaStreamHandler(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "test"}}, resp)
		require.Nil(t, apiErr)
		require.Equal(t, 12, usage.TotalTokens)
		var content, thinking strings.Builder
		var calls []dto.ToolCallResponse
		finished := false
		for _, line := range strings.Split(w.Body.String(), "\n") {
			data, ok := strings.CutPrefix(line, "data: ")
			if !ok || data == "[DONE]" {
				continue
			}
			var chunk dto.ChatCompletionsStreamResponse
			require.NoError(t, common.Unmarshal([]byte(data), &chunk))
			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]
			if choice.FinishReason != nil {
				finished = true
				if len(calls) > 0 {
					require.Equal(t, "tool_calls", *choice.FinishReason)
				}
				continue
			}
			require.False(t, finished)
			content.WriteString(choice.Delta.GetContentString())
			thinking.WriteString(choice.Delta.GetReasoningContent())
			calls = append(calls, choice.Delta.ToolCalls...)
		}
		require.Equal(t, "final-text", content.String())
		require.True(t, finished)
		if strings.Contains(payload, "tool_calls") {
			require.Equal(t, "final-thought", thinking.String())
			require.Len(t, calls, 2)
			require.Equal(t, "original-id", calls[0].ID)
			require.Equal(t, "call_1", calls[1].ID)
			require.Equal(t, 0, *calls[0].Index)
			require.Equal(t, 1, *calls[1].Index)
		}
	}
}
