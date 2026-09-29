package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestChatToClaudeFirstParallelToolChunkPreservesEveryBlock(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	var events []*dto.ClaudeResponse
	for _, raw := range []string{
		`{"choices":[{"delta":{"reasoning_content":"plan","content":"checking","tool_calls":[{"index":7,"id":"a","function":{"name":"first","arguments":"{"}},{"index":1000000000,"id":"b","function":{"name":"second","arguments":"{}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":7,"id":"a","function":{"name":"first","arguments":" "}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":7,"function":{"arguments":"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":20,"completion_tokens":12}}`,
	} {
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(raw, &chunk))
		result, err := StreamResponseOpenAI2Claude(&chunk, info)
		require.NoError(t, err)
		events = append(events, result...)
	}
	starts, stops := map[int]string{}, map[int]int{}
	args := map[int]string{}
	var texts, thinking strings.Builder
	counts := map[string]int{}
	for _, event := range events {
		counts[event.Type]++
		switch event.Type {
		case "content_block_start":
			require.NotContains(t, starts, *event.Index)
			starts[*event.Index] = event.ContentBlock.Type
		case "content_block_stop":
			require.Contains(t, starts, *event.Index)
			stops[*event.Index]++
		case "content_block_delta":
			require.Contains(t, starts, *event.Index)
			switch event.Delta.Type {
			case "thinking_delta":
				thinking.WriteString(*event.Delta.Thinking)
			case "text_delta":
				texts.WriteString(*event.Delta.Text)
			case "input_json_delta":
				args[*event.Index] += *event.Delta.PartialJson
			}
		}
	}
	require.Equal(t, 1, counts["message_start"])
	require.Equal(t, 1, counts["message_stop"])
	require.Equal(t, 4, counts["content_block_start"])
	require.Equal(t, map[int]int{0: 1, 1: 1, 2: 1, 3: 1}, stops)
	require.Equal(t, map[int]string{2: "{ }", 3: "{}"}, args)
	require.Equal(t, "plan", thinking.String())
	require.Equal(t, "checking", texts.String())
	require.Equal(t, "tool_use", *events[len(events)-2].Delta.StopReason)
	require.Equal(t, 12, events[len(events)-2].Usage.OutputTokens)
}

func TestChatToClaudeToolIdentityMayArriveAfterArguments(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	for _, raw := range []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"a","function":{"arguments":" "}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"lookup","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`,
	} {
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(raw, &chunk))
		events, err := StreamResponseOpenAI2Claude(&chunk, info)
		require.NoError(t, err)
		if info.Done {
			require.Equal(t, "content_block_start", events[0].Type)
			require.Equal(t, " {}", *events[1].Delta.PartialJson)
		}
	}
	require.True(t, info.Done)
}

func TestChatToClaudeMalformedToolsFailClosed(t *testing.T) {
	for _, raw := range []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":-1,"id":"a","function":{"name":"lookup","arguments":"{}"}}]}}]}`,
		`{"choices":[{"delta":{}},{"delta":{}}]}`,
	} {
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(raw, &chunk))
		events, err := StreamResponseOpenAI2Claude(&chunk, &relaycommon.RelayInfo{})
		require.Error(t, err)
		require.Empty(t, events)
	}
}

func TestChatToClaudeToolIdentityCannotChange(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	for i, raw := range []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"a","function":{"name":"lookup","arguments":"{"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"b","function":{"name":"other","arguments":"}"}}]},"finish_reason":"tool_calls"}]}`,
	} {
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(raw, &chunk))
		events, err := StreamResponseOpenAI2Claude(&chunk, info)
		if i == 0 {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.Empty(t, events)
			require.False(t, info.Done)
		}
	}
}
