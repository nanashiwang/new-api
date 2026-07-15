package xai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeRequestUsesOpenAIChatPlan(t *testing.T) {
	c, _ := newXAITestContext()
	info := newXAITestRelayInfo(types.RelayFormatClaude, true)
	temperature := 0.2

	converted, err := (&Adaptor{}).ConvertClaudeRequest(c, info, &dto.ClaudeRequest{
		Model:       "grok-4-1-fast-reasoning",
		MaxTokens:   128,
		Stream:      true,
		Temperature: &temperature,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	})

	require.NoError(t, err)
	request, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "grok-4-1-fast-reasoning", request.Model)
	require.Len(t, request.Messages, 1)
	require.Equal(t, "hello", request.Messages[0].StringContent())
	require.NotNil(t, request.StreamOptions)
	require.True(t, request.StreamOptions.IncludeUsage)
	require.Equal(t, []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI}, info.RequestConversionChain)
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAI), info.GetFinalRequestRelayFormat())
}

func TestConvertClaudeRequestRejectsNilRequest(t *testing.T) {
	c, _ := newXAITestContext()
	_, err := (&Adaptor{}).ConvertClaudeRequest(c, newXAITestRelayInfo(types.RelayFormatClaude, false), nil)

	require.ErrorContains(t, err, "request is nil")
}

func TestXAIChatCompletionsURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{name: "host", baseURL: "https://api.x.ai", expected: "https://api.x.ai/v1/chat/completions"},
		{name: "versioned", baseURL: "https://proxy.example/openai/v1/", expected: "https://proxy.example/openai/v1/chat/completions"},
		{name: "complete", baseURL: "https://proxy.example/v1/chat/completions", expected: "https://proxy.example/v1/chat/completions"},
		{name: "query", baseURL: "https://proxy.example/v1?region=us", expected: "https://proxy.example/v1/chat/completions?region=us"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := xAIChatCompletionsURL(test.baseURL)
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}

	_, err := xAIChatCompletionsURL("not-a-url")
	require.ErrorContains(t, err, "invalid xAI channel base URL")
	_, err = xAIChatCompletionsURL("ftp://api.x.ai/v1")
	require.ErrorContains(t, err, "invalid xAI channel base URL")
}

func TestGetRequestURLUsesChatCompletionsForClaude(t *testing.T) {
	info := newXAITestRelayInfo(types.RelayFormatClaude, false)
	info.ChannelBaseUrl = "https://api.x.ai/v1"
	info.RequestURLPath = "/v1/messages?beta=true"

	requestURL, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.x.ai/v1/chat/completions", requestURL)
}

func TestGetRequestURLKeepsResponsesEndpoint(t *testing.T) {
	info := newXAITestRelayInfo(types.RelayFormatOpenAIResponses, false)
	info.RelayMode = relayconstant.RelayModeResponses
	info.RequestURLPath = "/v1/responses"

	requestURL, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.x.ai/v1/responses", requestURL)
}

func TestXAIHandlerConvertsClaudeResponse(t *testing.T) {
	c, recorder := newXAITestContext()
	info := newXAITestRelayInfo(types.RelayFormatClaude, false)
	info.SetEstimatePromptTokens(3)
	responseBody := `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":0,"total_tokens":5}}`

	usage, newAPIError := xAIHandler(c, info, newXAIHTTPResponse(responseBody))

	require.Nil(t, newAPIError)
	require.Equal(t, 3, usage.PromptTokens)
	require.Equal(t, 2, usage.CompletionTokens)
	var response dto.ClaudeResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "message", response.Type)
	require.Len(t, response.Content, 1)
	require.Equal(t, "pong", response.Content[0].GetText())
	require.Equal(t, 3, response.Usage.InputTokens)
	require.Equal(t, 2, response.Usage.OutputTokens)
}

func TestXAIHandlerEstimatesMissingUsage(t *testing.T) {
	c, recorder := newXAITestContext()
	info := newXAITestRelayInfo(types.RelayFormatClaude, false)
	info.SetEstimatePromptTokens(4)
	responseBody := `{"id":"chatcmpl-2","object":"chat.completion","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"message":{"role":"assistant","content":"estimated output"},"finish_reason":"stop"}]}`

	usage, newAPIError := xAIHandler(c, info, newXAIHTTPResponse(responseBody))

	require.Nil(t, newAPIError)
	require.Equal(t, 4, usage.PromptTokens)
	require.Greater(t, usage.CompletionTokens, 0)
	require.Equal(t, usage.PromptTokens+usage.CompletionTokens, usage.TotalTokens)
	var response dto.ClaudeResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, usage.CompletionTokens, response.Usage.OutputTokens)
}

func TestXAIHandlerConvertsToolCallResponse(t *testing.T) {
	c, recorder := newXAITestContext()
	info := newXAITestRelayInfo(types.RelayFormatClaude, false)
	responseBody := `{"id":"chatcmpl-tool","object":"chat.completion","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"weather","arguments":"{\"city\":\"Paris\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":5,"total_tokens":9}}`

	_, newAPIError := xAIHandler(c, info, newXAIHTTPResponse(responseBody))

	require.Nil(t, newAPIError)
	var response dto.ClaudeResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Content, 1)
	require.Equal(t, "tool_use", response.Content[0].Type)
	require.Equal(t, "weather", response.Content[0].Name)
	require.Equal(t, "call_1", response.Content[0].Id)
}

func TestXAIHandlerReturnsEmbeddedUpstreamError(t *testing.T) {
	c, _ := newXAITestContext()
	responseBody := `{"error":{"message":"model at capacity","type":"server_error","code":"capacity"}}`

	_, newAPIError := xAIHandler(c, newXAITestRelayInfo(types.RelayFormatClaude, false), newXAIHTTPResponse(responseBody))

	require.NotNil(t, newAPIError)
	require.Contains(t, newAPIError.Error(), "model at capacity")
	require.Equal(t, http.StatusBadGateway, newAPIError.StatusCode)
}

func TestNormalizeXAIUsageClampsInvalidCounts(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     -3,
		CompletionTokens: -2,
		TotalTokens:      -5,
		CompletionTokenDetails: dto.OutputTokenDetails{
			ReasoningTokens: -1,
		},
	}

	normalizeXAIUsage(usage)

	require.Zero(t, usage.PromptTokens)
	require.Zero(t, usage.CompletionTokens)
	require.Zero(t, usage.TotalTokens)
	require.Zero(t, usage.CompletionTokenDetails.ReasoningTokens)
}

func TestXAIStreamHandlerConvertsClaudeSSE(t *testing.T) {
	c, recorder := newXAITestContext()
	info := newXAITestRelayInfo(types.RelayFormatClaude, true)
	info.SetEstimatePromptTokens(3)
	responseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"delta":{"content":"pong"},"finish_reason":null}],"usage":null}`,
		"",
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":0,"total_tokens":5}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	usage, newAPIError := xAIStreamHandler(c, info, newXAIHTTPResponse(responseBody))

	require.Nil(t, newAPIError)
	require.Equal(t, 3, usage.PromptTokens)
	require.Equal(t, 2, usage.CompletionTokens)
	body := recorder.Body.String()
	require.Contains(t, body, "event: message_start")
	require.Contains(t, body, "event: content_block_delta")
	require.Contains(t, body, "event: message_stop")
	require.Contains(t, body, `"text":"pong"`)
	require.Equal(t, 1, strings.Count(body, `"text":"pong"`))
	require.Equal(t, 1, strings.Count(body, "event: message_stop"))
	require.NotContains(t, body, "[DONE]")
}

func TestXAIStreamHandlerUsesTrailingUsageWithoutDuplicateStop(t *testing.T) {
	c, recorder := newXAITestContext()
	info := newXAITestRelayInfo(types.RelayFormatClaude, true)
	responseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-usage","object":"chat.completion.chunk","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"delta":{"content":"done"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-usage","object":"chat.completion.chunk","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		"",
		`data: {"id":"chatcmpl-usage","object":"chat.completion.chunk","created":1,"model":"grok-4-1-fast-reasoning","choices":[],"usage":{"prompt_tokens":3,"total_tokens":5}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	usage, newAPIError := xAIStreamHandler(c, info, newXAIHTTPResponse(responseBody))

	require.Nil(t, newAPIError)
	require.Equal(t, 3, usage.PromptTokens)
	require.Equal(t, 2, usage.CompletionTokens)
	body := recorder.Body.String()
	require.Equal(t, 1, strings.Count(body, `"text":"done"`))
	require.Equal(t, 1, strings.Count(body, "event: message_stop"))
	require.Contains(t, body, `"input_tokens":3`)
}

func TestXAIStreamHandlerGracefullyFinishesEOFWithoutUsage(t *testing.T) {
	c, recorder := newXAITestContext()
	info := newXAITestRelayInfo(types.RelayFormatClaude, true)
	info.SetEstimatePromptTokens(6)
	responseBody := "data: {\"id\":\"chatcmpl-eof\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"grok-4-1-fast-reasoning\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"partial\"},\"finish_reason\":null}]}\n\n"

	usage, newAPIError := xAIStreamHandler(c, info, newXAIHTTPResponse(responseBody))

	require.Nil(t, newAPIError)
	require.Equal(t, 6, usage.PromptTokens)
	require.Greater(t, usage.CompletionTokens, 0)
	body := recorder.Body.String()
	require.Contains(t, body, `"text":"partial"`)
	require.Equal(t, 1, strings.Count(body, `"text":"partial"`))
	require.Contains(t, body, "event: message_stop")
}

func TestXAIStreamHandlerConvertsToolCallFragments(t *testing.T) {
	c, recorder := newXAITestContext()
	info := newXAITestRelayInfo(types.RelayFormatClaude, true)
	responseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-tool","object":"chat.completion.chunk","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"weather","arguments":"{\"city\":"}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-tool","object":"chat.completion.chunk","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Paris\"}"}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-tool","object":"chat.completion.chunk","created":1,"model":"grok-4-1-fast-reasoning","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":5,"total_tokens":10}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	_, newAPIError := xAIStreamHandler(c, info, newXAIHTTPResponse(responseBody))

	require.Nil(t, newAPIError)
	body := recorder.Body.String()
	require.Contains(t, body, `"type":"tool_use"`)
	require.Contains(t, body, `"type":"input_json_delta"`)
	require.Contains(t, body, "event: message_stop")
}

func TestXAIStreamHandlerRejectsMalformedAndEmptyStreams(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: "data: {not-json}\n\n"},
		{name: "null", body: "data: null\n\n"},
		{name: "empty", body: "data: [DONE]\n\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, _ := newXAITestContext()
			_, newAPIError := xAIStreamHandler(c, newXAITestRelayInfo(types.RelayFormatClaude, true), newXAIHTTPResponse(test.body))
			require.NotNil(t, newAPIError)
		})
	}
}

func TestXAIStreamHandlerReturnsEmbeddedUpstreamError(t *testing.T) {
	c, _ := newXAITestContext()
	body := "data: {\"error\":{\"message\":\"stream capacity\",\"type\":\"server_error\"}}\n\n"

	_, newAPIError := xAIStreamHandler(c, newXAITestRelayInfo(types.RelayFormatClaude, true), newXAIHTTPResponse(body))

	require.NotNil(t, newAPIError)
	require.Contains(t, newAPIError.Error(), "stream capacity")
	require.Equal(t, http.StatusBadGateway, newAPIError.StatusCode)
}

func newXAITestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	return c, recorder
}

func newXAITestRelayInfo(format types.RelayFormat, stream bool) *relaycommon.RelayInfo {
	info := &relaycommon.RelayInfo{
		RelayFormat:            format,
		RelayMode:              relayconstant.RelayModeChatCompletions,
		IsStream:               stream,
		OriginModelName:        "grok-4-1-fast-reasoning",
		RequestURLPath:         "/v1/messages",
		RequestConversionChain: []types.RelayFormat{format},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:               constant.ChannelTypeXai,
			ChannelBaseUrl:            "https://api.x.ai",
			UpstreamModelName:         "grok-4-1-fast-reasoning",
			SupportsChatStreamOptions: true,
			NativeTextFormats: relaycommon.NewTextProtocolSet(
				types.RelayFormatOpenAI,
				types.RelayFormatOpenAIResponses,
			),
		},
	}
	if format == types.RelayFormatClaude {
		info.ClaudeConvertInfo = &relaycommon.ClaudeConvertInfo{LastMessagesType: relaycommon.LastMessageTypeNone}
	}
	return info
}

func newXAIHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
