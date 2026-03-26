package model

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
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

type ProfitBoardTokenPricingConfig struct {
	CostSource         string   `json:"cost_source,omitempty"`
	PricingMode        string   `json:"pricing_mode,omitempty"`
	InputPrice         float64  `json:"input_price"`
	OutputPrice        float64  `json:"output_price"`
	CacheReadPrice     float64  `json:"cache_read_price"`
	CacheCreationPrice float64  `json:"cache_creation_price"`
	FixedAmount        float64  `json:"fixed_amount"`
	ModelNames         []string `json:"model_names,omitempty"`
	Group              string   `json:"group,omitempty"`
	UseRechargePrice   bool     `json:"use_recharge_price,omitempty"`
}

type ProfitBoardConfigPayload struct {
	Selection ProfitBoardSelection          `json:"selection"`
	Upstream  ProfitBoardTokenPricingConfig `json:"upstream"`
	Site      ProfitBoardTokenPricingConfig `json:"site"`
}

type ProfitBoardQuery struct {
	Selection      ProfitBoardSelection          `json:"selection"`
	Upstream       ProfitBoardTokenPricingConfig `json:"upstream"`
	Site           ProfitBoardTokenPricingConfig `json:"site"`
	StartTimestamp int64                         `json:"start_timestamp"`
	EndTimestamp   int64                         `json:"end_timestamp"`
	Granularity    string                        `json:"granularity"`
	DetailLimit    int                           `json:"detail_limit"`
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
	UpstreamCostKnown        bool    `json:"upstream_cost_known"`
	UpstreamCostSource       string  `json:"upstream_cost_source"`
	SitePricingSource        string  `json:"site_pricing_source"`
}

type ProfitBoardSelectionInfo struct {
	ScopeType        string                     `json:"scope_type"`
	Signature        string                     `json:"signature"`
	ChannelIDs       []int                      `json:"channel_ids"`
	Tags             []string                   `json:"tags,omitempty"`
	ResolvedChannels []ProfitBoardChannelOption `json:"resolved_channels"`
}

type ProfitBoardReport struct {
	Selection        ProfitBoardSelectionInfo     `json:"selection"`
	Summary          ProfitBoardSummary           `json:"summary"`
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

func validateProfitBoardPricingConfig(config ProfitBoardTokenPricingConfig, isSite bool) error {
	numbers := []float64{
		config.InputPrice,
		config.OutputPrice,
		config.CacheReadPrice,
		config.CacheCreationPrice,
		config.FixedAmount,
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

func GetProfitBoardConfig(selection ProfitBoardSelection) (*ProfitBoardConfigPayload, string, error) {
	normalized, signature, err := normalizeProfitBoardSelection(selection)
	if err != nil {
		return nil, "", err
	}

	config := &ProfitBoardConfig{}
	if err := DB.Where("selection_signature = ?", signature).First(config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ProfitBoardConfigPayload{
				Selection: normalized,
				Upstream: ProfitBoardTokenPricingConfig{
					CostSource: ProfitBoardCostSourceReturnedFirst,
				},
				Site: ProfitBoardTokenPricingConfig{
					PricingMode: ProfitBoardSitePricingManual,
				},
			}, signature, nil
		}
		return nil, "", err
	}

	payload := &ProfitBoardConfigPayload{
		Selection: normalized,
		Upstream: ProfitBoardTokenPricingConfig{
			CostSource: ProfitBoardCostSourceReturnedFirst,
		},
		Site: ProfitBoardTokenPricingConfig{
			PricingMode: ProfitBoardSitePricingManual,
		},
	}
	_ = common.UnmarshalJsonStr(config.UpstreamConfig, &payload.Upstream)
	_ = common.UnmarshalJsonStr(config.SiteConfig, &payload.Site)
	return payload, signature, nil
}

func SaveProfitBoardConfig(payload ProfitBoardConfigPayload) (*ProfitBoardConfigPayload, string, error) {
	normalized, signature, err := normalizeProfitBoardSelection(payload.Selection)
	if err != nil {
		return nil, "", err
	}
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
			SelectionType:      normalized.ScopeType,
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
		record.SelectionType = normalized.ScopeType
		record.SelectionValues = string(selectionBytes)
		record.UpstreamConfig = string(upstreamBytes)
		record.SiteConfig = string(siteBytes)
		record.UpdatedAt = now
		if err := DB.Save(record).Error; err != nil {
			return nil, "", err
		}
	}

	return &ProfitBoardConfigPayload{
		Selection: normalized,
		Upstream:  payload.Upstream,
		Site:      payload.Site,
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
	normalizedSelection, signature, err := normalizeProfitBoardSelection(query.Selection)
	if err != nil {
		return ProfitBoardQuery{}, "", err
	}
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
		} else {
			query.Granularity = "day"
		}
	case "hour", "day":
		query.Granularity = strings.ToLower(strings.TrimSpace(query.Granularity))
	default:
		return ProfitBoardQuery{}, "", errors.New("无效的时间粒度")
	}
	if query.DetailLimit <= 0 {
		query.DetailLimit = 300
	}
	if query.DetailLimit > 2000 {
		query.DetailLimit = 2000
	}
	if query.Upstream.CostSource == "" {
		query.Upstream.CostSource = ProfitBoardCostSourceReturnedFirst
	}
	if query.Site.PricingMode == "" {
		query.Site.PricingMode = ProfitBoardSitePricingManual
	}
	query.Selection = normalizedSelection
	return query, signature, nil
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

func buildProfitBoardBucket(timestamp int64, granularity string) (int64, string) {
	t := time.Unix(timestamp, 0).In(time.Local)
	switch granularity {
	case "hour":
		bucket := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
		return bucket.Unix(), bucket.Format("2006-01-02 15:00")
	default:
		bucket := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		return bucket.Unix(), bucket.Format("2006-01-02")
	}
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

func profitBoardPriceFactor(useRechargePrice bool) float64 {
	if !useRechargePrice {
		return 1
	}
	if operation_setting.USDExchangeRate <= 0 {
		return 1
	}
	return operation_setting.Price / operation_setting.USDExchangeRate
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
	if pricing.QuotaType == 1 {
		return pricing.ModelPrice*groupRatio*priceFactor + config.FixedAmount, "site_model", true
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
		config.FixedAmount, "site_model", true
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
	}
	amount := profitBoardTokenMoneyUSD(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, config)
	if amount == 0 && config.PricingMode == ProfitBoardSitePricingSiteModel {
		return 0, "", false
	}
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
	hasReturnedCost := other.UpstreamCost > 0
	switch config.CostSource {
	case ProfitBoardCostSourceReturnedOnly:
		if hasReturnedCost {
			return other.UpstreamCost, "returned_cost", true
		}
		return 0, "", false
	case ProfitBoardCostSourceManualOnly:
		return profitBoardTokenMoneyUSD(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, config), "manual", true
	default:
		if hasReturnedCost {
			return other.UpstreamCost, "returned_cost", true
		}
		manual := profitBoardTokenMoneyUSD(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, config)
		if manual == 0 {
			return 0, "", false
		}
		return manual, "manual", true
	}
}

func GenerateProfitBoardReport(query ProfitBoardQuery) (*ProfitBoardReport, error) {
	normalizedQuery, signature, err := normalizeProfitBoardQuery(query)
	if err != nil {
		return nil, err
	}

	resolvedChannels, channelIDs, err := resolveProfitBoardChannels(normalizedQuery.Selection)
	if err != nil {
		return nil, err
	}
	channelNameMap := make(map[int]string, len(resolvedChannels))
	for _, channel := range resolvedChannels {
		channelNameMap[channel.Id] = channel.Name
	}

	pricingMap := make(map[string]Pricing)
	for _, pricing := range GetPricing() {
		pricingMap[pricing.ModelName] = pricing
	}
	groupRatios := ratio_setting.GetGroupRatioCopy()

	report := &ProfitBoardReport{
		Selection: ProfitBoardSelectionInfo{
			ScopeType:        normalizedQuery.Selection.ScopeType,
			Signature:        signature,
			ChannelIDs:       channelIDs,
			Tags:             normalizedQuery.Selection.Tags,
			ResolvedChannels: resolvedChannels,
		},
		DetailRows: make([]ProfitBoardDetailRow, 0),
	}

	timeBuckets := make(map[int64]*ProfitBoardTimeseriesPoint)
	channelBreakdown := make(map[string]*ProfitBoardBreakdownItem)
	modelBreakdown := make(map[string]*ProfitBoardBreakdownItem)

	tx := LOG_DB.Table("logs").
		Select("id, created_at, request_id, channel_id, model_name, quota, prompt_tokens, completion_tokens, other").
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? AND created_at <= ?", normalizedQuery.StartTimestamp, normalizedQuery.EndTimestamp).
		Where("channel_id IN ?", channelIDs).
		Order("id desc")

	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var row profitBoardLogRow
		if err := LOG_DB.ScanRows(rows, &row); err != nil {
			return nil, err
		}

		report.Summary.RequestCount++

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

		actualSiteRevenueUSD := float64(row.Quota) / common.QuotaPerUnit
		configuredSiteRevenueUSD, sitePricingSource, sitePricingKnown := profitBoardSiteRevenueUSD(
			row,
			inputTokens,
			cacheReadTokens,
			cacheCreationTokens,
			normalizedQuery.Site,
			pricingMap,
			groupRatios,
		)
		if sitePricingKnown && sitePricingSource == "site_model" {
			report.Summary.SiteModelMatchCount++
		}
		if !sitePricingKnown {
			report.Summary.MissingSitePricingCount++
		}

		upstreamCostUSD, upstreamCostSource, upstreamCostKnown := profitBoardUpstreamCostUSD(
			row,
			other,
			inputTokens,
			cacheReadTokens,
			cacheCreationTokens,
			normalizedQuery.Upstream,
		)
		if upstreamCostKnown {
			report.Summary.KnownUpstreamCostCount++
			report.Summary.UpstreamCostUSD += upstreamCostUSD
			switch upstreamCostSource {
			case "returned_cost":
				report.Summary.ReturnedCostCount++
			case "manual":
				report.Summary.ManualCostCount++
			}
		} else {
			report.Summary.MissingUpstreamCostCount++
		}

		report.Summary.ActualSiteRevenueUSD += actualSiteRevenueUSD
		if sitePricingKnown {
			report.Summary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
		}

		configuredProfitUSD := 0.0
		actualProfitUSD := 0.0
		if upstreamCostKnown && sitePricingKnown {
			configuredProfitUSD = configuredSiteRevenueUSD - upstreamCostUSD
			report.Summary.ConfiguredProfitUSD += configuredProfitUSD
		}
		if upstreamCostKnown {
			actualProfitUSD = actualSiteRevenueUSD - upstreamCostUSD
			report.Summary.ActualProfitUSD += actualProfitUSD
		}

		bucketTimestamp, bucketLabel := buildProfitBoardBucket(row.CreatedAt, normalizedQuery.Granularity)
		point, ok := timeBuckets[bucketTimestamp]
		if !ok {
			point = &ProfitBoardTimeseriesPoint{
				Bucket:          bucketLabel,
				BucketTimestamp: bucketTimestamp,
			}
			timeBuckets[bucketTimestamp] = point
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
		if sitePricingKnown && sitePricingSource == "site_model" {
			point.SiteModelMatchCount++
		}
		if !sitePricingKnown {
			point.MissingSitePricingCount++
		}
		if upstreamCostKnown && sitePricingKnown {
			point.ConfiguredProfitUSD += configuredProfitUSD
		}

		channelKey := strconv.Itoa(row.ChannelId)
		channelItem, ok := channelBreakdown[channelKey]
		if !ok {
			channelItem = &ProfitBoardBreakdownItem{
				Key:   channelKey,
				Label: channelNameMap[row.ChannelId],
			}
			if channelItem.Label == "" {
				channelItem.Label = fmt.Sprintf("渠道 #%d", row.ChannelId)
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

		modelItem, ok := modelBreakdown[row.ModelName]
		if !ok {
			modelItem = &ProfitBoardBreakdownItem{
				Key:   row.ModelName,
				Label: row.ModelName,
			}
			modelBreakdown[row.ModelName] = modelItem
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

		if len(report.DetailRows) < normalizedQuery.DetailLimit {
			report.DetailRows = append(report.DetailRows, ProfitBoardDetailRow{
				Id:                       row.Id,
				CreatedAt:                row.CreatedAt,
				RequestId:                row.RequestId,
				ChannelId:                row.ChannelId,
				ChannelName:              channelNameMap[row.ChannelId],
				ModelName:                row.ModelName,
				PromptTokens:             row.PromptTokens,
				CompletionTokens:         row.CompletionTokens,
				InputTokens:              inputTokens,
				CacheReadTokens:          cacheReadTokens,
				CacheCreationTokens:      cacheCreationTokens,
				ActualSiteRevenueUSD:     roundProfitBoardAmount(actualSiteRevenueUSD),
				ConfiguredSiteRevenueUSD: roundProfitBoardAmount(configuredSiteRevenueUSD),
				UpstreamCostUSD:          roundProfitBoardAmount(upstreamCostUSD),
				ConfiguredProfitUSD:      roundProfitBoardAmount(configuredProfitUSD),
				ActualProfitUSD:          roundProfitBoardAmount(actualProfitUSD),
				UpstreamCostKnown:        upstreamCostKnown,
				UpstreamCostSource:       upstreamCostSource,
				SitePricingSource:        sitePricingSource,
			})
		} else {
			report.DetailTruncated = true
		}
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

	timestamps := make([]int64, 0, len(timeBuckets))
	for timestamp := range timeBuckets {
		timestamps = append(timestamps, timestamp)
	}
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i] < timestamps[j]
	})
	report.Timeseries = make([]ProfitBoardTimeseriesPoint, 0, len(timestamps))
	for _, timestamp := range timestamps {
		point := *timeBuckets[timestamp]
		point.ActualSiteRevenueUSD = roundProfitBoardAmount(point.ActualSiteRevenueUSD)
		point.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(point.ConfiguredSiteRevenueUSD)
		point.UpstreamCostUSD = roundProfitBoardAmount(point.UpstreamCostUSD)
		point.ConfiguredProfitUSD = roundProfitBoardAmount(point.ConfiguredProfitUSD)
		point.ActualProfitUSD = roundProfitBoardAmount(point.ActualProfitUSD)
		report.Timeseries = append(report.Timeseries, point)
	}

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
		if report.ModelBreakdown[i].ConfiguredProfitUSD == report.ModelBreakdown[j].ConfiguredProfitUSD {
			return report.ModelBreakdown[i].RequestCount > report.ModelBreakdown[j].RequestCount
		}
		return report.ModelBreakdown[i].ConfiguredProfitUSD > report.ModelBreakdown[j].ConfiguredProfitUSD
	})

	report.Summary.ActualSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ActualSiteRevenueUSD)
	report.Summary.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ConfiguredSiteRevenueUSD)
	report.Summary.UpstreamCostUSD = roundProfitBoardAmount(report.Summary.UpstreamCostUSD)
	report.Summary.ConfiguredProfitUSD = roundProfitBoardAmount(report.Summary.ConfiguredProfitUSD)
	report.Summary.ActualProfitUSD = roundProfitBoardAmount(report.Summary.ActualProfitUSD)
	report.Summary.ConfiguredProfitCoverageRate = roundProfitBoardAmount(report.Summary.ConfiguredProfitCoverageRate)
	return report, nil
}

func ExportProfitBoardCSV(query ProfitBoardQuery) ([]byte, string, error) {
	normalizedQuery, _, err := normalizeProfitBoardQuery(query)
	if err != nil {
		return nil, "", err
	}

	resolvedChannels, channelIDs, err := resolveProfitBoardChannels(normalizedQuery.Selection)
	if err != nil {
		return nil, "", err
	}
	channelNameMap := make(map[int]string, len(resolvedChannels))
	for _, channel := range resolvedChannels {
		channelNameMap[channel.Id] = channel.Name
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
		"upstream_cost_usd",
		"upstream_cost_source",
		"configured_profit_usd",
		"actual_profit_usd",
	}); err != nil {
		return nil, "", err
	}

	tx := LOG_DB.Table("logs").
		Select("id, created_at, request_id, channel_id, model_name, quota, prompt_tokens, completion_tokens, other").
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? AND created_at <= ?", normalizedQuery.StartTimestamp, normalizedQuery.EndTimestamp).
		Where("channel_id IN ?", channelIDs).
		Order("id desc")

	rows, err := tx.Rows()
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	for rows.Next() {
		var row profitBoardLogRow
		if err := LOG_DB.ScanRows(rows, &row); err != nil {
			return nil, "", err
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

		actualSiteRevenueUSD := float64(row.Quota) / common.QuotaPerUnit
		configuredSiteRevenueUSD, sitePricingSource, sitePricingKnown := profitBoardSiteRevenueUSD(
			row,
			inputTokens,
			cacheReadTokens,
			cacheCreationTokens,
			normalizedQuery.Site,
			pricingMap,
			groupRatios,
		)
		upstreamCostUSD, upstreamCostSource, upstreamCostKnown := profitBoardUpstreamCostUSD(
			row,
			other,
			inputTokens,
			cacheReadTokens,
			cacheCreationTokens,
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

		if err := writer.Write([]string{
			row.RequestId,
			time.Unix(row.CreatedAt, 0).In(time.Local).Format("2006-01-02 15:04:05"),
			strconv.Itoa(row.ChannelId),
			channelNameMap[row.ChannelId],
			row.ModelName,
			strconv.Itoa(row.PromptTokens),
			strconv.Itoa(row.CompletionTokens),
			strconv.Itoa(inputTokens),
			strconv.Itoa(cacheReadTokens),
			strconv.Itoa(cacheCreationTokens),
			profitBoardCSVMoney(actualSiteRevenueUSD),
			profitBoardCSVMoney(configuredSiteRevenueUSD),
			sitePricingSource,
			profitBoardCSVMoney(upstreamCostUSD),
			upstreamCostSource,
			profitBoardCSVMoney(configuredProfitUSD),
			profitBoardCSVMoney(actualProfitUSD),
		}); err != nil {
			return nil, "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("profit-board-%s.csv", time.Now().In(time.Local).Format("20060102-150405"))
	return buffer.Bytes(), filename, nil
}
