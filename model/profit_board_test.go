package model

import (
	"testing"

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

	if err := db.AutoMigrate(&Channel{}, &Log{}, &ProfitBoardConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestSaveProfitBoardConfigUsesStableSignature(t *testing.T) {
	setupProfitBoardTestDB(t)

	payload := ProfitBoardConfigPayload{
		Selection: ProfitBoardSelection{
			ScopeType:  ProfitBoardScopeChannel,
			ChannelIDs: []int{9, 3, 9, 1},
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

	loaded, loadedSignature, err := GetProfitBoardConfig(ProfitBoardSelection{
		ScopeType:  ProfitBoardScopeChannel,
		ChannelIDs: []int{3, 1, 9},
	})
	if err != nil {
		t.Fatalf("GetProfitBoardConfig: %v", err)
	}
	if loadedSignature != signature {
		t.Fatalf("signature mismatch: got=%s want=%s", loadedSignature, signature)
	}
	if len(loaded.Selection.ChannelIDs) != 3 {
		t.Fatalf("expected 3 channel ids, got %d", len(loaded.Selection.ChannelIDs))
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
			Other:            `{"cache_tokens":200000,"cache_creation_tokens":100000,"upstream_cost":5.5}`,
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
		Selection: ProfitBoardSelection{
			ScopeType: ProfitBoardScopeTag,
			Tags:      []string{"tag-a"},
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
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if report.Summary.RequestCount != 2 {
		t.Fatalf("unexpected request count: %d", report.Summary.RequestCount)
	}
	if report.Summary.ReturnedCostCount != 1 || report.Summary.ManualCostCount != 1 {
		t.Fatalf("unexpected cost source counts: returned=%d manual=%d", report.Summary.ReturnedCostCount, report.Summary.ManualCostCount)
	}
	if report.Summary.MissingUpstreamCostCount != 0 {
		t.Fatalf("unexpected missing upstream count: %d", report.Summary.MissingUpstreamCostCount)
	}
	if report.Summary.ActualSiteRevenueUSD != 8.5 {
		t.Fatalf("unexpected actual site revenue: %v", report.Summary.ActualSiteRevenueUSD)
	}
	if report.Summary.ConfiguredSiteRevenueUSD != 9.4 {
		t.Fatalf("unexpected configured site revenue: %v", report.Summary.ConfiguredSiteRevenueUSD)
	}
	if report.Summary.UpstreamCostUSD != 7.35 {
		t.Fatalf("unexpected upstream cost: %v", report.Summary.UpstreamCostUSD)
	}
	if report.Summary.ConfiguredProfitUSD != 2.05 {
		t.Fatalf("unexpected configured profit: %v", report.Summary.ConfiguredProfitUSD)
	}
	if len(report.ChannelBreakdown) != 2 {
		t.Fatalf("expected 2 channel breakdown items, got %d", len(report.ChannelBreakdown))
	}
	if len(report.DetailRows) != 2 {
		t.Fatalf("expected 2 detail rows, got %d", len(report.DetailRows))
	}
}
