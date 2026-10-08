package controller

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Full local controller flow: outgoing HTTP is intercepted; real preconsume,
// wallet settlement, token quota and consumption logs use isolated SQLite.
func TestClaudeCacheBillingAcrossEndpoints(t *testing.T) {
	service.InitTokenEncoders()
	if service.GetHttpClient() == nil {
		service.InitHttpClient()
	}
	for _, tiered := range []bool{false, true} {
		for _, stream := range []bool{false, true} {
			for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI} {
				for _, tc := range []struct {
					name, usage                            string
					quota, total, five, hour, unclassified int
					invalid                                bool
				}{
					{"mixed", `"input_tokens":400,"output_tokens":80,"cache_read_input_tokens":120,"cache_creation_input_tokens":80,"cache_creation":{"ephemeral_5m_input_tokens":20,"ephemeral_1h_input_tokens":60}`, 957, 80, 20, 60, 0, false},
					{"only_hour", `"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":100,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":100}`, 200, 100, 0, 100, 0, false},
					{"partial", `"input_tokens":400,"output_tokens":0,"cache_creation_input_tokens":400,"cache_creation":{"ephemeral_5m_input_tokens":80,"ephemeral_1h_input_tokens":120}`, 990, 400, 80, 120, 200, false},
					{"total_only", `"input_tokens":400,"output_tokens":0,"cache_creation_input_tokens":400`, 900, 400, 0, 0, 400, false},
					{"split_only", `"input_tokens":400,"output_tokens":0,"cache_creation":{"ephemeral_5m_input_tokens":80,"ephemeral_1h_input_tokens":120}`, 740, 200, 80, 120, 0, false},
					{"zero", `"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":0`, 0, 0, 0, 0, 0, false},
					{"conflict", `"input_tokens":400,"output_tokens":0,"cache_creation_input_tokens":80,"cache_creation":{"ephemeral_5m_input_tokens":20,"ephemeral_1h_input_tokens":100}`, 0, 0, 0, 0, 0, true},
					{"late_conflict", `"input_tokens":400,"output_tokens":0,"cache_creation_input_tokens":80,"cache_creation":{"ephemeral_5m_input_tokens":20,"ephemeral_1h_input_tokens":100}`, 0, 0, 0, 0, 0, true},
					{"negative", `"input_tokens":400,"output_tokens":0,"cache_creation_input_tokens":-1`, 0, 0, 0, 0, 0, true},
				} {
					t.Run(fmt.Sprintf("%s/%s/stream=%t/tiered=%t", tc.name, format, stream, tiered), func(t *testing.T) {
						db := useControllerCapacityTestDB(t)
						sqlDB, err := db.DB()
						require.NoError(t, err)
						sqlDB.SetMaxOpenConns(1)
						t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
						require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Log{}, &model.ContentSafetyViolation{}, &model.PulseWalletReservation{}, &model.PulseFundingLedger{}))
						oldRedis, oldCount, oldBatch, oldConsume, oldSQLite := common.RedisEnabled, constant.CountToken, common.BatchUpdateEnabled, common.LogConsumeEnabled, common.UsingSQLite
						common.RedisEnabled, constant.CountToken, common.BatchUpdateEnabled, common.LogConsumeEnabled, common.UsingSQLite = false, false, false, true, true
						t.Cleanup(func() {
							common.RedisEnabled, constant.CountToken, common.BatchUpdateEnabled, common.LogConsumeEnabled, common.UsingSQLite = oldRedis, oldCount, oldBatch, oldConsume, oldSQLite
						})
						mr, cr, cc, outr := ratio_setting.ModelRatio2JSONString(), ratio_setting.CacheRatio2JSONString(), ratio_setting.CreateCacheRatio2JSONString(), ratio_setting.CompletionRatio2JSONString()
						bundle := billing_setting.TieredBundle{BillingMode: billing_setting.GetBillingModeCopy(), BillingExpr: billing_setting.GetBillingExprCopy()}
						require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"claude-cache-audit":1}`))
						require.NoError(t, ratio_setting.UpdateCacheRatioByJSONString(`{"claude-cache-audit":0.1}`))
						require.NoError(t, ratio_setting.UpdateCreateCacheRatioByJSONString(`{"claude-cache-audit":1.25}`))
						require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"claude-cache-audit":5}`))
						billing_setting.ReplaceBundle(billing_setting.TieredBundle{BillingMode: map[string]string{}, BillingExpr: map[string]string{}})
						if tiered {
							billing_setting.ReplaceBundle(billing_setting.TieredBundle{
								BillingMode: map[string]string{"claude-cache-audit": "tiered_expr"},
								BillingExpr: map[string]string{"claude-cache-audit": `tier("base", p*2+c*10+cr*0.2+cc*2.5+cc1h*4)`},
							})
						}
						t.Cleanup(func() {
							require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(mr))
							require.NoError(t, ratio_setting.UpdateCacheRatioByJSONString(cr))
							require.NoError(t, ratio_setting.UpdateCreateCacheRatioByJSONString(cc))
							require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(outr))
							billing_setting.ReplaceBundle(bundle)
						})
						const initial = 1000000
						user := model.User{Id: 995701, Username: "cache-audit", Quota: initial, Status: common.UserStatusEnabled}
						token := model.Token{Id: 995702, UserId: user.Id, Key: "cache-audit", RemainQuota: initial, Status: common.TokenStatusEnabled}
						channel := model.Channel{Id: 995703, Type: constant.ChannelTypeAnthropic, Status: common.ChannelStatusEnabled, Key: "fixture"}
						for _, row := range []any{&user, &token, &channel} {
							require.NoError(t, db.Create(row).Error)
						}
						body := `{"type":"message","id":"msg_fixture","model":"claude-cache-audit","content":[],"stop_reason":"end_turn","usage":{` + tc.usage + `}}`
						transport := &kimiUsageBillingTransport{body: body, contentType: "application/json"}
						if stream {
							transport.contentType = "text/event-stream"
							transport.body = `data: {"type":"message_start","message":{"id":"msg_fixture","model":"claude-cache-audit","usage":{` + tc.usage + `}}}` + "\n\n" +
								`data: {"type":"message_stop"}` + "\n\n"
							// A final output snapshot is present in these fixtures.
							transport.body = strings.Replace(transport.body, `{"type":"message_stop"}`, `{"type":"message_stop","usage":{"output_tokens":`+fmt.Sprint(gjson.Get("{"+tc.usage+"}", "output_tokens").Int())+`}}`, 1)
							if tc.name == "late_conflict" {
								transport.body = `data: {"type":"message_start","message":{"id":"msg_fixture","model":"claude-cache-audit","usage":{"input_tokens":400,"output_tokens":0}}}` + "\n\n" +
									`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial"}}` + "\n\n" +
									`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{` + tc.usage + `}}` + "\n\n"
							}
						}
						client := service.GetHttpClient()
						previousTransport := client.Transport
						client.Transport = transport
						t.Cleanup(func() { client.Transport = previousTransport })
						endpoint := "/v1/messages"
						if format == types.RelayFormatOpenAI {
							endpoint = "/v1/chat/completions"
						}
						w := httptest.NewRecorder()
						ctx, _ := gin.CreateTestContext(w)
						ctx.Request = httptest.NewRequest("POST", endpoint, strings.NewReader(fmt.Sprintf(`{"model":"claude-cache-audit","max_tokens":256,"messages":[{"role":"user","content":"OK"}],"stream":%t}`, stream)))
						ctx.Request.Header.Set("Content-Type", "application/json")
						requestID := "cache-audit-fixture"
						ctx.Set(common.RequestIdKey, requestID)
						common.SetContextKey(ctx, constant.ContextKeyUserId, user.Id)
						common.SetContextKey(ctx, constant.ContextKeyTokenId, token.Id)
						common.SetContextKey(ctx, constant.ContextKeyTokenKey, token.Key)
						common.SetContextKey(ctx, constant.ContextKeyUsingGroup, "default")
						common.SetContextKey(ctx, constant.ContextKeyUserSetting, dto.UserSetting{BillingPreference: "wallet_only"})
						require.Nil(t, middleware.SetupContextForSelectedChannel(ctx, &channel, "claude-cache-audit"))
						Relay(ctx, format)
						common.CleanupBodyStorage(ctx)
						require.Equal(t, 1, transport.calls, "invalid usage must not be retried")
						require.Eventually(t, func() bool {
							var u model.User
							var tok model.Token
							if db.First(&u, user.Id).Error != nil || db.First(&tok, token.Id).Error != nil {
								return false
							}
							return u.Quota == initial-tc.quota && tok.RemainQuota == initial-tc.quota && u.UsedQuota == tc.quota
						}, 2*time.Second, 10*time.Millisecond)
						var logs []model.Log
						require.NoError(t, db.Where("request_id = ? AND type = ?", requestID, model.LogTypeConsume).Find(&logs).Error)
						if tc.invalid {
							require.Empty(t, logs)
							return
						}
						require.Equal(t, 200, w.Code, w.Body.String())
						require.Len(t, logs, 1)
						require.Equal(t, tc.quota, logs[0].Quota)
						for key, want := range map[string]int{"cache_creation_tokens": tc.total, "cache_creation_tokens_5m": tc.five, "cache_creation_tokens_1h": tc.hour, "cache_creation_tokens_unclassified": tc.unclassified} {
							require.EqualValues(t, want, gjson.Get(logs[0].Other, key).Int(), key)
						}
						if tiered {
							require.EqualValues(t, tc.five+tc.unclassified, gjson.Get(logs[0].Other, "tiered_params.cc").Int())
							require.EqualValues(t, tc.hour, gjson.Get(logs[0].Other, "tiered_params.cc1h").Int())
						}
					})
				}
			}
		}
	}
}
