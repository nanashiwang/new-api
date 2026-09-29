package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestVideoPayloadPreservesWireShapeAndMetadata(t *testing.T) {
	for _, video := range []string{`"data:video/mp4;base64,AAAA"`, `{"url":"data:video/mp4;base64,AAAA","fps":2,"detail":"low"}`} {
		body := `{"model":"kimi-k3","messages":[{"role":"user","content":[{"type":"video_url","video_url":` + video + `}]}]}`
		var req GeneralOpenAIRequest
		require.NoError(t, common.UnmarshalJsonStr(body, &req))
		copy, err := common.DeepCopy(&req)
		require.NoError(t, err)
		parts := copy.Messages[0].ParseContent()
		require.Len(t, parts, 1)
		require.NotNil(t, parts[0].GetVideoUrl())
		require.Equal(t, "data:video/mp4;base64,AAAA", parts[0].GetVideoUrl().Url)
		wireVideo, err := common.Marshal(parts[0].VideoUrl)
		require.NoError(t, err)
		require.JSONEq(t, video, string(wireVideo))
		meta := copy.GetTokenCountMeta()
		require.Len(t, meta.Files, 1)
		require.Equal(t, types.FileTypeVideo, meta.Files[0].FileType)
		copy.Messages[0].SetMediaContent(parts)
		wire, err := common.Marshal(copy)
		require.NoError(t, err)
		require.JSONEq(t, body, string(wire))
	}
}

func TestOutputBudgetMetadataUsesEffectiveLimit(t *testing.T) {
	req := GeneralOpenAIRequest{MaxTokens: 4096, MaxCompletionTokens: 64}
	require.Equal(t, int(req.GetMaxTokens()), req.GetTokenCountMeta().MaxTokens)
}

func TestEffectiveOutputBudgetDoesNotRewriteRequest(t *testing.T) {
	for _, model := range []string{"kimi-k3", "k3", "gpt-4.1", "kimi-k2.5", "other-kimi-k3"} {
		for _, limits := range [][2]uint{{4096, 64}, {64, 4096}, {256, 0}, {0, 256}, {0, 0}} {
			req := GeneralOpenAIRequest{Model: model, MaxTokens: limits[0], MaxCompletionTokens: limits[1]}
			before := req
			effective := req.GetMaxTokens()
			require.EqualValues(t, effective, req.GetTokenCountMeta().MaxTokens)
			require.Equal(t, before, req, "inspecting a budget must not mutate the request")
		}
	}
}
