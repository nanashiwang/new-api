package model

import (
	"bytes"
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"gorm.io/gorm"
)

const (
	ProfitBoardScopeChannel = "channel"
	ProfitBoardScopeTag     = "tag"
	ProfitBoardScopeBatch   = "batch_set"

	ProfitBoardCostSourceReturnedFirst = "returned_cost_first"
	ProfitBoardCostSourceReturnedOnly  = "returned_cost_only"
	ProfitBoardCostSourceManualOnly    = "manual_only"

	ProfitBoardSitePricingManual    = "manual"
	ProfitBoardSitePricingSiteModel = "site_model"
)

type ProfitBoardConfig struct {
	Id                 int    `json:"id"`
	SelectionType      string `json:"selection_type" gorm:"type:varchar(24);index;not null"`
	SelectionSignature string `json:"selection_signature" gorm:"type:varchar(255);uniqueIndex;not null"`
	SelectionValues    string `json:"selection_values" gorm:"type:text;not null"`
	UpstreamConfig     string `json:"upstream_config" gorm:"type:text;not null"`
	SiteConfig         string `json:"site_config" gorm:"type:text;not null"`
	CreatedAt          int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt          int64  `json:"updated_at" gorm:"bigint;index"`
}

type ProfitBoardSelection struct {
	ScopeType  string   `json:"scope_type"`
	ChannelIDs []int    `json:"channel_ids,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

type ProfitBoardBatch struct {
	Id         string   `json:"id,omitempty"`
	Name       string   `json:"name,omitempty"`
	ScopeType  string   `json:"scope_type"`
	ChannelIDs []int    `json:"channel_ids,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

type ProfitBoardTokenPricingConfig struct {
	CostSource         string   `json:"cost_source,omitempty"`
	PricingMode        string   `json:"pricing_mode,omitempty"`
	InputPrice         float64  `json:"input_price"`
	OutputPrice        float64  `json:"output_price"`
	CacheReadPrice     float64  `json:"cache_read_price"`
	CacheCreationPrice float64  `json:"cache_creation_price"`
	FixedAmount        float64  `json:"fixed_amount"`
	FixedTotalAmount   float64  `json:"fixed_total_amount"`
	ModelNames         []string `json:"model_names,omitempty"`
	Group              string   `json:"group,omitempty"`
	UseRechargePrice   bool     `json:"use_recharge_price,omitempty"`
}

type ProfitBoardConfigPayload struct {
	Batches   []ProfitBoardBatch            `json:"batches,omitempty"`
	Selection ProfitBoardSelection          `json:"selection,omitempty"`
	Upstream  ProfitBoardTokenPricingConfig `json:"upstream"`
	Site      ProfitBoardTokenPricingConfig `json:"site"`
}

type ProfitBoardQuery struct {
	Batches               []ProfitBoardBatch            `json:"batches,omitempty"`
	Selection             ProfitBoardSelection          `json:"selection,omitempty"`
	Upstream              ProfitBoardTokenPricingConfig `json:"upstream"`
	Site                  ProfitBoardTokenPricingConfig `json:"site"`
	StartTimestamp        int64                         `json:"start_timestamp"`
	EndTimestamp          int64                         `json:"end_timestamp"`
	Granularity           string                        `json:"granularity"`
	CustomIntervalMinutes int                           `json:"custom_interval_minutes,omitempty"`
	DetailLimit           int                           `json:"detail_limit"`
}

type ProfitBoardChannelOption struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Tag    string `json:"tag,omitempty"`
	Status int    `json:"status,omitempty"`
}

type ProfitBoardLocalModelOption struct {
	ModelName             string   `json:"model_name"`
	QuotaType             int      `json:"quota_type"`
	EnableGroups          []string `json:"enable_groups"`
	SupportsCacheRead     bool     `json:"supports_cache_read"`
	SupportsCacheCreation bool     `json:"supports_cache_creation"`
}

type ProfitBoardOptions struct {
	Channels    []ProfitBoardChannelOption    `json:"channels"`
	Tags        []string                      `json:"tags"`
	Groups      []string                      `json:"groups"`
	LocalModels []ProfitBoardLocalModelOption `json:"local_models"`
}

type ProfitBoardSummary struct {
	RequestCount                 int     `json:"request_count"`
	ActualSiteRevenueUSD         float64 `json:"actual_site_revenue_usd"`
	ConfiguredSiteRevenueUSD     float64 `json:"configured_site_revenue_usd"`
	UpstreamCostUSD              float64 `json:"upstream_cost_usd"`
	ConfiguredProfitUSD          float64 `json:"configured_profit_usd"`
	ActualProfitUSD              float64 `json:"actual_profit_usd"`
	KnownUpstreamCostCount       int     `json:"known_upstream_cost_count"`
	MissingUpstreamCostCount     int     `json:"missing_upstream_cost_count"`
	ReturnedCostCount            int     `json:"returned_cost_count"`
	ManualCostCount              int     `json:"manual_cost_count"`
	SiteModelMatchCount          int     `json:"site_model_match_count"`
	MissingSitePricingCount      int     `json:"missing_site_pricing_count"`
	ConfiguredProfitCoverageRate float64 `json:"configured_profit_coverage_rate"`
}

type ProfitBoardTimeseriesPoint struct {
	BatchId                  string  `json:"batch_id,omitempty"`
	BatchName                string  `json:"batch_name,omitempty"`
	Bucket                   string  `json:"bucket"`
	BucketTimestamp          int64   `json:"bucket_timestamp"`
	RequestCount             int     `json:"request_count"`
	ActualSiteRevenueUSD     float64 `json:"actual_site_revenue_usd"`
	ConfiguredSiteRevenueUSD float64 `json:"configured_site_revenue_usd"`
	UpstreamCostUSD          float64 `json:"upstream_cost_usd"`
	ConfiguredProfitUSD      float64 `json:"configured_profit_usd"`
	ActualProfitUSD          float64 `json:"actual_profit_usd"`
	KnownUpstreamCostCount   int     `json:"known_upstream_cost_count"`
	MissingUpstreamCostCount int     `json:"missing_upstream_cost_count"`
	SiteModelMatchCount      int     `json:"site_model_match_count"`
	MissingSitePricingCount  int     `json:"missing_site_pricing_count"`
}

type ProfitBoardBreakdownItem struct {
	BatchId                  string  `json:"batch_id,omitempty"`
	BatchName                string  `json:"batch_name,omitempty"`
	Key                      string  `json:"key"`
	Label                    string  `json:"label"`
	RequestCount             int     `json:"request_count"`
	ActualSiteRevenueUSD     float64 `json:"actual_site_revenue_usd"`
	ConfiguredSiteRevenueUSD float64 `json:"configured_site_revenue_usd"`
	UpstreamCostUSD          float64 `json:"upstream_cost_usd"`
	ConfiguredProfitUSD      float64 `json:"configured_profit_usd"`
	ActualProfitUSD          float64 `json:"actual_profit_usd"`
	KnownUpstreamCostCount   int     `json:"known_upstream_cost_count"`
	MissingUpstreamCostCount int     `json:"missing_upstream_cost_count"`
}

type ProfitBoardDetailRow struct {
	Id                       int     `json:"id"`
	BatchId                  string  `json:"batch_id"`
	BatchName                string  `json:"batch_name"`
	CreatedAt                int64   `json:"created_at"`
	RequestId                string  `json:"request_id,omitempty"`
	ChannelId                int     `json:"channel_id"`
	ChannelName              string  `json:"channel_name"`
	ModelName                string  `json:"model_name"`
	PromptTokens             int     `json:"prompt_tokens"`
	CompletionTokens         int     `json:"completion_tokens"`
	InputTokens              int     `json:"input_tokens"`
	CacheReadTokens          int     `json:"cache_read_tokens"`
	CacheCreationTokens      int     `json:"cache_creation_tokens"`
	ActualSiteRevenueUSD     float64 `json:"actual_site_revenue_usd"`
	ConfiguredSiteRevenueUSD float64 `json:"configured_site_revenue_usd"`
	UpstreamCostUSD          float64 `json:"upstream_cost_usd"`
	ConfiguredProfitUSD      float64 `json:"configured_profit_usd"`
	ActualProfitUSD          float64 `json:"actual_profit_usd"`
	ConfiguredActualDeltaUSD float64 `json:"configured_actual_delta_usd"`
	UpstreamCostKnown        bool    `json:"upstream_cost_known"`
	UpstreamCostSource       string  `json:"upstream_cost_source"`
	SitePricingSource        string  `json:"site_pricing_source"`
	SitePricingKnown         bool    `json:"site_pricing_known"`
}

type ProfitBoardBatchInfo struct {
	Id               string                     `json:"id"`
	Name             string                     `json:"name"`
	ScopeType        string                     `json:"scope_type"`
	Signature        string                     `json:"signature"`
	ChannelIDs       []int                      `json:"channel_ids"`
	Tags             []string                   `json:"tags,omitempty"`
	ResolvedChannels []ProfitBoardChannelOption `json:"resolved_channels"`
}

type ProfitBoardBatchSummary struct {
	BatchId   string `json:"batch_id"`
	BatchName string `json:"batch_name"`
	ProfitBoardSummary
}

type ProfitBoardMeta struct {
	SiteUseRechargePrice      bool    `json:"site_use_recharge_price"`
	SitePriceFactor           float64 `json:"site_price_factor"`
	SitePriceFactorNote       string  `json:"site_price_factor_note"`
	GeneratedAt               int64   `json:"generated_at"`
	ActivityWatermark         string  `json:"activity_watermark"`
	LatestLogId               int     `json:"latest_log_id"`
	LatestLogCreatedAt        int64   `json:"latest_log_created_at"`
	CumulativeScope           string  `json:"cumulative_scope,omitempty"`
	FixedTotalAmountScope     string  `json:"fixed_total_amount_scope,omitempty"`
	FixedAmountAllocationMode string  `json:"fixed_amount_allocation_mode,omitempty"`
	UpstreamFixedTotalAmount  float64 `json:"upstream_fixed_total_amount_usd,omitempty"`
	SiteFixedTotalAmount      float64 `json:"site_fixed_total_amount_usd,omitempty"`
	LegacyUpstreamFixedAmount bool    `json:"legacy_upstream_fixed_amount,omitempty"`
	LegacySiteFixedAmount     bool    `json:"legacy_site_fixed_amount,omitempty"`
}

type ProfitBoardActivity struct {
	Signature          string `json:"signature"`
	GeneratedAt        int64  `json:"generated_at"`
	ActivityWatermark  string `json:"activity_watermark"`
	LatestLogId        int    `json:"latest_log_id"`
	LatestLogCreatedAt int64  `json:"latest_log_created_at"`
	RequestCount       int    `json:"request_count"`
}

type ProfitBoardReport struct {
	Signature        string                       `json:"signature"`
	Batches          []ProfitBoardBatchInfo       `json:"batches"`
	BatchSummaries   []ProfitBoardBatchSummary    `json:"batch_summaries"`
	Summary          ProfitBoardSummary           `json:"summary"`
	Meta             ProfitBoardMeta              `json:"meta"`
	Timeseries       []ProfitBoardTimeseriesPoint `json:"timeseries"`
	ChannelBreakdown []ProfitBoardBreakdownItem   `json:"channel_breakdown"`
	ModelBreakdown   []ProfitBoardBreakdownItem   `json:"model_breakdown"`
	DetailRows       []ProfitBoardDetailRow       `json:"detail_rows"`
	DetailTruncated  bool                         `json:"detail_truncated"`
	Warnings         []string                     `json:"warnings,omitempty"`
}

type profitBoardOtherInfo struct {
	tokenUsageOtherInfo
	UpstreamCost         float64 `json:"upstream_cost"`
	UpstreamCostReported bool    `json:"upstream_cost_reported"`
	UpstreamCostSource   string  `json:"upstream_cost_source"`
	UpstreamCostCurrency string  `json:"upstream_cost_currency"`
}

type profitBoardLogRow struct {
	Id               int    `gorm:"column:id"`
	CreatedAt        int64  `gorm:"column:created_at"`
	RequestId        string `gorm:"column:request_id"`
	ChannelId        int    `gorm:"column:channel_id"`
	ModelName        string `gorm:"column:model_name"`
	Quota            int    `gorm:"column:quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens"`
	Other            string `gorm:"column:other"`
}

func normalizeProfitBoardSelection(selection ProfitBoardSelection) (ProfitBoardSelection, string, error) {
	scopeType := strings.ToLower(strings.TrimSpace(selection.ScopeType))
	switch scopeType {
	case ProfitBoardScopeChannel:
		ids := make([]int, 0, len(selection.ChannelIDs))
		exists := make(map[int]struct{}, len(selection.ChannelIDs))
		for _, id := range selection.ChannelIDs {
			if id <= 0 {
				continue
			}
			if _, ok := exists[id]; ok {
				continue
			}
			exists[id] = struct{}{}
			ids = append(ids, id)
		}
		sort.Ints(ids)
		if len(ids) == 0 {
			return ProfitBoardSelection{}, "", errors.New("请至少选择一个渠道")
		}
		parts := make([]string, 0, len(ids))
		for _, id := range ids {
			parts = append(parts, strconv.Itoa(id))
		}
		return ProfitBoardSelection{
			ScopeType:  scopeType,
			ChannelIDs: ids,
		}, scopeType + ":" + strings.Join(parts, ","), nil
	case ProfitBoardScopeTag:
		tags := make([]string, 0, len(selection.Tags))
		exists := make(map[string]struct{}, len(selection.Tags))
		for _, tag := range selection.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if _, ok := exists[tag]; ok {
				continue
			}
			exists[tag] = struct{}{}
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		if len(tags) == 0 {
			return ProfitBoardSelection{}, "", errors.New("请至少选择一个标签")
		}
		return ProfitBoardSelection{
			ScopeType: scopeType,
			Tags:      tags,
		}, scopeType + ":" + strings.Join(tags, "|"), nil
	default:
		return ProfitBoardSelection{}, "", errors.New("无效的收益看板选择类型")
	}
}

func profitBoardLegacyBatches(selection ProfitBoardSelection) []ProfitBoardBatch {
	if strings.TrimSpace(selection.ScopeType) == "" {
		return nil
	}
	return []ProfitBoardBatch{
		{
			Id:         "batch-1",
			Name:       "批次 1",
			ScopeType:  selection.ScopeType,
			ChannelIDs: selection.ChannelIDs,
			Tags:       selection.Tags,
		},
	}
}

func normalizeProfitBoardBatch(batch ProfitBoardBatch, index int) (ProfitBoardBatch, string, error) {
	normalizedSelection, signature, err := normalizeProfitBoardSelection(ProfitBoardSelection{
		ScopeType:  batch.ScopeType,
		ChannelIDs: batch.ChannelIDs,
		Tags:       batch.Tags,
	})
	if err != nil {
		return ProfitBoardBatch{}, "", err
	}
	id := strings.TrimSpace(batch.Id)
	if id == "" {
		id = fmt.Sprintf("batch-%d", index+1)
	}
	name := strings.TrimSpace(batch.Name)
	if name == "" {
		name = fmt.Sprintf("批次 %d", index+1)
	}
	return ProfitBoardBatch{
		Id:         id,
		Name:       name,
		ScopeType:  normalizedSelection.ScopeType,
		ChannelIDs: normalizedSelection.ChannelIDs,
		Tags:       normalizedSelection.Tags,
	}, signature, nil
}

func buildProfitBoardBatchSetSignature(batchSignatures []string) string {
	if len(batchSignatures) == 1 {
		return batchSignatures[0]
	}
	sorted := append([]string(nil), batchSignatures...)
	sort.Strings(sorted)
	hash := sha1.Sum([]byte(strings.Join(sorted, ";")))
	return "batches:" + hex.EncodeToString(hash[:])
}

func normalizeProfitBoardBatches(batches []ProfitBoardBatch, legacySelection ProfitBoardSelection) ([]ProfitBoardBatch, string, string, error) {
	sourceBatches := batches
	if len(sourceBatches) == 0 {
		sourceBatches = profitBoardLegacyBatches(legacySelection)
	}
	if len(sourceBatches) == 0 {
		return nil, "", "", errors.New("请至少添加一个批次")
	}

	normalized := make([]ProfitBoardBatch, 0, len(sourceBatches))
	batchSignatures := make([]string, 0, len(sourceBatches))
	idExists := make(map[string]struct{}, len(sourceBatches))
	for index, batch := range sourceBatches {
		current, signature, err := normalizeProfitBoardBatch(batch, index)
		if err != nil {
			return nil, "", "", err
		}
		if _, ok := idExists[current.Id]; ok {
			return nil, "", "", errors.New("批次标识重复，请删除后重新添加批次")
		}
		idExists[current.Id] = struct{}{}
		normalized = append(normalized, current)
		batchSignatures = append(batchSignatures, signature)
	}

	selectionType := ProfitBoardScopeBatch
	if len(normalized) == 1 {
		selectionType = normalized[0].ScopeType
	}
	return normalized, buildProfitBoardBatchSetSignature(batchSignatures), selectionType, nil
}

func parseProfitBoardConfigBatches(raw string) []ProfitBoardBatch {
	batches := make([]ProfitBoardBatch, 0)
	if raw != "" && common.UnmarshalJsonStr(raw, &batches) == nil && len(batches) > 0 {
		return batches
	}
	selection := ProfitBoardSelection{}
	if raw != "" && common.UnmarshalJsonStr(raw, &selection) == nil && strings.TrimSpace(selection.ScopeType) != "" {
		return profitBoardLegacyBatches(selection)
	}
	return nil
}

func normalizeProfitBoardPricingConfig(config ProfitBoardTokenPricingConfig, isSite bool) ProfitBoardTokenPricingConfig {
	if !isSite {
		config.CostSource = ProfitBoardCostSourceManualOnly
	}
	if isSite && config.PricingMode == "" {
		config.PricingMode = ProfitBoardSitePricingManual
	}
	if !isSite {
		config.PricingMode = ""
	}
	return config
}

func validateProfitBoardPricingConfig(config ProfitBoardTokenPricingConfig, isSite bool) error {
	numbers := []float64{
		config.InputPrice,
		config.OutputPrice,
		config.CacheReadPrice,
		config.CacheCreationPrice,
		config.FixedAmount,
		config.FixedTotalAmount,
	}
	for _, num := range numbers {
		if math.IsNaN(num) || math.IsInf(num, 0) || num < 0 {
			return errors.New("价格配置必须是非负数字")
		}
	}
	switch config.CostSource {
	case "", ProfitBoardCostSourceReturnedFirst, ProfitBoardCostSourceReturnedOnly, ProfitBoardCostSourceManualOnly:
	default:
		return errors.New("无效的上游费用来源配置")
	}
	if !isSite {
		return nil
	}
	switch config.PricingMode {
	case "", ProfitBoardSitePricingManual, ProfitBoardSitePricingSiteModel:
		return nil
	default:
		return errors.New("无效的本站价格来源配置")
	}
}

func GetProfitBoardConfig(batches []ProfitBoardBatch, selection ProfitBoardSelection) (*ProfitBoardConfigPayload, string, error) {
	normalized, signature, _, err := normalizeProfitBoardBatches(batches, selection)
	if err != nil {
		return nil, "", err
	}

	config := &ProfitBoardConfig{}
	if err := DB.Where("selection_signature = ?", signature).First(config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ProfitBoardConfigPayload{
				Batches: normalized,
				Upstream: normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
					CostSource: ProfitBoardCostSourceManualOnly,
				}, false),
				Site: normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
					PricingMode: ProfitBoardSitePricingManual,
				}, true),
			}, signature, nil
		}
		return nil, "", err
	}

	payload := &ProfitBoardConfigPayload{
		Batches: normalized,
		Upstream: normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceManualOnly,
		}, false),
		Site: normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		}, true),
	}
	if parsedBatches := parseProfitBoardConfigBatches(config.SelectionValues); len(parsedBatches) > 0 {
		payload.Batches = parsedBatches
	}
	_ = common.UnmarshalJsonStr(config.UpstreamConfig, &payload.Upstream)
	_ = common.UnmarshalJsonStr(config.SiteConfig, &payload.Site)
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
	return payload, signature, nil
}

func SaveProfitBoardConfig(payload ProfitBoardConfigPayload) (*ProfitBoardConfigPayload, string, error) {
	normalized, signature, selectionType, err := normalizeProfitBoardBatches(payload.Batches, payload.Selection)
	if err != nil {
		return nil, "", err
	}
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
	if err := validateProfitBoardPricingConfig(payload.Upstream, false); err != nil {
		return nil, "", err
	}
	if err := validateProfitBoardPricingConfig(payload.Site, true); err != nil {
		return nil, "", err
	}

	selectionBytes, err := common.Marshal(normalized)
	if err != nil {
		return nil, "", err
	}
	upstreamBytes, err := common.Marshal(payload.Upstream)
	if err != nil {
		return nil, "", err
	}
	siteBytes, err := common.Marshal(payload.Site)
	if err != nil {
		return nil, "", err
	}

	now := common.GetTimestamp()
	record := &ProfitBoardConfig{}
	if err := DB.Where("selection_signature = ?", signature).First(record).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", err
		}
		record = &ProfitBoardConfig{
			SelectionType:      selectionType,
			SelectionSignature: signature,
			SelectionValues:    string(selectionBytes),
			UpstreamConfig:     string(upstreamBytes),
			SiteConfig:         string(siteBytes),
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := DB.Create(record).Error; err != nil {
			return nil, "", err
		}
	} else {
		record.SelectionType = selectionType
		record.SelectionValues = string(selectionBytes)
		record.UpstreamConfig = string(upstreamBytes)
		record.SiteConfig = string(siteBytes)
		record.UpdatedAt = now
		if err := DB.Save(record).Error; err != nil {
			return nil, "", err
		}
	}

	return &ProfitBoardConfigPayload{
		Batches:  normalized,
		Upstream: payload.Upstream,
		Site:     payload.Site,
	}, signature, nil
}

func GetProfitBoardOptions() (*ProfitBoardOptions, error) {
	options := &ProfitBoardOptions{}

	channels := make([]ProfitBoardChannelOption, 0)
	if err := DB.Model(&Channel{}).
		Select("id, name, tag, status").
		Order("priority desc, id desc").
		Scan(&channels).Error; err != nil {
		return nil, err
	}
	options.Channels = channels

	type tagRow struct {
		Tag string `gorm:"column:tag"`
	}
	tagRows := make([]tagRow, 0)
	if err := DB.Model(&Channel{}).
		Select("DISTINCT tag").
		Where("tag IS NOT NULL AND tag != ''").
		Order("tag asc").
		Scan(&tagRows).Error; err != nil {
		return nil, err
	}
	options.Tags = make([]string, 0, len(tagRows))
	for _, row := range tagRows {
		if row.Tag != "" {
			options.Tags = append(options.Tags, row.Tag)
		}
	}

	groupNames := make([]string, 0)
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupNames = append(groupNames, groupName)
	}
	sort.Strings(groupNames)
	options.Groups = groupNames

	localModels := GetPricing()
	options.LocalModels = make([]ProfitBoardLocalModelOption, 0, len(localModels))
	for _, item := range localModels {
		options.LocalModels = append(options.LocalModels, ProfitBoardLocalModelOption{
			ModelName:             item.ModelName,
			QuotaType:             item.QuotaType,
			EnableGroups:          item.EnableGroup,
			SupportsCacheRead:     item.SupportsCacheRead,
			SupportsCacheCreation: item.SupportsCacheCreation,
		})
	}
	sort.Slice(options.LocalModels, func(i, j int) bool {
		return options.LocalModels[i].ModelName < options.LocalModels[j].ModelName
	})

	return options, nil
}

func normalizeProfitBoardQuery(query ProfitBoardQuery) (ProfitBoardQuery, string, error) {
	normalizedBatches, signature, _, err := normalizeProfitBoardBatches(query.Batches, query.Selection)
	if err != nil {
		return ProfitBoardQuery{}, "", err
	}
	query.Upstream = normalizeProfitBoardPricingConfig(query.Upstream, false)
	query.Site = normalizeProfitBoardPricingConfig(query.Site, true)
	if err := validateProfitBoardPricingConfig(query.Upstream, false); err != nil {
		return ProfitBoardQuery{}, "", err
	}
	if err := validateProfitBoardPricingConfig(query.Site, true); err != nil {
		return ProfitBoardQuery{}, "", err
	}

	if query.StartTimestamp <= 0 {
		query.StartTimestamp = time.Now().Add(-7 * 24 * time.Hour).Unix()
	}
	if query.EndTimestamp <= 0 {
		query.EndTimestamp = time.Now().Unix()
	}
	if query.EndTimestamp < query.StartTimestamp {
		return ProfitBoardQuery{}, "", errors.New("结束时间不能早于开始时间")
	}
	switch strings.ToLower(strings.TrimSpace(query.Granularity)) {
	case "", "auto":
		if query.EndTimestamp-query.StartTimestamp <= 72*3600 {
			query.Granularity = "hour"
		} else if query.EndTimestamp-query.StartTimestamp <= 45*24*3600 {
			query.Granularity = "day"
		} else {
			query.Granularity = "week"
		}
	case "hour", "day", "week", "month":
		query.Granularity = strings.ToLower(strings.TrimSpace(query.Granularity))
	case "custom":
		query.Granularity = "custom"
		if query.CustomIntervalMinutes <= 0 {
			return ProfitBoardQuery{}, "", errors.New("自定义时间粒度必须大于 0 分钟")
		}
		if query.CustomIntervalMinutes > 43200 {
			return ProfitBoardQuery{}, "", errors.New("自定义时间粒度不能超过 43200 分钟")
		}
	default:
		return ProfitBoardQuery{}, "", errors.New("无效的时间粒度")
	}
	if query.DetailLimit <= 0 {
		query.DetailLimit = 300
	}
	if query.DetailLimit > 2000 {
		query.DetailLimit = 2000
	}
	query.Batches = normalizedBatches
	query.Selection = ProfitBoardSelection{}
	return query, signature, nil
}

func buildProfitBoardActivityWatermark(requestCount int, latestLogID int, latestCreatedAt int64) string {
	return fmt.Sprintf("%d:%d:%d", requestCount, latestLogID, latestCreatedAt)
}

func resolveProfitBoardChannels(selection ProfitBoardSelection) ([]ProfitBoardChannelOption, []int, error) {
	switch selection.ScopeType {
	case ProfitBoardScopeChannel:
		channels := make([]ProfitBoardChannelOption, 0, len(selection.ChannelIDs))
		if err := DB.Model(&Channel{}).
			Select("id, name, tag, status").
			Where("id IN ?", selection.ChannelIDs).
			Order("id asc").
			Scan(&channels).Error; err != nil {
			return nil, nil, err
		}
		if len(channels) == 0 {
			return nil, nil, errors.New("所选渠道不存在")
		}
		ids := make([]int, 0, len(channels))
		for _, channel := range channels {
			ids = append(ids, channel.Id)
		}
		return channels, ids, nil
	case ProfitBoardScopeTag:
		channels := make([]ProfitBoardChannelOption, 0)
		if err := DB.Model(&Channel{}).
			Select("id, name, tag, status").
			Where("tag IN ?", selection.Tags).
			Order("tag asc, priority desc, id desc").
			Scan(&channels).Error; err != nil {
			return nil, nil, err
		}
		if len(channels) == 0 {
			return nil, nil, errors.New("所选标签下没有渠道")
		}
		ids := make([]int, 0, len(channels))
		exists := make(map[int]struct{}, len(channels))
		for _, channel := range channels {
			if _, ok := exists[channel.Id]; ok {
				continue
			}
			exists[channel.Id] = struct{}{}
			ids = append(ids, channel.Id)
		}
		sort.Ints(ids)
		return channels, ids, nil
	default:
		return nil, nil, errors.New("无效的收益看板选择类型")
	}
}

func resolveProfitBoardBatch(batch ProfitBoardBatch) (ProfitBoardBatchInfo, error) {
	resolvedChannels, channelIDs, err := resolveProfitBoardChannels(ProfitBoardSelection{
		ScopeType:  batch.ScopeType,
		ChannelIDs: batch.ChannelIDs,
		Tags:       batch.Tags,
	})
	if err != nil {
		return ProfitBoardBatchInfo{}, err
	}
	_, signature, err := normalizeProfitBoardSelection(ProfitBoardSelection{
		ScopeType:  batch.ScopeType,
		ChannelIDs: batch.ChannelIDs,
		Tags:       batch.Tags,
	})
	if err != nil {
		return ProfitBoardBatchInfo{}, err
	}
	return ProfitBoardBatchInfo{
		Id:               batch.Id,
		Name:             batch.Name,
		ScopeType:        batch.ScopeType,
		Signature:        signature,
		ChannelIDs:       channelIDs,
		Tags:             batch.Tags,
		ResolvedChannels: resolvedChannels,
	}, nil
}

func resolveProfitBoardBatches(batches []ProfitBoardBatch) ([]ProfitBoardBatchInfo, error) {
	resolved := make([]ProfitBoardBatchInfo, 0, len(batches))
	channelOwners := make(map[int]string)
	channelNames := make(map[int]string)

	for _, batch := range batches {
		current, err := resolveProfitBoardBatch(batch)
		if err != nil {
			return nil, err
		}
		for _, channel := range current.ResolvedChannels {
			name := strings.TrimSpace(channel.Name)
			if name == "" {
				name = fmt.Sprintf("渠道 #%d", channel.Id)
			}
			channelNames[channel.Id] = name
			if owner, ok := channelOwners[channel.Id]; ok {
				return nil, fmt.Errorf("%s 同时出现在批次“%s”和“%s”中，请调整批次避免重复统计", name, owner, current.Name)
			}
			channelOwners[channel.Id] = current.Name
		}
		resolved = append(resolved, current)
	}
	return resolved, nil
}

func buildProfitBoardBucket(timestamp int64, granularity string, customIntervalMinutes int) (int64, string) {
	t := time.Unix(timestamp, 0).In(time.Local)
	switch granularity {
	case "hour":
		bucket := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
		return bucket.Unix(), bucket.Format("2006-01-02 15:00")
	case "month":
		bucket := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
		return bucket.Unix(), bucket.Format("2006-01")
	case "week":
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		bucket := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, -(weekday - 1))
		year, week := bucket.ISOWeek()
		return bucket.Unix(), fmt.Sprintf("%d-W%02d", year, week)
	case "custom":
		intervalSeconds := int64(customIntervalMinutes) * 60
		current := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
		minutesSinceMidnight := int64(current.Hour()*60 + current.Minute())
		bucketMinutes := (minutesSinceMidnight / int64(customIntervalMinutes)) * int64(customIntervalMinutes)
		bucket := time.Date(
			current.Year(),
			current.Month(),
			current.Day(),
			int(bucketMinutes/60),
			int(bucketMinutes%60),
			0,
			0,
			current.Location(),
		)
		if intervalSeconds <= 3600 {
			return bucket.Unix(), bucket.Format("2006-01-02 15:04")
		}
		return bucket.Unix(), bucket.Format("2006-01-02 15:04")
	default:
		bucket := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		return bucket.Unix(), bucket.Format("2006-01-02")
	}
}

func collectProfitBoardChannelIDs(batches []ProfitBoardBatchInfo) []int {
	ids := make([]int, 0)
	seen := make(map[int]struct{})
	for _, batch := range batches {
		for _, channelID := range batch.ChannelIDs {
			if _, ok := seen[channelID]; ok {
				continue
			}
			seen[channelID] = struct{}{}
			ids = append(ids, channelID)
		}
	}
	sort.Ints(ids)
	return ids
}

func GetProfitBoardActivity(query ProfitBoardQuery) (*ProfitBoardActivity, error) {
	normalizedQuery, signature, err := normalizeProfitBoardQuery(query)
	if err != nil {
		return nil, err
	}

	resolvedBatches, err := resolveProfitBoardBatches(normalizedQuery.Batches)
	if err != nil {
		return nil, err
	}

	channelIDs := collectProfitBoardChannelIDs(resolvedBatches)
	if len(channelIDs) == 0 {
		return &ProfitBoardActivity{
			Signature:         signature,
			GeneratedAt:       common.GetTimestamp(),
			ActivityWatermark: buildProfitBoardActivityWatermark(0, 0, 0),
		}, nil
	}

	baseQuery := LOG_DB.Table("logs").
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? AND created_at <= ?", normalizedQuery.StartTimestamp, normalizedQuery.EndTimestamp).
		Where("channel_id IN ?", channelIDs)

	var requestCount int64
	if err := baseQuery.Count(&requestCount).Error; err != nil {
		return nil, err
	}

	type latestLogRow struct {
		Id        int   `gorm:"column:id"`
		CreatedAt int64 `gorm:"column:created_at"`
	}
	latestRow := latestLogRow{}
	if err := LOG_DB.Table("logs").
		Select("id, created_at").
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? AND created_at <= ?", normalizedQuery.StartTimestamp, normalizedQuery.EndTimestamp).
		Where("channel_id IN ?", channelIDs).
		Order("id desc").
		Limit(1).
		Scan(&latestRow).Error; err != nil {
		return nil, err
	}

	return &ProfitBoardActivity{
		Signature:          signature,
		GeneratedAt:        common.GetTimestamp(),
		ActivityWatermark:  buildProfitBoardActivityWatermark(int(requestCount), latestRow.Id, latestRow.CreatedAt),
		LatestLogId:        latestRow.Id,
		LatestLogCreatedAt: latestRow.CreatedAt,
		RequestCount:       int(requestCount),
	}, nil
}

func roundProfitBoardAmount(value float64) float64 {
	return math.Round(value*1000000) / 1000000
}

func profitBoardCSVMoney(value float64) string {
	return strconv.FormatFloat(roundProfitBoardAmount(value), 'f', 6, 64)
}

func profitBoardTokenMoneyUSD(inputTokens, completionTokens, cacheReadTokens, cacheCreationTokens int, config ProfitBoardTokenPricingConfig) float64 {
	return float64(inputTokens)*config.InputPrice/1_000_000 +
		float64(completionTokens)*config.OutputPrice/1_000_000 +
		float64(cacheReadTokens)*config.CacheReadPrice/1_000_000 +
		float64(cacheCreationTokens)*config.CacheCreationPrice/1_000_000 +
		config.FixedAmount
}

func profitBoardLegacyFixedAmountEnabled(config ProfitBoardTokenPricingConfig) bool {
	return config.FixedAmount > 0
}

func profitBoardFixedAllocationShare(total float64, totalRequests int, itemRequests int) float64 {
	if total == 0 || totalRequests <= 0 || itemRequests <= 0 {
		return 0
	}
	return total * float64(itemRequests) / float64(totalRequests)
}

func applyProfitBoardFixedTotals(report *ProfitBoardReport, siteFixedTotal float64, upstreamFixedTotal float64) {
	if report == nil {
		return
	}
	if report.Meta.FixedAmountAllocationMode == "" {
		report.Meta.FixedAmountAllocationMode = "request_count"
	}
	if report.Meta.FixedTotalAmountScope == "" {
		report.Meta.FixedTotalAmountScope = "period_only"
	}
	report.Meta.SiteFixedTotalAmount = roundProfitBoardAmount(siteFixedTotal)
	report.Meta.UpstreamFixedTotalAmount = roundProfitBoardAmount(upstreamFixedTotal)

	report.Summary.ConfiguredSiteRevenueUSD += siteFixedTotal
	report.Summary.UpstreamCostUSD += upstreamFixedTotal
	report.Summary.ConfiguredProfitUSD += siteFixedTotal - upstreamFixedTotal
	report.Summary.ActualProfitUSD -= upstreamFixedTotal

	if report.Summary.RequestCount <= 0 {
		return
	}

	for index := range report.BatchSummaries {
		siteShare := profitBoardFixedAllocationShare(siteFixedTotal, report.Summary.RequestCount, report.BatchSummaries[index].RequestCount)
		upstreamShare := profitBoardFixedAllocationShare(upstreamFixedTotal, report.Summary.RequestCount, report.BatchSummaries[index].RequestCount)
		report.BatchSummaries[index].ConfiguredSiteRevenueUSD += siteShare
		report.BatchSummaries[index].UpstreamCostUSD += upstreamShare
		report.BatchSummaries[index].ConfiguredProfitUSD += siteShare - upstreamShare
		report.BatchSummaries[index].ActualProfitUSD -= upstreamShare
	}

	for index := range report.Timeseries {
		siteShare := profitBoardFixedAllocationShare(siteFixedTotal, report.Summary.RequestCount, report.Timeseries[index].RequestCount)
		upstreamShare := profitBoardFixedAllocationShare(upstreamFixedTotal, report.Summary.RequestCount, report.Timeseries[index].RequestCount)
		report.Timeseries[index].ConfiguredSiteRevenueUSD += siteShare
		report.Timeseries[index].UpstreamCostUSD += upstreamShare
		report.Timeseries[index].ConfiguredProfitUSD += siteShare - upstreamShare
		report.Timeseries[index].ActualProfitUSD -= upstreamShare
	}

	for index := range report.ChannelBreakdown {
		siteShare := profitBoardFixedAllocationShare(siteFixedTotal, report.Summary.RequestCount, report.ChannelBreakdown[index].RequestCount)
		upstreamShare := profitBoardFixedAllocationShare(upstreamFixedTotal, report.Summary.RequestCount, report.ChannelBreakdown[index].RequestCount)
		report.ChannelBreakdown[index].ConfiguredSiteRevenueUSD += siteShare
		report.ChannelBreakdown[index].UpstreamCostUSD += upstreamShare
		report.ChannelBreakdown[index].ConfiguredProfitUSD += siteShare - upstreamShare
		report.ChannelBreakdown[index].ActualProfitUSD -= upstreamShare
	}

	for index := range report.ModelBreakdown {
		siteShare := profitBoardFixedAllocationShare(siteFixedTotal, report.Summary.RequestCount, report.ModelBreakdown[index].RequestCount)
		upstreamShare := profitBoardFixedAllocationShare(upstreamFixedTotal, report.Summary.RequestCount, report.ModelBreakdown[index].RequestCount)
		report.ModelBreakdown[index].ConfiguredSiteRevenueUSD += siteShare
		report.ModelBreakdown[index].UpstreamCostUSD += upstreamShare
		report.ModelBreakdown[index].ConfiguredProfitUSD += siteShare - upstreamShare
		report.ModelBreakdown[index].ActualProfitUSD -= upstreamShare
	}
}

func profitBoardPriceFactor(useRechargePrice bool) float64 {
	if !useRechargePrice {
		return 1
	}
	if operation_setting.USDExchangeRate <= 0 {
		return 1
	}
	return operation_setting.Price / operation_setting.USDExchangeRate
}

func profitBoardPriceFactorMeta(useRechargePrice bool) (float64, string) {
	factor := profitBoardPriceFactor(useRechargePrice)
	if !useRechargePrice {
		return factor, "当前按本站模型原价重算"
	}
	if math.Abs(factor-1) < 0.000001 {
		return factor, "已开启按充值价格读取，但当前充值倍率为 1.000x，所以结果会与原价一致"
	}
	return factor, fmt.Sprintf("已开启按充值价格读取，当前充值倍率为 %.3fx", factor)
}

func profitBoardResolveGroupRatio(groups []string, preferredGroup string, groupRatios map[string]float64) (float64, bool) {
	if preferredGroup != "" {
		for _, group := range groups {
			if group == preferredGroup {
				if ratio, ok := groupRatios[group]; ok {
					return ratio, true
				}
				return 1, true
			}
		}
		return 0, false
	}
	best := math.MaxFloat64
	found := false
	for _, group := range groups {
		ratio := 1.0
		if current, ok := groupRatios[group]; ok {
			ratio = current
		}
		if ratio < best {
			best = ratio
			found = true
		}
	}
	if found {
		return best, true
	}
	return 1, true
}

func profitBoardSiteModelRevenueUSD(
	row profitBoardLogRow,
	inputTokens int,
	cacheReadTokens int,
	cacheCreationTokens int,
	config ProfitBoardTokenPricingConfig,
	pricingMap map[string]Pricing,
	groupRatios map[string]float64,
) (float64, string, bool) {
	pricing, ok := pricingMap[row.ModelName]
	if !ok {
		return 0, "", false
	}
	if len(config.ModelNames) > 0 {
		matched := false
		for _, modelName := range config.ModelNames {
			if modelName == row.ModelName {
				matched = true
				break
			}
		}
		if !matched {
			return 0, "", false
		}
	}

	groupRatio, ok := profitBoardResolveGroupRatio(pricing.EnableGroup, config.Group, groupRatios)
	if !ok {
		return 0, "", false
	}
	priceFactor := profitBoardPriceFactor(config.UseRechargePrice)
	source := "site_model_standard"
	if config.UseRechargePrice {
		source = "site_model_recharge"
	}
	if pricing.QuotaType == 1 {
		return pricing.ModelPrice*groupRatio*priceFactor + config.FixedAmount, source, true
	}

	baseInputPrice := pricing.ModelRatio * 2 * groupRatio * priceFactor
	cacheReadPrice := 0.0
	if pricing.SupportsCacheRead {
		cacheReadPrice = baseInputPrice * pricing.CacheRatio
	}
	cacheCreationPrice := 0.0
	if pricing.SupportsCacheCreation {
		cacheCreationPrice = baseInputPrice * pricing.CacheCreationRatio
	}
	outputPrice := pricing.ModelRatio * pricing.CompletionRatio * 2 * groupRatio * priceFactor

	return float64(inputTokens)*baseInputPrice/1_000_000 +
		float64(row.CompletionTokens)*outputPrice/1_000_000 +
		float64(cacheReadTokens)*cacheReadPrice/1_000_000 +
		float64(cacheCreationTokens)*cacheCreationPrice/1_000_000 +
		config.FixedAmount, source, true
}

func profitBoardSiteRevenueUSD(
	row profitBoardLogRow,
	inputTokens int,
	cacheReadTokens int,
	cacheCreationTokens int,
	config ProfitBoardTokenPricingConfig,
	pricingMap map[string]Pricing,
	groupRatios map[string]float64,
) (float64, string, bool) {
	if config.PricingMode == ProfitBoardSitePricingSiteModel {
		if amount, source, ok := profitBoardSiteModelRevenueUSD(row, inputTokens, cacheReadTokens, cacheCreationTokens, config, pricingMap, groupRatios); ok {
			return amount, source, true
		}
		amount := profitBoardTokenMoneyUSD(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, config)
		if amount == 0 {
			return 0, "site_model_missing", false
		}
		return amount, "manual_fallback", true
	}
	amount := profitBoardTokenMoneyUSD(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, config)
	return amount, "manual", true
}

func profitBoardUpstreamCostUSD(
	row profitBoardLogRow,
	other profitBoardOtherInfo,
	inputTokens int,
	cacheReadTokens int,
	cacheCreationTokens int,
	config ProfitBoardTokenPricingConfig,
) (float64, string, bool) {
	_ = other
	return profitBoardTokenMoneyUSD(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, config), "manual", true
}

type profitBoardPreparedRow struct {
	Batch               ProfitBoardBatchInfo
	Row                 profitBoardLogRow
	Other               profitBoardOtherInfo
	InputTokens         int
	CacheReadTokens     int
	CacheCreationTokens int
}

func iterateProfitBoardRows(query ProfitBoardQuery, batches []ProfitBoardBatchInfo, callback func(prepared profitBoardPreparedRow) error) error {
	for _, batch := range batches {
		tx := LOG_DB.Table("logs").
			Select("id, created_at, request_id, channel_id, model_name, quota, prompt_tokens, completion_tokens, other").
			Where("type = ?", LogTypeConsume).
			Where("channel_id IN ?", batch.ChannelIDs)
		if query.StartTimestamp > 0 {
			tx = tx.Where("created_at >= ?", query.StartTimestamp)
		}
		if query.EndTimestamp > 0 {
			tx = tx.Where("created_at <= ?", query.EndTimestamp)
		}
		tx = tx.Order("id desc")

		rows, err := tx.Rows()
		if err != nil {
			return err
		}

		var rowIterErr error
		for rows.Next() {
			var row profitBoardLogRow
			if err := LOG_DB.ScanRows(rows, &row); err != nil {
				rowIterErr = err
				break
			}
			other := profitBoardOtherInfo{}
			if row.Other != "" {
				_ = common.UnmarshalJsonStr(row.Other, &other)
			}
			cacheReadTokens := other.CacheTokens
			if cacheReadTokens < 0 {
				cacheReadTokens = 0
			}
			cacheCreationTokens := sumCacheCreationTokens(other.tokenUsageOtherInfo)
			if cacheCreationTokens < 0 {
				cacheCreationTokens = 0
			}
			inputTokens := normalizeInputTokens(row.PromptTokens, cacheReadTokens, cacheCreationTokens, other.tokenUsageOtherInfo)

			if err := callback(profitBoardPreparedRow{
				Batch:               batch,
				Row:                 row,
				Other:               other,
				InputTokens:         inputTokens,
				CacheReadTokens:     cacheReadTokens,
				CacheCreationTokens: cacheCreationTokens,
			}); err != nil {
				rowIterErr = err
				break
			}
		}
		if rowIterErr == nil {
			rowIterErr = rows.Err()
		}
		if closeErr := rows.Close(); rowIterErr == nil && closeErr != nil {
			rowIterErr = closeErr
		}
		if rowIterErr != nil {
			return rowIterErr
		}
	}
	return nil
}

func generateProfitBoardReport(query ProfitBoardQuery, applyDetailLimit bool) (*ProfitBoardReport, error) {
	normalizedQuery, signature, err := normalizeProfitBoardQuery(query)
	if err != nil {
		return nil, err
	}

	resolvedBatches, err := resolveProfitBoardBatches(normalizedQuery.Batches)
	if err != nil {
		return nil, err
	}

	pricingMap := make(map[string]Pricing)
	for _, pricing := range GetPricing() {
		pricingMap[pricing.ModelName] = pricing
	}
	groupRatios := ratio_setting.GetGroupRatioCopy()
	sitePriceFactor, sitePriceFactorNote := profitBoardPriceFactorMeta(normalizedQuery.Site.UseRechargePrice)

	report := &ProfitBoardReport{
		Signature:      signature,
		Batches:        resolvedBatches,
		BatchSummaries: make([]ProfitBoardBatchSummary, 0, len(resolvedBatches)),
		Meta: ProfitBoardMeta{
			SiteUseRechargePrice:      normalizedQuery.Site.UseRechargePrice,
			SitePriceFactor:           roundProfitBoardAmount(sitePriceFactor),
			SitePriceFactorNote:       sitePriceFactorNote,
			GeneratedAt:               common.GetTimestamp(),
			FixedTotalAmountScope:     "period_only",
			FixedAmountAllocationMode: "request_count",
			LegacyUpstreamFixedAmount: profitBoardLegacyFixedAmountEnabled(normalizedQuery.Upstream),
			LegacySiteFixedAmount:     profitBoardLegacyFixedAmountEnabled(normalizedQuery.Site),
		},
		DetailRows: make([]ProfitBoardDetailRow, 0),
	}

	timeBuckets := make(map[string]*ProfitBoardTimeseriesPoint)
	channelBreakdown := make(map[string]*ProfitBoardBreakdownItem)
	modelBreakdown := make(map[string]*ProfitBoardBreakdownItem)
	batchSummaryMap := make(map[string]*ProfitBoardBatchSummary, len(resolvedBatches))
	channelNameMap := make(map[int]string)
	latestLogId := 0
	latestLogCreatedAt := int64(0)
	for _, batch := range resolvedBatches {
		batchSummaryMap[batch.Id] = &ProfitBoardBatchSummary{
			BatchId:   batch.Id,
			BatchName: batch.Name,
		}
		for _, channel := range batch.ResolvedChannels {
			channelNameMap[channel.Id] = channel.Name
		}
	}

	if err := iterateProfitBoardRows(normalizedQuery, resolvedBatches, func(prepared profitBoardPreparedRow) error {
		row := prepared.Row
		batch := prepared.Batch
		batchSummary := batchSummaryMap[batch.Id]
		if row.Id > latestLogId {
			latestLogId = row.Id
			latestLogCreatedAt = row.CreatedAt
		} else if row.Id == latestLogId && row.CreatedAt > latestLogCreatedAt {
			latestLogCreatedAt = row.CreatedAt
		}
		actualSiteRevenueUSD := float64(row.Quota) / common.QuotaPerUnit
		configuredSiteRevenueUSD, sitePricingSource, sitePricingKnown := profitBoardSiteRevenueUSD(
			row,
			prepared.InputTokens,
			prepared.CacheReadTokens,
			prepared.CacheCreationTokens,
			normalizedQuery.Site,
			pricingMap,
			groupRatios,
		)
		upstreamCostUSD, upstreamCostSource, upstreamCostKnown := profitBoardUpstreamCostUSD(
			row,
			prepared.Other,
			prepared.InputTokens,
			prepared.CacheReadTokens,
			prepared.CacheCreationTokens,
			normalizedQuery.Upstream,
		)

		report.Summary.RequestCount++
		batchSummary.RequestCount++
		if sitePricingKnown && strings.HasPrefix(sitePricingSource, "site_model_") {
			report.Summary.SiteModelMatchCount++
			batchSummary.SiteModelMatchCount++
		}
		if !sitePricingKnown {
			report.Summary.MissingSitePricingCount++
			batchSummary.MissingSitePricingCount++
		}
		if upstreamCostKnown {
			report.Summary.KnownUpstreamCostCount++
			batchSummary.KnownUpstreamCostCount++
			report.Summary.UpstreamCostUSD += upstreamCostUSD
			batchSummary.UpstreamCostUSD += upstreamCostUSD
			switch upstreamCostSource {
			case "returned_cost":
				report.Summary.ReturnedCostCount++
				batchSummary.ReturnedCostCount++
			case "manual":
				report.Summary.ManualCostCount++
				batchSummary.ManualCostCount++
			}
		} else {
			report.Summary.MissingUpstreamCostCount++
			batchSummary.MissingUpstreamCostCount++
		}

		report.Summary.ActualSiteRevenueUSD += actualSiteRevenueUSD
		batchSummary.ActualSiteRevenueUSD += actualSiteRevenueUSD
		if sitePricingKnown {
			report.Summary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
			batchSummary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
		}

		configuredProfitUSD := 0.0
		actualProfitUSD := 0.0
		if upstreamCostKnown && sitePricingKnown {
			configuredProfitUSD = configuredSiteRevenueUSD - upstreamCostUSD
			report.Summary.ConfiguredProfitUSD += configuredProfitUSD
			batchSummary.ConfiguredProfitUSD += configuredProfitUSD
		}
		if upstreamCostKnown {
			actualProfitUSD = actualSiteRevenueUSD - upstreamCostUSD
			report.Summary.ActualProfitUSD += actualProfitUSD
			batchSummary.ActualProfitUSD += actualProfitUSD
		}

		bucketTimestamp, bucketLabel := buildProfitBoardBucket(
			row.CreatedAt,
			normalizedQuery.Granularity,
			normalizedQuery.CustomIntervalMinutes,
		)
		timeKey := fmt.Sprintf("%s:%d", batch.Id, bucketTimestamp)
		point, ok := timeBuckets[timeKey]
		if !ok {
			point = &ProfitBoardTimeseriesPoint{
				BatchId:         batch.Id,
				BatchName:       batch.Name,
				Bucket:          bucketLabel,
				BucketTimestamp: bucketTimestamp,
			}
			timeBuckets[timeKey] = point
		}
		point.RequestCount++
		point.ActualSiteRevenueUSD += actualSiteRevenueUSD
		if sitePricingKnown {
			point.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
		}
		if upstreamCostKnown {
			point.UpstreamCostUSD += upstreamCostUSD
			point.KnownUpstreamCostCount++
			point.ActualProfitUSD += actualProfitUSD
		} else {
			point.MissingUpstreamCostCount++
		}
		if sitePricingKnown && strings.HasPrefix(sitePricingSource, "site_model_") {
			point.SiteModelMatchCount++
		}
		if !sitePricingKnown {
			point.MissingSitePricingCount++
		}
		if upstreamCostKnown && sitePricingKnown {
			point.ConfiguredProfitUSD += configuredProfitUSD
		}

		channelLabel := channelNameMap[row.ChannelId]
		if channelLabel == "" {
			channelLabel = fmt.Sprintf("渠道 #%d", row.ChannelId)
		}
		channelKey := batch.Id + "|" + strconv.Itoa(row.ChannelId)
		channelItem, ok := channelBreakdown[channelKey]
		if !ok {
			channelItem = &ProfitBoardBreakdownItem{
				BatchId:   batch.Id,
				BatchName: batch.Name,
				Key:       strconv.Itoa(row.ChannelId),
				Label:     channelLabel,
			}
			channelBreakdown[channelKey] = channelItem
		}
		channelItem.RequestCount++
		channelItem.ActualSiteRevenueUSD += actualSiteRevenueUSD
		if sitePricingKnown {
			channelItem.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
		}
		if upstreamCostKnown {
			channelItem.UpstreamCostUSD += upstreamCostUSD
			channelItem.KnownUpstreamCostCount++
			channelItem.ActualProfitUSD += actualProfitUSD
		} else {
			channelItem.MissingUpstreamCostCount++
		}
		if upstreamCostKnown && sitePricingKnown {
			channelItem.ConfiguredProfitUSD += configuredProfitUSD
		}

		modelKey := batch.Id + "|" + row.ModelName
		modelItem, ok := modelBreakdown[modelKey]
		if !ok {
			modelItem = &ProfitBoardBreakdownItem{
				BatchId:   batch.Id,
				BatchName: batch.Name,
				Key:       row.ModelName,
				Label:     row.ModelName,
			}
			modelBreakdown[modelKey] = modelItem
		}
		modelItem.RequestCount++
		modelItem.ActualSiteRevenueUSD += actualSiteRevenueUSD
		if sitePricingKnown {
			modelItem.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
		}
		if upstreamCostKnown {
			modelItem.UpstreamCostUSD += upstreamCostUSD
			modelItem.KnownUpstreamCostCount++
			modelItem.ActualProfitUSD += actualProfitUSD
		} else {
			modelItem.MissingUpstreamCostCount++
		}
		if upstreamCostKnown && sitePricingKnown {
			modelItem.ConfiguredProfitUSD += configuredProfitUSD
		}

		report.DetailRows = append(report.DetailRows, ProfitBoardDetailRow{
			Id:                       row.Id,
			BatchId:                  batch.Id,
			BatchName:                batch.Name,
			CreatedAt:                row.CreatedAt,
			RequestId:                row.RequestId,
			ChannelId:                row.ChannelId,
			ChannelName:              channelLabel,
			ModelName:                row.ModelName,
			PromptTokens:             row.PromptTokens,
			CompletionTokens:         row.CompletionTokens,
			InputTokens:              prepared.InputTokens,
			CacheReadTokens:          prepared.CacheReadTokens,
			CacheCreationTokens:      prepared.CacheCreationTokens,
			ActualSiteRevenueUSD:     roundProfitBoardAmount(actualSiteRevenueUSD),
			ConfiguredSiteRevenueUSD: roundProfitBoardAmount(configuredSiteRevenueUSD),
			UpstreamCostUSD:          roundProfitBoardAmount(upstreamCostUSD),
			ConfiguredProfitUSD:      roundProfitBoardAmount(configuredProfitUSD),
			ActualProfitUSD:          roundProfitBoardAmount(actualProfitUSD),
			ConfiguredActualDeltaUSD: roundProfitBoardAmount(configuredSiteRevenueUSD - actualSiteRevenueUSD),
			UpstreamCostKnown:        upstreamCostKnown,
			UpstreamCostSource:       upstreamCostSource,
			SitePricingSource:        sitePricingSource,
			SitePricingKnown:         sitePricingKnown,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if report.Summary.RequestCount > 0 {
		report.Summary.ConfiguredProfitCoverageRate = float64(report.Summary.KnownUpstreamCostCount) / float64(report.Summary.RequestCount)
	}
	if report.Summary.MissingUpstreamCostCount > 0 {
		report.Warnings = append(report.Warnings, "部分日志没有上游返回费用，当前统计已按你的上游费用策略回退或标记为未知")
	}
	if report.Summary.MissingSitePricingCount > 0 {
		report.Warnings = append(report.Warnings, "部分日志没有命中本站模型定价，已按手动价格或零值处理")
	}
	if report.Meta.LegacyUpstreamFixedAmount {
		report.Warnings = append(report.Warnings, "当前上游价格配置仍包含旧版按次固定金额，请确认后改成固定总金额")
	}
	if report.Meta.LegacySiteFixedAmount {
		report.Warnings = append(report.Warnings, "当前本站价格配置仍包含旧版按次固定金额，请确认后改成固定总金额")
	}
	applyProfitBoardFixedTotals(report, normalizedQuery.Site.FixedTotalAmount, normalizedQuery.Upstream.FixedTotalAmount)

	report.BatchSummaries = make([]ProfitBoardBatchSummary, 0, len(batchSummaryMap))
	for _, batch := range resolvedBatches {
		current := *batchSummaryMap[batch.Id]
		if current.RequestCount > 0 {
			current.ConfiguredProfitCoverageRate = float64(current.KnownUpstreamCostCount) / float64(current.RequestCount)
		}
		current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
		current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
		current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
		current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
		current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
		current.ConfiguredProfitCoverageRate = roundProfitBoardAmount(current.ConfiguredProfitCoverageRate)
		report.BatchSummaries = append(report.BatchSummaries, current)
	}

	report.Timeseries = make([]ProfitBoardTimeseriesPoint, 0, len(timeBuckets))
	for _, point := range timeBuckets {
		current := *point
		current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
		current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
		current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
		current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
		current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
		report.Timeseries = append(report.Timeseries, current)
	}
	sort.Slice(report.Timeseries, func(i, j int) bool {
		if report.Timeseries[i].BucketTimestamp == report.Timeseries[j].BucketTimestamp {
			return report.Timeseries[i].BatchName < report.Timeseries[j].BatchName
		}
		return report.Timeseries[i].BucketTimestamp < report.Timeseries[j].BucketTimestamp
	})

	report.ChannelBreakdown = make([]ProfitBoardBreakdownItem, 0, len(channelBreakdown))
	for _, item := range channelBreakdown {
		current := *item
		current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
		current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
		current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
		current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
		current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
		report.ChannelBreakdown = append(report.ChannelBreakdown, current)
	}
	sort.Slice(report.ChannelBreakdown, func(i, j int) bool {
		if report.ChannelBreakdown[i].BatchName != report.ChannelBreakdown[j].BatchName {
			return report.ChannelBreakdown[i].BatchName < report.ChannelBreakdown[j].BatchName
		}
		if report.ChannelBreakdown[i].ConfiguredProfitUSD == report.ChannelBreakdown[j].ConfiguredProfitUSD {
			return report.ChannelBreakdown[i].ActualSiteRevenueUSD > report.ChannelBreakdown[j].ActualSiteRevenueUSD
		}
		return report.ChannelBreakdown[i].ConfiguredProfitUSD > report.ChannelBreakdown[j].ConfiguredProfitUSD
	})

	report.ModelBreakdown = make([]ProfitBoardBreakdownItem, 0, len(modelBreakdown))
	for _, item := range modelBreakdown {
		current := *item
		current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
		current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
		current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
		current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
		current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
		report.ModelBreakdown = append(report.ModelBreakdown, current)
	}
	sort.Slice(report.ModelBreakdown, func(i, j int) bool {
		if report.ModelBreakdown[i].BatchName != report.ModelBreakdown[j].BatchName {
			return report.ModelBreakdown[i].BatchName < report.ModelBreakdown[j].BatchName
		}
		if report.ModelBreakdown[i].ConfiguredProfitUSD == report.ModelBreakdown[j].ConfiguredProfitUSD {
			return report.ModelBreakdown[i].RequestCount > report.ModelBreakdown[j].RequestCount
		}
		return report.ModelBreakdown[i].ConfiguredProfitUSD > report.ModelBreakdown[j].ConfiguredProfitUSD
	})

	sort.Slice(report.DetailRows, func(i, j int) bool {
		if report.DetailRows[i].CreatedAt == report.DetailRows[j].CreatedAt {
			return report.DetailRows[i].Id > report.DetailRows[j].Id
		}
		return report.DetailRows[i].CreatedAt > report.DetailRows[j].CreatedAt
	})
	if applyDetailLimit && len(report.DetailRows) > normalizedQuery.DetailLimit {
		report.DetailRows = report.DetailRows[:normalizedQuery.DetailLimit]
		report.DetailTruncated = true
	}

	report.Summary.ActualSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ActualSiteRevenueUSD)
	report.Summary.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ConfiguredSiteRevenueUSD)
	report.Summary.UpstreamCostUSD = roundProfitBoardAmount(report.Summary.UpstreamCostUSD)
	report.Summary.ConfiguredProfitUSD = roundProfitBoardAmount(report.Summary.ConfiguredProfitUSD)
	report.Summary.ActualProfitUSD = roundProfitBoardAmount(report.Summary.ActualProfitUSD)
	report.Summary.ConfiguredProfitCoverageRate = roundProfitBoardAmount(report.Summary.ConfiguredProfitCoverageRate)
	report.Meta.LatestLogId = latestLogId
	report.Meta.LatestLogCreatedAt = latestLogCreatedAt
	report.Meta.ActivityWatermark = buildProfitBoardActivityWatermark(
		report.Summary.RequestCount,
		latestLogId,
		latestLogCreatedAt,
	)
	return report, nil
}

func GenerateProfitBoardOverview(payload ProfitBoardConfigPayload) (*ProfitBoardReport, error) {
	normalizedBatches, signature, _, err := normalizeProfitBoardBatches(payload.Batches, payload.Selection)
	if err != nil {
		return nil, err
	}
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
	if err := validateProfitBoardPricingConfig(payload.Upstream, false); err != nil {
		return nil, err
	}
	if err := validateProfitBoardPricingConfig(payload.Site, true); err != nil {
		return nil, err
	}

	resolvedBatches, err := resolveProfitBoardBatches(normalizedBatches)
	if err != nil {
		return nil, err
	}
	pricingMap := make(map[string]Pricing)
	for _, pricing := range GetPricing() {
		pricingMap[pricing.ModelName] = pricing
	}
	groupRatios := ratio_setting.GetGroupRatioCopy()
	sitePriceFactor, sitePriceFactorNote := profitBoardPriceFactorMeta(payload.Site.UseRechargePrice)

	report := &ProfitBoardReport{
		Signature:      signature,
		Batches:        resolvedBatches,
		BatchSummaries: make([]ProfitBoardBatchSummary, 0, len(resolvedBatches)),
		Meta: ProfitBoardMeta{
			SiteUseRechargePrice:      payload.Site.UseRechargePrice,
			SitePriceFactor:           roundProfitBoardAmount(sitePriceFactor),
			SitePriceFactorNote:       sitePriceFactorNote,
			GeneratedAt:               common.GetTimestamp(),
			CumulativeScope:           "all_time",
			FixedTotalAmountScope:     "period_only",
			FixedAmountAllocationMode: "request_count",
			UpstreamFixedTotalAmount:  roundProfitBoardAmount(payload.Upstream.FixedTotalAmount),
			SiteFixedTotalAmount:      roundProfitBoardAmount(payload.Site.FixedTotalAmount),
			LegacyUpstreamFixedAmount: profitBoardLegacyFixedAmountEnabled(payload.Upstream),
			LegacySiteFixedAmount:     profitBoardLegacyFixedAmountEnabled(payload.Site),
		},
	}

	batchSummaryMap := make(map[string]*ProfitBoardBatchSummary, len(resolvedBatches))
	latestLogId := 0
	latestLogCreatedAt := int64(0)
	for _, batch := range resolvedBatches {
		batchSummaryMap[batch.Id] = &ProfitBoardBatchSummary{BatchId: batch.Id, BatchName: batch.Name}
	}

	query := ProfitBoardQuery{Batches: normalizedBatches}
	if err := iterateProfitBoardRows(query, resolvedBatches, func(prepared profitBoardPreparedRow) error {
		row := prepared.Row
		batchSummary := batchSummaryMap[prepared.Batch.Id]
		if row.Id > latestLogId {
			latestLogId = row.Id
			latestLogCreatedAt = row.CreatedAt
		} else if row.Id == latestLogId && row.CreatedAt > latestLogCreatedAt {
			latestLogCreatedAt = row.CreatedAt
		}

		actualSiteRevenueUSD := float64(row.Quota) / common.QuotaPerUnit
		configuredSiteRevenueUSD, sitePricingSource, sitePricingKnown := profitBoardSiteRevenueUSD(
			row,
			prepared.InputTokens,
			prepared.CacheReadTokens,
			prepared.CacheCreationTokens,
			payload.Site,
			pricingMap,
			groupRatios,
		)
		upstreamCostUSD, upstreamCostSource, upstreamCostKnown := profitBoardUpstreamCostUSD(
			row,
			prepared.Other,
			prepared.InputTokens,
			prepared.CacheReadTokens,
			prepared.CacheCreationTokens,
			payload.Upstream,
		)

		report.Summary.RequestCount++
		batchSummary.RequestCount++
		if sitePricingKnown && strings.HasPrefix(sitePricingSource, "site_model_") {
			report.Summary.SiteModelMatchCount++
			batchSummary.SiteModelMatchCount++
		}
		if !sitePricingKnown {
			report.Summary.MissingSitePricingCount++
			batchSummary.MissingSitePricingCount++
		}
		if upstreamCostKnown {
			report.Summary.KnownUpstreamCostCount++
			batchSummary.KnownUpstreamCostCount++
			report.Summary.UpstreamCostUSD += upstreamCostUSD
			batchSummary.UpstreamCostUSD += upstreamCostUSD
			switch upstreamCostSource {
			case "returned_cost":
				report.Summary.ReturnedCostCount++
				batchSummary.ReturnedCostCount++
			case "manual":
				report.Summary.ManualCostCount++
				batchSummary.ManualCostCount++
			}
		} else {
			report.Summary.MissingUpstreamCostCount++
			batchSummary.MissingUpstreamCostCount++
		}

		report.Summary.ActualSiteRevenueUSD += actualSiteRevenueUSD
		batchSummary.ActualSiteRevenueUSD += actualSiteRevenueUSD
		if sitePricingKnown {
			report.Summary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
			batchSummary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
		}
		if upstreamCostKnown && sitePricingKnown {
			configuredProfitUSD := configuredSiteRevenueUSD - upstreamCostUSD
			report.Summary.ConfiguredProfitUSD += configuredProfitUSD
			batchSummary.ConfiguredProfitUSD += configuredProfitUSD
		}
		if upstreamCostKnown {
			actualProfitUSD := actualSiteRevenueUSD - upstreamCostUSD
			report.Summary.ActualProfitUSD += actualProfitUSD
			batchSummary.ActualProfitUSD += actualProfitUSD
		}
		return nil
	}); err != nil {
		return nil, err
	}

	for _, summary := range batchSummaryMap {
		summary.ActualSiteRevenueUSD = roundProfitBoardAmount(summary.ActualSiteRevenueUSD)
		summary.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(summary.ConfiguredSiteRevenueUSD)
		summary.UpstreamCostUSD = roundProfitBoardAmount(summary.UpstreamCostUSD)
		summary.ConfiguredProfitUSD = roundProfitBoardAmount(summary.ConfiguredProfitUSD)
		summary.ActualProfitUSD = roundProfitBoardAmount(summary.ActualProfitUSD)
		report.BatchSummaries = append(report.BatchSummaries, *summary)
	}
	sort.Slice(report.BatchSummaries, func(i, j int) bool {
		return report.BatchSummaries[i].BatchName < report.BatchSummaries[j].BatchName
	})
	if report.Summary.RequestCount > 0 {
		report.Summary.ConfiguredProfitCoverageRate = float64(report.Summary.KnownUpstreamCostCount) / float64(report.Summary.RequestCount)
	}
	if report.Summary.MissingUpstreamCostCount > 0 {
		report.Warnings = append(report.Warnings, "累计总览中部分日志没有上游返回费用，当前已按你的上游费用策略回退或标记为未知")
	}
	if report.Summary.MissingSitePricingCount > 0 {
		report.Warnings = append(report.Warnings, "累计总览中部分日志没有命中本站模型定价，已按手动价格或零值处理")
	}
	if report.Meta.LegacyUpstreamFixedAmount {
		report.Warnings = append(report.Warnings, "当前上游价格配置仍包含旧版按次固定金额，请确认后改成固定总金额")
	}
	if report.Meta.LegacySiteFixedAmount {
		report.Warnings = append(report.Warnings, "当前本站价格配置仍包含旧版按次固定金额，请确认后改成固定总金额")
	}
	if payload.Upstream.FixedTotalAmount > 0 || payload.Site.FixedTotalAmount > 0 {
		report.Warnings = append(report.Warnings, "固定总金额只参与时间分析，不计入累计总览")
	}

	report.Summary.ActualSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ActualSiteRevenueUSD)
	report.Summary.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ConfiguredSiteRevenueUSD)
	report.Summary.UpstreamCostUSD = roundProfitBoardAmount(report.Summary.UpstreamCostUSD)
	report.Summary.ConfiguredProfitUSD = roundProfitBoardAmount(report.Summary.ConfiguredProfitUSD)
	report.Summary.ActualProfitUSD = roundProfitBoardAmount(report.Summary.ActualProfitUSD)
	report.Summary.ConfiguredProfitCoverageRate = roundProfitBoardAmount(report.Summary.ConfiguredProfitCoverageRate)
	report.Meta.LatestLogId = latestLogId
	report.Meta.LatestLogCreatedAt = latestLogCreatedAt
	report.Meta.ActivityWatermark = buildProfitBoardActivityWatermark(report.Summary.RequestCount, latestLogId, latestLogCreatedAt)
	return report, nil
}

func GenerateProfitBoardReport(query ProfitBoardQuery) (*ProfitBoardReport, error) {
	return generateProfitBoardReport(query, true)
}

func profitBoardSitePricingSourceLabel(source string) string {
	switch source {
	case "manual":
		return "手动价格"
	case "manual_fallback":
		return "手动价格回退"
	case "site_model_standard":
		return "读取本站模型原价"
	case "site_model_recharge":
		return "读取本站模型充值价"
	case "site_model_missing":
		return "未命中本站模型"
	default:
		return source
	}
}

func profitBoardUpstreamCostSourceLabel(source string) string {
	switch source {
	case "returned_cost":
		return "上游返回费用"
	case "manual":
		return "手动价格回退"
	default:
		return source
	}
}

func profitBoardExcelCell(value string) string {
	return html.EscapeString(value)
}

func profitBoardExcelMoney(value float64, known bool) string {
	if !known {
		return "-"
	}
	return profitBoardCSVMoney(value)
}

func buildProfitBoardExcelHTML(report *ProfitBoardReport) string {
	summary := report.Summary
	summaryRows := [][]string{
		{"请求数", strconv.Itoa(summary.RequestCount)},
		{"本站实际收入", profitBoardCSVMoney(summary.ActualSiteRevenueUSD)},
		{"本站配置收入", profitBoardCSVMoney(summary.ConfiguredSiteRevenueUSD)},
		{"上游费用", profitBoardCSVMoney(summary.UpstreamCostUSD)},
		{"配置利润", profitBoardCSVMoney(summary.ConfiguredProfitUSD)},
		{"实际利润", profitBoardCSVMoney(summary.ActualProfitUSD)},
		{"费用覆盖率", profitBoardCSVMoney(summary.ConfiguredProfitCoverageRate)},
		{"缺失上游费用", strconv.Itoa(summary.MissingUpstreamCostCount)},
		{"命中本站模型价格", strconv.Itoa(summary.SiteModelMatchCount)},
		{"缺失本站价格", strconv.Itoa(summary.MissingSitePricingCount)},
		{"上游返回费用条数", strconv.Itoa(summary.ReturnedCostCount)},
		{"手动回退条数", strconv.Itoa(summary.ManualCostCount)},
	}

	var buffer bytes.Buffer
	buffer.WriteString("<html><head><meta charset=\"utf-8\" /></head><body>")
	buffer.WriteString("<h2>收益看板</h2>")
	buffer.WriteString("<table border=\"1\" cellspacing=\"0\" cellpadding=\"6\" style=\"border-collapse:collapse;margin-bottom:24px;\">")
	for _, row := range summaryRows {
		buffer.WriteString("<tr><td><strong>")
		buffer.WriteString(profitBoardExcelCell(row[0]))
		buffer.WriteString("</strong></td><td>")
		buffer.WriteString(profitBoardExcelCell(row[1]))
		buffer.WriteString("</td></tr>")
	}
	buffer.WriteString("</table>")
	buffer.WriteString("<table border=\"1\" cellspacing=\"0\" cellpadding=\"6\" style=\"border-collapse:collapse;width:100%;\">")
	buffer.WriteString("<thead><tr>")
	headers := []string{
		"时间",
		"批次",
		"请求 ID",
		"渠道",
		"模型",
		"本站实际收入",
		"本站配置收入",
		"配置与实际差值",
		"本站配置来源",
		"上游费用",
		"上游费用来源",
		"配置利润",
		"实际利润",
	}
	for _, header := range headers {
		buffer.WriteString("<th>")
		buffer.WriteString(profitBoardExcelCell(header))
		buffer.WriteString("</th>")
	}
	buffer.WriteString("</tr></thead><tbody>")
	for _, row := range report.DetailRows {
		buffer.WriteString("<tr>")
		values := []string{
			time.Unix(row.CreatedAt, 0).In(time.Local).Format("2006-01-02 15:04:05"),
			row.BatchName,
			row.RequestId,
			row.ChannelName,
			row.ModelName,
			profitBoardCSVMoney(row.ActualSiteRevenueUSD),
			profitBoardCSVMoney(row.ConfiguredSiteRevenueUSD),
			profitBoardCSVMoney(row.ConfiguredActualDeltaUSD),
			profitBoardSitePricingSourceLabel(row.SitePricingSource),
			profitBoardExcelMoney(row.UpstreamCostUSD, row.UpstreamCostKnown),
			profitBoardUpstreamCostSourceLabel(row.UpstreamCostSource),
			profitBoardExcelMoney(row.ConfiguredProfitUSD, row.UpstreamCostKnown && row.SitePricingKnown),
			profitBoardExcelMoney(row.ActualProfitUSD, row.UpstreamCostKnown),
		}
		for index, value := range values {
			buffer.WriteString("<td>")
			if index == 9 || index == 10 {
				if !row.UpstreamCostKnown {
					buffer.WriteString("-")
				} else {
					buffer.WriteString(profitBoardExcelCell(value))
				}
			} else if index == 11 && !(row.UpstreamCostKnown && row.SitePricingKnown) {
				buffer.WriteString("-")
			} else {
				buffer.WriteString(profitBoardExcelCell(value))
			}
			buffer.WriteString("</td>")
		}
		buffer.WriteString("</tr>")
	}
	buffer.WriteString("</tbody></table></body></html>")
	return buffer.String()
}

func ExportProfitBoardCSV(query ProfitBoardQuery) ([]byte, string, error) {
	normalizedQuery, _, err := normalizeProfitBoardQuery(query)
	if err != nil {
		return nil, "", err
	}

	resolvedBatches, err := resolveProfitBoardBatches(normalizedQuery.Batches)
	if err != nil {
		return nil, "", err
	}
	channelNameMap := make(map[int]string)
	for _, batch := range resolvedBatches {
		for _, channel := range batch.ResolvedChannels {
			channelNameMap[channel.Id] = channel.Name
		}
	}

	pricingMap := make(map[string]Pricing)
	for _, pricing := range GetPricing() {
		pricingMap[pricing.ModelName] = pricing
	}
	groupRatios := ratio_setting.GetGroupRatioCopy()

	buffer := &bytes.Buffer{}
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(buffer)
	if err := writer.Write([]string{
		"batch_id",
		"batch_name",
		"request_id",
		"created_at",
		"channel_id",
		"channel_name",
		"model_name",
		"prompt_tokens",
		"completion_tokens",
		"input_tokens",
		"cache_read_tokens",
		"cache_creation_tokens",
		"actual_site_revenue_usd",
		"configured_site_revenue_usd",
		"site_pricing_source",
		"configured_actual_delta_usd",
		"upstream_cost_usd",
		"upstream_cost_source",
		"configured_profit_usd",
		"actual_profit_usd",
	}); err != nil {
		return nil, "", err
	}

	if err := iterateProfitBoardRows(normalizedQuery, resolvedBatches, func(prepared profitBoardPreparedRow) error {
		row := prepared.Row
		actualSiteRevenueUSD := float64(row.Quota) / common.QuotaPerUnit
		configuredSiteRevenueUSD, sitePricingSource, sitePricingKnown := profitBoardSiteRevenueUSD(
			row,
			prepared.InputTokens,
			prepared.CacheReadTokens,
			prepared.CacheCreationTokens,
			normalizedQuery.Site,
			pricingMap,
			groupRatios,
		)
		upstreamCostUSD, upstreamCostSource, upstreamCostKnown := profitBoardUpstreamCostUSD(
			row,
			prepared.Other,
			prepared.InputTokens,
			prepared.CacheReadTokens,
			prepared.CacheCreationTokens,
			normalizedQuery.Upstream,
		)
		configuredProfitUSD := 0.0
		actualProfitUSD := 0.0
		if upstreamCostKnown && sitePricingKnown {
			configuredProfitUSD = configuredSiteRevenueUSD - upstreamCostUSD
		}
		if upstreamCostKnown {
			actualProfitUSD = actualSiteRevenueUSD - upstreamCostUSD
		}
		channelLabel := channelNameMap[row.ChannelId]
		if channelLabel == "" {
			channelLabel = fmt.Sprintf("渠道 #%d", row.ChannelId)
		}

		return writer.Write([]string{
			prepared.Batch.Id,
			prepared.Batch.Name,
			row.RequestId,
			time.Unix(row.CreatedAt, 0).In(time.Local).Format("2006-01-02 15:04:05"),
			strconv.Itoa(row.ChannelId),
			channelLabel,
			row.ModelName,
			strconv.Itoa(row.PromptTokens),
			strconv.Itoa(row.CompletionTokens),
			strconv.Itoa(prepared.InputTokens),
			strconv.Itoa(prepared.CacheReadTokens),
			strconv.Itoa(prepared.CacheCreationTokens),
			profitBoardCSVMoney(actualSiteRevenueUSD),
			profitBoardCSVMoney(configuredSiteRevenueUSD),
			sitePricingSource,
			profitBoardCSVMoney(configuredSiteRevenueUSD - actualSiteRevenueUSD),
			profitBoardCSVMoney(upstreamCostUSD),
			upstreamCostSource,
			profitBoardCSVMoney(configuredProfitUSD),
			profitBoardCSVMoney(actualProfitUSD),
		})
	}); err != nil {
		return nil, "", err
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("profit-board-%s.csv", time.Now().In(time.Local).Format("20060102-150405"))
	return buffer.Bytes(), filename, nil
}

func ExportProfitBoardExcel(query ProfitBoardQuery) ([]byte, string, error) {
	report, err := generateProfitBoardReport(query, false)
	if err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("profit-board-%s.xls", time.Now().In(time.Local).Format("20060102-150405"))
	return []byte(buildProfitBoardExcelHTML(report)), filename, nil
}
