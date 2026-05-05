package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCacheGetRandomSatisfiedChannel_CodexAutoReviewAllowsOpenAICompatibleChannel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originDB := model.DB
	originLogDB := model.LOG_DB
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	model.DB = db
	model.LOG_DB = db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = originDB
		model.LOG_DB = originLogDB
		common.MemoryCacheEnabled = originMemoryCacheEnabled
	})

	if err := db.AutoMigrate(&model.Channel{}, &model.Ability{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	channels := []model.Channel{
		{Id: 1, Name: "openai-codex-model", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled},
		{Id: 2, Name: "claude-codex-model", Type: constant.ChannelTypeAnthropic, Status: common.ChannelStatusEnabled},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}
	abilities := []model.Ability{
		{Group: "default", Model: constant.CodexAutoReviewRoutingModel, ChannelId: 1, Enabled: true, Priority: common.GetPointer[int64](0), Weight: 100},
		{Group: "default", Model: constant.CodexAutoReviewRoutingModel, ChannelId: 2, Enabled: true, Priority: common.GetPointer[int64](0), Weight: 100},
	}
	if err := db.Create(&abilities).Error; err != nil {
		t.Fatalf("seed abilities: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	got, _, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        ctx,
		TokenGroup: "default",
		ModelName:  constant.CodexAutoReviewModel,
		Retry:      common.GetPointer(0),
	})
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if got == nil {
		t.Fatal("expected openai channel, got nil")
	}
	if got.Id != 1 {
		t.Fatalf("expected openai channel 1, got %d", got.Id)
	}
}
