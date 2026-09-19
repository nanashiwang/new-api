package openai

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiConversionRequestsSupportedStreamUsage(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, supported := range []bool{false, true} {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := &relaycommon.RelayInfo{IsStream: stream, ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType: constant.ChannelTypeOpenAI, UpstreamModelName: "gpt-4o", SupportsChatStreamOptions: supported,
			}}
			converted, err := (&Adaptor{}).ConvertGeminiRequest(c, info, &dto.GeminiChatRequest{
				Contents: []dto.GeminiChatContent{{Role: "user", Parts: []dto.GeminiPart{{Text: "hello"}}}},
			})
			require.NoError(t, err)
			request := converted.(*dto.GeneralOpenAIRequest)
			if stream && supported {
				require.NotNil(t, request.StreamOptions)
				require.True(t, request.StreamOptions.IncludeUsage)
			} else {
				require.Nil(t, request.StreamOptions)
			}
		}
	}
}
