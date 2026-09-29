package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestClaudeConversionRejectsUnsupportedMediaAndDynamicTools(t *testing.T) {
	for _, message := range []string{
		`{"role":"user","content":[{"type":"video_url","video_url":"data:video/mp4;base64,AAAA"}]}`,
		`{"role":"system","content":[{"type":"video_url","video_url":{"url":"https://example.invalid/video.mp4"}}]}`,
		`{"role":"system","tools":[]}`,
	} {
		var req dto.GeneralOpenAIRequest
		require.NoError(t, common.UnmarshalJsonStr(`{"model":"kimi-k3","messages":[`+message+`]}`, &req))
		_, err := RequestOpenAI2ClaudeMessage(nil, req)
		var apiErr *types.NewAPIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, 400, apiErr.StatusCode)
		require.True(t, types.IsSkipRetryError(apiErr))
	}
}
