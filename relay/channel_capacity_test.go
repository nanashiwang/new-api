package relay

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func enableLocalChannelCapacityForTest(t *testing.T) {
	t.Helper()
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = originalRedisEnabled })
	t.Setenv("CHANNEL_CONCURRENCY_ENABLED", "")
	t.Setenv("CHANNEL_RPM_WINDOW_SECONDS", "")

	setting := operation_setting.GetChannelConcurrencySetting()
	original := *setting
	setting.Enabled = true
	setting.DefaultMaxConcurrency = 10
	setting.DefaultRpmLimit = 0
	setting.RpmWindowSeconds = 60
	t.Cleanup(func() { *setting = original })
}

func useRelayCapacityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	model.DB = db
	model.LOG_DB = db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})
	require.NoError(t, db.AutoMigrate(&model.Channel{}))
	return db
}

func TestAcquireAdditionalChannelRequestRpmCountsEveryUpstreamRequest(t *testing.T) {
	enableLocalChannelCapacityForTest(t)
	db := useRelayCapacityTestDB(t)

	channelSetting := `{"max_concurrency":10,"rpm_limit":1}`
	channel := model.Channel{Id: 992001, Name: "responses", Status: common.ChannelStatusEnabled, Setting: &channelSetting}
	require.NoError(t, db.Create(&channel).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelId: channel.Id}}

	require.Nil(t, acquireAdditionalChannelRequestRpm(ctx, info))
	apiErr := acquireAdditionalChannelRequestRpm(ctx, info)
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	require.Greater(t, apiErr.RetryAfter, time.Duration(0))
}

func TestAcquireRealtimeTaskChannelCapacityFallsBackWhenChannelFull(t *testing.T) {
	enableLocalChannelCapacityForTest(t)
	channelSetting := `{"max_concurrency":1,"rpm_limit":10}`
	channel := &model.Channel{Id: 992002, Setting: &channelSetting}

	occupied := middleware.TryAcquireChannelCapacity(channel.Id, 1, 10, time.Minute, "occupied")
	require.True(t, occupied.Acquired)
	t.Cleanup(occupied.Release)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task", nil)
	release, acquired := acquireRealtimeTaskChannelCapacity(ctx, channel)
	require.False(t, acquired)
	require.Nil(t, release)
}

func TestApplyMidjourneyRateLimitMetadataForcesRateLimitCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	result := &dto.MidjourneyResponseWithStatusCode{
		StatusCode:         http.StatusTooManyRequests,
		UpstreamStatusCode: http.StatusTooManyRequests,
		RetryAfter:         7 * time.Second,
		Response: dto.MidjourneyResponse{
			Code:        4,
			Description: "rate limited",
		},
	}

	applyMidjourneyRateLimitMetadata(ctx, result)
	require.Equal(t, 30, result.Response.Code)
	require.Equal(t, "7", recorder.Header().Get("Retry-After"))
}
