package relay

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const (
	testResponsesSessionBridgeEnabledOption  = "OpenAIResponsesSessionBridgeEnabled"
	testResponsesSessionBridgeUseRedisOption = "OpenAIResponsesSessionBridgeUseRedis"
)

type responsesRetryTestAdaptor struct {
	requests  []string
	responses []*http.Response
}

var _ channel.Adaptor = (*responsesRetryTestAdaptor)(nil)

func (a *responsesRetryTestAdaptor) Init(*relaycommon.RelayInfo) {}

func (a *responsesRetryTestAdaptor) GetRequestURL(*relaycommon.RelayInfo) (string, error) {
	return "", nil
}

func (a *responsesRetryTestAdaptor) SetupRequestHeader(*gin.Context, *http.Header, *relaycommon.RelayInfo) error {
	return nil
}

func (a *responsesRetryTestAdaptor) ConvertOpenAIRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeneralOpenAIRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertOpenAIRequest call")
}

func (a *responsesRetryTestAdaptor) ConvertRerankRequest(*gin.Context, int, dto.RerankRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertRerankRequest call")
}

func (a *responsesRetryTestAdaptor) ConvertEmbeddingRequest(*gin.Context, *relaycommon.RelayInfo, dto.EmbeddingRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertEmbeddingRequest call")
}

func (a *responsesRetryTestAdaptor) ConvertAudioRequest(*gin.Context, *relaycommon.RelayInfo, dto.AudioRequest) (io.Reader, error) {
	return nil, fmt.Errorf("unexpected ConvertAudioRequest call")
}

func (a *responsesRetryTestAdaptor) ConvertImageRequest(*gin.Context, *relaycommon.RelayInfo, dto.ImageRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertImageRequest call")
}

func (a *responsesRetryTestAdaptor) ConvertOpenAIResponsesRequest(_ *gin.Context, _ *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return request, nil
}

func (a *responsesRetryTestAdaptor) DoRequest(_ *gin.Context, _ *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	body, err := io.ReadAll(requestBody)
	if err != nil {
		return nil, err
	}
	a.requests = append(a.requests, string(body))
	if len(a.responses) == 0 {
		return nil, fmt.Errorf("unexpected extra DoRequest call")
	}
	resp := a.responses[0]
	a.responses = a.responses[1:]
	return resp, nil
}

func (a *responsesRetryTestAdaptor) DoResponse(*gin.Context, *http.Response, *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	return nil, types.NewOpenAIError(fmt.Errorf("unexpected DoResponse call"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
}

func (a *responsesRetryTestAdaptor) GetModelList() []string {
	return nil
}

func (a *responsesRetryTestAdaptor) GetChannelName() string {
	return "test"
}

func (a *responsesRetryTestAdaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertClaudeRequest call")
}

func (a *responsesRetryTestAdaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, fmt.Errorf("unexpected ConvertGeminiRequest call")
}

func setResponsesRetryTestOptions(t *testing.T, updates map[string]string) {
	t.Helper()

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalValues := make(map[string]string, len(updates))
	originalExists := make(map[string]bool, len(updates))
	for key, value := range updates {
		originalValues[key], originalExists[key] = common.OptionMap[key]
		common.OptionMap[key] = value
	}
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		for key := range updates {
			if originalExists[key] {
				common.OptionMap[key] = originalValues[key]
			} else {
				delete(common.OptionMap, key)
			}
		}
		common.OptionMapRWMutex.Unlock()
	})
}

func setResponsesRetryStreamTestTimeout(t *testing.T) {
	t.Helper()

	original := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = original
	})
}

func newResponsesRetryHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func newResponsesRetryStreamHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func TestChatCompletionsViaResponses_RetriesWithoutPreviousResponseID(t *testing.T) {
	setResponsesRetryTestOptions(t, map[string]string{
		testResponsesSessionBridgeEnabledOption:  "true",
		testResponsesSessionBridgeUseRedisOption: "false",
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	info := &relaycommon.RelayInfo{
		TokenId:         2,
		UserId:          1,
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: "gpt-5",
		RelayFormat:     types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:                      3,
			UpstreamModelName:              "gpt-5",
			SupportsResponsesStreamOptions: true,
		},
	}

	initialReq := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello"},
		},
	}
	err := service.StoreResponsesSessionBridge(
		info,
		initialReq,
		dto.Message{Role: "assistant", Content: "hi"},
		"resp_1",
	)
	require.NoError(t, err)

	request := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "what's next?"},
		},
	}

	successBody, err := common.Marshal(map[string]any{
		"id":         "resp_2",
		"model":      "gpt-5",
		"created_at": 1700000000,
		"status":     "completed",
		"output": []map[string]any{
			{
				"type": "message",
				"role": "assistant",
				"content": []map[string]any{
					{
						"type": "output_text",
						"text": "Fallback ok",
					},
				},
			},
		},
		"usage": map[string]any{
			"input_tokens":  10,
			"output_tokens": 2,
			"total_tokens":  12,
		},
	})
	require.NoError(t, err)

	adaptor := &responsesRetryTestAdaptor{
		responses: []*http.Response{
			newResponsesRetryHTTPResponse(http.StatusBadRequest, `{"error":{"message":"Unsupported parameter: previous_response_id"}}`),
			newResponsesRetryHTTPResponse(http.StatusOK, string(successBody)),
		},
	}

	usage, newAPIError := chatCompletionsViaResponses(ctx, info, adaptor, request)
	require.Nil(t, newAPIError)
	require.NotNil(t, usage)
	require.Len(t, adaptor.requests, 2)
	require.Contains(t, adaptor.requests[0], `"previous_response_id":"resp_1"`)
	require.Contains(t, adaptor.requests[0], `what's next?`)
	require.NotContains(t, adaptor.requests[0], `"hello"`)
	require.NotContains(t, adaptor.requests[0], `"hi"`)
	require.NotContains(t, adaptor.requests[1], `"previous_response_id"`)
	require.Contains(t, adaptor.requests[1], `"hello"`)
	require.Contains(t, adaptor.requests[1], `"hi"`)
	require.Contains(t, adaptor.requests[1], `what's next?`)
}

func TestChatCompletionsViaResponses_RetriesWithoutStreamOptionsThenPreviousResponseID(t *testing.T) {
	setResponsesRetryTestOptions(t, map[string]string{
		testResponsesSessionBridgeEnabledOption:  "true",
		testResponsesSessionBridgeUseRedisOption: "false",
	})
	setResponsesRetryStreamTestTimeout(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	info := &relaycommon.RelayInfo{
		TokenId:         12,
		UserId:          11,
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: "gpt-5",
		RelayFormat:     types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:                      13,
			UpstreamModelName:              "gpt-5",
			SupportsResponsesStreamOptions: true,
		},
	}

	initialReq := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello"},
		},
	}
	err := service.StoreResponsesSessionBridge(
		info,
		initialReq,
		dto.Message{Role: "assistant", Content: "hi"},
		"resp_1",
	)
	require.NoError(t, err)

	request := &dto.GeneralOpenAIRequest{
		Model:  "gpt-5",
		Stream: true,
		StreamOptions: &dto.StreamOptions{
			IncludeUsage: true,
		},
		Messages: []dto.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "what's next?"},
		},
	}

	successBody := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_2","model":"gpt-5","created_at":1700000000}}`,
		`data: {"type":"response.output_text.delta","delta":"Fallback ok"}`,
		`data: {"type":"response.completed","response":{"id":"resp_2","model":"gpt-5","created_at":1700000000,"usage":{"input_tokens":10,"output_tokens":2,"total_tokens":12},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Fallback ok"}]}]}}`,
		`data: [DONE]`,
	}, "\n")

	adaptor := &responsesRetryTestAdaptor{
		responses: []*http.Response{
			newResponsesRetryHTTPResponse(http.StatusBadRequest, `{"error":{"message":"Unsupported parameter: stream_options"}}`),
			newResponsesRetryHTTPResponse(http.StatusBadRequest, `{"error":{"message":"Unsupported parameter: previous_response_id"}}`),
			newResponsesRetryStreamHTTPResponse(successBody),
		},
	}

	usage, newAPIError := chatCompletionsViaResponses(ctx, info, adaptor, request)
	require.Nil(t, newAPIError)
	require.NotNil(t, usage)
	require.Len(t, adaptor.requests, 3)
	require.Contains(t, adaptor.requests[0], `"stream_options"`)
	require.Contains(t, adaptor.requests[0], `"previous_response_id":"resp_1"`)
	require.NotContains(t, adaptor.requests[1], `"stream_options"`)
	require.Contains(t, adaptor.requests[1], `"previous_response_id":"resp_1"`)
	require.NotContains(t, adaptor.requests[2], `"stream_options"`)
	require.NotContains(t, adaptor.requests[2], `"previous_response_id"`)
	require.Contains(t, adaptor.requests[2], `"hello"`)
	require.Contains(t, adaptor.requests[2], `"hi"`)
	require.Contains(t, adaptor.requests[2], `what's next?`)
}

func TestChatCompletionsViaResponses_RemembersPreviousResponseIDUnsupported(t *testing.T) {
	setResponsesRetryTestOptions(t, map[string]string{
		testResponsesSessionBridgeEnabledOption:  "true",
		testResponsesSessionBridgeUseRedisOption: "false",
	})

	gin.SetMode(gin.TestMode)

	info := &relaycommon.RelayInfo{
		TokenId:         202,
		UserId:          201,
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: "gpt-5",
		RelayFormat:     types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:                      203,
			ApiType:                        constant.APITypeOpenAI,
			ChannelBaseUrl:                 "https://example.com",
			UpstreamModelName:              "gpt-5",
			SupportsResponsesStreamOptions: true,
		},
	}

	initialReq := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello"},
		},
	}
	err := service.StoreResponsesSessionBridge(
		info,
		initialReq,
		dto.Message{Role: "assistant", Content: "hi"},
		"resp_1",
	)
	require.NoError(t, err)

	request := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "what's next?"},
		},
	}

	successBody, err := common.Marshal(map[string]any{
		"id":         "resp_2",
		"model":      "gpt-5",
		"created_at": 1700000000,
		"status":     "completed",
		"output": []map[string]any{
			{
				"type": "message",
				"role": "assistant",
				"content": []map[string]any{
					{
						"type": "output_text",
						"text": "Fallback ok",
					},
				},
			},
		},
		"usage": map[string]any{
			"input_tokens":  10,
			"output_tokens": 2,
			"total_tokens":  12,
		},
	})
	require.NoError(t, err)

	firstRecorder := httptest.NewRecorder()
	firstCtx, _ := gin.CreateTestContext(firstRecorder)
	firstCtx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	firstAdaptor := &responsesRetryTestAdaptor{
		responses: []*http.Response{
			newResponsesRetryHTTPResponse(http.StatusBadRequest, `{"error":{"message":"Unsupported parameter: previous_response_id"}}`),
			newResponsesRetryHTTPResponse(http.StatusOK, string(successBody)),
		},
	}

	usage, newAPIError := chatCompletionsViaResponses(firstCtx, info, firstAdaptor, request)
	require.Nil(t, newAPIError)
	require.NotNil(t, usage)
	require.Len(t, firstAdaptor.requests, 2)
	require.Contains(t, firstAdaptor.requests[0], `"previous_response_id":"resp_1"`)
	require.NotContains(t, firstAdaptor.requests[1], `"previous_response_id"`)

	secondRecorder := httptest.NewRecorder()
	secondCtx, _ := gin.CreateTestContext(secondRecorder)
	secondCtx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	secondAdaptor := &responsesRetryTestAdaptor{
		responses: []*http.Response{
			newResponsesRetryHTTPResponse(http.StatusOK, string(successBody)),
		},
	}

	usage, newAPIError = chatCompletionsViaResponses(secondCtx, info, secondAdaptor, request)
	require.Nil(t, newAPIError)
	require.NotNil(t, usage)
	require.Len(t, secondAdaptor.requests, 1)
	require.NotContains(t, secondAdaptor.requests[0], `"previous_response_id"`)
	require.Contains(t, secondAdaptor.requests[0], `"hello"`)
	require.Contains(t, secondAdaptor.requests[0], `"hi"`)
	require.Contains(t, secondAdaptor.requests[0], `what's next?`)
}

func TestRestoreResponsesInstructionsFromOriginalChatBackfillsTrimmedSessionRequest(t *testing.T) {
	t.Parallel()

	trimmedResponsesReq := &dto.OpenAIResponsesRequest{}
	originalChatReq := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "be helpful"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "what's next?"},
		},
	}

	err := restoreResponsesInstructionsFromOriginalChat(trimmedResponsesReq, originalChatReq)
	require.NoError(t, err)

	var instructions string
	err = common.Unmarshal(trimmedResponsesReq.Instructions, &instructions)
	require.NoError(t, err)
	require.Equal(t, "be helpful", instructions)
}

func TestRestoreResponsesInstructionsFromOriginalChatKeepsExistingInstructions(t *testing.T) {
	t.Parallel()

	trimmedResponsesReq := &dto.OpenAIResponsesRequest{
		Instructions: []byte(`"keep me"`),
	}
	originalChatReq := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "be helpful"},
			{Role: "user", Content: "hello"},
		},
	}

	err := restoreResponsesInstructionsFromOriginalChat(trimmedResponsesReq, originalChatReq)
	require.NoError(t, err)

	var instructions string
	err = common.Unmarshal(trimmedResponsesReq.Instructions, &instructions)
	require.NoError(t, err)
	require.Equal(t, "keep me", instructions)
}
