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
	body  []byte
	calls int
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
	}
	if err = common.Unmarshal(body, &request); err != nil {
		return nil, err
	}
	status := http.StatusOK
	response := `{"id":"kimi-test","model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","content":"{\"ok\":true}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	for _, message := range request.Messages {
		if _, tools := message["tools"]; tools {
			_, content := message["content"]
			if string(message["role"]) != `"system"` || content {
				status = http.StatusBadRequest
				response = `{"error":{"type":"invalid_request_error","message":"invalid dynamic tool message"}}`
			}
		}
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(response)), Request: req}, nil
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
	for _, channelType := range []int{constant.ChannelTypeOpenAI, constant.ChannelTypeMoonshot} {
		for _, tc := range []struct {
			name, message string
			status        int
		}{
			{"valid", `{"role":"system","tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}]}`, 200},
			{"content_null", `{"role":"system","content":null,"tools":[]}`, 400},
			{"content_text", `{"role":"system","content":"bad","tools":[]}`, 400},
			{"user_tools", `{"role":"user","content":"bad","tools":[]}`, 400},
		} {
			t.Run(fmt.Sprintf("%d/%s", channelType, tc.name), func(t *testing.T) {
				transport := &kimiContractTransport{}
				client.Transport = transport
				channel := &model.Channel{Id: 994000 + channelType, Name: "kimi-contract", Type: channelType, Status: common.ChannelStatusEnabled, Key: "test-key", BaseURL: common.GetPointer("https://example.invalid")}
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				body := `{"model":"kimi-k3","messages":[` + tc.message + `,{"role":"user","content":"test"}],"reasoning_effort":"low","temperature":1,"top_p":0.95,"prompt_cache_options":{"mode":"implicit","ttl":"1h"},"response_format":{"type":"json_schema","json_schema":{"name":"answer","strict":true,"schema":{"type":"object"}}}}`
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				require.Nil(t, middleware.SetupContextForSelectedChannel(c, channel, "kimi-k3"))
				Relay(c, types.RelayFormatOpenAI)
				require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
				require.Equal(t, 1, transport.calls)
				require.JSONEq(t, body, string(transport.body))
				if tc.status == 200 {
					require.Contains(t, recorder.Body.String(), `"content":"{\"ok\":true}"`)
				}
			})
		}
	}
}
