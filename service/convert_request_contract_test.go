package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestClaudeToChatPreservesToolChoice(t *testing.T) {
	for _, tc := range []struct {
		choice, want string
		parallel     *bool
	}{
		{`{"type":"auto"}`, `"auto"`, nil},
		{`{"type":"any"}`, `"required"`, nil},
		{`{"type":"none"}`, `"none"`, nil},
		{`{"type":"tool","name":"lookup","disable_parallel_tool_use":true}`, `{"type":"function","function":{"name":"lookup"}}`, common.GetPointer(false)},
		{`{"type":"auto","disable_parallel_tool_use":false}`, `"auto"`, common.GetPointer(true)},
	} {
		t.Run(tc.choice, func(t *testing.T) {
			var req dto.ClaudeRequest
			require.NoError(t, common.UnmarshalJsonStr(`{"model":"kimi-k3","max_tokens":256,"tool_choice":`+tc.choice+`}`, &req))
			out, err := ClaudeToOpenAIRequest(nil, req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}})
			require.NoError(t, err)
			wire, err := common.Marshal(out.ToolChoice)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(wire))
			require.Equal(t, tc.parallel, out.ParallelTooCalls)
			require.Equal(t, uint(256), out.MaxTokens)
		})
	}
}

func TestClaudeToChatRejectsUnrepresentableToolChoice(t *testing.T) {
	for _, choice := range []string{`{"type":"tool"}`, `{"type":"future"}`, `{"type":"auto","disable_parallel_tool_use":"yes"}`, `"required"`} {
		var req dto.ClaudeRequest
		require.NoError(t, common.UnmarshalJsonStr(`{"model":"kimi-k3","tool_choice":`+choice+`}`, &req))
		_, err := ClaudeToOpenAIRequest(nil, req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}})
		require.Error(t, err, choice)
	}
}

func TestClaudeToChatRetainsAssistantThinkingAlongsideToolCall(t *testing.T) {
	var req dto.ClaudeRequest
	require.NoError(t, common.UnmarshalJsonStr(`{"model":"kimi-k3","messages":[
		{"role":"assistant","content":[{"type":"thinking","thinking":"plan "},{"type":"thinking","thinking":"next"},{"type":"text","text":"Checking"},{"type":"tool_use","id":"call_1","name":"lookup","input":{}}]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"found"}]}
	]}`, &req))
	out, err := ClaudeToOpenAIRequest(nil, req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}})
	require.NoError(t, err)
	require.Len(t, out.Messages, 2)
	require.Equal(t, "plan next", out.Messages[0].ReasoningContent)
	require.Equal(t, "Checking", out.Messages[0].ParseContent()[0].Text)
	require.NotEmpty(t, out.Messages[0].ToolCalls)
	require.Equal(t, "call_1", out.Messages[1].ToolCallId)
	require.Equal(t, "found", out.Messages[1].StringContent())
}
