package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestClaudeToolImagesFollowCompleteToolBatch(t *testing.T) {
	setResponsesBridgeTestOptions(t, map[string]string{responsesMediaTransportModeOption: "data"})
	for _, split := range []bool{false, true} {
		t.Run(map[bool]string{false: "one message", true: "adjacent messages"}[split], func(t *testing.T) {
			results := []dto.ClaudeMediaMessage{
				{Type: "tool_result", ToolUseId: "a", Content: []any{
					map[string]any{"type": "text", "text": "screenshot"},
					map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": "aGVsbG8="}},
				}},
				{Type: "tool_result", ToolUseId: "b", Content: []any{
					map[string]any{"type": "image", "source": map[string]any{"type": "url", "url": "https://example.com/image.png"}},
				}},
			}
			request := dto.ClaudeRequest{Model: "claude-test", Messages: []dto.ClaudeMessage{
				{Role: "assistant", Content: []dto.ClaudeMediaMessage{
					{Type: "tool_use", Id: "a", Name: "capture", Input: map[string]any{}},
					{Type: "tool_use", Id: "b", Name: "lookup", Input: map[string]any{}},
				}},
			}}
			if split {
				for _, result := range results {
					request.Messages = append(request.Messages, dto.ClaudeMessage{Role: "user", Content: []dto.ClaudeMediaMessage{result}})
				}
			} else {
				request.Messages = append(request.Messages, dto.ClaudeMessage{Role: "user", Content: results})
			}
			request.Messages = append(request.Messages, dto.ClaudeMessage{Role: "assistant", Content: "checked"})
			before, err := common.Marshal(request)
			require.NoError(t, err)
			out, err := ClaudeToOpenAIRequest(nil, request, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}})
			require.NoError(t, err)
			require.Len(t, out.Messages, 5)
			require.Equal(t, "tool", out.Messages[1].Role)
			require.Equal(t, "tool", out.Messages[2].Role)
			require.Equal(t, "capture", *out.Messages[1].Name)
			require.Equal(t, "lookup", *out.Messages[2].Name)
			require.Equal(t, "screenshot", out.Messages[1].StringContent())
			require.Equal(t, "[image]", out.Messages[2].StringContent())
			require.Equal(t, "user", out.Messages[3].Role)
			media := out.Messages[3].ParseContent()
			require.Len(t, media, 2)
			require.Equal(t, "data:image/png;base64,aGVsbG8=", media[0].GetImageMedia().Url)
			require.Equal(t, "https://example.com/image.png", media[1].GetImageMedia().Url)
			require.Equal(t, "checked", out.Messages[4].StringContent())
			after, err := common.Marshal(request)
			require.NoError(t, err)
			require.Equal(t, string(before), string(after))
		})
	}
}

func TestClaudeToolResultContentPreservesFallbacks(t *testing.T) {
	for _, body := range []string{
		`[{"type":"document","source":{"data":"unknown"},"future_field":42}]`,
		`[{"type":"image","source":{}},{"type":"text","text":"keep"}]`,
		`{"ok":true,"extra":[1,2]}`, `[1,2]`, `[]`, `null`,
	} {
		var content any
		require.NoError(t, common.UnmarshalJsonStr(body, &content))
		text, media, err := claudeToolResultToChat(nil, content)
		require.NoError(t, err)
		require.Empty(t, media)
		require.JSONEq(t, body, text)
	}
	text, media, err := claudeToolResultToChat(nil, []dto.ClaudeMediaMessage{
		{Type: "text", Text: common.GetPointer("one")},
		{Type: "input_text", Text: common.GetPointer("two")},
	})
	require.NoError(t, err)
	require.Equal(t, "one\ntwo", text)
	require.Empty(t, media)
}

func TestClaudeToolResultPropagatesImageBridgeErrors(t *testing.T) {
	setResponsesBridgeTestOptions(t, map[string]string{
		responsesMediaTransportModeOption: "bridge",
		responsesMediaBridgeEnabledOption: "true",
	})
	_, _, err := claudeToolResultToChat(nil, []dto.ClaudeMediaMessage{{
		Type: "image", Source: &dto.ClaudeMessageSource{Type: "base64", MediaType: "image/png", Data: "!invalid!"},
	}})
	require.Error(t, err)
}
