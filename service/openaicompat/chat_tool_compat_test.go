package openaicompat

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestNormalizeChatToolProtocol_ConvertsLegacyRequestAndHistory(t *testing.T) {
	functionName := "lookup"
	req := &dto.GeneralOpenAIRequest{
		Model:        "gpt-5",
		Functions:    json.RawMessage(`[{"name":"lookup","description":"Look up data","parameters":{"type":"object","properties":{"q":{"type":"string"}}}}]`),
		FunctionCall: json.RawMessage(`{"name":"lookup"}`),
		Messages: []dto.Message{
			{Role: "user", Content: "find it"},
			{Role: "assistant", Content: nil, FunctionCall: json.RawMessage(`{"name":"lookup","arguments":"{\"q\":\"x\"}"}`)},
			{Role: "function", Name: &functionName, Content: `{"result":"ok"}`},
			{Role: "user", Content: "continue"},
		},
	}

	protocol, err := NormalizeChatToolProtocol(req)
	require.NoError(t, err)
	require.Equal(t, dto.ChatToolProtocolLegacy, protocol)
	require.Nil(t, req.Functions)
	require.Nil(t, req.FunctionCall)
	require.Len(t, req.Tools, 1)
	require.Equal(t, "lookup", req.Tools[0].Function.Name)
	require.NotNil(t, req.ParallelTooCalls)
	require.False(t, *req.ParallelTooCalls)

	choice, ok := req.ToolChoice.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "function", choice["type"])

	historyCalls := req.Messages[1].ParseToolCalls()
	require.Len(t, historyCalls, 1)
	require.Equal(t, "legacy_call_1", historyCalls[0].ID)
	require.Nil(t, req.Messages[1].FunctionCall)
	require.Equal(t, "tool", req.Messages[2].Role)
	require.Equal(t, historyCalls[0].ID, req.Messages[2].ToolCallId)

	responsesReq, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.JSONEq(t, `[{"type":"function","name":"lookup","description":"Look up data","parameters":{"type":"object","properties":{"q":{"type":"string"}}}}]`, string(responsesReq.Tools))
	require.JSONEq(t, `{"type":"function","name":"lookup"}`, string(responsesReq.ToolChoice))

	var input []map[string]any
	require.NoError(t, common.Unmarshal(responsesReq.Input, &input))
	require.Equal(t, "function_call", input[2]["type"])
	require.Equal(t, "function_call_output", input[3]["type"])
	require.Equal(t, input[2]["call_id"], input[3]["call_id"])
}

func TestNormalizeChatToolProtocol_LeavesModernRequestModern(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Tools: []dto.ToolCallRequest{{
			Type: "function",
			Function: dto.FunctionRequest{
				Name:       "lookup",
				Parameters: map[string]any{"type": "object"},
			},
		}},
		ToolChoice: "auto",
	}

	protocol, err := NormalizeChatToolProtocol(req)
	require.NoError(t, err)
	require.Equal(t, dto.ChatToolProtocolModern, protocol)
	require.Len(t, req.Tools, 1)
	require.Equal(t, "auto", req.ToolChoice)
	require.Nil(t, req.ParallelTooCalls)
}

func TestNormalizeChatToolProtocol_RejectsAmbiguousTopLevelProtocol(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Functions: json.RawMessage(`[{"name":"legacy"}]`),
		Tools: []dto.ToolCallRequest{{
			Type:     "function",
			Function: dto.FunctionRequest{Name: "modern"},
		}},
	}

	_, err := NormalizeChatToolProtocol(req)
	require.ErrorContains(t, err, "ambiguous tool protocol")
}

func TestNormalizeChatToolProtocol_RejectsUndefinedForcedFunction(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Functions:    json.RawMessage(`[{"name":"lookup"}]`),
		FunctionCall: json.RawMessage(`{"name":"missing"}`),
	}

	_, err := NormalizeChatToolProtocol(req)
	require.ErrorContains(t, err, "undefined function")
}

func TestNormalizeChatToolProtocol_RejectsParallelLegacyFunctions(t *testing.T) {
	parallel := true
	req := &dto.GeneralOpenAIRequest{
		Functions:        json.RawMessage(`[{"name":"lookup"}]`),
		ParallelTooCalls: &parallel,
	}

	_, err := NormalizeChatToolProtocol(req)
	require.ErrorContains(t, err, "parallel_tool_calls=true")
}

func TestNormalizeChatToolProtocol_MapsLegacyStringChoices(t *testing.T) {
	for _, choice := range []string{"auto", "none"} {
		t.Run(choice, func(t *testing.T) {
			req := &dto.GeneralOpenAIRequest{
				Functions:    json.RawMessage(`[{"name":"lookup"}]`),
				FunctionCall: json.RawMessage(`"` + choice + `"`),
			}

			protocol, err := NormalizeChatToolProtocol(req)
			require.NoError(t, err)
			require.Equal(t, dto.ChatToolProtocolLegacy, protocol)
			require.Equal(t, choice, req.ToolChoice)
		})
	}
}

func TestNormalizeChatToolProtocol_RejectsInvalidLegacyDefinitions(t *testing.T) {
	tests := []struct {
		name      string
		functions string
		contains  string
	}{
		{name: "duplicate name", functions: `[{"name":"lookup"},{"name":"lookup"}]`, contains: "duplicate function name"},
		{name: "array parameters", functions: `[{"name":"lookup","parameters":[]}]`, contains: "parameters must be a JSON object"},
		{name: "missing name", functions: `[{"description":"missing"}]`, contains: "name is required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &dto.GeneralOpenAIRequest{Functions: json.RawMessage(test.functions)}
			_, err := NormalizeChatToolProtocol(req)
			require.ErrorContains(t, err, test.contains)
		})
	}
}

func TestNormalizeChatToolProtocol_RejectsMalformedLegacyHistory(t *testing.T) {
	tests := []struct {
		name     string
		messages []dto.Message
		contains string
	}{
		{
			name: "missing result",
			messages: []dto.Message{{
				Role: "assistant", FunctionCall: json.RawMessage(`{"name":"lookup","arguments":"{}"}`),
			}},
			contains: "missing its role=function result",
		},
		{
			name: "orphan result",
			messages: []dto.Message{{
				Role: "function", Content: "ok",
			}},
			contains: "no preceding function_call",
		},
		{
			name: "mixed assistant formats",
			messages: []dto.Message{{
				Role:         "assistant",
				FunctionCall: json.RawMessage(`{"name":"lookup","arguments":"{}"}`),
				ToolCalls:    json.RawMessage(`[{"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{}"}}]`),
			}},
			contains: "mixes function_call and tool_calls",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &dto.GeneralOpenAIRequest{Messages: test.messages}
			_, err := NormalizeChatToolProtocol(req)
			require.ErrorContains(t, err, test.contains)
		})
	}
}

func TestNormalizeChatToolProtocol_AvoidsSyntheticCallIDCollision(t *testing.T) {
	functionName := "legacy"
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "assistant", ToolCalls: json.RawMessage(`[{"id":"legacy_call_1","type":"function","function":{"name":"modern","arguments":"{}"}}]`)},
			{Role: "tool", ToolCallId: "legacy_call_1", Content: "ok"},
			{Role: "assistant", FunctionCall: json.RawMessage(`{"name":"legacy","arguments":"{}"}`)},
			{Role: "function", Name: &functionName, Content: "ok"},
		},
	}

	protocol, err := NormalizeChatToolProtocol(req)
	require.NoError(t, err)
	require.Equal(t, dto.ChatToolProtocolLegacy, protocol)
	legacyCalls := req.Messages[2].ParseToolCalls()
	require.Len(t, legacyCalls, 1)
	require.Equal(t, "legacy_call_2", legacyCalls[0].ID)
	require.Equal(t, "legacy_call_2", req.Messages[3].ToolCallId)
}
