package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestCacheGetRandomSatisfiedChannel_SlowTTFTStateDoesNotFilter(t *testing.T) {
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

	channel := model.Channel{Id: 1, Name: "only-slow-tag", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled}
	channel.SetTag("slow-tag")
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	ability := model.Ability{Group: "Pro", Model: "gpt-5.5", ChannelId: 1, Enabled: true, Priority: common.GetPointer[int64](0), Weight: 100}
	if err := db.Create(&ability).Error; err != nil {
		t.Fatalf("seed ability: %v", err)
	}

	now := time.Now()
	guard := newSlowTTFTGuardState()
	guard.evidence[slowTTFTScope{Model: "gpt-5.5", Group: "Pro", Tag: "slow-tag"}] = &slowTTFTEvidenceState{
		OpenUntil: now.Add(time.Minute),
		LastSeen:  now,
	}
	useSlowTTFTGuardForTest(t, guard)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Set(ginKeySlowTTFTModel, "gpt-5.5")
	ctx.Set(ginKeySlowTTFTGroup, "Pro")
	ctx.Set(ginKeySlowTTFTSetting, slowTTFTTestSetting())

	got, _, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        ctx,
		TokenGroup: "Pro",
		ModelName:  "gpt-5.5",
		Retry:      common.GetPointer(0),
	})
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if got == nil || got.Id != 1 {
		t.Fatalf("expected slow TTFT state not to filter channel 1, got %#v", got)
	}
}

func TestCacheGetRandomSatisfiedChannel_NoMemoryCacheFallsBackAfterFilter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	originDB, originLogDB := model.DB, model.LOG_DB
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	model.DB, model.LOG_DB = db, db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originDB, originLogDB
		common.MemoryCacheEnabled = originMemoryCacheEnabled
	})
	if err := db.AutoMigrate(&model.Channel{}, &model.Ability{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	high, low := int64(10), int64(1)
	channels := []model.Channel{
		{Id: 1, Name: "filtered-high", Status: common.ChannelStatusEnabled, AutoBan: common.GetPointer(1), Priority: &high},
		{Id: 2, Name: "available-low", Status: common.ChannelStatusEnabled, AutoBan: common.GetPointer(1), Priority: &low},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}
	abilities := []model.Ability{
		{Group: "default", Model: "gpt-5.6-sol", ChannelId: 1, Enabled: true, Priority: &high, Weight: 100},
		{Group: "default", Model: "gpt-5.6-sol", ChannelId: 2, Enabled: true, Priority: &low, Weight: 100},
	}
	if err := db.Create(&abilities).Error; err != nil {
		t.Fatalf("seed abilities: %v", err)
	}
	got, err := model.GetRandomSatisfiedChannel("default", "gpt-5.6-sol", 0, nil, nil, func(channel *model.Channel) bool {
		return channel.Id != 1
	})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got == nil || got.Id != 2 {
		t.Fatalf("expected lower-priority channel 2, got %#v", got)
	}
}

func TestCacheGetSatisfiedChannelCandidatesReturnsAllSamePriority(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	originDB, originLogDB := model.DB, model.LOG_DB
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	model.DB, model.LOG_DB = db, db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originDB, originLogDB
		common.MemoryCacheEnabled = originMemoryCacheEnabled
	})
	if err := db.AutoMigrate(&model.Channel{}, &model.Ability{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	high, low := int64(10), int64(1)
	channels := []model.Channel{
		{Id: 1, Name: "high-a", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Priority: &high},
		{Id: 2, Name: "high-b", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Priority: &high},
		{Id: 3, Name: "low", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Priority: &low},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}
	abilities := []model.Ability{
		{Group: "default", Model: "gpt-5.6-sol", ChannelId: 1, Enabled: true, Priority: &high, Weight: 100},
		{Group: "default", Model: "gpt-5.6-sol", ChannelId: 2, Enabled: true, Priority: &high, Weight: 100},
		{Group: "default", Model: "gpt-5.6-sol", ChannelId: 3, Enabled: true, Priority: &low, Weight: 100},
	}
	if err := db.Create(&abilities).Error; err != nil {
		t.Fatalf("seed abilities: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	got, group, err := CacheGetSatisfiedChannelCandidates(&RetryParam{
		Ctx:        ctx,
		TokenGroup: "default",
		ModelName:  "gpt-5.6-sol",
		Retry:      common.GetPointer(0),
	})
	if err != nil {
		t.Fatalf("get candidates: %v", err)
	}
	if group != "default" {
		t.Fatalf("expected group default, got %s", group)
	}
	if len(got) != 2 || got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("expected same-priority candidates [1,2], got %#v", got)
	}
}
