package dto

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestDynamicToolMessagesSurviveCopyAndSerialization(t *testing.T) {
	for _, body := range []string{
		`{"role":"system","tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}]}`,
		`{"role":"system","content":null,"tools":[]}`,
		`{"role":"system","content":"","tools":[{"type":"function","function":{"name":"lookup"}}]}`,
		`{"role":"system","content":"invalid combination","tools":[]}`,
		`{"role":"user","content":"invalid role","tools":[]}`,
		`{"role":"assistant","content":null,"reasoning_content":"plan","tool_calls":[{"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{}"}}]}`,
		`{"role":"user","content":[{"type":"text","text":"hello"},{"type":"image_url","image_url":{"url":"https://example.com/image.png"}}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			var request GeneralOpenAIRequest
			require.NoError(t, common.UnmarshalJsonStr(`{"model":"kimi-k3","messages":[`+body+`]}`, &request))
			copy, err := common.DeepCopy(&request)
			require.NoError(t, err)
			encoded, err := common.Marshal(copy)
			require.NoError(t, err)
			require.JSONEq(t, `{"model":"kimi-k3","messages":[`+body+`]}`, string(encoded))
		})
	}
}

func TestKimiRequestPreservesStructuredOutputParametersAndCacheTTL(t *testing.T) {
	body := `{"model":"kimi-k3","messages":[{"role":"system","tools":[{"type":"function","function":{"name":"lookup"}}]}],"reasoning_effort":"low","temperature":0.5,"top_p":0.8,"frequency_penalty":0.5,"prompt_cache_options":{"mode":"implicit","ttl":"1h"},"response_format":{"type":"json_schema","json_schema":{"name":"answer","strict":true,"schema":{"type":"object"}}},"tools":[{"type":"function","function":{"name":"base","strict":true}}]}`
	var request GeneralOpenAIRequest
	require.NoError(t, common.UnmarshalJsonStr(body, &request))
	copy, err := common.DeepCopy(&request)
	require.NoError(t, err)
	encoded, err := common.Marshal(copy)
	require.NoError(t, err)
	require.JSONEq(t, body, string(encoded))
	require.Contains(t, request.GetTokenCountMeta().CombineText, "lookup")
	copy.Messages[0].Tools[0] = ' '
	require.Equal(t, byte('['), request.Messages[0].Tools[0], "retry copies must not share tool payloads")
	legacy, err := common.Marshal(Message{Role: "assistant"})
	require.NoError(t, err)
	require.JSONEq(t, `{"role":"assistant","content":null}`, string(legacy))
	constructed, err := common.Marshal(GeneralOpenAIRequest{Messages: []Message{{Role: "system", Tools: json.RawMessage(`[]`)}}})
	require.NoError(t, err)
	require.JSONEq(t, `{"messages":[{"role":"system","tools":[]}]}`, string(constructed))
}

func TestOpenAIResponseMessageDoesNotHideChoiceFields(t *testing.T) {
	body := `{"index":2,"message":{"role":"assistant","content":"hello"},"finish_reason":"content_filter"}`
	var choice OpenAITextResponseChoice
	require.NoError(t, common.UnmarshalJsonStr(body, &choice))
	require.Equal(t, 2, choice.Index)
	require.Equal(t, "content_filter", choice.FinishReason)
	require.Equal(t, "hello", choice.Message.StringContent())
	wire, err := common.Marshal(choice)
	require.NoError(t, err)
	require.JSONEq(t, body, string(wire))
}
