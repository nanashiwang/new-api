package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
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

func TestResponsesResponseToChatCompletionsResponse_ProjectsLegacyFunctionCallWithText(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		Model:     "gpt-5",
		CreatedAt: 1700000000,
		Output: []dto.ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: "I will check it."},
				},
			},
			{
				Type:      "function_call",
				CallId:    "call_lookup",
				Name:      "lookup",
				Arguments: dto.ResponsesArguments(`{"q":"status"}`),
			},
		},
	}

	out, _, err := ResponsesResponseToChatCompletionsResponseWithToolProtocol(resp, "chatcmpl-legacy", dto.ChatToolProtocolLegacy)
	require.NoError(t, err)
	require.Equal(t, "function_call", out.Choices[0].FinishReason)
	require.Equal(t, "I will check it.", out.Choices[0].Message.StringContent())
	require.Nil(t, out.Choices[0].Message.ToolCalls)

	var functionCall dto.FunctionResponse
	require.NoError(t, common.Unmarshal(out.Choices[0].Message.FunctionCall, &functionCall))
	require.Equal(t, "lookup", functionCall.Name)
	require.JSONEq(t, `{"q":"status"}`, functionCall.Arguments)
}

func TestResponsesResponseToChatCompletionsResponse_RejectsParallelLegacyFunctionCalls(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		Output: []dto.ResponsesOutput{
			{Type: "function_call", CallId: "call_1", Name: "first", Arguments: dto.ResponsesArguments(`{}`)},
			{Type: "function_call", CallId: "call_2", Name: "second", Arguments: dto.ResponsesArguments(`{}`)},
		},
	}

	_, _, err := ResponsesResponseToChatCompletionsResponseWithToolProtocol(resp, "chatcmpl-legacy", dto.ChatToolProtocolLegacy)
	require.ErrorContains(t, err, "cannot represent multiple function calls")
}

func TestResponsesResponseToChatCompletionsResponse_PreservesReasoningSummary(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		Model:     "gpt-5",
		CreatedAt: 1700000000,
		Output: []dto.ResponsesOutput{
			{
				Type: "reasoning",
				Summary: []dto.ResponsesReasoningSummaryPart{
					{Type: "summary_text", Text: "Checked constraints."},
					{Type: "summary_text", Text: "Picked the shortest plan."},
				},
			},
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: "Here is the answer."},
				},
			},
		},
	}

	out, _, err := ResponsesResponseToChatCompletionsResponse(resp, "chatcmpl-reasoning")
	require.NoError(t, err)
	require.Equal(t, "Here is the answer.", out.Choices[0].Message.StringContent())
	require.Equal(t, "Checked constraints.\n\nPicked the shortest plan.", out.Choices[0].Message.ReasoningContent)
}
