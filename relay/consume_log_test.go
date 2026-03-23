package relay

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	model.DB = db
	model.LOG_DB = db

	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	if err := db.AutoMigrate(
		&model.User{},
		&model.Token{},
		&model.Log{},
		&model.Channel{},
		&model.UserSubscription{},
	); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	os.Exit(m.Run())
}

func truncateRelayTables(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM users")
		model.DB.Exec("DELETE FROM tokens")
		model.DB.Exec("DELETE FROM logs")
		model.DB.Exec("DELETE FROM channels")
		model.DB.Exec("DELETE FROM user_subscriptions")
	})
}

func seedRelayUser(t *testing.T, id int, quota int) {
	t.Helper()
	user := &model.User{Id: id, Username: "relay_test_user", Quota: quota, Status: common.UserStatusEnabled}
	require.NoError(t, model.DB.Create(user).Error)
}

func seedRelayPackageToken(t *testing.T, id int, userID int, key string, remainQuota int, packageLimit int) {
	t.Helper()
	token := &model.Token{
		Id:                id,
		UserId:            userID,
		Key:               key,
		Name:              "relay_package_token",
		Status:            common.TokenStatusEnabled,
		RemainQuota:       remainQuota,
		PackageEnabled:    true,
		PackageLimitQuota: packageLimit,
		PackagePeriod:     model.TokenPackagePeriodHourly,
		PackagePeriodMode: model.TokenPackagePeriodModeRelative,
	}
	require.NoError(t, model.DB.Create(token).Error)
}

func seedRelayChannel(t *testing.T, id int) {
	t.Helper()
	channel := &model.Channel{Id: id, Name: "relay_test_channel", Key: "sk-test", Status: common.ChannelStatusEnabled}
	require.NoError(t, model.DB.Create(channel).Error)
}

func getLastRelayLog(t *testing.T) *model.Log {
	t.Helper()
	var logEntry model.Log
	err := model.LOG_DB.Order("id desc").First(&logEntry).Error
	require.NoError(t, err)
	return &logEntry
}

func TestPostConsumeQuota_LogsChargedQuotaWhenPackageSettleFails(t *testing.T) {
	truncateRelayTables(t)

	const userID = 5101
	const tokenID = 6101
	const channelID = 7101
	const tokenKey = "relay_package_token_key"

	seedRelayUser(t, userID, 0)
	seedRelayPackageToken(t, tokenID, userID, tokenKey, 100, 10)
	seedRelayChannel(t, channelID)

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	common.SetContextKey(ctx, constant.ContextKeyTokenPackageEnabled, true)
	common.SetContextKey(ctx, constant.ContextKeyTokenBillingMode, model.TokenBillingModeTokenOnly)

	relayInfo := &relaycommon.RelayInfo{
		UserId:            userID,
		TokenId:           tokenID,
		TokenKey:          tokenKey,
		OriginModelName:   "test-model",
		UsingGroup:        "default",
		StartTime:         time.Now(),
		FirstResponseTime: time.Now(),
		RequestURLPath:    "/v1/chat/completions",
		RequestConversionChain: []types.RelayFormat{
			types.RelayFormatOpenAI,
		},
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 1,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: channelID,
		},
	}

	apiErr := service.PreConsumeBilling(ctx, 8, relayInfo)
	require.Nil(t, apiErr)
	require.Equal(t, 8, relayInfo.FinalPreConsumedQuota)

	postConsumeQuota(ctx, relayInfo, &dto.Usage{
		PromptTokens:     8,
		CompletionTokens: 4,
		TotalTokens:      12,
	})

	logEntry := getLastRelayLog(t)
	require.Equal(t, model.LogTypeConsume, logEntry.Type)
	require.Equal(t, 8, logEntry.Quota)
	require.Contains(t, logEntry.Content, "实际消耗")
	require.Contains(t, logEntry.Content, "令牌仅成功扣费")

	other := make(map[string]interface{})
	require.NoError(t, common.UnmarshalJsonStr(logEntry.Other, &other))
	require.Equal(t, float64(12), other["actual_quota"])
	require.Equal(t, float64(8), other["charged_quota"])
	require.Equal(t, true, other["settle_failed"])
	require.NotEmpty(t, other["settle_error"])
}
