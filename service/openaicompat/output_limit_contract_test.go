package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestResponsesBridgeDoesNotIncreaseExplicitCompletionLimit(t *testing.T) {
	for _, limits := range [][2]uint{{4096, 64}, {64, 4096}, {64, 0}, {0, 64}, {0, 0}} {
		req := &dto.GeneralOpenAIRequest{Model: "test-model", MaxTokens: limits[0], MaxCompletionTokens: limits[1]}
		out, err := ChatCompletionsRequestToResponsesRequest(req)
		require.NoError(t, err)
		require.Equal(t, req.GetMaxTokens(), out.MaxOutputTokens)
	}
}

func TestResponsesBridgeRejectsDynamicToolsInsteadOfDroppingThem(t *testing.T) {
	var req dto.GeneralOpenAIRequest
	require.NoError(t, common.UnmarshalJsonStr(`{"model":"kimi-k3","messages":[{"role":"system","tools":[{"type":"function","function":{"name":"lookup"}}]},{"role":"user","content":"hi"}]}`, &req))
	_, err := ChatCompletionsRequestToResponsesRequest(&req)
	require.ErrorContains(t, err, "dynamic tools")
}

func TestResponsesBridgePreservesVideoForms(t *testing.T) {
	for _, video := range []string{`"data:video/mp4;base64,AAAA"`, `{"url":"data:video/mp4;base64,AAAA","fps":2}`} {
		var req dto.GeneralOpenAIRequest
		require.NoError(t, common.UnmarshalJsonStr(`{"model":"kimi-k3","messages":[{"role":"user","content":[{"type":"video_url","video_url":`+video+`}]}]}`, &req))
		out, err := ChatCompletionsRequestToResponsesRequest(&req)
		require.NoError(t, err)
		require.JSONEq(t, `[{"role":"user","content":[{"type":"input_video","video_url":`+video+`}]}]`, string(out.Input))
	}
}
