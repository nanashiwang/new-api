package controller

import (
	"encoding/json"
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
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayKimiFinalOutboundRequestIntegrity(t *testing.T) {
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
	global := model_setting.GetGlobalSettings()
	oldPass := global.PassThroughRequestEnabled
	t.Cleanup(func() { global.PassThroughRequestEnabled = oldPass })
	if service.GetHttpClient() == nil {
		service.InitHttpClient()
	}
	client := service.GetHttpClient()
	oldTransport := client.Transport
	t.Cleanup(func() { client.Transport = oldTransport })
	for _, tc := range []struct {
		name, path, body string
		format           types.RelayFormat
		channelType      int
		passThrough      bool
	}{
		{"chat_openai", "/v1/chat/completions", `{"model":"kimi-k3","messages":[{"role":"user","content":[{"type":"video_url","video_url":{"url":"data:video/mp4;base64,AAAA","fps":2}}]}],"max_tokens":4096,"max_completion_tokens":64,"tool_choice":"required","tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],"reasoning_effort":"low"}`, types.RelayFormatOpenAI, constant.ChannelTypeOpenAI, false},
		{"chat_moonshot", "/v1/chat/completions", `{"model":"kimi-k3","messages":[{"role":"user","content":[{"type":"video_url","video_url":"data:video/mp4;base64,AAAA"}]}],"max_completion_tokens":64,"tool_choice":"required","tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],"reasoning_effort":"low"}`, types.RelayFormatOpenAI, constant.ChannelTypeMoonshot, false},
		{"messages_openai", "/v1/messages", `{"model":"kimi-k3","messages":[{"role":"user","content":"look up"}],"max_tokens":64,"tool_choice":{"type":"any","disable_parallel_tool_use":true},"tools":[{"name":"lookup","input_schema":{"type":"object"}}]}`, types.RelayFormatClaude, constant.ChannelTypeOpenAI, false},
		{"passthrough_unchanged", "/v1/chat/completions", `{"model":"kimi-k3","messages":[{"role":"user","content":"test"}],"max_tokens":4096,"max_completion_tokens":64}`, types.RelayFormatOpenAI, constant.ChannelTypeOpenAI, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := &kimiContractTransport{}
			client.Transport = transport
			global.PassThroughRequestEnabled = tc.passThrough
			channel := &model.Channel{Id: 994101, Type: tc.channelType, Status: common.ChannelStatusEnabled, Key: "fixture", BaseURL: common.GetPointer("https://example.invalid")}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			require.Nil(t, middleware.SetupContextForSelectedChannel(c, channel, "kimi-k3"))
			Relay(c, tc.format)
			require.Equal(t, 200, recorder.Code, recorder.Body.String())
			require.Equal(t, 1, transport.calls)
			if tc.passThrough {
				require.JSONEq(t, tc.body, string(transport.body))
				return
			}
			var out dto.GeneralOpenAIRequest
			require.NoError(t, common.Unmarshal(transport.body, &out))
			require.Equal(t, uint(64), out.MaxTokens)
			require.Zero(t, out.MaxCompletionTokens)
			require.Equal(t, "required", out.ToolChoice)
			require.Equal(t, "lookup", out.Tools[0].Function.Name)
			if tc.format == types.RelayFormatClaude {
				require.Equal(t, common.GetPointer(false), out.ParallelTooCalls)
			} else {
				var before, after map[string]json.RawMessage
				require.NoError(t, common.UnmarshalJsonStr(tc.body, &before))
				require.NoError(t, common.Unmarshal(transport.body, &after))
				require.JSONEq(t, string(before["messages"]), string(after["messages"]))
				require.Equal(t, "low", out.ReasoningEffort)
			}
		})
	}
}
