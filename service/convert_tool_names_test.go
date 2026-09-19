package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestClaudeToolNameIndexPreservesFirstAndExplicitNames(t *testing.T) {
	req := dto.ClaudeRequest{Model: "claude-test", Messages: []dto.ClaudeMessage{
		{Role: "user", Content: []dto.ClaudeMediaMessage{{Type: "tool_result", ToolUseId: "call_1", Content: "before"}}},
		{Role: "assistant", Content: []dto.ClaudeMediaMessage{{Type: "tool_use", Id: "call_1", Name: "first", Input: map[string]any{}}}},
		{Role: "user", Content: []dto.ClaudeMediaMessage{
			{Type: "tool_result", ToolUseId: "call_1", Content: "after"},
			{Type: "tool_result", ToolUseId: "missing", Content: "unknown"},
			{Type: "tool_result", ToolUseId: "call_1", Name: "explicit", Content: "named"},
		}},
		{Role: "assistant", Content: []dto.ClaudeMediaMessage{{Type: "tool_use", Id: "call_1", Name: "later", Input: map[string]any{}}}},
	}}
	converted, err := ClaudeToOpenAIRequest(nil, req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}})
	require.NoError(t, err)
	require.Len(t, converted.Messages, 6)
	for _, test := range []struct {
		index int
		name  string
	}{{0, "first"}, {2, "first"}, {3, ""}, {4, "explicit"}} {
		msg := converted.Messages[test.index]
		require.Equal(t, "tool", msg.Role)
		require.NotNil(t, msg.Name)
		require.Equal(t, test.name, *msg.Name)
	}
}
