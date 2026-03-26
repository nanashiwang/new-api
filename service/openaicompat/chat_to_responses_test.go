package openaicompat

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionsRequestToResponsesRequest_MapsAssistantContentAndFields(t *testing.T) {
	t.Parallel()

	assistant := dto.Message{Role: "assistant"}
	assistant.SetMediaContent([]dto.MediaContent{{Type: dto.ContentTypeText, Text: "done"}})

	req := &dto.GeneralOpenAIRequest{
		Model:                "gpt-5",
		Messages:             []dto.Message{{Role: "system", Content: "be helpful"}, assistant, {Role: "user", Content: "next"}},
		Stream:               true,
		StreamOptions:        &dto.StreamOptions{IncludeUsage: true, IncludeObfuscation: true},
		ServiceTier:          "flex",
		SafetyIdentifier:     "user-1",
		PromptCacheKey:       "cache-key",
		PromptCacheRetention: []byte(`{"type":"ephemeral"}`),
		TopLogProbs:          5,
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, req.StreamOptions, out.StreamOptions)
	require.Equal(t, "flex", out.ServiceTier)
	require.Equal(t, "user-1", out.SafetyIdentifier)
	require.NotNil(t, out.TopLogProbs)
	require.Equal(t, 5, *out.TopLogProbs)

	var promptCacheKey string
	err = common.Unmarshal(out.PromptCacheKey, &promptCacheKey)
	require.NoError(t, err)
	require.Equal(t, "cache-key", promptCacheKey)
	require.JSONEq(t, `{"type":"ephemeral"}`, string(out.PromptCacheRetention))

	var inputItems []map[string]any
	err = common.Unmarshal(out.Input, &inputItems)
	require.NoError(t, err)
	require.Len(t, inputItems, 2)

	assistantContent, ok := inputItems[0]["content"].([]any)
	require.True(t, ok)
	require.Len(t, assistantContent, 1)

	assistantPart, ok := assistantContent[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "output_text", assistantPart["type"])
	require.Equal(t, "done", assistantPart["text"])
}

func TestChatCompletionsRequestToResponsesRequest_OmitsEmptyInstructionsWithoutSystemMessages(t *testing.T) {
	t.Parallel()

	req := &dto.GeneralOpenAIRequest{
		Model:    "gpt-5",
		Messages: []dto.Message{{Role: "user", Content: "hello"}},
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.Len(t, out.Instructions, 0)
}

func TestChatCompletionsRequestToResponsesRequest_MapsSystemMessagesToInstructions(t *testing.T) {
	t.Parallel()

	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "be helpful"},
			{Role: "developer", Content: "format as json"},
			{Role: "user", Content: "hello"},
		},
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)

	var instructions string
	err = common.Unmarshal(out.Instructions, &instructions)
	require.NoError(t, err)
	require.Equal(t, "be helpful\n\nformat as json", instructions)
}

func TestChatToResponses_ToolStrictFieldPreserved(t *testing.T) {
	t.Parallel()

	req := &dto.GeneralOpenAIRequest{
		Model:    "gpt-4o",
		Messages: []dto.Message{{Role: "user", Content: "weather?"}},
		Tools: []dto.ToolCallRequest{
			{
				Type: "function",
				Function: dto.FunctionRequest{
					Name:        "get_weather",
					Description: "Get weather",
					Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      json.RawMessage(`true`),
				},
			},
			{
				Type: "function",
				Function: dto.FunctionRequest{
					Name:        "get_time",
					Description: "Get time",
					Parameters:  map[string]any{"type": "object"},
				},
			},
		},
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.NotNil(t, out.Tools)

	var tools []map[string]any
	err = common.Unmarshal(out.Tools, &tools)
	require.NoError(t, err)
	require.Len(t, tools, 2)

	// First tool should have strict = true
	require.Equal(t, "function", tools[0]["type"])
	require.Equal(t, "get_weather", tools[0]["name"])
	strictVal, ok := tools[0]["strict"]
	require.True(t, ok, "strict field should be present")
	require.Equal(t, true, strictVal)

	// Second tool should NOT have strict field
	_, hasStrict := tools[1]["strict"]
	require.False(t, hasStrict, "strict field should be absent when not set")
}

func TestResponsesToChat_ToolCallsExtractedWithText(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		Model:     "gpt-4o",
		CreatedAt: 1700000000,
		Output: []dto.ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: "Let me check the weather."},
				},
			},
			{
				Type:      "function_call",
				CallId:    "call_123",
				Name:      "get_weather",
				Arguments: `{"city":"Tokyo"}`,
			},
		},
	}

	out, _, err := ResponsesResponseToChatCompletionsResponse(resp, "chatcmpl-1")
	require.NoError(t, err)
	require.Len(t, out.Choices, 1)

	msg := out.Choices[0].Message

	// Both text and tool_calls should be present
	require.Equal(t, "Let me check the weather.", msg.Content)
	require.Equal(t, "tool_calls", out.Choices[0].FinishReason)

	var toolCalls []dto.ToolCallResponse
	err = common.Unmarshal(msg.ToolCalls, &toolCalls)
	require.NoError(t, err)
	require.Len(t, toolCalls, 1)
	require.Equal(t, "call_123", toolCalls[0].ID)
	require.Equal(t, "get_weather", toolCalls[0].Function.Name)
	require.Equal(t, `{"city":"Tokyo"}`, toolCalls[0].Function.Arguments)
}

func TestResponsesToChat_ToolCallsOnlyNilContent(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		Model:     "gpt-4o",
		CreatedAt: 1700000000,
		Output: []dto.ResponsesOutput{
			{
				Type:      "function_call",
				CallId:    "call_456",
				Name:      "search",
				Arguments: `{"q":"test"}`,
			},
		},
	}

	out, _, err := ResponsesResponseToChatCompletionsResponse(resp, "chatcmpl-2")
	require.NoError(t, err)

	msg := out.Choices[0].Message
	require.Nil(t, msg.Content, "content should be nil when no text output")
	require.Equal(t, "tool_calls", out.Choices[0].FinishReason)

	var toolCalls []dto.ToolCallResponse
	err = common.Unmarshal(msg.ToolCalls, &toolCalls)
	require.NoError(t, err)
	require.Len(t, toolCalls, 1)
	require.Equal(t, "search", toolCalls[0].Function.Name)
}
