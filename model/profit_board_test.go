package model

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func decodeProfitBoardJSONMap(t *testing.T, value any) map[string]any {
	t.Helper()

	raw, err := common.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}

	decoded := map[string]any{}
	if err := common.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode value: %v", err)
	}
	return decoded
}

func requireProfitBoardFloat(t *testing.T, value any, expected float64, field string) {
	t.Helper()

	actual, ok := value.(float64)
	if !ok {
		t.Fatalf("field %s is not a number: %#v", field, value)
	}
	if math.Abs(actual-expected) > 0.000001 {
		t.Fatalf("unexpected %s: got %.6f want %.6f", field, actual, expected)
	}
}

func findProfitBoardWarningItem(items []ProfitBoardWarningItem, code string) *ProfitBoardWarningItem {
	for i := range items {
		if items[i].Code == code {
			return &items[i]
		}
	}
	return nil
}

func findProfitBoardWarningDetail(items []ProfitBoardWarningDetail, scopeType, scopeLabel, modelName string) *ProfitBoardWarningDetail {
	for i := range items {
		if items[i].ScopeType == scopeType && items[i].ScopeLabel == scopeLabel && items[i].ModelName == modelName {
			return &items[i]
		}
	}
	return nil
}

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

func TestGetProfitBoardUserOptionsFiltersByRoleGroupAndKeyword(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	users := []User{
		{Id: 1, Username: "admin-user", DisplayName: "Admin User", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AffCode: "pb-user-option-admin"},
		{Id: 2, Username: "root-user", DisplayName: "Root User", Role: common.RoleRootUser, Status: common.UserStatusEnabled, AffCode: "pb-user-option-root"},
		{Id: 3, Username: "common-alpha", DisplayName: "Alpha Nick", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "pb-user-option-common-a"},
		{Id: 4, Username: "common-beta", DisplayName: "Beta Nick", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "pb-user-option-common-b"},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}

	adminUsers, adminTotal, err := GetProfitBoardUserOptions(ProfitBoardUserOptionQuery{
		RoleGroup: "admin",
		Limit:     100,
	})
	if err != nil {
		t.Fatalf("GetProfitBoardUserOptions admin: %v", err)
	}
	if adminTotal != 2 {
		t.Fatalf("expected 2 admin users, got total=%d data=%+v", adminTotal, adminUsers)
	}
	if len(adminUsers) != 2 || adminUsers[0].Id != 2 || adminUsers[1].Id != 1 {
		t.Fatalf("unexpected admin users order: %+v", adminUsers)
	}

	commonUsers, commonTotal, err := GetProfitBoardUserOptions(ProfitBoardUserOptionQuery{
		RoleGroup: "common",
		Keyword:   "Nick",
		Limit:     100,
	})
	if err != nil {
		t.Fatalf("GetProfitBoardUserOptions common: %v", err)
	}
	if commonTotal != 2 {
		t.Fatalf("expected 2 common users, got total=%d data=%+v", commonTotal, commonUsers)
	}
	if len(commonUsers) != 2 || commonUsers[0].Id != 4 || commonUsers[1].Id != 3 {
		t.Fatalf("unexpected common users order: %+v", commonUsers)
	}

	byIDUsers, byIDTotal, err := GetProfitBoardUserOptions(ProfitBoardUserOptionQuery{
		RoleGroup: "all",
		Keyword:   "3",
		Limit:     100,
	})
	if err != nil {
		t.Fatalf("GetProfitBoardUserOptions by id: %v", err)
	}
	if byIDTotal != 1 || len(byIDUsers) != 1 || byIDUsers[0].Id != 3 {
		t.Fatalf("expected exact id match, got total=%d data=%+v", byIDTotal, byIDUsers)
	}
}

func TestGetProfitBoardUserOptionsSupportsIDsAndSkipsDeletedUsers(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	active := User{Id: 10, Username: "keep-me", DisplayName: "Keep Me", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "pb-user-option-keep"}
	deleted := User{Id: 11, Username: "delete-me", DisplayName: "Delete Me", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "pb-user-option-delete"}
	if err := db.Create(&active).Error; err != nil {
		t.Fatalf("seed active user: %v", err)
	}
	if err := db.Create(&deleted).Error; err != nil {
		t.Fatalf("seed deleted user: %v", err)
	}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	users, total, err := GetProfitBoardUserOptions(ProfitBoardUserOptionQuery{
		IDs:   []int{11, 10},
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("GetProfitBoardUserOptions ids: %v", err)
	}
	if total != 1 || len(users) != 1 || users[0].Id != 10 {
		t.Fatalf("expected only active user returned, got total=%d data=%+v", total, users)
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

func TestBuildProfitBoardReportCacheKeyIncludesPricingAndGranularity(t *testing.T) {
	base := ProfitBoardQuery{
		Batches: []ProfitBoardBatch{{
			Id:         "batch-1",
			Name:       "批次 1",
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{1},
		}},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceReturnedOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
			InputPrice:  1,
		},
		StartTimestamp:        100,
		EndTimestamp:          200,
		Granularity:           "custom",
		CustomIntervalMinutes: 15,
		Sections:              []string{"timeseries", "warning_items"},
	}

	keyA := buildProfitBoardReportCacheKey(base)
	if keyA == "" {
		t.Fatal("expected non-empty cache key")
	}

	withDifferentSite := base
	withDifferentSite.Site.InputPrice = 2
	keyB := buildProfitBoardReportCacheKey(withDifferentSite)
	if keyA == keyB {
		t.Fatalf("expected cache key to change when site pricing changes: %q", keyA)
	}

	withDifferentGranularity := base
	withDifferentGranularity.CustomIntervalMinutes = 30
	keyC := buildProfitBoardReportCacheKey(withDifferentGranularity)
	if keyA == keyC {
		t.Fatalf("expected cache key to change when custom interval changes: %q", keyA)
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
	if options[0].PeriodUsedUSD != 0 || options[0].ObservedCostUSD != 0 {
		t.Fatalf("expected lightweight account options to skip observed aggregate: %+v", options[0])
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

func TestGetLatestProfitBoardConfigPreservesComboExchangeRates(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	batches := []ProfitBoardBatch{{
		Id:         "batch-1",
		Name:       "组合 1",
		ScopeType:  ProfitBoardScopeChannel,
		ChannelIDs: []int{1},
		CreatedAt:  common.GetTimestamp() - 60,
	}}
	selectionBytes, err := common.Marshal(batches)
	if err != nil {
		t.Fatalf("marshal batches: %v", err)
	}

	record := ProfitBoardConfig{
		SelectionType:      ProfitBoardScopeBatch,
		SelectionSignature: "profit-board:test:exchange-rates",
		SelectionValues:    string(selectionBytes),
		UpstreamConfig:     `{"cost_source":"manual_only"}`,
		SiteConfig: `{
			"legacy_site":{"pricing_mode":"manual"},
			"combo_configs":[
				{
					"combo_id":"batch-1",
					"site_exchange_rate":0.89,
					"upstream_exchange_rate":1.5
				}
			]
		}`,
		CreatedAt: common.GetTimestamp(),
		UpdatedAt: common.GetTimestamp(),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	payload, _, err := GetLatestProfitBoardConfig()
	if err != nil {
		t.Fatalf("GetLatestProfitBoardConfig: %v", err)
	}
	if payload == nil {
		t.Fatal("expected payload, got nil")
	}

	decoded := decodeProfitBoardJSONMap(t, payload)
	comboConfigs, ok := decoded["combo_configs"].([]any)
	if !ok || len(comboConfigs) != 1 {
		t.Fatalf("unexpected combo_configs: %#v", decoded["combo_configs"])
	}
	combo, ok := comboConfigs[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected combo config: %#v", comboConfigs[0])
	}
	requireProfitBoardFloat(t, combo["site_exchange_rate"], 0.89, "site_exchange_rate")
	requireProfitBoardFloat(t, combo["upstream_exchange_rate"], 1.5, "upstream_exchange_rate")
}

func TestProfitBoardReportIncludesConfiguredCNYMetrics(t *testing.T) {
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

	batches := []ProfitBoardBatch{{
		Id:         "batch-1",
		Name:       "组合 1",
		ScopeType:  ProfitBoardScopeChannel,
		ChannelIDs: []int{1},
		CreatedAt:  now - 120,
	}}
	selectionBytes, err := common.Marshal(batches)
	if err != nil {
		t.Fatalf("marshal batches: %v", err)
	}

	record := ProfitBoardConfig{
		SelectionType:      ProfitBoardScopeBatch,
		SelectionSignature: "profit-board:test:cny-report",
		SelectionValues:    string(selectionBytes),
		UpstreamConfig:     `{"cost_source":"manual_only"}`,
		SiteConfig: `{
			"legacy_site":{"pricing_mode":"manual"},
			"combo_configs":[
				{
					"combo_id":"batch-1",
					"site_rules":[{"is_default":true,"input_price":5}],
					"upstream_rules":[{"is_default":true,"input_price":2}],
					"site_exchange_rate":0.89,
					"upstream_exchange_rate":1.5
				}
			]
		}`,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	payload, _, err := GetLatestProfitBoardConfig()
	if err != nil {
		t.Fatalf("GetLatestProfitBoardConfig: %v", err)
	}
	if payload == nil {
		t.Fatal("expected payload, got nil")
	}

	overview, err := GenerateProfitBoardOverview(*payload)
	if err != nil {
		t.Fatalf("GenerateProfitBoardOverview: %v", err)
	}
	overviewJSON := decodeProfitBoardJSONMap(t, overview)
	overviewSummary, ok := overviewJSON["summary"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected overview summary: %#v", overviewJSON["summary"])
	}
	requireProfitBoardFloat(t, overviewSummary["configured_site_revenue_cny"], 0.00445, "overview configured_site_revenue_cny")
	requireProfitBoardFloat(t, overviewSummary["upstream_cost_cny"], 0.003, "overview upstream_cost_cny")
	requireProfitBoardFloat(t, overviewSummary["configured_profit_cny"], 0.00145, "overview configured_profit_cny")

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
		Batches:         payload.Batches,
		SharedSite:      payload.SharedSite,
		ComboConfigs:    payload.ComboConfigs,
		ExcludedUserIDs: payload.ExcludedUserIDs,
		Upstream:        payload.Upstream,
		Site:            payload.Site,
		StartTimestamp:  now - 300,
		EndTimestamp:    now,
		Granularity:     "day",
		IncludeDetails:  true,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	reportJSON := decodeProfitBoardJSONMap(t, report)
	reportSummary, ok := reportJSON["summary"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected report summary: %#v", reportJSON["summary"])
	}
	requireProfitBoardFloat(t, reportSummary["configured_site_revenue_cny"], 0.00445, "report configured_site_revenue_cny")
	requireProfitBoardFloat(t, reportSummary["upstream_cost_cny"], 0.003, "report upstream_cost_cny")
	requireProfitBoardFloat(t, reportSummary["configured_profit_cny"], 0.00145, "report configured_profit_cny")

	timeseries, ok := reportJSON["timeseries"].([]any)
	if !ok || len(timeseries) != 1 {
		t.Fatalf("unexpected timeseries: %#v", reportJSON["timeseries"])
	}
	point, ok := timeseries[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected timeseries point: %#v", timeseries[0])
	}
	requireProfitBoardFloat(t, point["configured_site_revenue_cny"], 0.00445, "timeseries configured_site_revenue_cny")
	requireProfitBoardFloat(t, point["upstream_cost_cny"], 0.003, "timeseries upstream_cost_cny")
	requireProfitBoardFloat(t, point["configured_profit_cny"], 0.00145, "timeseries configured_profit_cny")

	channelBreakdown, ok := reportJSON["channel_breakdown"].([]any)
	if !ok || len(channelBreakdown) != 1 {
		t.Fatalf("unexpected channel_breakdown: %#v", reportJSON["channel_breakdown"])
	}
	channelRow, ok := channelBreakdown[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected channel row: %#v", channelBreakdown[0])
	}
	requireProfitBoardFloat(t, channelRow["configured_site_revenue_cny"], 0.00445, "channel configured_site_revenue_cny")
	requireProfitBoardFloat(t, channelRow["upstream_cost_cny"], 0.003, "channel upstream_cost_cny")
	requireProfitBoardFloat(t, channelRow["configured_profit_cny"], 0.00145, "channel configured_profit_cny")

	modelBreakdown, ok := reportJSON["model_breakdown"].([]any)
	if !ok || len(modelBreakdown) != 1 {
		t.Fatalf("unexpected model_breakdown: %#v", reportJSON["model_breakdown"])
	}
	modelRow, ok := modelBreakdown[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected model row: %#v", modelBreakdown[0])
	}
	requireProfitBoardFloat(t, modelRow["configured_site_revenue_cny"], 0.00445, "model configured_site_revenue_cny")
	requireProfitBoardFloat(t, modelRow["upstream_cost_cny"], 0.003, "model upstream_cost_cny")
	requireProfitBoardFloat(t, modelRow["configured_profit_cny"], 0.00145, "model configured_profit_cny")

	detailPage, err := QueryProfitBoardDetails(ProfitBoardDetailQuery{
		ProfitBoardQuery: ProfitBoardQuery{
			Batches:         payload.Batches,
			SharedSite:      payload.SharedSite,
			ComboConfigs:    payload.ComboConfigs,
			ExcludedUserIDs: payload.ExcludedUserIDs,
			Upstream:        payload.Upstream,
			Site:            payload.Site,
			StartTimestamp:  now - 300,
			EndTimestamp:    now,
			Granularity:     "day",
		},
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("QueryProfitBoardDetails: %v", err)
	}
	detailJSON := decodeProfitBoardJSONMap(t, detailPage)
	detailRows, ok := detailJSON["rows"].([]any)
	if !ok || len(detailRows) != 1 {
		t.Fatalf("unexpected detail rows: %#v", detailJSON["rows"])
	}
	detailRow, ok := detailRows[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected detail row: %#v", detailRows[0])
	}
	requireProfitBoardFloat(t, detailRow["configured_site_revenue_cny"], 0.00445, "detail configured_site_revenue_cny")
	requireProfitBoardFloat(t, detailRow["upstream_cost_cny"], 0.003, "detail upstream_cost_cny")
	requireProfitBoardFloat(t, detailRow["configured_profit_cny"], 0.00145, "detail configured_profit_cny")
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

func TestProfitBoardWarningsExposeStructuredDetails(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	originQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})

	channels := []Channel{
		{Id: 1, Name: "alpha", Tag: common.GetPointer("vip"), Status: common.ChannelStatusEnabled},
		{Id: 2, Name: "beta", Tag: common.GetPointer("vip"), Status: common.ChannelStatusEnabled},
		{Id: 3, Name: "gamma", Status: common.ChannelStatusEnabled},
		{Id: 4, Name: "delta", Tag: common.GetPointer("billing"), Status: common.ChannelStatusEnabled},
		{Id: 5, Name: "epsilon", Status: common.ChannelStatusEnabled},
		{Id: 6, Name: "zeta", Status: common.ChannelStatusEnabled},
	}
	for _, channel := range channels {
		if err := db.Create(&channel).Error; err != nil {
			t.Fatalf("seed channel: %v", err)
		}
	}

	now := common.GetTimestamp()
	logs := []Log{
		{Id: 1, Type: LogTypeConsume, CreatedAt: now - 90, ChannelId: 1, ModelName: "missing-site-model", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 2, Type: LogTypeConsume, CreatedAt: now - 80, ChannelId: 2, ModelName: "missing-site-model", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 3, Type: LogTypeConsume, CreatedAt: now - 70, ChannelId: 3, ModelName: "missing-site-model", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 4, Type: LogTypeConsume, CreatedAt: now - 60, ChannelId: 4, ModelName: "known-site-model", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 5, Type: LogTypeConsume, CreatedAt: now - 50, ChannelId: 4, ModelName: "known-site-model", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 6, Type: LogTypeConsume, CreatedAt: now - 40, ChannelId: 5, ModelName: "known-site-model", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
		{Id: 31, Type: LogTypeConsume, CreatedAt: now - 39, ChannelId: 6, ModelName: "manual-only-missing", PromptTokens: 1000, CompletionTokens: 0, Quota: 5},
	}
	for _, logRow := range logs {
		if err := db.Create(&logRow).Error; err != nil {
			t.Fatalf("seed log: %v", err)
		}
	}

	query := ProfitBoardQuery{
		Batches: []ProfitBoardBatch{
			{
				Id:         "site-missing-batch",
				Name:       "站内缺失",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{1, 2, 3},
				CreatedAt:  now - 120,
			},
			{
				Id:         "upstream-missing-batch",
				Name:       "上游缺失",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{4, 5},
				CreatedAt:  now - 120,
			},
			{
				Id:         "manual-site-batch",
				Name:       "手动缺失",
				ScopeType:  ProfitBoardScopeChannel,
				ChannelIDs: []int{6},
				CreatedAt:  now - 120,
			},
		},
		ComboConfigs: []ProfitBoardComboPricingConfig{
			{
				ComboId:    "site-missing-batch",
				SiteMode:   ProfitBoardComboSiteModeSharedSite,
				CostSource: ProfitBoardCostSourceManualOnly,
				SiteRules: []ProfitBoardModelPricingRule{{
					ModelName:  "known-site-fallback",
					InputPrice: 5,
				}},
				UpstreamRules: []ProfitBoardModelPricingRule{{
					IsDefault:  true,
					InputPrice: 2,
				}},
			},
			{
				ComboId:    "upstream-missing-batch",
				CostSource: ProfitBoardCostSourceReturnedOnly,
				SiteRules: []ProfitBoardModelPricingRule{{
					IsDefault:  true,
					InputPrice: 5,
				}},
			},
			{
				ComboId:    "manual-site-batch",
				CostSource: ProfitBoardCostSourceManualOnly,
			},
		},
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
		StartTimestamp: now - 120,
		EndTimestamp:   now,
		Granularity:    "day",
	}

	report, err := GenerateProfitBoardReport(query)
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if warningText := strings.Join(report.Warnings, "\n"); !strings.Contains(warningText, "部分日志没有命中本站模型定价") || !strings.Contains(warningText, "部分日志未命中上游成本配置") {
		t.Fatalf("expected warning texts preserved, got %v", report.Warnings)
	}

	siteWarning := findProfitBoardWarningItem(report.WarningItems, "missing_site_pricing")
	if siteWarning == nil {
		t.Fatalf("expected missing_site_pricing warning item, got %+v", report.WarningItems)
	}
	if siteWarning.TotalCount != 4 || len(siteWarning.Details) != 3 {
		t.Fatalf("unexpected site warning item: %+v", siteWarning)
	}
	siteVipDetail := findProfitBoardWarningDetail(siteWarning.Details, "tag", "vip", "missing-site-model")
	if siteVipDetail == nil || siteVipDetail.Count != 2 {
		t.Fatalf("unexpected site vip detail: %+v", siteWarning.Details)
	}
	if siteVipDetail.ReasonCode == "" || siteVipDetail.ReasonLabel == "" {
		t.Fatalf("expected site vip detail reason fields, got %+v", siteVipDetail)
	}
	gammaSharedSiteDetail := findProfitBoardWarningDetail(siteWarning.Details, "channel", "gamma", "missing-site-model")
	if gammaSharedSiteDetail == nil || gammaSharedSiteDetail.Count != 1 {
		t.Fatalf("unexpected gamma shared-site detail: %+v", siteWarning.Details)
	}
	if gammaSharedSiteDetail.ReasonCode == "" || gammaSharedSiteDetail.ReasonLabel == "" {
		t.Fatalf("expected gamma shared-site reason fields, got %+v", gammaSharedSiteDetail)
	}
	manualOnlyDetail := findProfitBoardWarningDetail(siteWarning.Details, "channel", "zeta", "manual-only-missing")
	if manualOnlyDetail == nil || manualOnlyDetail.Count != 1 {
		t.Fatalf("unexpected manual-only detail: %+v", siteWarning.Details)
	}
	if manualOnlyDetail.ReasonCode == "" || manualOnlyDetail.ReasonLabel == "" {
		t.Fatalf("expected manual-only reason fields, got %+v", manualOnlyDetail)
	}
	if gammaSharedSiteDetail.ReasonCode == manualOnlyDetail.ReasonCode {
		t.Fatalf("expected different site missing reasons, got shared=%s manual=%s", gammaSharedSiteDetail.ReasonCode, manualOnlyDetail.ReasonCode)
	}

	upstreamWarning := findProfitBoardWarningItem(report.WarningItems, "missing_upstream_cost")
	if upstreamWarning == nil {
		t.Fatalf("expected missing_upstream_cost warning item, got %+v", report.WarningItems)
	}
	if upstreamWarning.TotalCount != 4 || len(upstreamWarning.Details) != 3 {
		t.Fatalf("unexpected upstream warning item: %+v", upstreamWarning)
	}
	if upstreamWarning.Details[0].ScopeType != "tag" || upstreamWarning.Details[0].ScopeLabel != "billing" || upstreamWarning.Details[0].ModelName != "known-site-model" || upstreamWarning.Details[0].Count != 2 {
		t.Fatalf("unexpected first upstream warning detail: %+v", upstreamWarning.Details[0])
	}
	if upstreamWarning.Details[1].ScopeType != "channel" || upstreamWarning.Details[1].ScopeLabel != "epsilon" || upstreamWarning.Details[1].ModelName != "known-site-model" || upstreamWarning.Details[1].Count != 1 {
		t.Fatalf("unexpected second upstream warning detail: %+v", upstreamWarning.Details[1])
	}
	if upstreamWarning.Details[2].ScopeType != "channel" || upstreamWarning.Details[2].ScopeLabel != "zeta" || upstreamWarning.Details[2].ModelName != "manual-only-missing" || upstreamWarning.Details[2].Count != 1 {
		t.Fatalf("unexpected third upstream warning detail: %+v", upstreamWarning.Details[2])
	}

	overview, err := GenerateProfitBoardOverview(ProfitBoardConfigPayload{
		Batches:      query.Batches,
		ComboConfigs: query.ComboConfigs,
		Upstream:     query.Upstream,
		Site:         query.Site,
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardOverview: %v", err)
	}

	overviewSiteWarning := findProfitBoardWarningItem(overview.WarningItems, "missing_site_pricing")
	if overviewSiteWarning == nil || overviewSiteWarning.Message == "" || !strings.Contains(overviewSiteWarning.Message, "累计总览") {
		t.Fatalf("expected overview site warning message, got %+v", overview.WarningItems)
	}
	overviewUpstreamWarning := findProfitBoardWarningItem(overview.WarningItems, "missing_upstream_cost")
	if overviewUpstreamWarning == nil || overviewUpstreamWarning.Message == "" || !strings.Contains(overviewUpstreamWarning.Message, "累计总览") {
		t.Fatalf("expected overview upstream warning message, got %+v", overview.WarningItems)
	}
}

func TestGenerateProfitBoardReportSupportsSections(t *testing.T) {
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

	report, err := GenerateProfitBoardReport(ProfitBoardQuery{
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
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
		StartTimestamp: now - 60,
		EndTimestamp:   now,
		Granularity:    "day",
		Sections:       []string{"timeseries", "warning_items"},
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if len(report.Timeseries) != 1 {
		t.Fatalf("expected timeseries section loaded, got %+v", report.Timeseries)
	}
	if len(report.ChannelBreakdown) != 0 {
		t.Fatalf("expected channel breakdown skipped, got %+v", report.ChannelBreakdown)
	}
	if len(report.ModelBreakdown) != 0 {
		t.Fatalf("expected model breakdown skipped, got %+v", report.ModelBreakdown)
	}
	if len(report.Meta.LoadedSections) == 0 {
		t.Fatalf("expected loaded sections metadata, got %+v", report.Meta)
	}
	if report.Meta.LoadedSections[0] == "" {
		t.Fatalf("expected non-empty loaded sections, got %+v", report.Meta.LoadedSections)
	}
}
