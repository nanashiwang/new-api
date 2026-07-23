package service

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestSanitizeClaudeToolArguments_RemovesReasonFromShutdownApproval(t *testing.T) {
	raw := `{"to":"team-lead","type":"shutdown_response","approve":true,"reason":"done","message":{"type":"shutdown_response","request_id":"shutdown-1","approve":true,"reason":"done"}}`

	sanitized := SanitizeClaudeToolArguments("SendMessage", raw)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(sanitized), &decoded))
	require.Equal(t, true, decoded["approve"])
	require.NotContains(t, decoded, "reason")
	message := decoded["message"].(map[string]any)
	require.Equal(t, true, message["approve"])
	require.NotContains(t, message, "reason")
}

func TestSanitizeClaudeToolArguments_KeepsShutdownRejectionReason(t *testing.T) {
	raw := `{"to":"team-lead","type":"shutdown_response","approve":false,"reason":"still working","message":{"type":"shutdown_response","request_id":"shutdown-1","approve":false,"reason":"still working"}}`

	sanitized := SanitizeClaudeToolArguments("SendMessage", raw)
	require.Equal(t, raw, sanitized)
}

func TestSanitizeClaudeToolArguments_LeavesOtherToolsAndInvalidJSONUntouched(t *testing.T) {
	raw := `{"message":{"type":"shutdown_response","approve":true,"reason":"done"}}`
	require.Equal(t, raw, SanitizeClaudeToolArguments("Read", raw))
	require.Equal(t, "{", SanitizeClaudeToolArguments("SendMessage", "{"))
}

func TestResponseOpenAI2Claude_SanitizesShutdownApprovalToolCall(t *testing.T) {
	message := dto.Message{}
	message.SetToolCalls([]dto.ToolCallRequest{
		{
			ID:   "call_shutdown",
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      "SendMessage",
				Arguments: `{"type":"shutdown_response","approve":true,"reason":"done"}`,
			},
		},
	})
	response := &dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{
			{Message: message, FinishReason: "tool_calls"},
		},
	}

	claudeResp := ResponseOpenAI2Claude(response, &relaycommon.RelayInfo{})
	require.Len(t, claudeResp.Content, 1)
	input, ok := claudeResp.Content[0].Input.(map[string]any)
	require.True(t, ok)
	require.NotContains(t, input, "reason")
}
