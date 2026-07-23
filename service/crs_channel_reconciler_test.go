package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCRSReconcilerTest(t *testing.T) *gorm.DB {
	t.Helper()
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
	return db
}

func crsSnapshot(t *testing.T, active, schedulable, rateLimited bool) *model.CRSAccountSnapshot {
	t.Helper()
	raw, err := common.Marshal(map[string]any{
		"isActive":        active,
		"schedulable":     schedulable,
		"rateLimitStatus": map[string]any{"isRateLimited": rateLimited},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &model.CRSAccountSnapshot{Platform: "openai-responses", RawAccount: string(raw)}
}

func TestCRSReconcilerRequiresTwoObservationsAndOnlyRecoversOwnedChannel(t *testing.T) {
	db := setupCRSReconcilerTest(t)
	channel := model.Channel{Id: 1, Name: "managed", Status: common.ChannelStatusEnabled, AutoBan: common.GetPointer(1)}
	channel.SetSetting(dto.ChannelSettings{CRSSiteID: 6, CRSPlatform: "openai-responses", CRSAutoManage: true})
	if err := db.Create(&channel).Error; err != nil {
		t.Fatal(err)
	}
	ability := model.Ability{Group: "default", Model: "gpt-5.6-sol", ChannelId: 1, Enabled: true, Priority: common.GetPointer[int64](0)}
	if err := db.Create(&ability).Error; err != nil {
		t.Fatal(err)
	}
	unhealthy := []*model.CRSAccountSnapshot{crsSnapshot(t, true, false, true)}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", unhealthy, 100); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&channel, 1).Error; err != nil || channel.Status != common.ChannelStatusEnabled {
		t.Fatalf("first observation must not disable: status=%d err=%v", channel.Status, err)
	}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", unhealthy, 101); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&channel, 1).Error; err != nil || !model.IsCRSAutoDisabledChannel(&channel) {
		t.Fatalf("second observation should CRS-disable: status=%d err=%v", channel.Status, err)
	}
	if err := model.AutoDisableChannelForPeriodQuota(1, "channel", "1", 200, "period quota exceeded"); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&channel, 1).Error; err != nil || !model.HasPeriodQuotaMeta(&channel) {
		t.Fatalf("quota blocker should coexist with CRS blocker: err=%v", err)
	}
	if recovered, err := model.RecoverPeriodQuotaChannel(&channel, 200); err != nil || recovered {
		t.Fatalf("quota expiry must not bypass CRS blocker: recovered=%v err=%v", recovered, err)
	}
	if err := db.First(&channel, 1).Error; err != nil || model.HasPeriodQuotaMeta(&channel) || !model.IsCRSAutoDisabledChannel(&channel) {
		t.Fatalf("quota blocker should clear while CRS blocker remains: status=%d err=%v", channel.Status, err)
	}
	healthy := []*model.CRSAccountSnapshot{crsSnapshot(t, true, true, false)}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", healthy, 102); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", healthy, 103); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&channel, 1).Error; err != nil || channel.Status != common.ChannelStatusAutoDisabled {
		t.Fatalf("healthy snapshots without a probe must stay disabled: status=%d err=%v", channel.Status, err)
	}
	if err := model.MarkCRSRecoveryProbeSuccess(1, 103); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", healthy, 104); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&channel, 1).Error; err != nil || channel.Status != common.ChannelStatusEnabled {
		t.Fatalf("two healthy observations should recover owned channel: status=%d err=%v", channel.Status, err)
	}
}

func TestCRSReconcilerSkipsMissingHealthFieldsAndManualDisable(t *testing.T) {
	db := setupCRSReconcilerTest(t)
	channel := model.Channel{Id: 1, Name: "manual", Status: common.ChannelStatusManuallyDisabled, AutoBan: common.GetPointer(1)}
	channel.SetSetting(dto.ChannelSettings{CRSSiteID: 6, CRSPlatform: "openai-responses", CRSAutoManage: true})
	if err := db.Create(&channel).Error; err != nil {
		t.Fatal(err)
	}
	missing := []*model.CRSAccountSnapshot{{Platform: "openai-responses", RawAccount: `{"isActive":true}`}}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", missing, 100); err != nil {
		t.Fatal(err)
	}
	healthy := []*model.CRSAccountSnapshot{crsSnapshot(t, true, true, false)}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", healthy, 101); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", healthy, 102); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&channel, 1).Error; err != nil || channel.Status != common.ChannelStatusManuallyDisabled {
		t.Fatalf("manual disable must never recover: status=%d err=%v", channel.Status, err)
	}
}

func TestCRSReconcilerDoesNotRecoverAfterAnotherBlockerClaimsChannel(t *testing.T) {
	db := setupCRSReconcilerTest(t)
	channel := model.Channel{Id: 1, Name: "claimed", Status: common.ChannelStatusEnabled, AutoBan: common.GetPointer(1)}
	channel.SetSetting(dto.ChannelSettings{CRSSiteID: 6, CRSPlatform: "openai-responses", CRSAutoManage: true})
	if err := db.Create(&channel).Error; err != nil {
		t.Fatal(err)
	}
	unhealthy := []*model.CRSAccountSnapshot{crsSnapshot(t, true, false, true)}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", unhealthy, 100); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", unhealthy, 101); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&channel, 1).Error; err != nil {
		t.Fatal(err)
	}
	claimed, err := model.ClaimCRSAutoDisabledChannel(&channel, "low balance")
	if err != nil || !claimed {
		t.Fatalf("claim ownership: claimed=%v err=%v", claimed, err)
	}
	healthy := []*model.CRSAccountSnapshot{crsSnapshot(t, true, true, false)}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", healthy, 102); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileCRSManagedChannels(6, "openai-responses", healthy, 103); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&channel, 1).Error; err != nil || channel.Status != common.ChannelStatusAutoDisabled {
		t.Fatalf("claimed channel must stay disabled: status=%d err=%v", channel.Status, err)
	}
}

func TestSummarizeCRSOpenAIHealthSeparatesPlatforms(t *testing.T) {
	healthyOpenAI := crsSnapshot(t, true, true, false)
	healthyOpenAI.Platform = "openai"
	unhealthyResponses := crsSnapshot(t, true, false, true)
	total, healthy, complete := summarizeCRSOpenAIHealth("openai-responses", []*model.CRSAccountSnapshot{healthyOpenAI, unhealthyResponses})
	if !complete || total != 1 || healthy != 0 {
		t.Fatalf("unexpected responses pool health: total=%d healthy=%d complete=%v", total, healthy, complete)
	}
}

func TestOpenChannelShortCircuitIsChannelScoped(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	resetChannelModelCircuitForTest()
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		resetChannelModelCircuitForTest()
	})
	channelA := &model.Channel{Id: 1}
	channelB := &model.Channel{Id: 2}
	OpenChannelShortCircuit(channelA)
	if !IsChannelModelCircuitOpen(channelA, "gpt-5.6-sol") {
		t.Fatal("channel-wide circuit should cover every model on the channel")
	}
	if IsChannelModelCircuitOpen(channelB, "gpt-5.6-sol") {
		t.Fatal("channel-wide circuit must not affect another channel")
	}
}
