package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSystemPromptDoesNotCorruptDynamicToolMessages(t *testing.T) {
	for _, existing := range []bool{false, true} {
		request := &dto.GeneralOpenAIRequest{}
		require.NoError(t, common.UnmarshalJsonStr(`{"model":"kimi-k3","messages":[{"role":"system","tools":[{"type":"function","function":{"name":"lookup"}}]},{"role":"user","content":"hello"}]}`, request))
		if existing {
			request.Messages = append(request.Messages, dto.Message{Role: "system", Content: "original"})
		}
		info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
		info.ChannelSetting.SystemPrompt = "configured"
		info.ChannelSetting.SystemPromptOverride = true
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		applySystemPromptIfNeeded(c, info, request)
		var dynamic, prompts int
		for _, message := range request.Messages {
			if len(message.Tools) > 0 {
				dynamic++
				wire, err := common.Marshal(dto.GeneralOpenAIRequest{Messages: []dto.Message{message}})
				require.NoError(t, err)
				require.NotContains(t, string(wire), `"content"`)
			} else if message.Role == "system" {
				prompts++
				require.Contains(t, message.StringContent(), "configured")
				if existing {
					require.Contains(t, message.StringContent(), "original")
				}
			}
		}
		require.Equal(t, 1, dynamic)
		require.Equal(t, 1, prompts)
	}
}
