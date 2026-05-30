package model

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDashboardModelChannelStatsTestDB(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()

	mainDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open main db: %v", err)
	}
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open log db: %v", err)
	}

	originDB := DB
	originLogDB := LOG_DB
	originUsingSQLite := common.UsingSQLite
	originUsingMySQL := common.UsingMySQL
	originUsingPostgreSQL := common.UsingPostgreSQL
	DB = mainDB
	LOG_DB = logDB
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	initCol()
	if dashboardModelChannelStatsCache != nil {
		_ = dashboardModelChannelStatsCache.Purge()
	}

	t.Cleanup(func() {
		if dashboardModelChannelStatsCache != nil {
			_ = dashboardModelChannelStatsCache.Purge()
		}
		DB = originDB
		LOG_DB = originLogDB
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
		initCol()
	})

	if err := mainDB.AutoMigrate(&Channel{}); err != nil {
		t.Fatalf("migrate channels: %v", err)
	}
	if err := logDB.AutoMigrate(&Log{}); err != nil {
		t.Fatalf("migrate logs: %v", err)
	}

	return mainDB, logDB
}

func TestGetDashboardModelChannelTagStatsAggregatesByChannelTag(t *testing.T) {
	mainDB, logDB := setupDashboardModelChannelStatsTestDB(t)

	vipTag := "vip"
	svipTag := "svip"
	channels := []Channel{
		{Id: 1, Name: "vip-a", Key: "key-1", Tag: &vipTag},
		{Id: 2, Name: "vip-b", Key: "key-2", Tag: &vipTag},
		{Id: 3, Name: "untagged", Key: "key-3"},
		{Id: 4, Name: "svip", Key: "key-4", Tag: &svipTag},
	}
	if err := mainDB.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	logs := []Log{
		{Username: "alice", Type: LogTypeConsume, ModelName: "gpt-4o", Quota: 100, ChannelId: 1, CreatedAt: 110},
		{Username: "alice", Type: LogTypeConsume, ModelName: "gpt-4o", Quota: 50, ChannelId: 2, CreatedAt: 120},
		{Username: "alice", Type: LogTypeConsume, ModelName: "gpt-4o", Quota: 25, ChannelId: 3, CreatedAt: 130},
		{Username: "alice", Type: LogTypeConsume, ModelName: "claude", Quota: 200, ChannelId: 4, CreatedAt: 140},
		{Username: "alice", Type: LogTypeConsume, ModelName: "old", Quota: 999, ChannelId: 4, CreatedAt: 10},
		{Username: "alice", Type: LogTypeError, ModelName: "gpt-4o", Quota: 999, ChannelId: 1, CreatedAt: 150},
	}
	if err := logDB.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	stats, err := GetDashboardModelChannelTagStats(100, 200, "", 10)
	if err != nil {
		t.Fatalf("GetDashboardModelChannelTagStats: %v", err)
	}
	if stats.TotalQuota != 375 {
		t.Fatalf("expected total quota 375, got %d", stats.TotalQuota)
	}
	if len(stats.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(stats.Models))
	}
	if stats.Models[0].ModelName != "claude" || stats.Models[0].Quota != 200 {
		t.Fatalf("expected claude first with quota 200, got %#v", stats.Models[0])
	}

	gptTags := make(map[string]DashboardChannelTagSpendStat)
	for _, item := range stats.ChannelTagShares {
		if item.ModelName == "gpt-4o" {
			gptTags[item.Tag] = item
		}
	}
	if gptTags["vip"].Quota != 150 {
		t.Fatalf("expected gpt-4o vip quota 150, got %#v", gptTags["vip"])
	}
	if gptTags[dashboardModelChannelUntaggedLabel].Quota != 25 {
		t.Fatalf("expected gpt-4o untagged quota 25, got %#v", gptTags[dashboardModelChannelUntaggedLabel])
	}
	if math.Abs(gptTags["vip"].Share-(150.0/175.0)) > 0.0001 {
		t.Fatalf("unexpected vip share: %f", gptTags["vip"].Share)
	}
}

func TestGetDashboardModelChannelTagStatsFiltersUsername(t *testing.T) {
	mainDB, logDB := setupDashboardModelChannelStatsTestDB(t)

	tag := "pool"
	if err := mainDB.Create(&Channel{Id: 1, Name: "pool", Key: "key", Tag: &tag}).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	logs := []Log{
		{Username: "alice", Type: LogTypeConsume, ModelName: "gpt-4o", Quota: 100, ChannelId: 1, CreatedAt: 110},
		{Username: "bob", Type: LogTypeConsume, ModelName: "gpt-4o", Quota: 300, ChannelId: 1, CreatedAt: 120},
	}
	if err := logDB.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	stats, err := GetDashboardModelChannelTagStats(100, 200, "alice", 10)
	if err != nil {
		t.Fatalf("GetDashboardModelChannelTagStats: %v", err)
	}
	if stats.TotalQuota != 100 {
		t.Fatalf("expected alice quota 100, got %d", stats.TotalQuota)
	}
	if len(stats.ChannelTagShares) != 1 || stats.ChannelTagShares[0].Quota != 100 {
		t.Fatalf("unexpected tag shares: %#v", stats.ChannelTagShares)
	}
}
