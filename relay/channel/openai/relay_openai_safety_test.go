package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionsStreamEventErrorClassifiesExplicitSafetySignals(t *testing.T) {
	errorEvent := `{"error":{"message":"rejected","type":"invalid_request","code":"cyber_policy"}}`
	apiErr := chatCompletionsStreamEventError(errorEvent)
	require.NotNil(t, apiErr)
	require.True(t, service.IsContentSafetyPolicyError(apiErr))
	require.Contains(t, decorateChatCompletionsPolicyFailureData(errorEvent, apiErr), "本站警告")

	filterEvent := `{"choices":[{"delta":{},"finish_reason":"content_filter"}]}`
	apiErr = chatCompletionsStreamEventError(filterEvent)
	require.NotNil(t, apiErr)
	require.Equal(t, types.ErrorCode("content_filter"), apiErr.GetErrorCode())

	require.Nil(t, chatCompletionsStreamEventError(`{"choices":[{"delta":{"content":"ok"}}]}`))
}

func TestOpenaiHandlerReturnsContentFilterAsErrorBeforeWritingSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"choices":[{"finish_reason":"content_filter","message":{"role":"assistant","content":""}}],"usage":{"prompt_tokens":1,"completion_tokens":0,"total_tokens":1}}`)),
	}
	_, apiErr := OpenaiHandler(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI}}, resp)
	require.NotNil(t, apiErr)
	require.True(t, service.IsContentSafetyPolicyError(apiErr))
	require.Empty(t, recorder.Body.String())
}
