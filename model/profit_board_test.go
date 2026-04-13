package model

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupProfitBoardTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originDB := DB
	originLogDB := LOG_DB
	originSQLite := common.UsingSQLite
	originMySQL := common.UsingMySQL
	originPostgres := common.UsingPostgreSQL

	DB = db
	LOG_DB = db
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	initCol()

	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.UsingSQLite = originSQLite
		common.UsingMySQL = originMySQL
		common.UsingPostgreSQL = originPostgres
		initCol()
	})

	if err := db.AutoMigrate(&User{}, &Channel{}, &Log{}, &Ability{}, &Model{}, &Vendor{}, &ProfitBoardConfig{}, &ProfitBoardUpstreamAccount{}, &ProfitBoardRemoteSnapshot{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestGetProfitBoardOptionsIncludesAdminUsers(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	users := []User{
		{Id: 1, Username: "admin-user", DisplayName: "Admin User", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AffCode: "pb-admin-user"},
		{Id: 2, Username: "root-user", DisplayName: "Root User", Role: common.RoleRootUser, Status: common.UserStatusEnabled, AffCode: "pb-root-user"},
		{Id: 3, Username: "common-user", DisplayName: "Common User", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "pb-common-user"},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}

	options, err := GetProfitBoardOptions()
	if err != nil {
		t.Fatalf("GetProfitBoardOptions: %v", err)
	}

	if len(options.AdminUsers) != 2 {
		t.Fatalf("expected 2 admin users, got %+v", options.AdminUsers)
	}
	if options.AdminUsers[0].Id != 2 || options.AdminUsers[1].Id != 1 {
		t.Fatalf("unexpected admin users order: %+v", options.AdminUsers)
	}
}

func TestProfitBoardExcludedUsersSkipConfiguredRevenueAndKeepUpstreamCost(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	users := []User{
		{Id: 1, Username: "admin-user", DisplayName: "Admin User", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AffCode: "pb-excluded-admin"},
		{Id: 2, Username: "common-user", DisplayName: "Common User", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "pb-excluded-common"},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := common.GetTimestamp()
	logs := []Log{
		{Id: 1, UserId: 1, Username: "admin-user", Type: LogTypeConsume, CreatedAt: now - 120, ChannelId: 1, ModelName: "gpt-4.1", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 2, UserId: 2, Username: "common-user", Type: LogTypeConsume, CreatedAt: now - 60, ChannelId: 1, ModelName: "gpt-4.1", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
	}
	for _, logRow := range logs {
		if err := db.Create(&logRow).Error; err != nil {
			t.Fatalf("seed log: %v", err)
		}
	}

	batches := []ProfitBoardBatch{{
		Id:         "batch-1",
		Name:       "组合 1",
		ScopeType:  ProfitBoardScopeChannel,
		ChannelIDs: []int{1},
		CreatedAt:  now - 300,
	}}
	comboConfigs := []ProfitBoardComboPricingConfig{{
		ComboId:              "batch-1",
		SiteRules:            []ProfitBoardModelPricingRule{{IsDefault: true, InputPrice: 5}},
		UpstreamRules:        []ProfitBoardModelPricingRule{{IsDefault: true, InputPrice: 2}},
		SiteFixedTotalAmount: 0.2,
	}}
	upstreamConfig := ProfitBoardTokenPricingConfig{
		CostSource: ProfitBoardCostSourceManualOnly,
	}
	siteConfig := ProfitBoardTokenPricingConfig{
		PricingMode: ProfitBoardSitePricingManual,
	}

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches:         batches,
		ComboConfigs:    comboConfigs,
		ExcludedUserIDs: []int{1},
		Upstream:        upstreamConfig,
		Site:            siteConfig,
		StartTimestamp:  now - 300,
		EndTimestamp:    now,
		Granularity:     "day",
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if report.Summary.RequestCount != 2 {
		t.Fatalf("unexpected request count: %+v", report.Summary)
	}
	if report.Summary.ConfiguredSiteRevenueUSD != 0.205 {
		t.Fatalf("expected excluded admin revenue to be skipped, got %+v", report.Summary)
	}
	if report.Summary.UpstreamCostUSD != 0.004 {
		t.Fatalf("expected upstream cost kept for both requests, got %+v", report.Summary)
	}
	if report.Summary.ConfiguredProfitUSD != 0.201 {
		t.Fatalf("unexpected configured profit: %+v", report.Summary)
	}

	overview, err := GenerateProfitBoardOverview(ProfitBoardConfigPayload{
		Batches:         batches,
		ComboConfigs:    comboConfigs,
		ExcludedUserIDs: []int{1},
		Upstream:        upstreamConfig,
		Site:            siteConfig,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardOverview: %v", err)
	}
	if overview.Summary.ConfiguredSiteRevenueUSD != 0.205 || overview.Summary.UpstreamCostUSD != 0.004 || overview.Summary.ConfiguredProfitUSD != 0.201 {
		t.Fatalf("unexpected overview summary: %+v", overview.Summary)
	}
}

func seedProfitBoardRemoteSnapshot(t *testing.T, selectionSignature string, comboID string, config ProfitBoardRemoteObserverConfig, syncedAt int64, walletQuota int64, walletUsedQuota int64, subscriptions []ProfitBoardRemoteSubscriptionSnapshot) {
	t.Helper()

	payload, err := common.Marshal(subscriptions)
	if err != nil {
		t.Fatalf("marshal subscriptions: %v", err)
	}
	snapshot := ProfitBoardRemoteSnapshot{
		SelectionSignature: selectionSignature,
		ComboId:            comboID,
		ConfigHash:         profitBoardRemoteObserverConfigHash(config),
		Status:             profitBoardRemoteSnapshotStatusSuccess,
		RemoteQuotaPerUnit: common.QuotaPerUnit,
		WalletQuota:        walletQuota,
		WalletUsedQuota:    walletUsedQuota,
		SubscriptionStates: string(payload),
		SyncedAt:           syncedAt,
		CreatedAt:          syncedAt,
	}
	if err = DB.Create(&snapshot).Error; err != nil {
		t.Fatalf("seed remote snapshot: %v", err)
	}
}

func TestSaveProfitBoardConfigUsesStableSignature(t *testing.T) {
	setupProfitBoardTestDB(t)

	payload := ProfitBoardConfigPayload{
		Batches: []ProfitBoardBatch{
			{
				Id:         "batch-stable",
				Name:       "批次 A",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{9, 3, 9, 1},
			},
		},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceReturnedFirst,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
	}

	_, signature, err := SaveProfitBoardConfig(payload)
	if err != nil {
		t.Fatalf("SaveProfitBoardConfig: %v", err)
	}
	if signature != "channel:1,3,9" {
		t.Fatalf("unexpected signature: %s", signature)
	}

	loaded, loadedSignature, err := GetProfitBoardConfig([]ProfitBoardBatch{
		{
			Id:         "batch-lookup",
			Name:       "批次 B",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{3, 1, 9},
		},
	}, ProfitBoardSelection{})
	if err != nil {
		t.Fatalf("GetProfitBoardConfig: %v", err)
	}
	if loadedSignature != signature {
		t.Fatalf("signature mismatch: got=%s want=%s", loadedSignature, signature)
	}
	if len(loaded.Batches) != 1 || len(loaded.Batches[0].ChannelIDs) != 3 {
		t.Fatalf("expected 1 batch with 3 channel ids, got %+v", loaded.Batches)
	}
}

func TestSaveProfitBoardConfigAssignsBatchCreatedAtWhenMissing(t *testing.T) {
	setupProfitBoardTestDB(t)

	before := common.GetTimestamp()
	saved, _, err := SaveProfitBoardConfig(ProfitBoardConfigPayload{
		Batches: []ProfitBoardBatch{{
			Id:         "batch-created-at",
			Name:       "组合起点",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		}},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
	})
	if err != nil {
		t.Fatalf("SaveProfitBoardConfig: %v", err)
	}
	if len(saved.Batches) != 1 {
		t.Fatalf("expected one saved batch, got %+v", saved.Batches)
	}
	if saved.Batches[0].CreatedAt < before {
		t.Fatalf("expected created_at >= %d, got %+v", before, saved.Batches)
	}

	record := ProfitBoardConfig{}
	if err := DB.First(&record).Error; err != nil {
		t.Fatalf("load record: %v", err)
	}
	loadedBatches := parseProfitBoardConfigBatches(record.SelectionValues)
	if len(loadedBatches) != 1 {
		t.Fatalf("expected one persisted batch, got %+v", loadedBatches)
	}
	if loadedBatches[0].CreatedAt != saved.Batches[0].CreatedAt {
		t.Fatalf("expected persisted created_at %d, got %+v", saved.Batches[0].CreatedAt, loadedBatches)
	}
}

func TestGetProfitBoardConfigRemapsComboConfigToCurrentBatchID(t *testing.T) {
	setupProfitBoardTestDB(t)

	payload := ProfitBoardConfigPayload{
		Batches: []ProfitBoardBatch{
			{
				Id:         "batch-saved",
				Name:       "组合 A",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{3, 1},
			},
		},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
		ComboConfigs: []ProfitBoardComboPricingConfig{
			{
				ComboId:                  "batch-saved",
				SiteMode:                 ProfitBoardComboSiteModeManual,
				UpstreamMode:             ProfitBoardUpstreamModeManual,
				SiteFixedTotalAmount:     12.5,
				UpstreamFixedTotalAmount: 4.25,
				SiteRules: []ProfitBoardModelPricingRule{
					{
						IsDefault:   true,
						InputPrice:  1.23,
						OutputPrice: 4.56,
					},
				},
				UpstreamRules: []ProfitBoardModelPricingRule{
					{
						IsDefault:   true,
						InputPrice:  0.78,
						OutputPrice: 0.9,
					},
				},
			},
		},
	}

	if _, _, err := SaveProfitBoardConfig(payload); err != nil {
		t.Fatalf("SaveProfitBoardConfig: %v", err)
	}

	loaded, _, err := GetProfitBoardConfig([]ProfitBoardBatch{
		{
			Id:         "batch-current",
			Name:       "组合 B",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1, 3},
		},
	}, ProfitBoardSelection{})
	if err != nil {
		t.Fatalf("GetProfitBoardConfig: %v", err)
	}
	if len(loaded.ComboConfigs) != 1 {
		t.Fatalf("unexpected combo configs: %+v", loaded.ComboConfigs)
	}

	combo := loaded.ComboConfigs[0]
	if combo.ComboId != "batch-current" {
		t.Fatalf("unexpected combo id: %+v", combo)
	}
	if combo.SiteFixedTotalAmount != 12.5 || combo.UpstreamFixedTotalAmount != 4.25 {
		t.Fatalf("fixed totals not remapped: %+v", combo)
	}
	if len(combo.SiteRules) != 1 || combo.SiteRules[0].InputPrice != 1.23 || combo.SiteRules[0].OutputPrice != 4.56 {
		t.Fatalf("site rules not preserved: %+v", combo.SiteRules)
	}
	if len(combo.UpstreamRules) != 1 || combo.UpstreamRules[0].InputPrice != 0.78 || combo.UpstreamRules[0].OutputPrice != 0.9 {
		t.Fatalf("upstream rules not preserved: %+v", combo.UpstreamRules)
	}
}

func TestGenerateProfitBoardReportUsesReturnedAndManualFallback(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	channels := []Channel{
		{Id: 1, Name: "alpha", Tag: common.GetPointer("tag-a"), Status: common.ChannelStatusEnabled},
		{Id: 2, Name: "beta", Tag: common.GetPointer("tag-a"), Status: common.ChannelStatusEnabled},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	logs := []Log{
		{
			Id:               1,
			Type:             LogTypeConsume,
			CreatedAt:        1710000000,
			ChannelId:        1,
			ModelName:        "gpt-4o",
			Quota:            3000000,
			PromptTokens:     2000000,
			CompletionTokens: 500000,
			RequestId:        "req-1",
			Other:            `{"cache_tokens":200000,"cache_creation_tokens":100000,"upstream_cost":5.5,"upstream_cost_reported":true,"upstream_cost_source":"provider"}`,
		},
		{
			Id:               2,
			Type:             LogTypeConsume,
			CreatedAt:        1710003600,
			ChannelId:        2,
			ModelName:        "gpt-4o-mini",
			Quota:            1250000,
			PromptTokens:     900000,
			CompletionTokens: 250000,
			RequestId:        "req-2",
			Other:            `{"cache_tokens":100000}`,
		},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches: []ProfitBoardBatch{
			{
				Id:        "batch-tag-a",
				Name:      "Tag A",
				ScopeType: ProfitBoardScopeTag,
				Tags:      []string{"tag-a"},
			},
		},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource:         ProfitBoardCostSourceReturnedFirst,
			InputPrice:         1,
			OutputPrice:        2,
			CacheReadPrice:     0.5,
			CacheCreationPrice: 0.5,
			FixedAmount:        0.5,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode:        ProfitBoardSitePricingManual,
			InputPrice:         2,
			OutputPrice:        4,
			CacheReadPrice:     1,
			CacheCreationPrice: 1,
			FixedAmount:        0.5,
		},
		StartTimestamp: 1709990000,
		EndTimestamp:   1710010000,
		Granularity:    "hour",
		IncludeDetails: true,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if report.Summary.RequestCount != 2 {
		t.Fatalf("unexpected request count: %d", report.Summary.RequestCount)
	}
	if report.Summary.ActualSiteRevenueUSD <= 0 {
		t.Fatalf("expected actual site revenue, got %v", report.Summary.ActualSiteRevenueUSD)
	}
	if report.Summary.ConfiguredSiteRevenueUSD <= 0 {
		t.Fatalf("expected configured site revenue, got %v", report.Summary.ConfiguredSiteRevenueUSD)
	}
	if report.Summary.UpstreamCostUSD <= 0 {
		t.Fatalf("expected upstream cost, got %v", report.Summary.UpstreamCostUSD)
	}
	if report.Summary.ConfiguredProfitUSD == 0 {
		t.Fatalf("expected configured profit, got %v", report.Summary.ConfiguredProfitUSD)
	}
	if len(report.ChannelBreakdown) != 2 {
		t.Fatalf("expected 2 channel breakdown items, got %d", len(report.ChannelBreakdown))
	}
	if len(report.BatchSummaries) != 1 || report.BatchSummaries[0].BatchName != "Tag A" {
		t.Fatalf("unexpected batch summaries: %+v", report.BatchSummaries)
	}
}

func TestGetProfitBoardConfigUsesManualOnlyForNewSelection(t *testing.T) {
	setupProfitBoardTestDB(t)

	config, _, err := GetProfitBoardConfig([]ProfitBoardBatch{
		{
			Name:       "批次 A",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		},
	}, ProfitBoardSelection{})
	if err != nil {
		t.Fatalf("GetProfitBoardConfig: %v", err)
	}
	if config.Upstream.CostSource != ProfitBoardCostSourceManualOnly {
		t.Fatalf("unexpected default cost source: %s", config.Upstream.CostSource)
	}
}

func TestGetProfitBoardConfigKeepsLegacySavedDefault(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	record := ProfitBoardConfig{
		SelectionType:      ProfitBoardScopeChannel,
		SelectionSignature: "channel:1",
		SelectionValues:    `[{"id":"batch-1","name":"批次 1","scope_type":"channel","channel_ids":[1]}]`,
		UpstreamConfig:     `{}`,
		SiteConfig:         `{"pricing_mode":"manual"}`,
		CreatedAt:          1,
		UpdatedAt:          1,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	config, _, err := GetProfitBoardConfig([]ProfitBoardBatch{
		{
			Name:       "批次 A",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		},
	}, ProfitBoardSelection{})
	if err != nil {
		t.Fatalf("GetProfitBoardConfig: %v", err)
	}
	if config.Upstream.CostSource != ProfitBoardCostSourceManualOnly {
		t.Fatalf("unexpected saved default cost source: %s", config.Upstream.CostSource)
	}
}

func TestGenerateProfitBoardReportTreatsZeroReturnedCostAsKnown(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	logEntry := Log{
		Id:               1,
		Type:             LogTypeConsume,
		CreatedAt:        1710000000,
		ChannelId:        1,
		ModelName:        "gpt-4o",
		Quota:            1000000,
		PromptTokens:     1000,
		CompletionTokens: 500,
		RequestId:        "req-zero",
		Other:            `{"upstream_cost":0,"upstream_cost_reported":true,"upstream_cost_source":"provider"}`,
	}
	if err := db.Create(&logEntry).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches: []ProfitBoardBatch{
			{
				Id:         "batch-1",
				Name:       "批次 1",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{1},
			},
		},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceReturnedOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
			InputPrice:  1,
		},
		StartTimestamp: 1709990000,
		EndTimestamp:   1710010000,
		Granularity:    "hour",
		IncludeDetails: true,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if report.Summary.UpstreamCostUSD != 0 {
		t.Fatalf("unexpected upstream cost amount: %v", report.Summary.UpstreamCostUSD)
	}
}

func TestGenerateProfitBoardReportSupportsMonthGranularity(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	loc := time.Local
	logs := []Log{
		{
			Id:               1,
			Type:             LogTypeConsume,
			CreatedAt:        time.Date(2026, 1, 15, 10, 0, 0, 0, loc).Unix(),
			ChannelId:        1,
			ModelName:        "gpt-4o",
			Quota:            1000000,
			PromptTokens:     1000,
			CompletionTokens: 500,
			RequestId:        "req-jan",
			Other:            `{"upstream_cost":1,"upstream_cost_reported":true}`,
		},
		{
			Id:               2,
			Type:             LogTypeConsume,
			CreatedAt:        time.Date(2026, 2, 2, 9, 0, 0, 0, loc).Unix(),
			ChannelId:        1,
			ModelName:        "gpt-4o",
			Quota:            1000000,
			PromptTokens:     1000,
			CompletionTokens: 500,
			RequestId:        "req-feb",
			Other:            `{"upstream_cost":1,"upstream_cost_reported":true}`,
		},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches: []ProfitBoardBatch{{
			Id:         "batch-1",
			Name:       "批次 1",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		}},
		Upstream: ProfitBoardTokenPricingConfig{CostSource: ProfitBoardCostSourceReturnedOnly},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
			InputPrice:  1,
		},
		StartTimestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, loc).Unix(),
		EndTimestamp:   time.Date(2026, 2, 28, 23, 59, 59, 0, loc).Unix(),
		Granularity:    "month",
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if len(report.Timeseries) != 2 {
		t.Fatalf("expected 2 month buckets, got %d", len(report.Timeseries))
	}
	if report.Timeseries[0].Bucket != "2026-01" || report.Timeseries[1].Bucket != "2026-02" {
		t.Fatalf("unexpected month buckets: %+v", report.Timeseries)
	}
}

func TestGenerateProfitBoardReportSupportsCustomMinuteGranularity(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	loc := time.Local
	logs := []Log{
		{
			Id:               1,
			Type:             LogTypeConsume,
			CreatedAt:        time.Date(2026, 3, 1, 10, 5, 0, 0, loc).Unix(),
			ChannelId:        1,
			ModelName:        "gpt-4o",
			Quota:            1000000,
			PromptTokens:     1000,
			CompletionTokens: 500,
			RequestId:        "req-a",
			Other:            `{"upstream_cost":1,"upstream_cost_reported":true}`,
		},
		{
			Id:               2,
			Type:             LogTypeConsume,
			CreatedAt:        time.Date(2026, 3, 1, 10, 17, 0, 0, loc).Unix(),
			ChannelId:        1,
			ModelName:        "gpt-4o",
			Quota:            1000000,
			PromptTokens:     1000,
			CompletionTokens: 500,
			RequestId:        "req-b",
			Other:            `{"upstream_cost":1,"upstream_cost_reported":true}`,
		},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches: []ProfitBoardBatch{{
			Id:         "batch-1",
			Name:       "批次 1",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		}},
		Upstream: ProfitBoardTokenPricingConfig{CostSource: ProfitBoardCostSourceReturnedOnly},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
			InputPrice:  1,
		},
		StartTimestamp:        time.Date(2026, 3, 1, 10, 0, 0, 0, loc).Unix(),
		EndTimestamp:          time.Date(2026, 3, 1, 10, 30, 0, 0, loc).Unix(),
		Granularity:           "custom",
		CustomIntervalMinutes: 15,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if len(report.Timeseries) != 2 {
		t.Fatalf("expected 2 custom buckets, got %d", len(report.Timeseries))
	}
	if report.Timeseries[0].Bucket != "2026-03-01 10:00" || report.Timeseries[1].Bucket != "2026-03-01 10:15" {
		t.Fatalf("unexpected custom buckets: %+v", report.Timeseries)
	}
}

func TestGetProfitBoardActivityReturnsWatermark(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	logs := []Log{
		{
			Id:               11,
			Type:             LogTypeConsume,
			CreatedAt:        1710000000,
			ChannelId:        1,
			ModelName:        "gpt-4o",
			Quota:            1000000,
			PromptTokens:     1000,
			CompletionTokens: 500,
			RequestId:        "req-1",
			Other:            `{"upstream_cost":1,"upstream_cost_reported":true}`,
		},
		{
			Id:               12,
			Type:             LogTypeConsume,
			CreatedAt:        1710003600,
			ChannelId:        1,
			ModelName:        "gpt-4o",
			Quota:            1000000,
			PromptTokens:     1000,
			CompletionTokens: 500,
			RequestId:        "req-2",
			Other:            `{"upstream_cost":1,"upstream_cost_reported":true}`,
		},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	activity, err := GetProfitBoardActivity(ProfitBoardQuery{
		Batches: []ProfitBoardBatch{{
			Id:         "batch-1",
			Name:       "批次 1",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		}},
		Upstream:       ProfitBoardTokenPricingConfig{CostSource: ProfitBoardCostSourceReturnedOnly},
		Site:           ProfitBoardTokenPricingConfig{PricingMode: ProfitBoardSitePricingManual},
		StartTimestamp: 1709990000,
		EndTimestamp:   1710010000,
		Granularity:    "day",
	})
	if err != nil {
		t.Fatalf("GetProfitBoardActivity: %v", err)
	}

	if activity.RequestCount != 0 || activity.LatestLogId != 12 || activity.LatestLogCreatedAt != 1710003600 {
		t.Fatalf("unexpected activity payload: %+v", activity)
	}
	if activity.ActivityWatermark != "0:12:1710003600" {
		t.Fatalf("unexpected watermark: %s", activity.ActivityWatermark)
	}
}

func TestGetProfitBoardActivityWatermarkChangesWhenWalletSnapshotChanges(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := common.GetTimestamp()
	account := ProfitBoardUpstreamAccount{
		Name:        "钱包账户",
		AccountType: ProfitBoardUpstreamAccountTypeNewAPI,
		BaseURL:     "https://remote.example.com",
		UserID:      9,
		Enabled:     true,
		CreatedAt:   now - 300,
		UpdatedAt:   now - 300,
	}
	encryptedToken, err := encryptProfitBoardRemoteSecret("wallet-token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	account.AccessTokenEncrypted = encryptedToken
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create wallet account: %v", err)
	}

	query := ProfitBoardQuery{
		Batches: []ProfitBoardBatch{{
			Id:         "wallet-batch",
			Name:       "钱包组合",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		}},
		ComboConfigs: []ProfitBoardComboPricingConfig{{
			ComboId:           "wallet-batch",
			UpstreamMode:      ProfitBoardUpstreamModeWallet,
			UpstreamAccountID: account.Id,
		}},
		Upstream:       ProfitBoardTokenPricingConfig{CostSource: ProfitBoardCostSourceManualOnly},
		Site:           ProfitBoardTokenPricingConfig{PricingMode: ProfitBoardSitePricingManual},
		StartTimestamp: now - 600,
		EndTimestamp:   now,
		Granularity:    "day",
	}

	before, err := GetProfitBoardActivity(query)
	if err != nil {
		t.Fatalf("GetProfitBoardActivity before snapshot: %v", err)
	}

	config := account.remoteObserverConfig()
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now-60, 800, 120, nil)

	after, err := GetProfitBoardActivity(query)
	if err != nil {
		t.Fatalf("GetProfitBoardActivity after snapshot: %v", err)
	}

	if before.LatestLogId != after.LatestLogId || before.LatestLogCreatedAt != after.LatestLogCreatedAt {
		t.Fatalf("log watermark should stay unchanged: before=%+v after=%+v", before, after)
	}
	if before.ActivityWatermark == after.ActivityWatermark {
		t.Fatalf("expected wallet snapshot to change activity watermark: before=%s after=%s", before.ActivityWatermark, after.ActivityWatermark)
	}
	if !strings.Contains(after.ActivityWatermark, "|") {
		t.Fatalf("expected wallet snapshot watermark suffix, got %s", after.ActivityWatermark)
	}
}

func TestCollectProfitBoardUpstreamAccountObservedAggregateIncludesLatestSnapshotBeyondQueryEnd(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	now := common.GetTimestamp()
	account := ProfitBoardUpstreamAccount{
		Name:        "钱包账户",
		AccountType: ProfitBoardUpstreamAccountTypeNewAPI,
		BaseURL:     "https://remote.example.com",
		UserID:      9,
		Enabled:     true,
		CreatedAt:   now - 300,
		UpdatedAt:   now - 300,
	}
	encryptedToken, err := encryptProfitBoardRemoteSecret("wallet-token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	account.AccessTokenEncrypted = encryptedToken
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create wallet account: %v", err)
	}

	config := account.remoteObserverConfig()
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	seedProfitBoardRemoteSnapshot(
		t,
		signature,
		profitBoardUpstreamAccountSnapshotComboID,
		config,
		now-2,
		800,
		120,
		nil,
	)

	aggregate, err := collectProfitBoardUpstreamAccountObservedAggregate(
		account.Id,
		now-600,
		now-10,
		"day",
		0,
		false,
	)
	if err != nil {
		t.Fatalf("collectProfitBoardUpstreamAccountObservedAggregate: %v", err)
	}

	if aggregate.State.SnapshotCount != 1 {
		t.Fatalf("expected latest snapshot to be included, got snapshot_count=%d", aggregate.State.SnapshotCount)
	}
	warnings := strings.Join(aggregate.Warnings, "\n")
	if strings.Contains(warnings, "在所选时间范围内没有成功同步的远端快照") {
		t.Fatalf("expected latest snapshot to suppress no-snapshot warning, got warnings=%v", aggregate.Warnings)
	}
	if !strings.Contains(warnings, "当前仅有 1 个成功快照") {
		t.Fatalf("expected single-snapshot warning, got warnings=%v", aggregate.Warnings)
	}
}

func TestSaveProfitBoardConfigMasksRemoteObserverToken(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	payload := ProfitBoardConfigPayload{
		Batches: []ProfitBoardBatch{{
			Id:         "batch-1",
			Name:       "批次 1",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		}},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
		ComboConfigs: []ProfitBoardComboPricingConfig{{
			ComboId: "batch-1",
			RemoteObserver: ProfitBoardRemoteObserverConfig{
				Enabled:     true,
				BaseURL:     "https://remote.example.com",
				UserID:      9,
				AccessToken: "secret-token-value",
			},
		}},
	}

	saved, signature, err := SaveProfitBoardConfig(payload)
	if err != nil {
		t.Fatalf("SaveProfitBoardConfig: %v", err)
	}

	observer := saved.ComboConfigs[0].RemoteObserver
	if observer.AccessToken != "" || observer.AccessTokenEncrypted != "" {
		t.Fatalf("remote observer token should be stripped from response: %+v", observer)
	}
	if observer.AccessTokenMasked == "" {
		t.Fatalf("expected masked token in response")
	}

	record := ProfitBoardConfig{}
	if err = db.Where("selection_signature = ?", signature).First(&record).Error; err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if strings.Contains(record.SiteConfig, "secret-token-value") {
		t.Fatalf("plain text token leaked into site config")
	}

	loaded, _, err := GetProfitBoardConfig(payload.Batches, ProfitBoardSelection{})
	if err != nil {
		t.Fatalf("GetProfitBoardConfig: %v", err)
	}
	loadedObserver := loaded.ComboConfigs[0].RemoteObserver
	if loadedObserver.AccessToken != "" || loadedObserver.AccessTokenEncrypted != "" {
		t.Fatalf("loaded remote observer should keep token hidden: %+v", loadedObserver)
	}
	if loadedObserver.AccessTokenMasked == "" {
		t.Fatalf("expected masked token after reload")
	}
}

func TestGetProfitBoardConfigMigratesGlobalSharedSiteIntoComboConfig(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	record := ProfitBoardConfig{
		SelectionType:      ProfitBoardScopeChannel,
		SelectionSignature: "channel:1",
		SelectionValues:    `[{"id":"batch-1","name":"批次 1","scope_type":"channel","channel_ids":[1]}]`,
		UpstreamConfig:     `{}`,
		SiteConfig:         `{"legacy_site":{"pricing_mode":"manual"},"shared_site":{"model_names":["gpt-4.1"],"group":"vip","use_recharge_price":true},"combo_configs":[{"combo_id":"batch-1","site_mode":"shared_site_model"}]}`,
		CreatedAt:          1,
		UpdatedAt:          1,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	config, _, err := GetProfitBoardConfig([]ProfitBoardBatch{{
		Id:         "batch-1",
		Name:       "批次 1",
		ScopeType:  ProfitBoardScopeChannel,
		ChannelIDs: []int{1},
	}}, ProfitBoardSelection{})
	if err != nil {
		t.Fatalf("GetProfitBoardConfig: %v", err)
	}
	if len(config.ComboConfigs) != 1 {
		t.Fatalf("unexpected combo configs: %+v", config.ComboConfigs)
	}
	shared := config.ComboConfigs[0].SharedSite
	if len(shared.ModelNames) != 1 || shared.ModelNames[0] != "gpt-4.1" {
		t.Fatalf("unexpected combo shared models: %+v", shared)
	}
	if shared.Group != "vip" || !shared.UseRechargePrice {
		t.Fatalf("unexpected combo shared config: %+v", shared)
	}
}

func TestSaveProfitBoardConfigKeepsComboSharedSiteIndependent(t *testing.T) {
	setupProfitBoardTestDB(t)

	payload := ProfitBoardConfigPayload{
		Batches: []ProfitBoardBatch{
			{
				Id:         "batch-a",
				Name:       "组合 A",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{1},
			},
			{
				Id:         "batch-b",
				Name:       "组合 B",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{2},
			},
		},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
		ComboConfigs: []ProfitBoardComboPricingConfig{
			{
				ComboId:  "batch-a",
				SiteMode: ProfitBoardComboSiteModeSharedSite,
				SharedSite: ProfitBoardSharedSitePricingConfig{
					ModelNames:       []string{"gpt-4.1"},
					Group:            "vip",
					UseRechargePrice: true,
				},
			},
			{
				ComboId:  "batch-b",
				SiteMode: ProfitBoardComboSiteModeSharedSite,
				SharedSite: ProfitBoardSharedSitePricingConfig{
					ModelNames: []string{"gpt-4o-mini"},
					Group:      "default",
				},
			},
		},
	}

	saved, _, err := SaveProfitBoardConfig(payload)
	if err != nil {
		t.Fatalf("SaveProfitBoardConfig: %v", err)
	}
	if len(saved.ComboConfigs) != 2 {
		t.Fatalf("unexpected saved combo configs: %+v", saved.ComboConfigs)
	}

	comboMap := map[string]ProfitBoardSharedSitePricingConfig{}
	for _, combo := range saved.ComboConfigs {
		comboMap[combo.ComboId] = combo.SharedSite
	}
	if comboMap["batch-a"].Group != "vip" || !comboMap["batch-a"].UseRechargePrice {
		t.Fatalf("unexpected batch-a shared site: %+v", comboMap["batch-a"])
	}
	if comboMap["batch-b"].Group != "default" || comboMap["batch-b"].UseRechargePrice {
		t.Fatalf("unexpected batch-b shared site: %+v", comboMap["batch-b"])
	}
	if strings.Join(comboMap["batch-a"].ModelNames, ",") == strings.Join(comboMap["batch-b"].ModelNames, ",") {
		t.Fatalf("combo shared site configs should stay independent: %+v", comboMap)
	}
}

func TestSaveProfitBoardConfigKeepsComboWalletModeIndependent(t *testing.T) {
	setupProfitBoardTestDB(t)

	payload := ProfitBoardConfigPayload{
		Batches: []ProfitBoardBatch{
			{
				Id:         "batch-a",
				Name:       "组合 A",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{1},
			},
			{
				Id:         "batch-b",
				Name:       "组合 B",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{2},
			},
		},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
		ComboConfigs: []ProfitBoardComboPricingConfig{
			{
				ComboId:           "batch-a",
				UpstreamMode:      ProfitBoardUpstreamModeWallet,
				UpstreamAccountID: 21,
			},
			{
				ComboId:      "batch-b",
				UpstreamMode: ProfitBoardUpstreamModeManual,
			},
		},
	}

	saved, _, err := SaveProfitBoardConfig(payload)
	if err != nil {
		t.Fatalf("SaveProfitBoardConfig: %v", err)
	}
	if len(saved.ComboConfigs) != 2 {
		t.Fatalf("unexpected combo configs: %+v", saved.ComboConfigs)
	}

	comboMap := map[string]ProfitBoardComboPricingConfig{}
	for _, combo := range saved.ComboConfigs {
		comboMap[combo.ComboId] = combo
	}
	if comboMap["batch-a"].UpstreamMode != ProfitBoardUpstreamModeWallet || comboMap["batch-a"].UpstreamAccountID != 21 {
		t.Fatalf("unexpected batch-a wallet config: %+v", comboMap["batch-a"])
	}
	if comboMap["batch-b"].UpstreamMode != ProfitBoardUpstreamModeManual || comboMap["batch-b"].UpstreamAccountID != 0 {
		t.Fatalf("unexpected batch-b upstream config: %+v", comboMap["batch-b"])
	}
}

func TestBuildProfitBoardUpstreamAccountStateUsesWalletSnapshotKey(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	account := ProfitBoardUpstreamAccount{
		Name:                   "主账户",
		AccountType:            ProfitBoardUpstreamAccountTypeNewAPI,
		BaseURL:                "https://remote.example.com",
		UserID:                 42,
		Enabled:                true,
		LowBalanceThresholdUSD: 0.9,
		CreatedAt:              1,
		UpdatedAt:              1,
	}
	encryptedToken, err := encryptProfitBoardRemoteSecret("remote-token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	account.AccessTokenEncrypted = encryptedToken
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}

	config := account.remoteObserverConfig()
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	now := common.GetTimestamp()
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now-120, 800, 120, nil)
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now-60, 800, 200, nil)

	options, err := GetProfitBoardUpstreamAccountOptions()
	if err != nil {
		t.Fatalf("GetProfitBoardUpstreamAccountOptions: %v", err)
	}
	if len(options) != 1 {
		t.Fatalf("unexpected account options: %+v", options)
	}
	if options[0].Status != profitBoardRemoteObserverStatusReady || !options[0].BaselineReady {
		t.Fatalf("unexpected option status: %+v", options[0])
	}
	if options[0].ResourceDisplayMode != ProfitBoardResourceDisplayBoth {
		t.Fatalf("unexpected resource display mode: %+v", options[0])
	}
	if options[0].WalletBalanceUSD != 0.8 || options[0].WalletQuotaUSD != 0.8 {
		t.Fatalf("unexpected wallet balance: %+v", options[0])
	}
	if options[0].WalletUsedTotalUSD != 0.2 || options[0].WalletUsedQuotaUSD != 0.2 {
		t.Fatalf("unexpected wallet amounts: %+v", options[0])
	}
	if !options[0].LowBalanceAlert || options[0].LowBalanceThresholdUSD != 0.9 {
		t.Fatalf("unexpected low balance state: %+v", options[0])
	}
}

func TestGetProfitBoardUpstreamAccountTrendUsesPeriodUsedUSD(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	account := ProfitBoardUpstreamAccount{
		Name:                   "趋势账户",
		AccountType:            ProfitBoardUpstreamAccountTypeNewAPI,
		BaseURL:                "https://remote.example.com",
		UserID:                 7,
		Enabled:                true,
		LowBalanceThresholdUSD: 0.75,
		CreatedAt:              1,
		UpdatedAt:              1,
	}
	encryptedToken, err := encryptProfitBoardRemoteSecret("remote-trend-token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	account.AccessTokenEncrypted = encryptedToken
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}

	config := account.remoteObserverConfig()
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	now := time.Now().Unix()
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now-2*24*60*60, 900, 100, nil)
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now-24*60*60, 850, 150, nil)
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now, 700, 300, nil)

	trend, err := GetProfitBoardUpstreamAccountTrend(account.Id, now-3*24*60*60, now+1, "day", 0)
	if err != nil {
		t.Fatalf("GetProfitBoardUpstreamAccountTrend: %v", err)
	}
	if len(trend.Points) != 2 {
		t.Fatalf("unexpected trend points: %+v", trend.Points)
	}
	if trend.Points[0].PeriodUsedUSD != 0.05 || trend.Points[1].PeriodUsedUSD != 0.15 {
		t.Fatalf("unexpected trend values: %+v", trend.Points)
	}
	if trend.Account.WalletBalanceUSD != 0.7 || trend.Account.PeriodUsedUSD != 0.2 {
		t.Fatalf("unexpected account summary: %+v", trend.Account)
	}
	if !trend.Account.LowBalanceAlert {
		t.Fatalf("expected low balance alert: %+v", trend.Account)
	}
}

func TestParseProfitBoardRemoteSubscriptionsSupportsIDField(t *testing.T) {
	raw := `[{"id":302,"plan_id":13,"amount_total":60000000,"amount_used":0,"status":"active"}]`
	subscriptions := parseProfitBoardRemoteSubscriptions(raw)
	if len(subscriptions) != 1 {
		t.Fatalf("unexpected subscriptions: %+v", subscriptions)
	}
	if subscriptions[0].SubscriptionID != 302 || subscriptions[0].ID != 302 {
		t.Fatalf("unexpected normalized subscription id: %+v", subscriptions[0])
	}
}

func TestCollectProfitBoardRemoteObserverAggregateUsesUsedQuotaDelta(t *testing.T) {
	setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	now := common.GetTimestamp()
	config := ProfitBoardRemoteObserverConfig{
		Enabled:     true,
		BaseURL:     "https://remote.example.com",
		UserID:      7,
		AccessToken: "remote-token",
	}
	selectionSignature := "channel:1"
	subscriptionA := []ProfitBoardRemoteSubscriptionSnapshot{{
		SubscriptionID: 1,
		AmountTotal:    400,
		AmountUsed:     50,
		NextResetTime:  now + 1800,
		EndTime:        now + 7200,
	}}
	subscriptionB := []ProfitBoardRemoteSubscriptionSnapshot{{
		SubscriptionID: 1,
		AmountTotal:    400,
		AmountUsed:     80,
		NextResetTime:  now + 1800,
		EndTime:        now + 7200,
	}}
	subscriptionC := []ProfitBoardRemoteSubscriptionSnapshot{{
		SubscriptionID: 1,
		AmountTotal:    400,
		AmountUsed:     20,
		LastResetTime:  now - 90,
		NextResetTime:  now + 3600,
		EndTime:        now + 10800,
	}}
	seedProfitBoardRemoteSnapshot(t, selectionSignature, "batch-1", config, now-240, 600, 100, subscriptionA)
	seedProfitBoardRemoteSnapshot(t, selectionSignature, "batch-1", config, now-120, 600, 160, subscriptionB)
	seedProfitBoardRemoteSnapshot(t, selectionSignature, "batch-1", config, now-60, 620, 180, subscriptionC)

	aggregate, err := collectProfitBoardRemoteObserverAggregate(
		selectionSignature,
		[]ProfitBoardBatchInfo{{Id: "batch-1", Name: "批次 1"}},
		[]ProfitBoardComboPricingConfig{{ComboId: "batch-1", RemoteObserver: config}},
		now-180,
		now,
		"day",
		0,
		false,
		true,
	)
	if err != nil {
		t.Fatalf("collectProfitBoardRemoteObserverAggregate: %v", err)
	}

	if aggregate.TotalCostUSD != 0.08 {
		t.Fatalf("unexpected remote observed cost: %v", aggregate.TotalCostUSD)
	}
	if aggregate.BatchCostUSD["batch-1"] != 0.08 {
		t.Fatalf("unexpected batch remote observed cost: %+v", aggregate.BatchCostUSD)
	}
	if len(aggregate.Timeseries) != 1 || aggregate.Timeseries[0].RemoteObservedCostUSD != 0.08 {
		t.Fatalf("unexpected remote timeseries: %+v", aggregate.Timeseries)
	}
	if len(aggregate.States) != 1 || aggregate.States[0].Status != profitBoardRemoteObserverStatusReady || !aggregate.States[0].BaselineReady {
		t.Fatalf("unexpected remote observer states: %+v", aggregate.States)
	}
}

func TestProfitBoardReportAndOverviewIncludeRemoteObservedCost(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := common.GetTimestamp()
	config := ProfitBoardRemoteObserverConfig{
		Enabled:     true,
		BaseURL:     "https://remote.example.com",
		UserID:      11,
		AccessToken: "remote-token",
	}
	selectionSignature := "channel:1"
	seedProfitBoardRemoteSnapshot(t, selectionSignature, "batch-1", config, now-240, 600, 100, []ProfitBoardRemoteSubscriptionSnapshot{{
		SubscriptionID: 1,
		AmountTotal:    400,
		AmountUsed:     50,
		NextResetTime:  now + 1800,
		EndTime:        now + 7200,
	}})
	seedProfitBoardRemoteSnapshot(t, selectionSignature, "batch-1", config, now-120, 600, 160, []ProfitBoardRemoteSubscriptionSnapshot{{
		SubscriptionID: 1,
		AmountTotal:    400,
		AmountUsed:     80,
		NextResetTime:  now + 1800,
		EndTime:        now + 7200,
	}})
	seedProfitBoardRemoteSnapshot(t, selectionSignature, "batch-1", config, now-60, 620, 180, []ProfitBoardRemoteSubscriptionSnapshot{{
		SubscriptionID: 1,
		AmountTotal:    400,
		AmountUsed:     20,
		LastResetTime:  now - 90,
		NextResetTime:  now + 3600,
		EndTime:        now + 10800,
	}})

	batches := []ProfitBoardBatch{{
		Id:         "batch-1",
		Name:       "批次 1",
		ScopeType:  ProfitBoardScopeChannel,
		ChannelIDs: []int{1},
	}}
	comboConfigs := []ProfitBoardComboPricingConfig{{
		ComboId:        "batch-1",
		RemoteObserver: config,
	}}

	overview, err := GenerateProfitBoardOverview(ProfitBoardConfigPayload{
		Batches:      batches,
		ComboConfigs: comboConfigs,
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardOverview: %v", err)
	}
	if overview.Summary.RemoteObservedCostUSD != 0.08 {
		t.Fatalf("unexpected overview remote observed cost: %v", overview.Summary.RemoteObservedCostUSD)
	}
	if len(overview.BatchSummaries) != 1 || overview.BatchSummaries[0].RemoteObservedCostUSD != 0.08 {
		t.Fatalf("unexpected overview batch summaries: %+v", overview.BatchSummaries)
	}
	if len(overview.RemoteObserverStates) != 1 {
		t.Fatalf("expected remote observer state in overview")
	}

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches:               batches,
		ComboConfigs:          comboConfigs,
		Upstream:              ProfitBoardTokenPricingConfig{CostSource: ProfitBoardCostSourceManualOnly},
		Site:                  ProfitBoardTokenPricingConfig{PricingMode: ProfitBoardSitePricingManual},
		StartTimestamp:        now - 180,
		EndTimestamp:          now,
		Granularity:           "day",
		CustomIntervalMinutes: 0,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}
	if report.Summary.RemoteObservedCostUSD != 0.08 {
		t.Fatalf("unexpected report remote observed cost: %v", report.Summary.RemoteObservedCostUSD)
	}
	if len(report.BatchSummaries) != 1 || report.BatchSummaries[0].RemoteObservedCostUSD != 0.08 {
		t.Fatalf("unexpected report batch summaries: %+v", report.BatchSummaries)
	}
	if len(report.RemoteObserverStates) != 1 {
		t.Fatalf("expected remote observer state in report")
	}
	if len(report.Timeseries) != 1 || report.Timeseries[0].RemoteObservedCostUSD != 0.08 {
		t.Fatalf("unexpected report remote timeseries: %+v", report.Timeseries)
	}
}

func TestGenerateProfitBoardReportSupportsComboWalletObserverAndManualMix(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	channels := []Channel{
		{Id: 1, Name: "alpha", Tag: common.GetPointer("vip"), Status: common.ChannelStatusEnabled},
		{Id: 2, Name: "beta", Tag: common.GetPointer("shared"), Status: common.ChannelStatusEnabled},
	}
	for _, channel := range channels {
		if err := db.Create(&channel).Error; err != nil {
			t.Fatalf("seed channel: %v", err)
		}
	}

	now := common.GetTimestamp()
	account := ProfitBoardUpstreamAccount{
		Name:        "钱包账户",
		AccountType: ProfitBoardUpstreamAccountTypeNewAPI,
		BaseURL:     "https://remote.example.com",
		UserID:      9,
		Enabled:     true,
		CreatedAt:   now - 300,
		UpdatedAt:   now - 300,
	}
	encryptedToken, err := encryptProfitBoardRemoteSecret("wallet-token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	account.AccessTokenEncrypted = encryptedToken
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create wallet account: %v", err)
	}

	config := account.remoteObserverConfig()
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now-180, 900, 100, nil)
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now-60, 800, 200, nil)

	logs := []Log{
		{Id: 1, Type: LogTypeConsume, CreatedAt: now - 120, ChannelId: 1, ModelName: "gpt-4.1", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 2, Type: LogTypeConsume, CreatedAt: now - 110, ChannelId: 1, ModelName: "gpt-4.1", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 3, Type: LogTypeConsume, CreatedAt: now - 100, ChannelId: 2, ModelName: "gpt-4.1", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
	}
	for _, logRow := range logs {
		if err := db.Create(&logRow).Error; err != nil {
			t.Fatalf("seed log: %v", err)
		}
	}

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches: []ProfitBoardBatch{
			{Id: "wallet-batch", Name: "钱包组合", ScopeType: ProfitBoardScopeChannel, ChannelIDs: []int{1}},
			{Id: "manual-batch", Name: "手动组合", ScopeType: ProfitBoardScopeChannel, ChannelIDs: []int{2}},
		},
		ComboConfigs: []ProfitBoardComboPricingConfig{
			{
				ComboId:           "wallet-batch",
				UpstreamMode:      ProfitBoardUpstreamModeWallet,
				UpstreamAccountID: account.Id,
				SiteRules: []ProfitBoardModelPricingRule{{
					IsDefault:  true,
					InputPrice: 5,
				}},
			},
			{
				ComboId:      "manual-batch",
				UpstreamMode: ProfitBoardUpstreamModeManual,
				SiteRules: []ProfitBoardModelPricingRule{{
					IsDefault:  true,
					InputPrice: 5,
				}},
				UpstreamRules: []ProfitBoardModelPricingRule{{
					IsDefault:  true,
					InputPrice: 2,
				}},
			},
		},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
		StartTimestamp:        now - 200,
		EndTimestamp:          now,
		Granularity:           "day",
		CustomIntervalMinutes: 0,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if report.Summary.UpstreamCostUSD != 0.102 {
		t.Fatalf("unexpected mixed upstream cost: %+v", report.Summary)
	}
	if report.Summary.ConfiguredProfitUSD != -0.087 {
		t.Fatalf("unexpected mixed configured profit: %+v", report.Summary)
	}
	if report.Summary.ConfiguredProfitCoverageRate != 1 {
		t.Fatalf("unexpected mixed coverage: %+v", report.Summary)
	}

	batchCosts := map[string]float64{}
	for _, summary := range report.BatchSummaries {
		batchCosts[summary.BatchId] = summary.UpstreamCostUSD
	}
	if batchCosts["wallet-batch"] != 0.1 {
		t.Fatalf("unexpected wallet batch cost: %+v", report.BatchSummaries)
	}
	if batchCosts["manual-batch"] != 0.002 {
		t.Fatalf("unexpected manual batch cost: %+v", report.BatchSummaries)
	}
}

func TestGenerateProfitBoardOverviewIncludesWalletObserverAccountCost(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := common.GetTimestamp()
	account := ProfitBoardUpstreamAccount{
		Name:        "钱包账户",
		AccountType: ProfitBoardUpstreamAccountTypeNewAPI,
		BaseURL:     "https://remote.example.com",
		UserID:      9,
		Enabled:     true,
		CreatedAt:   now - 300,
		UpdatedAt:   now - 300,
	}
	encryptedToken, err := encryptProfitBoardRemoteSecret("wallet-token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	account.AccessTokenEncrypted = encryptedToken
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create wallet account: %v", err)
	}

	config := account.remoteObserverConfig()
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now-180, 900, 100, nil)
	seedProfitBoardRemoteSnapshot(t, signature, profitBoardUpstreamAccountSnapshotComboID, config, now-60, 800, 200, nil)

	logs := []Log{
		{Id: 1, Type: LogTypeConsume, CreatedAt: now - 120, ChannelId: 1, ModelName: "gpt-4.1", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 2, Type: LogTypeConsume, CreatedAt: now - 110, ChannelId: 1, ModelName: "gpt-4.1", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
	}
	for _, logRow := range logs {
		if err := db.Create(&logRow).Error; err != nil {
			t.Fatalf("seed log: %v", err)
		}
	}

	overview, err := GenerateProfitBoardOverview(ProfitBoardConfigPayload{
		Batches: []ProfitBoardBatch{{
			Id:         "wallet-batch",
			Name:       "钱包组合",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
			CreatedAt:  now - 200,
		}},
		ComboConfigs: []ProfitBoardComboPricingConfig{{
			ComboId:           "wallet-batch",
			UpstreamMode:      ProfitBoardUpstreamModeWallet,
			UpstreamAccountID: account.Id,
			SiteRules: []ProfitBoardModelPricingRule{{
				IsDefault:  true,
				InputPrice: 5,
			}},
		}},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardOverview: %v", err)
	}

	if overview.Summary.UpstreamCostUSD != 0.1 {
		t.Fatalf("unexpected overview upstream cost: %+v", overview.Summary)
	}
	if overview.Summary.ConfiguredProfitUSD != -0.09 {
		t.Fatalf("unexpected overview configured profit: %+v", overview.Summary)
	}
	if overview.Summary.ActualProfitUSD != -0.09 {
		t.Fatalf("unexpected overview actual profit: %+v", overview.Summary)
	}
	if len(overview.BatchSummaries) != 1 || overview.BatchSummaries[0].UpstreamCostUSD != 0.1 {
		t.Fatalf("unexpected overview batch summaries: %+v", overview.BatchSummaries)
	}
}

func TestDeleteProfitBoardUpstreamAccountRejectsPersistedReferences(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := common.GetTimestamp()
	account := ProfitBoardUpstreamAccount{
		Name:        "钱包账户",
		AccountType: ProfitBoardUpstreamAccountTypeNewAPI,
		BaseURL:     "https://remote.example.com",
		UserID:      9,
		Enabled:     true,
		CreatedAt:   now - 300,
		UpdatedAt:   now - 300,
	}
	encryptedToken, err := encryptProfitBoardRemoteSecret("wallet-token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	account.AccessTokenEncrypted = encryptedToken
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create wallet account: %v", err)
	}

	_, _, err = SaveProfitBoardConfig(ProfitBoardConfigPayload{
		Batches: []ProfitBoardBatch{{
			Id:         "wallet-batch",
			Name:       "钱包组合",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		}},
		ComboConfigs: []ProfitBoardComboPricingConfig{{
			ComboId:           "wallet-batch",
			UpstreamMode:      ProfitBoardUpstreamModeWallet,
			UpstreamAccountID: account.Id,
		}},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
	})
	if err != nil {
		t.Fatalf("SaveProfitBoardConfig: %v", err)
	}

	err = DeleteProfitBoardUpstreamAccount(account.Id)
	if !errors.Is(err, ErrProfitBoardAccountInUse) {
		t.Fatalf("expected ErrProfitBoardAccountInUse, got %v", err)
	}

	loaded := ProfitBoardUpstreamAccount{}
	if err := db.First(&loaded, account.Id).Error; err != nil {
		t.Fatalf("account should still exist after protected delete: %v", err)
	}
}

func TestProfitBoardReportAndOverviewRespectBatchCreatedAt(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := common.GetTimestamp()
	logs := []Log{
		{Id: 1, Type: LogTypeConsume, CreatedAt: now - 300, ChannelId: 1, ModelName: "gpt-4.1", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 2, Type: LogTypeConsume, CreatedAt: now - 100, ChannelId: 1, ModelName: "gpt-4.1", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
	}
	for _, logRow := range logs {
		if err := db.Create(&logRow).Error; err != nil {
			t.Fatalf("seed log: %v", err)
		}
	}

	batches := []ProfitBoardBatch{{
		Id:         "batch-1",
		Name:       "组合 1",
		ScopeType:  ProfitBoardScopeChannel,
		ChannelIDs: []int{1},
		CreatedAt:  now - 150,
	}}

	query := ProfitBoardQuery{
		Batches: batches,
		ComboConfigs: []ProfitBoardComboPricingConfig{{
			ComboId: "batch-1",
			SiteRules: []ProfitBoardModelPricingRule{{
				IsDefault:  true,
				InputPrice: 5,
			}},
			UpstreamRules: []ProfitBoardModelPricingRule{{
				IsDefault:  true,
				InputPrice: 2,
			}},
		}},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
		StartTimestamp: now - 400,
		EndTimestamp:   now,
		Granularity:    "day",
	}

	report, err := GenerateProfitBoardReport(query)
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}
	if report.Summary.RequestCount != 1 {
		t.Fatalf("unexpected request count: %+v", report.Summary)
	}
	if report.Summary.ConfiguredSiteRevenueUSD != 0.005 || report.Summary.UpstreamCostUSD != 0.002 || report.Summary.ConfiguredProfitUSD != 0.003 {
		t.Fatalf("unexpected report summary: %+v", report.Summary)
	}

	overview, err := GenerateProfitBoardOverview(ProfitBoardConfigPayload{
		Batches:      batches,
		ComboConfigs: query.ComboConfigs,
		Upstream:     query.Upstream,
		Site:         query.Site,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardOverview: %v", err)
	}
	if overview.Summary.RequestCount != 1 {
		t.Fatalf("unexpected overview request count: %+v", overview.Summary)
	}
	if overview.Summary.ConfiguredSiteRevenueUSD != 0.005 || overview.Summary.UpstreamCostUSD != 0.002 || overview.Summary.ConfiguredProfitUSD != 0.003 {
		t.Fatalf("unexpected overview summary: %+v", overview.Summary)
	}
}

func TestProfitBoardManualOnlyEmptyUpstreamRulesUseOnlyFixedTotal(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := common.GetTimestamp()
	logRow := Log{
		Id:               1,
		Type:             LogTypeConsume,
		CreatedAt:        now - 60,
		ChannelId:        1,
		ModelName:        "gpt-4.1",
		PromptTokens:     1000,
		CompletionTokens: 0,
		Quota:            5,
	}
	if err := db.Create(&logRow).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	batches := []ProfitBoardBatch{{
		Id:         "batch-1",
		Name:       "组合 1",
		ScopeType:  ProfitBoardScopeChannel,
		ChannelIDs: []int{1},
		CreatedAt:  now - 120,
	}}
	comboConfigs := []ProfitBoardComboPricingConfig{{
		ComboId:                  "batch-1",
		SiteRules:                []ProfitBoardModelPricingRule{{IsDefault: true, InputPrice: 5}},
		UpstreamRules:            []ProfitBoardModelPricingRule{},
		UpstreamFixedTotalAmount: 0.2,
	}}
	upstreamConfig := ProfitBoardTokenPricingConfig{
		CostSource: ProfitBoardCostSourceManualOnly,
		InputPrice: 9,
	}
	siteConfig := ProfitBoardTokenPricingConfig{
		PricingMode: ProfitBoardSitePricingManual,
	}

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches:        batches,
		ComboConfigs:   comboConfigs,
		Upstream:       upstreamConfig,
		Site:           siteConfig,
		StartTimestamp: now - 300,
		EndTimestamp:   now,
		Granularity:    "day",
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}
	if report.Summary.ConfiguredSiteRevenueUSD != 0.005 || report.Summary.UpstreamCostUSD != 0.2 || report.Summary.ConfiguredProfitUSD != -0.195 {
		t.Fatalf("unexpected report summary: %+v", report.Summary)
	}
	if report.Summary.ConfiguredProfitCoverageRate != 1 {
		t.Fatalf("expected full configured profit coverage, got %+v", report.Summary)
	}
	if warningText := strings.Join(report.Warnings, "\n"); strings.Contains(warningText, "上游返回费用") {
		t.Fatalf("unexpected report warnings: %v", report.Warnings)
	}

	overview, err := GenerateProfitBoardOverview(ProfitBoardConfigPayload{
		Batches:      batches,
		ComboConfigs: comboConfigs,
		Upstream:     upstreamConfig,
		Site:         siteConfig,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardOverview: %v", err)
	}
	if overview.Summary.ConfiguredSiteRevenueUSD != 0.005 || overview.Summary.UpstreamCostUSD != 0.2 || overview.Summary.ConfiguredProfitUSD != -0.195 {
		t.Fatalf("unexpected overview summary: %+v", overview.Summary)
	}
	if overview.Summary.ConfiguredProfitCoverageRate != 1 {
		t.Fatalf("expected full overview configured profit coverage, got %+v", overview.Summary)
	}
	if warningText := strings.Join(overview.Warnings, "\n"); strings.Contains(warningText, "上游返回费用") {
		t.Fatalf("unexpected overview warnings: %v", overview.Warnings)
	}
}

func TestProfitBoardFixedTotalsApplyOnceFromBatchCreatedAt(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := common.GetTimestamp()
	createdAt := now - 120
	batches := []ProfitBoardBatch{{
		Id:         "batch-1",
		Name:       "组合 1",
		ScopeType:  ProfitBoardScopeChannel,
		ChannelIDs: []int{1},
		CreatedAt:  createdAt,
	}}
	comboConfigs := []ProfitBoardComboPricingConfig{{
		ComboId:                  "batch-1",
		SiteFixedTotalAmount:     1.2,
		UpstreamFixedTotalAmount: 0.2,
	}}

	overview, err := GenerateProfitBoardOverview(ProfitBoardConfigPayload{
		Batches:      batches,
		ComboConfigs: comboConfigs,
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardOverview: %v", err)
	}
	if overview.Summary.ConfiguredSiteRevenueUSD != 1.2 || overview.Summary.UpstreamCostUSD != 0.2 || overview.Summary.ConfiguredProfitUSD != 1 {
		t.Fatalf("unexpected overview fixed totals: %+v", overview.Summary)
	}

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches:      batches,
		ComboConfigs: comboConfigs,
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
		StartTimestamp: createdAt - 60,
		EndTimestamp:   now,
		Granularity:    "day",
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}
	if report.Summary.ConfiguredSiteRevenueUSD != 1.2 || report.Summary.UpstreamCostUSD != 0.2 || report.Summary.ConfiguredProfitUSD != 1 {
		t.Fatalf("unexpected report fixed totals: %+v", report.Summary)
	}
	if len(report.Timeseries) != 1 || report.Timeseries[0].ConfiguredSiteRevenueUSD != 1.2 || report.Timeseries[0].UpstreamCostUSD != 0.2 {
		t.Fatalf("unexpected report fixed total timeseries: %+v", report.Timeseries)
	}
}

func TestQueryProfitBoardDetailsSupportsTagFilter(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	channel := Channel{Id: 1, Name: "alpha", Tag: common.GetPointer("vip"), Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := common.GetTimestamp()
	logRow := Log{
		Id:               1,
		Type:             LogTypeConsume,
		CreatedAt:        now - 30,
		ChannelId:        1,
		ModelName:        "gpt-4.1",
		PromptTokens:     1000,
		CompletionTokens: 0,
		Quota:            5,
	}
	if err := db.Create(&logRow).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	page, err := QueryProfitBoardDetails(ProfitBoardDetailQuery{
		ProfitBoardQuery: ProfitBoardQuery{
			Batches: []ProfitBoardBatch{{
				Id:         "batch-1",
				Name:       "批次 1",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{1},
			}},
			ComboConfigs: []ProfitBoardComboPricingConfig{{
				ComboId: "batch-1",
				SiteRules: []ProfitBoardModelPricingRule{{
					IsDefault:  true,
					InputPrice: 5,
				}},
				UpstreamRules: []ProfitBoardModelPricingRule{{
					IsDefault:  true,
					InputPrice: 2,
				}},
			}},
			Upstream:       ProfitBoardTokenPricingConfig{CostSource: ProfitBoardCostSourceManualOnly},
			Site:           ProfitBoardTokenPricingConfig{PricingMode: ProfitBoardSitePricingManual},
			StartTimestamp: now - 60,
			EndTimestamp:   now,
			Granularity:    "day",
		},
		DetailFilter: ProfitBoardDetailFilter{
			Type:  "tag",
			Value: "vip",
		},
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("QueryProfitBoardDetails: %v", err)
	}
	if page.Total != 1 || len(page.Rows) != 1 {
		t.Fatalf("unexpected tag filtered page: %+v", page)
	}
}
