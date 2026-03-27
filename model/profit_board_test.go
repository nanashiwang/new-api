package model

import (
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

	if err := db.AutoMigrate(&Channel{}, &Log{}, &ProfitBoardConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestSaveProfitBoardConfigUsesStableSignature(t *testing.T) {
	setupProfitBoardTestDB(t)

	payload := ProfitBoardConfigPayload{
		Batches: []ProfitBoardBatch{
			{
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
	if len(report.BatchSummaries) != 1 || report.BatchSummaries[0].BatchName != "Tag A" {
		t.Fatalf("unexpected batch summaries: %+v", report.BatchSummaries)
	}
	if report.DetailRows[0].BatchName == "" {
		t.Fatalf("expected detail rows to include batch name")
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
	if config.Upstream.CostSource != ProfitBoardCostSourceReturnedFirst {
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
	})
	if err != nil {
		t.Fatalf("GenerateProfitBoardReport: %v", err)
	}

	if report.Summary.KnownUpstreamCostCount != 1 {
		t.Fatalf("unexpected known upstream count: %d", report.Summary.KnownUpstreamCostCount)
	}
	if report.Summary.ReturnedCostCount != 1 {
		t.Fatalf("unexpected returned cost count: %d", report.Summary.ReturnedCostCount)
	}
	if report.Summary.UpstreamCostUSD != 0 {
		t.Fatalf("unexpected upstream cost amount: %v", report.Summary.UpstreamCostUSD)
	}
	if len(report.DetailRows) != 1 || !report.DetailRows[0].UpstreamCostKnown {
		t.Fatalf("expected detail row to keep zero returned cost as known: %+v", report.DetailRows)
	}
}

func TestExportProfitBoardExcelIgnoresDetailLimitTruncation(t *testing.T) {
	db := setupProfitBoardTestDB(t)

	channel := Channel{Id: 1, Name: "alpha", Status: common.ChannelStatusEnabled}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	logs := []Log{
		{
			Id:               1,
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
			Id:               2,
			Type:             LogTypeConsume,
			CreatedAt:        1710003600,
			ChannelId:        1,
			ModelName:        "gpt-4o",
			Quota:            1000000,
			PromptTokens:     1000,
			CompletionTokens: 500,
			RequestId:        "req-2",
			Other:            `{"upstream_cost":2,"upstream_cost_reported":true}`,
		},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	data, filename, err := ExportProfitBoardExcel(ProfitBoardQuery{
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
		DetailLimit:    1,
	})
	if err != nil {
		t.Fatalf("ExportProfitBoardExcel: %v", err)
	}
	if !strings.HasSuffix(filename, ".xls") {
		t.Fatalf("unexpected filename: %s", filename)
	}
	content := string(data)
	if !strings.Contains(content, "req-1") || !strings.Contains(content, "req-2") {
		t.Fatalf("expected excel export to include all rows: %s", content)
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

	if activity.RequestCount != 2 || activity.LatestLogId != 12 || activity.LatestLogCreatedAt != 1710003600 {
		t.Fatalf("unexpected activity payload: %+v", activity)
	}
	if activity.ActivityWatermark != "2:12:1710003600" {
		t.Fatalf("unexpected watermark: %s", activity.ActivityWatermark)
	}
}
