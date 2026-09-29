package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

// This uses real loopback HTTP/SSE, but a fixture supplier and no live API key.
// It verifies local protocol delivery, not provider capability or production.
func TestKimiRetestThreeProtocolsOverLocalHTTP(t *testing.T) {
	db := useControllerCapacityTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.ContentSafetyViolation{}))
	oldRedis, oldCount, oldRetry, oldLog, oldConsume := common.RedisEnabled, constant.CountToken, common.RetryTimes, constant.ErrorLogEnabled, common.LogConsumeEnabled
	common.RedisEnabled, constant.CountToken, common.RetryTimes, constant.ErrorLogEnabled, common.LogConsumeEnabled = false, false, 0, false, false
	t.Cleanup(func() {
		common.RedisEnabled, constant.CountToken, common.RetryTimes, constant.ErrorLogEnabled, common.LogConsumeEnabled = oldRedis, oldCount, oldRetry, oldLog, oldConsume
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
	for _, tc := range []struct {
		path, upstreamPath, body, stream, terminal string
		format                                     types.RelayFormat
		channelType                                int
	}{
		{
			"/v1/chat/completions", "/v1/chat/completions",
			`{"model":"kimi-k3","messages":[{"role":"user","content":"OK"}],"max_tokens":16,"reasoning_effort":"none","temperature":0.6,"stream":true}`,
			"data: " + `{"id":"fixture-chat","model":"kimi-k3","choices":[{"index":0,"delta":{"content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":1,"total_tokens":11}}` + "\n\ndata: [DONE]\n\n",
			"[DONE]", types.RelayFormatOpenAI, constant.ChannelTypeMoonshot,
		},
		{
			"/v1/messages", "/anthropic/v1/messages",
			`{"model":"kimi-k3","messages":[{"role":"user","content":"OK"}],"max_tokens":16,"stream":true}`,
			"event: message_start\ndata: " + `{"type":"message_start","message":{"id":"fixture-message","type":"message","role":"assistant","content":[],"model":"kimi-k3","usage":{"input_tokens":10,"output_tokens":0}}}` + "\n\n" +
				"event: content_block_start\ndata: " + `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}` + "\n\n" +
				"event: content_block_delta\ndata: " + `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"OK"}}` + "\n\n" +
				"event: content_block_stop\ndata: " + `{"type":"content_block_stop","index":0}` + "\n\n" +
				"event: message_delta\ndata: " + `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":1}}` + "\n\n" +
				"event: message_stop\ndata: " + `{"type":"message_stop"}` + "\n\n",
			"message_stop", types.RelayFormatClaude, constant.ChannelTypeMoonshot,
		},
		{
			"/v1/responses", "/v1/responses",
			`{"model":"kimi-k3","input":"OK","max_output_tokens":16,"stream":true}`,
			"data: " + `{"type":"response.output_text.delta","delta":"OK"}` + "\n\n" +
				"data: " + `{"type":"response.completed","response":{"id":"fixture-response","status":"completed","model":"kimi-k3","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"OK"}]}],"usage":{"input_tokens":10,"output_tokens":1,"total_tokens":11}}}` + "\n\n",
			"response.completed", types.RelayFormatOpenAIResponses, constant.ChannelTypeOpenAI,
		},
	} {
		t.Run(tc.path, func(t *testing.T) {
			var calls atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != tc.upstreamPath {
					http.Error(w, "wrong fixture path", 400)
					return
				}
				_, _ = io.Copy(io.Discard, r.Body)
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, tc.stream)
				w.(http.Flusher).Flush()
			}))
			defer upstream.Close()
			channel := &model.Channel{Id: 994401, Type: tc.channelType, Status: common.ChannelStatusEnabled, Key: "fixture", BaseURL: common.GetPointer(upstream.URL)}
			r := gin.New()
			r.Use(middleware.DecompressRequestMiddleware(), middleware.BodyStorageCleanup())
			r.POST(tc.path, func(c *gin.Context) {
				if err := middleware.SetupContextForSelectedChannel(c, channel, "kimi-k3"); err != nil {
					c.Status(500)
					return
				}
				Relay(c, tc.format)
			})
			server := httptest.NewServer(r)
			defer server.Close()
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Post(server.URL+tc.path, "application/json", strings.NewReader(tc.body))
			require.NoError(t, err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode, string(body))
			require.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")
			require.Contains(t, string(body), "OK")
			require.Contains(t, string(body), tc.terminal)
			require.EqualValues(t, 1, calls.Load())
		})
	}
}
