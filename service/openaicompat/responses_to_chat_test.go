package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestResponsesResponseToChatCompletionsResponse_CountsWebSearchCallsWithoutToolCalls(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		Model:     "gpt-4.1",
		CreatedAt: 1700000000,
		Output: []dto.ResponsesOutput{
			{
				Type: dto.BuildInCallWebSearchCall,
				ID:   "ws_1",
			},
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: "Found the result."},
				},
			},
		},
	}

	out, usage, err := ResponsesResponseToChatCompletionsResponse(resp, "chatcmpl-test")
	require.NoError(t, err)
	require.Equal(t, 1, usage.WebSearchRequests)
	require.Equal(t, "stop", out.Choices[0].FinishReason)
	require.Nil(t, out.Choices[0].Message.ToolCalls)
	require.Equal(t, "Found the result.", out.Choices[0].Message.StringContent())
}
