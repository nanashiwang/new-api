package controller

import (
	"fmt"
	"net/http"
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
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayChatAndMessagesInterruptedOutputRefunds(t *testing.T) {
	service.InitTokenEncoders()
	if service.GetHttpClient() == nil {
		service.InitHttpClient()
	}
	for _, format := range []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude} {
		for _, funding := range []string{service.BillingSourceWallet, service.BillingSourceSubscription, service.BillingSourceToken} {
			t.Run(string(format)+"/"+funding, func(t *testing.T) {
				db := useControllerCapacityTestDB(t)
				sqlDB, err := db.DB()
				require.NoError(t, err)
				sqlDB.SetMaxOpenConns(1)
				t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
				require.NoError(t, db.AutoMigrate(&model.Log{}, &model.User{}, &model.Token{}, &model.ContentSafetyViolation{}, &model.PulseWalletReservation{}, &model.PulseFundingLedger{}, &model.SubscriptionPlan{}, &model.UserSubscription{}, &model.SubscriptionPreConsumeRecord{}))
				oldLog, oldRedis, oldCount, oldRetry, oldBatch, oldConsume, oldSQLite := constant.ErrorLogEnabled, common.RedisEnabled, constant.CountToken, common.RetryTimes, common.BatchUpdateEnabled, common.LogConsumeEnabled, common.UsingSQLite
				constant.ErrorLogEnabled, common.RedisEnabled, constant.CountToken, common.RetryTimes, common.BatchUpdateEnabled, common.LogConsumeEnabled, common.UsingSQLite = true, false, false, 3, false, true, true
				t.Cleanup(func() {
					constant.ErrorLogEnabled, common.RedisEnabled, constant.CountToken, common.RetryTimes, common.BatchUpdateEnabled, common.LogConsumeEnabled, common.UsingSQLite = oldLog, oldRedis, oldCount, oldRetry, oldBatch, oldConsume, oldSQLite
				})
				oldRatio := ratio_setting.ModelRatio2JSONString()
				require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"kimi-k3":1}`))
				t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(oldRatio)) })
				const initialQuota = 1000000
				user := &model.User{Id: 994201, Username: "stream-refund-fixture", Quota: initialQuota, Status: common.UserStatusEnabled}
				token := &model.Token{Id: 994202, UserId: user.Id, Key: "fixture-stream-refund", Status: common.TokenStatusEnabled, RemainQuota: initialQuota}
				plan := &model.SubscriptionPlan{Id: 994203, Title: "fixture", QuotaResetPeriod: "never"}
				sub := &model.UserSubscription{Id: 994204, UserId: user.Id, PlanId: plan.Id, AmountTotal: initialQuota, Status: "active", StartTime: time.Now().Unix(), EndTime: time.Now().Add(time.Hour).Unix()}
				for _, row := range []any{user, token, plan, sub} {
					require.NoError(t, db.Create(row).Error)
				}
				transport := &responsesBillingTransport{
					body: "data: " + `{"choices":[{"delta":{"content":"partial"}}],"usage":{"prompt_tokens":20,"completion_tokens":1,"total_tokens":21}}` + "\n\n",
					onRequest: func() error {
						var reserved model.Token
						if err := db.First(&reserved, token.Id).Error; err != nil {
							return err
						}
						if reserved.RemainQuota >= initialQuota {
							return fmt.Errorf("fixture did not exercise precharge")
						}
						return nil
					},
				}
				client := service.GetHttpClient()
				oldTransport := client.Transport
				client.Transport = transport
				t.Cleanup(func() { client.Transport = oldTransport })
				channel := &model.Channel{Id: 994205, Name: "fixture", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Key: "fixture"}
				require.NoError(t, db.Create(channel).Error)
				r := gin.New()
				r.Use(middleware.RequestFailureLog(), middleware.RouteTag("relay"))
				path := "/v1/chat/completions"
				if format == types.RelayFormatClaude {
					path = "/v1/messages"
				}
				requestID := "stream-refund-" + string(format) + "-" + funding
				r.POST(path, func(c *gin.Context) {
					common.SetContextKey(c, constant.ContextKeyUserId, user.Id)
					common.SetContextKey(c, constant.ContextKeyTokenId, token.Id)
					common.SetContextKey(c, constant.ContextKeyTokenKey, token.Key)
					common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
					c.Set(common.RequestIdKey, requestID)
					pref := "wallet_only"
					if funding == service.BillingSourceSubscription {
						pref = "subscription_only"
					}
					if funding == service.BillingSourceToken {
						common.SetContextKey(c, constant.ContextKeyTokenBillingMode, model.TokenBillingModeTokenOnly)
					}
					common.SetContextKey(c, constant.ContextKeyUserSetting, dto.UserSetting{BillingPreference: pref})
					require.Nil(t, middleware.SetupContextForSelectedChannel(c, channel, "kimi-k3"))
					Relay(c, format)
					require.False(t, service.RequestSucceeded(c))
				})
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"kimi-k3","messages":[{"role":"user","content":"hi"}],"max_tokens":64,"stream":true}`))
				req.Header.Set("Content-Type", "application/json")
				r.ServeHTTP(w, req)
				require.Equal(t, 1, transport.calls, "never replay partial output")
				require.Contains(t, w.Body.String(), "partial")
				require.Contains(t, w.Body.String(), `"error"`)
				require.NotContains(t, w.Body.String(), "[DONE]")
				require.NotContains(t, w.Body.String(), "message_stop")
				require.Eventually(t, func() bool {
					var actualUser model.User
					var actualToken model.Token
					var actualSub model.UserSubscription
					if db.First(&actualUser, user.Id).Error != nil || db.First(&actualToken, token.Id).Error != nil || db.First(&actualSub, sub.Id).Error != nil {
						return false
					}
					return actualUser.Quota == initialQuota && actualUser.UsedQuota == 0 && actualToken.RemainQuota == initialQuota && actualToken.UsedQuota == 0 && actualSub.AmountUsed == 0
				}, 3*time.Second, 10*time.Millisecond)
				var logs []model.Log
				require.NoError(t, db.Where("request_id = ?", requestID).Find(&logs).Error)
				require.Len(t, logs, 1)
				require.Equal(t, model.LogTypeError, logs[0].Type)
				require.Zero(t, logs[0].Quota)
			})
		}
	}
}
