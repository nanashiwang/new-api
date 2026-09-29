package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type kimiContractTransport struct {
	body         []byte
	calls        int
	toolResponse bool
}

func (transport *kimiContractTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.calls++
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	transport.body = body
	var request struct {
		Messages []map[string]json.RawMessage `json:"messages"`
		Stream   bool                         `json:"stream"`
	}
	if err = common.Unmarshal(body, &request); err != nil {
		return nil, err
	}
	status := http.StatusOK
	response := `{"id":"kimi-test","model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","content":"{\"ok\":true}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	contentType := "application/json"
	// A forwarding fixture, not an implementation of the full KVV validator.
	// KVV 66092cf permits content="" and rejects *non-empty* content with tools.
	// Explicit null is tested separately for preservation, not vendor behavior.
	for _, message := range request.Messages {
		if _, tools := message["tools"]; tools {
			var content *string
			if raw, present := message["content"]; present {
				if err := common.Unmarshal(raw, &content); err != nil {
					return nil, fmt.Errorf("fixture does not cover this content shape")
				}
			}
			if string(message["role"]) != `"system"` || (content != nil && *content != "") {
				status = http.StatusBadRequest
				response = `{"error":{"type":"invalid_request_error","message":"invalid dynamic tool message"}}`
			}
		}
	}
	if status == http.StatusOK && transport.toolResponse {
		response = `{"id":"kimi-test","model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"fixture-call","type":"function","function":{"name":"lookup","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
		if request.Stream {
			contentType = "text/event-stream"
			response = "data: " + `{"id":"kimi-test","model":"kimi-k3","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"fixture-call","type":"function","function":{"name":"lookup","arguments":"{"}}]}}]}` + "\n\n" +
				"data: " + `{"id":"kimi-test","model":"kimi-k3","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}` + "\n\ndata: [DONE]\n\n"
		}
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(response)), Request: req}, nil
}

func TestRelayKimiContractThroughOpenAIAndMoonshot(t *testing.T) {
	db := useControllerCapacityTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.ContentSafetyViolation{}))
	oldRedis, oldCount, oldRetry, oldLog := common.RedisEnabled, constant.CountToken, common.RetryTimes, constant.ErrorLogEnabled
	common.RedisEnabled, constant.CountToken, common.RetryTimes, constant.ErrorLogEnabled = false, false, 0, false
	t.Cleanup(func() {
		common.RedisEnabled, constant.CountToken, common.RetryTimes, constant.ErrorLogEnabled = oldRedis, oldCount, oldRetry, oldLog
	})
	oldRatio := ratio_setting.ModelRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"kimi-k3":0}`))
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(oldRatio)) })
	quota := operation_setting.GetQuotaSetting()
	oldFree := quota.EnableFreeModelPreConsume
	quota.EnableFreeModelPreConsume = false
	t.Cleanup(func() { quota.EnableFreeModelPreConsume = oldFree })
	if service.GetHttpClient() == nil {
		service.InitHttpClient()
	}
	client := service.GetHttpClient()
	oldTransport := client.Transport
	t.Cleanup(func() { client.Transport = oldTransport })
	toolMessage := `{"role":"system","content":"","tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}]}`
	for _, channelType := range []int{constant.ChannelTypeOpenAI, constant.ChannelTypeMoonshot} {
		for _, tc := range []struct {
			name, messages string
			status         int
		}{
			// Omission is an existing gateway compatibility check, not a KVV node.
			{"omitted_content_compatibility", `[{"role":"system","tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}]},{"role":"user","content":"test"}]`, 200},
			// Official KVV 66092cf: test_dynamic_tool_in_system_callable.
			{"official_empty_content", `[` + toolMessage + `,{"role":"user","content":"test"}]`, 200},
			{"official_subsequent_system", `[{"role":"system","content":"help"},{"role":"user","content":"hi"},{"role":"assistant","content":"hello"},` + toolMessage + `,{"role":"user","content":"test"}]`, 200},
			{"official_last_system", `[{"role":"user","content":"test"},` + toolMessage + `]`, 200},
			{"nonempty_content_rejected", `[{"role":"system","content":"bad","tools":[{"type":"function","function":{"name":"lookup"}}]},{"role":"user","content":"test"}]`, 400},
			{"user_tools_rejected", `[{"role":"user","content":"bad","tools":[{"type":"function","function":{"name":"lookup"}}]},{"role":"user","content":"test"}]`, 400},
			{"assistant_tools_rejected", `[{"role":"assistant","content":"bad","tools":[{"type":"function","function":{"name":"lookup"}}]},{"role":"user","content":"test"}]`, 400},
		} {
			for _, stream := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/%s/stream=%t", channelType, tc.name, stream), func(t *testing.T) {
					transport := &kimiContractTransport{toolResponse: true}
					client.Transport = transport
					channel := &model.Channel{Id: 994000 + channelType, Name: "kimi-contract", Type: channelType, Status: common.ChannelStatusEnabled, Key: "test-key", BaseURL: common.GetPointer("https://example.invalid")}
					// The fixture implements stream_options. Unknown compatible
					// hosts are conservative by default; declare this capability
					// for the test rather than changing production defaults.
					channel.SetOtherSettings(dto.ChannelOtherSettings{ChatStreamOptionsMode: dto.CapabilityModeEnabled})
					recorder := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(recorder)
					body := `{"model":"kimi-k3","messages":` + tc.messages + `,"tool_choice":"required","reasoning_effort":"low","temperature":1,"top_p":0.95}`
					if stream {
						body = strings.TrimSuffix(body, "}") + `,"stream":true,"stream_options":{"include_usage":true}}`
					}
					c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
					c.Request.Header.Set("Content-Type", "application/json")
					require.Nil(t, middleware.SetupContextForSelectedChannel(c, channel, "kimi-k3"))
					Relay(c, types.RelayFormatOpenAI)
					require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
					require.Equal(t, 1, transport.calls)
					require.JSONEq(t, body, string(transport.body))
					if tc.status == 200 {
						assertKimiForwardedToolResponse(t, recorder.Body.String(), stream)
					}
				})
			}
		}
	}
}

func assertKimiForwardedToolResponse(t *testing.T, body string, stream bool) {
	t.Helper()
	if !stream {
		var response dto.OpenAITextResponse
		require.NoError(t, common.UnmarshalJsonStr(body, &response))
		require.Len(t, response.Choices, 1)
		require.Equal(t, "tool_calls", response.Choices[0].FinishReason)
		calls := response.Choices[0].Message.ParseToolCalls()
		require.Len(t, calls, 1)
		require.Equal(t, "lookup", calls[0].Function.Name)
		require.JSONEq(t, `{}`, calls[0].Function.Arguments)
		return
	}
	var name, arguments, finish string
	identities, done := 0, 0
	for _, line := range strings.Split(body, "\n") {
		if line == "data: [DONE]" {
			done++
			continue
		}
		if !strings.HasPrefix(line, "data: {") {
			continue
		}
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(strings.TrimPrefix(line, "data: "), &chunk))
		for _, choice := range chunk.Choices {
			if choice.FinishReason != nil {
				finish = *choice.FinishReason
			}
			for _, call := range choice.Delta.ToolCalls {
				if call.ID != "" {
					identities++
				}
				if call.Function.Name != "" {
					name = call.Function.Name
				}
				arguments += call.Function.Arguments
			}
		}
	}
	require.Equal(t, "lookup", name)
	require.Equal(t, "tool_calls", finish)
	require.JSONEq(t, `{}`, arguments)
	require.Equal(t, 1, identities)
	require.Equal(t, 1, done)
}
