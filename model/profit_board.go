package model

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/samber/hot"
	"gorm.io/gorm"
)

const (
	ProfitBoardScopeChannel = "channel"
	ProfitBoardScopeTag     = "tag"
	ProfitBoardScopeBatch   = "batch_set"

	ProfitBoardCostSourceReturnedFirst     = "returned_cost_first"
	ProfitBoardCostSourceReturnedOnly      = "returned_cost_only"
	ProfitBoardCostSourceManualOnly        = "manual_only"
	ProfitBoardUpstreamModeManual          = "manual_rules"
	ProfitBoardUpstreamModeWallet          = "wallet_observer"
	ProfitBoardUpstreamAccountTypeNewAPI   = "newapi"
	ProfitBoardResourceDisplayBoth         = "both"
	ProfitBoardResourceDisplayWallet       = "wallet"
	ProfitBoardResourceDisplaySubscription = "subscription"

	ProfitBoardSitePricingManual       = "manual"
	ProfitBoardSitePricingSiteModel    = "site_model"
	ProfitBoardComboSiteModeManual     = "manual"
	ProfitBoardComboSiteModeSharedSite = "shared_site_model"
	ProfitBoardComboSiteModeLogQuota   = "log_quota"
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
	CreatedAt  int64    `json:"created_at,omitempty"`
}

type ProfitBoardTokenPricingConfig struct {
	CostSource         string   `json:"cost_source,omitempty"`
	UpstreamMode       string   `json:"upstream_mode,omitempty"`
	UpstreamAccountID  int      `json:"upstream_account_id,omitempty"`
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
	PlanID             int      `json:"plan_id,omitempty"`
}

type ProfitBoardModelPricingRule struct {
	ModelName          string  `json:"model_name,omitempty"`
	InputPrice         float64 `json:"input_price"`
	OutputPrice        float64 `json:"output_price"`
	CacheReadPrice     float64 `json:"cache_read_price"`
	CacheCreationPrice float64 `json:"cache_creation_price"`
	IsDefault          bool    `json:"is_default,omitempty"`
	IsCustom           bool    `json:"is_custom,omitempty"`
}

type ProfitBoardSharedSitePricingConfig struct {
	ModelNames       []string `json:"model_names,omitempty"`
	Group            string   `json:"group,omitempty"`
	UseRechargePrice bool     `json:"use_recharge_price,omitempty"`
	PlanID           int      `json:"plan_id,omitempty"`
}

type ProfitBoardComboPricingConfig struct {
	ComboId                  string                             `json:"combo_id"`
	SiteMode                 string                             `json:"site_mode,omitempty"`
	UpstreamMode             string                             `json:"upstream_mode,omitempty"`
	CostSource               string                             `json:"cost_source,omitempty"`
	UpstreamAccountID        int                                `json:"upstream_account_id,omitempty"`
	SiteExchangeRate         float64                            `json:"site_exchange_rate"`
	UpstreamExchangeRate     float64                            `json:"upstream_exchange_rate"`
	SharedSite               ProfitBoardSharedSitePricingConfig `json:"shared_site,omitempty"`
	SiteRules                []ProfitBoardModelPricingRule      `json:"site_rules,omitempty"`
	UpstreamRules            []ProfitBoardModelPricingRule      `json:"upstream_rules,omitempty"`
	SiteFixedTotalAmount     float64                            `json:"site_fixed_total_amount"`
	UpstreamFixedTotalAmount float64                            `json:"upstream_fixed_total_amount"`
	RemoteObserver           ProfitBoardRemoteObserverConfig    `json:"remote_observer,omitempty"`
}

type ProfitBoardRemoteObserverConfig struct {
	Enabled              bool   `json:"enabled,omitempty"`
	BaseURL              string `json:"base_url,omitempty"`
	UserID               int    `json:"user_id,omitempty"`
	AccessToken          string `json:"access_token,omitempty"`
	AccessTokenMasked    string `json:"access_token_masked,omitempty"`
	AccessTokenEncrypted string `json:"access_token_encrypted,omitempty"`
}

type ProfitBoardRemoteSubscriptionSnapshot struct {
	ID             int    `json:"id,omitempty"`
	SubscriptionID int    `json:"subscription_id"`
	PlanID         int    `json:"plan_id"`
	AmountTotal    int64  `json:"amount_total"`
	AmountUsed     int64  `json:"amount_used"`
	LastResetTime  int64  `json:"last_reset_time"`
	NextResetTime  int64  `json:"next_reset_time"`
	StartTime      int64  `json:"start_time"`
	EndTime        int64  `json:"end_time"`
	Status         string `json:"status,omitempty"`
}

type ProfitBoardRemoteObserverState struct {
	BatchId                      string  `json:"batch_id"`
	BatchName                    string  `json:"batch_name"`
	Enabled                      bool    `json:"enabled"`
	Configured                   bool    `json:"configured"`
	Status                       string  `json:"status"`
	ErrorMessage                 string  `json:"error_message,omitempty"`
	LastSyncedAt                 int64   `json:"last_synced_at"`
	LastSuccessAt                int64   `json:"last_success_at"`
	PeriodUsedUSD                float64 `json:"period_used_usd"`
	ObservedCostUSD              float64 `json:"observed_cost_usd"`
	WalletBalanceUSD             float64 `json:"wallet_balance_usd"`
	WalletQuotaUSD               float64 `json:"wallet_quota_usd"`
	WalletUsedTotalUSD           float64 `json:"wallet_used_total_usd"`
	WalletUsedQuotaUSD           float64 `json:"wallet_used_quota_usd"`
	SubscriptionRemainingUSD     float64 `json:"subscription_remaining_quota_usd"`
	SubscriptionTotalQuotaUSD    float64 `json:"subscription_total_quota_usd"`
	SubscriptionUsedQuotaUSD     float64 `json:"subscription_used_quota_usd"`
	SubscriptionCount            int     `json:"subscription_count"`
	SubscriptionEarliestExpireAt int64   `json:"subscription_earliest_expire_at"`
	HasSubscriptionData          bool    `json:"has_subscription_data"`
	SubscriptionHasUnlimited     bool    `json:"subscription_has_unlimited"`
	RemoteQuotaPerUnit           float64 `json:"remote_quota_per_unit"`
	QuotaPerUnitMismatch         bool    `json:"quota_per_unit_mismatch"`
	LowBalanceThresholdUSD       float64 `json:"low_balance_threshold_usd,omitempty"`
	LowBalanceAlert              bool    `json:"low_balance_alert,omitempty"`
	BaselineReady                bool    `json:"baseline_ready"`
}

type ProfitBoardUpstreamAccountSubscription struct {
	SubscriptionID    int     `json:"subscription_id"`
	PlanID            int     `json:"plan_id"`
	TotalQuotaUSD     float64 `json:"total_quota_usd"`
	UsedQuotaUSD      float64 `json:"used_quota_usd"`
	RemainingQuotaUSD float64 `json:"remaining_quota_usd"`
	HasUnlimited      bool    `json:"has_unlimited"`
	LastResetTime     int64   `json:"last_reset_time"`
	NextResetTime     int64   `json:"next_reset_time"`
	StartTime         int64   `json:"start_time"`
	EndTime           int64   `json:"end_time"`
	Status            string  `json:"status,omitempty"`
}

type ProfitBoardUpstreamAccountTrendPoint struct {
	Bucket          string  `json:"bucket"`
	BucketTimestamp int64   `json:"bucket_timestamp"`
	PeriodUsedUSD   float64 `json:"period_used_usd"`
}

type ProfitBoardUpstreamAccountTrend struct {
	Account               ProfitBoardUpstreamAccountOption         `json:"account"`
	Points                []ProfitBoardUpstreamAccountTrendPoint   `json:"points"`
	Subscriptions         []ProfitBoardUpstreamAccountSubscription `json:"subscriptions,omitempty"`
	StartTimestamp        int64                                    `json:"start_timestamp"`
	EndTimestamp          int64                                    `json:"end_timestamp"`
	Granularity           string                                   `json:"granularity"`
	CustomIntervalMinutes int                                      `json:"custom_interval_minutes"`
	Warnings              []string                                 `json:"warnings,omitempty"`
}

type ProfitBoardConfigPayload struct {
	Batches         []ProfitBoardBatch                 `json:"batches,omitempty"`
	Selection       ProfitBoardSelection               `json:"selection,omitempty"`
	SharedSite      ProfitBoardSharedSitePricingConfig `json:"shared_site,omitempty"`
	ComboConfigs    []ProfitBoardComboPricingConfig    `json:"combo_configs,omitempty"`
	ExcludedUserIDs []int                              `json:"excluded_user_ids,omitempty"`
	Upstream        ProfitBoardTokenPricingConfig      `json:"upstream"`
	Site            ProfitBoardTokenPricingConfig      `json:"site"`
}

type ProfitBoardQuery struct {
	Batches               []ProfitBoardBatch                 `json:"batches,omitempty"`
	Selection             ProfitBoardSelection               `json:"selection,omitempty"`
	SharedSite            ProfitBoardSharedSitePricingConfig `json:"shared_site,omitempty"`
	ComboConfigs          []ProfitBoardComboPricingConfig    `json:"combo_configs,omitempty"`
	ExcludedUserIDs       []int                              `json:"excluded_user_ids,omitempty"`
	Upstream              ProfitBoardTokenPricingConfig      `json:"upstream"`
	Site                  ProfitBoardTokenPricingConfig      `json:"site"`
	StartTimestamp        int64                              `json:"start_timestamp"`
	EndTimestamp          int64                              `json:"end_timestamp"`
	Granularity           string                             `json:"granularity"`
	CustomIntervalMinutes int                                `json:"custom_interval_minutes,omitempty"`
	IncludeDetails        bool                               `json:"include_details,omitempty"`
	DetailLimit           int                                `json:"detail_limit"`
}

type ProfitBoardChannelOption struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Tag    string `json:"tag,omitempty"`
	Status int    `json:"status,omitempty"`
	Models string `json:"models,omitempty"`
}

type ProfitBoardLocalModelOption struct {
	ModelName             string   `json:"model_name"`
	QuotaType             int      `json:"quota_type"`
	EnableGroups          []string `json:"enable_groups"`
	SupportsCacheRead     bool     `json:"supports_cache_read"`
	SupportsCacheCreation bool     `json:"supports_cache_creation"`
	ModelRatio            float64  `json:"model_ratio"`
	ModelPrice            float64  `json:"model_price"`
	CompletionRatio       float64  `json:"completion_ratio"`
	CacheRatio            float64  `json:"cache_ratio"`
	CacheCreationRatio    float64  `json:"cache_creation_ratio"`
}

type ProfitBoardOptions struct {
	Channels         []ProfitBoardChannelOption         `json:"channels"`
	Tags             []string                           `json:"tags"`
	Groups           []string                           `json:"groups"`
	LocalModels      []ProfitBoardLocalModelOption      `json:"local_models"`
	SiteModels       []string                           `json:"site_models"`
	AdminUsers       []ProfitBoardAdminUserOption       `json:"admin_users"`
	UpstreamAccounts []ProfitBoardUpstreamAccountOption `json:"upstream_accounts"`
}

type ProfitBoardAdminUserOption struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        int    `json:"role"`
}

type ProfitBoardSummary struct {
	RequestCount                 int     `json:"request_count"`
	ActualSiteRevenueUSD         float64 `json:"actual_site_revenue_usd"`
	ConfiguredSiteRevenueUSD     float64 `json:"configured_site_revenue_usd"`
	ConfiguredSiteRevenueCNY     float64 `json:"configured_site_revenue_cny"`
	UpstreamCostUSD              float64 `json:"upstream_cost_usd"`
	UpstreamCostCNY              float64 `json:"upstream_cost_cny"`
	RemoteObservedCostUSD        float64 `json:"remote_observed_cost_usd"`
	ConfiguredProfitUSD          float64 `json:"configured_profit_usd"`
	ConfiguredProfitCNY          float64 `json:"configured_profit_cny"`
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
	ConfiguredSiteRevenueCNY float64 `json:"configured_site_revenue_cny"`
	UpstreamCostUSD          float64 `json:"upstream_cost_usd"`
	UpstreamCostCNY          float64 `json:"upstream_cost_cny"`
	RemoteObservedCostUSD    float64 `json:"remote_observed_cost_usd"`
	ConfiguredProfitUSD      float64 `json:"configured_profit_usd"`
	ConfiguredProfitCNY      float64 `json:"configured_profit_cny"`
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
	ConfiguredSiteRevenueCNY float64 `json:"configured_site_revenue_cny"`
	UpstreamCostUSD          float64 `json:"upstream_cost_usd"`
	UpstreamCostCNY          float64 `json:"upstream_cost_cny"`
	ConfiguredProfitUSD      float64 `json:"configured_profit_usd"`
	ConfiguredProfitCNY      float64 `json:"configured_profit_cny"`
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
	ConfiguredSiteRevenueCNY float64 `json:"configured_site_revenue_cny"`
	UpstreamCostUSD          float64 `json:"upstream_cost_usd"`
	UpstreamCostCNY          float64 `json:"upstream_cost_cny"`
	ConfiguredProfitUSD      float64 `json:"configured_profit_usd"`
	ConfiguredProfitCNY      float64 `json:"configured_profit_cny"`
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
	CreatedAt        int64                      `json:"created_at,omitempty"`
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

type ProfitBoardDetailFilter struct {
	Type    string `json:"type,omitempty"`
	Value   string `json:"value,omitempty"`
	BatchId string `json:"batch_id,omitempty"`
}

type ProfitBoardDetailQuery struct {
	ProfitBoardQuery
	ViewBatchId  string                  `json:"view_batch_id,omitempty"`
	DetailFilter ProfitBoardDetailFilter `json:"detail_filter,omitempty"`
	Page         int                     `json:"page,omitempty"`
	PageSize     int                     `json:"page_size,omitempty"`
}

type ProfitBoardDetailPage struct {
	Rows     []ProfitBoardDetailRow `json:"rows"`
	Total    int                    `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

type ProfitBoardReport struct {
	Signature            string                           `json:"signature"`
	Batches              []ProfitBoardBatchInfo           `json:"batches"`
	BatchSummaries       []ProfitBoardBatchSummary        `json:"batch_summaries"`
	Summary              ProfitBoardSummary               `json:"summary"`
	Meta                 ProfitBoardMeta                  `json:"meta"`
	Timeseries           []ProfitBoardTimeseriesPoint     `json:"timeseries"`
	ChannelBreakdown     []ProfitBoardBreakdownItem       `json:"channel_breakdown"`
	ModelBreakdown       []ProfitBoardBreakdownItem       `json:"model_breakdown"`
	DetailRows           []ProfitBoardDetailRow           `json:"detail_rows"`
	DetailTruncated      bool                             `json:"detail_truncated"`
	RemoteObserverStates []ProfitBoardRemoteObserverState `json:"remote_observer_states,omitempty"`
	Warnings             []string                         `json:"warnings,omitempty"`
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
	UserId           int    `gorm:"column:user_id"`
	CreatedAt        int64  `gorm:"column:created_at"`
	RequestId        string `gorm:"column:request_id"`
	ChannelId        int    `gorm:"column:channel_id"`
	ModelName        string `gorm:"column:model_name"`
	Quota            int    `gorm:"column:quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens"`
	Other            string `gorm:"column:other"`
}

type profitBoardResolvedComboPricing struct {
	ComboId                  string
	SiteMode                 string
	UpstreamMode             string
	CostSource               string
	UpstreamAccountID        int
	SiteExchangeRate         float64
	UpstreamExchangeRate     float64
	SharedSite               ProfitBoardSharedSitePricingConfig
	SiteRules                []ProfitBoardModelPricingRule
	UpstreamRules            []ProfitBoardModelPricingRule
	SiteFixedTotalAmount     float64
	UpstreamFixedTotalAmount float64
}

type profitBoardPersistedSiteConfig struct {
	LegacySite      ProfitBoardTokenPricingConfig      `json:"legacy_site"`
	SharedSite      ProfitBoardSharedSitePricingConfig `json:"shared_site,omitempty"`
	ComboConfigs    []ProfitBoardComboPricingConfig    `json:"combo_configs,omitempty"`
	ExcludedUserIDs []int                              `json:"excluded_user_ids,omitempty"`
}

// Sentinel errors for i18n translation
var (
	ErrProfitBoardNoChannel                 = errors.New("profit_board:no_channel")
	ErrProfitBoardNoTag                     = errors.New("profit_board:no_tag")
	ErrProfitBoardInvalidScopeType          = errors.New("profit_board:invalid_scope_type")
	ErrProfitBoardRuleMustSpecifyModel      = errors.New("profit_board:rule_must_specify_model")
	ErrProfitBoardRuleOnlyOneDefault        = errors.New("profit_board:rule_only_one_default")
	ErrProfitBoardRuleNonNegative           = errors.New("profit_board:rule_non_negative")
	ErrProfitBoardComboMissingId            = errors.New("profit_board:combo_missing_id")
	ErrProfitBoardComboDuplicate            = errors.New("profit_board:combo_duplicate")
	ErrProfitBoardInvalidSitePricingMode    = errors.New("profit_board:invalid_site_pricing_mode")
	ErrProfitBoardComboSiteNonNegative      = errors.New("profit_board:combo_site_non_negative")
	ErrProfitBoardComboUpstreamNonNegative  = errors.New("profit_board:combo_upstream_non_negative")
	ErrProfitBoardComboSiteExchangeRate     = errors.New("profit_board:combo_site_exchange_rate")
	ErrProfitBoardComboUpstreamExchangeRate = errors.New("profit_board:combo_upstream_exchange_rate")
	ErrProfitBoardNoBatch                   = errors.New("profit_board:no_batch")
	ErrProfitBoardBatchDuplicate            = errors.New("profit_board:batch_duplicate")
	ErrProfitBoardPriceNonNegative          = errors.New("profit_board:price_non_negative")
	ErrProfitBoardInvalidCostSource         = errors.New("profit_board:invalid_cost_source")
	ErrProfitBoardInvalidUpstreamMode       = errors.New("profit_board:invalid_upstream_mode")
	ErrProfitBoardWalletRequireAccount      = errors.New("profit_board:wallet_require_account")
	ErrProfitBoardInvalidSiteSource         = errors.New("profit_board:invalid_site_source")
	ErrProfitBoardAccountInUse              = errors.New("profit_board:account_in_use")
	ErrProfitBoardEndBeforeStart            = errors.New("profit_board:end_before_start")
	ErrProfitBoardCustomGranularityMin      = errors.New("profit_board:custom_granularity_min")
	ErrProfitBoardCustomGranularityMax      = errors.New("profit_board:custom_granularity_max")
	ErrProfitBoardInvalidGranularity        = errors.New("profit_board:invalid_granularity")
	ErrProfitBoardChannelNotExist           = errors.New("profit_board:channel_not_exist")
	ErrProfitBoardTagNoChannel              = errors.New("profit_board:tag_no_channel")
	ErrProfitBoardChannelDuplicateBatch     = errors.New("profit_board:channel_duplicate_batch")
)

var (
	profitBoardReportCacheOnce sync.Once
	profitBoardReportCache     *cachex.HybridCache[ProfitBoardReport]
)

func profitBoardReportCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("PROFIT_BOARD_REPORT_CACHE_TTL", 60)
	if ttlSeconds <= 0 {
		ttlSeconds = 60
	}
	return time.Duration(ttlSeconds) * time.Second
}

func profitBoardReportCacheCapacity() int {
	capacity := common.GetEnvOrDefault("PROFIT_BOARD_REPORT_CACHE_CAP", 128)
	if capacity <= 0 {
		capacity = 128
	}
	return capacity
}

func getProfitBoardReportCache() *cachex.HybridCache[ProfitBoardReport] {
	profitBoardReportCacheOnce.Do(func() {
		ttl := profitBoardReportCacheTTL()
		profitBoardReportCache = cachex.NewHybridCache[ProfitBoardReport](cachex.HybridCacheConfig[ProfitBoardReport]{
			Namespace: cachex.Namespace("profit_board_report:v2"),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[ProfitBoardReport]{},
			Memory: func() *hot.HotCache[string, ProfitBoardReport] {
				return hot.NewHotCache[string, ProfitBoardReport](hot.LRU, profitBoardReportCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return profitBoardReportCache
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
			return ProfitBoardSelection{}, "", ErrProfitBoardNoChannel
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
			return ProfitBoardSelection{}, "", ErrProfitBoardNoTag
		}
		return ProfitBoardSelection{
			ScopeType: scopeType,
			Tags:      tags,
		}, scopeType + ":" + strings.Join(tags, "|"), nil
	default:
		return ProfitBoardSelection{}, "", ErrProfitBoardInvalidScopeType
	}
}

func normalizeProfitBoardExcludedUserIDs(userIDs []int) []int {
	if len(userIDs) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(userIDs))
	normalized := make([]int, 0, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		normalized = append(normalized, userID)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Ints(normalized)
	return normalized
}

func profitBoardExcludedUserSet(userIDs []int) map[int]struct{} {
	if len(userIDs) == 0 {
		return nil
	}
	userSet := make(map[int]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID > 0 {
			userSet[userID] = struct{}{}
		}
	}
	if len(userSet) == 0 {
		return nil
	}
	return userSet
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

func profitBoardLegacyRuleFromTokenConfig(config ProfitBoardTokenPricingConfig) []ProfitBoardModelPricingRule {
	if config.InputPrice == 0 &&
		config.OutputPrice == 0 &&
		config.CacheReadPrice == 0 &&
		config.CacheCreationPrice == 0 {
		return nil
	}
	return []ProfitBoardModelPricingRule{
		{
			IsDefault:          true,
			InputPrice:         clampProfitBoardNumber(config.InputPrice),
			OutputPrice:        clampProfitBoardNumber(config.OutputPrice),
			CacheReadPrice:     clampProfitBoardNumber(config.CacheReadPrice),
			CacheCreationPrice: clampProfitBoardNumber(config.CacheCreationPrice),
		},
	}
}

func clampProfitBoardNumber(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	return value
}

func normalizeProfitBoardModelPricingRules(rules []ProfitBoardModelPricingRule, legacyFallback ProfitBoardTokenPricingConfig) []ProfitBoardModelPricingRule {
	if rules == nil {
		return profitBoardLegacyRuleFromTokenConfig(legacyFallback)
	}
	normalized := make([]ProfitBoardModelPricingRule, 0, len(rules)+1)
	seen := make(map[string]struct{}, len(rules))
	defaultAdded := false
	for _, rule := range rules {
		modelName := strings.TrimSpace(rule.ModelName)
		isDefault := rule.IsDefault || modelName == "*"
		if isDefault {
			if defaultAdded {
				continue
			}
			defaultAdded = true
			normalized = append(normalized, ProfitBoardModelPricingRule{
				IsDefault:          true,
				IsCustom:           rule.IsCustom,
				InputPrice:         clampProfitBoardNumber(rule.InputPrice),
				OutputPrice:        clampProfitBoardNumber(rule.OutputPrice),
				CacheReadPrice:     clampProfitBoardNumber(rule.CacheReadPrice),
				CacheCreationPrice: clampProfitBoardNumber(rule.CacheCreationPrice),
			})
			continue
		}
		if modelName == "" {
			continue
		}
		lowerName := strings.ToLower(modelName)
		if _, ok := seen[lowerName]; ok {
			continue
		}
		seen[lowerName] = struct{}{}
		normalized = append(normalized, ProfitBoardModelPricingRule{
			ModelName:          modelName,
			IsCustom:           rule.IsCustom,
			InputPrice:         clampProfitBoardNumber(rule.InputPrice),
			OutputPrice:        clampProfitBoardNumber(rule.OutputPrice),
			CacheReadPrice:     clampProfitBoardNumber(rule.CacheReadPrice),
			CacheCreationPrice: clampProfitBoardNumber(rule.CacheCreationPrice),
		})
	}
	return normalized
}

func validateProfitBoardModelPricingRules(rules []ProfitBoardModelPricingRule) error {
	defaultCount := 0
	for _, rule := range rules {
		if strings.TrimSpace(rule.ModelName) == "" && !rule.IsDefault {
			return ErrProfitBoardRuleMustSpecifyModel
		}
		if rule.IsDefault {
			defaultCount++
			if defaultCount > 1 {
				return ErrProfitBoardRuleOnlyOneDefault
			}
		}
		numbers := []float64{
			rule.InputPrice,
			rule.OutputPrice,
			rule.CacheReadPrice,
			rule.CacheCreationPrice,
		}
		for _, num := range numbers {
			if math.IsNaN(num) || math.IsInf(num, 0) || num < 0 {
				return ErrProfitBoardRuleNonNegative
			}
		}
	}
	return nil
}

func normalizeProfitBoardSharedSiteConfig(config ProfitBoardSharedSitePricingConfig, legacySite ProfitBoardTokenPricingConfig) ProfitBoardSharedSitePricingConfig {
	if len(config.ModelNames) == 0 && len(legacySite.ModelNames) > 0 {
		config.ModelNames = append([]string(nil), legacySite.ModelNames...)
	}
	if config.Group == "" {
		config.Group = strings.TrimSpace(legacySite.Group)
	}
	if !config.UseRechargePrice {
		config.UseRechargePrice = legacySite.UseRechargePrice
	}
	if config.PlanID == 0 && legacySite.PlanID > 0 {
		config.PlanID = legacySite.PlanID
	}
	// 互斥：充值价和套餐价不能同时启用
	if config.UseRechargePrice && config.PlanID > 0 {
		config.PlanID = 0
	}
	// 验证套餐存在
	if config.PlanID > 0 {
		if plan, err := GetSubscriptionPlanById(config.PlanID); err != nil || plan == nil {
			config.PlanID = 0
		}
	}
	seen := make(map[string]struct{}, len(config.ModelNames))
	modelNames := make([]string, 0, len(config.ModelNames))
	for _, modelName := range config.ModelNames {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		lowerName := strings.ToLower(modelName)
		if _, ok := seen[lowerName]; ok {
			continue
		}
		seen[lowerName] = struct{}{}
		modelNames = append(modelNames, modelName)
	}
	sort.Strings(modelNames)
	config.ModelNames = modelNames
	config.Group = strings.TrimSpace(config.Group)
	return config
}

func profitBoardSharedSiteConfigEmpty(config ProfitBoardSharedSitePricingConfig) bool {
	return len(config.ModelNames) == 0 && strings.TrimSpace(config.Group) == "" && !config.UseRechargePrice && config.PlanID == 0
}

func defaultProfitBoardComboSiteMode(sharedSite ProfitBoardSharedSitePricingConfig, legacySite ProfitBoardTokenPricingConfig) string {
	if legacySite.PricingMode == ProfitBoardSitePricingSiteModel || len(sharedSite.ModelNames) > 0 {
		return ProfitBoardComboSiteModeSharedSite
	}
	return ProfitBoardComboSiteModeManual
}

func normalizeProfitBoardComboConfigs(batches []ProfitBoardBatch, comboConfigs []ProfitBoardComboPricingConfig, sharedSite ProfitBoardSharedSitePricingConfig, legacySite ProfitBoardTokenPricingConfig, legacyUpstream ProfitBoardTokenPricingConfig) []ProfitBoardComboPricingConfig {
	configMap := make(map[string]ProfitBoardComboPricingConfig, len(comboConfigs))
	legacyCostSource := normalizeProfitBoardCostSource(legacyUpstream.CostSource)
	if legacyCostSource == "" {
		legacyCostSource = ProfitBoardCostSourceManualOnly
	}
	for _, config := range comboConfigs {
		comboID := strings.TrimSpace(config.ComboId)
		if comboID == "" {
			continue
		}
		config.ComboId = comboID
		config.SiteMode = strings.ToLower(strings.TrimSpace(config.SiteMode))
		switch config.SiteMode {
		case "", ProfitBoardComboSiteModeManual, ProfitBoardComboSiteModeSharedSite, ProfitBoardComboSiteModeLogQuota:
		default:
			config.SiteMode = ProfitBoardComboSiteModeManual
		}
		switch strings.ToLower(strings.TrimSpace(config.UpstreamMode)) {
		case ProfitBoardUpstreamModeWallet:
			config.UpstreamMode = ProfitBoardUpstreamModeWallet
		default:
			if strings.TrimSpace(legacyUpstream.UpstreamMode) == ProfitBoardUpstreamModeWallet {
				config.UpstreamMode = ProfitBoardUpstreamModeWallet
			} else {
				config.UpstreamMode = ProfitBoardUpstreamModeManual
			}
		}
		config.CostSource = normalizeProfitBoardCostSource(config.CostSource)
		if config.CostSource == "" {
			config.CostSource = legacyCostSource
		}
		if config.UpstreamMode != ProfitBoardUpstreamModeWallet {
			config.UpstreamAccountID = 0
		} else if config.UpstreamAccountID <= 0 {
			config.UpstreamAccountID = legacyUpstream.UpstreamAccountID
		}
		config.SiteExchangeRate = normalizeProfitBoardExchangeRate(config.SiteExchangeRate)
		config.UpstreamExchangeRate = normalizeProfitBoardExchangeRate(config.UpstreamExchangeRate)
		if profitBoardSharedSiteConfigEmpty(config.SharedSite) {
			config.SharedSite = normalizeProfitBoardSharedSiteConfig(sharedSite, legacySite)
		} else {
			config.SharedSite = normalizeProfitBoardSharedSiteConfig(config.SharedSite, legacySite)
		}
		config.SiteRules = normalizeProfitBoardModelPricingRules(config.SiteRules, legacySite)
		config.UpstreamRules = normalizeProfitBoardModelPricingRules(config.UpstreamRules, legacyUpstream)
		config.SiteFixedTotalAmount = clampProfitBoardNumber(config.SiteFixedTotalAmount)
		config.UpstreamFixedTotalAmount = clampProfitBoardNumber(config.UpstreamFixedTotalAmount)
		config.RemoteObserver = normalizeProfitBoardRemoteObserverConfig(config.RemoteObserver)
		configMap[comboID] = config
	}

	normalized := make([]ProfitBoardComboPricingConfig, 0, len(batches))
	for _, batch := range batches {
		config, ok := configMap[batch.Id]
		if !ok {
			config = ProfitBoardComboPricingConfig{
				ComboId:                  batch.Id,
				SiteMode:                 defaultProfitBoardComboSiteMode(sharedSite, legacySite),
				UpstreamMode:             legacyUpstream.UpstreamMode,
				CostSource:               legacyCostSource,
				UpstreamAccountID:        legacyUpstream.UpstreamAccountID,
				SiteExchangeRate:         1,
				UpstreamExchangeRate:     1,
				SharedSite:               normalizeProfitBoardSharedSiteConfig(sharedSite, legacySite),
				SiteRules:                normalizeProfitBoardModelPricingRules(nil, legacySite),
				UpstreamRules:            normalizeProfitBoardModelPricingRules(nil, legacyUpstream),
				SiteFixedTotalAmount:     clampProfitBoardNumber(legacySite.FixedTotalAmount),
				UpstreamFixedTotalAmount: clampProfitBoardNumber(legacyUpstream.FixedTotalAmount),
			}
		}
		if config.SiteMode == "" {
			config.SiteMode = defaultProfitBoardComboSiteMode(sharedSite, legacySite)
		}
		if config.UpstreamMode == "" {
			config.UpstreamMode = legacyUpstream.UpstreamMode
		}
		if config.UpstreamMode == "" {
			config.UpstreamMode = ProfitBoardUpstreamModeManual
		}
		config.CostSource = normalizeProfitBoardCostSource(config.CostSource)
		if config.CostSource == "" {
			config.CostSource = legacyCostSource
		}
		if config.UpstreamMode != ProfitBoardUpstreamModeWallet {
			config.UpstreamAccountID = 0
		} else if config.UpstreamAccountID <= 0 {
			config.UpstreamAccountID = legacyUpstream.UpstreamAccountID
		}
		config.SiteExchangeRate = normalizeProfitBoardExchangeRate(config.SiteExchangeRate)
		config.UpstreamExchangeRate = normalizeProfitBoardExchangeRate(config.UpstreamExchangeRate)
		if profitBoardSharedSiteConfigEmpty(config.SharedSite) {
			config.SharedSite = normalizeProfitBoardSharedSiteConfig(sharedSite, legacySite)
		}
		normalized = append(normalized, config)
	}
	return normalized
}

func validateProfitBoardComboConfigs(comboConfigs []ProfitBoardComboPricingConfig) error {
	seen := make(map[string]struct{}, len(comboConfigs))
	for _, config := range comboConfigs {
		comboID := strings.TrimSpace(config.ComboId)
		if comboID == "" {
			return ErrProfitBoardComboMissingId
		}
		if _, ok := seen[comboID]; ok {
			return ErrProfitBoardComboDuplicate
		}
		seen[comboID] = struct{}{}
		switch config.SiteMode {
		case "", ProfitBoardComboSiteModeManual, ProfitBoardComboSiteModeSharedSite, ProfitBoardComboSiteModeLogQuota:
		default:
			return ErrProfitBoardInvalidSitePricingMode
		}
		switch strings.TrimSpace(config.UpstreamMode) {
		case "", ProfitBoardUpstreamModeManual, ProfitBoardUpstreamModeWallet:
		default:
			return ErrProfitBoardInvalidUpstreamMode
		}
		switch normalizeProfitBoardCostSource(config.CostSource) {
		case "", ProfitBoardCostSourceReturnedFirst, ProfitBoardCostSourceReturnedOnly, ProfitBoardCostSourceManualOnly:
		default:
			return ErrProfitBoardInvalidCostSource
		}
		if strings.TrimSpace(config.UpstreamMode) == ProfitBoardUpstreamModeWallet && config.UpstreamAccountID <= 0 {
			return ErrProfitBoardWalletRequireAccount
		}
		if err := validateProfitBoardModelPricingRules(config.SiteRules); err != nil {
			return err
		}
		if err := validateProfitBoardModelPricingRules(config.UpstreamRules); err != nil {
			return err
		}
		if math.IsNaN(config.SiteFixedTotalAmount) || math.IsInf(config.SiteFixedTotalAmount, 0) || config.SiteFixedTotalAmount < 0 {
			return ErrProfitBoardComboSiteNonNegative
		}
		if math.IsNaN(config.UpstreamFixedTotalAmount) || math.IsInf(config.UpstreamFixedTotalAmount, 0) || config.UpstreamFixedTotalAmount < 0 {
			return ErrProfitBoardComboUpstreamNonNegative
		}
		if !validateProfitBoardExchangeRate(config.SiteExchangeRate) {
			return ErrProfitBoardComboSiteExchangeRate
		}
		if !validateProfitBoardExchangeRate(config.UpstreamExchangeRate) {
			return ErrProfitBoardComboUpstreamExchangeRate
		}
		if err := validateProfitBoardRemoteObserverConfig(config.RemoteObserver); err != nil {
			return err
		}
	}
	return nil
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
		CreatedAt:  batch.CreatedAt,
	}, signature, nil
}

func fillProfitBoardBatchCreatedAt(batches []ProfitBoardBatch, fallbackTimestamp int64) []ProfitBoardBatch {
	if len(batches) == 0 {
		return batches
	}
	if fallbackTimestamp <= 0 {
		fallbackTimestamp = common.GetTimestamp()
	}
	filled := make([]ProfitBoardBatch, 0, len(batches))
	for _, batch := range batches {
		current := batch
		if current.CreatedAt <= 0 {
			current.CreatedAt = fallbackTimestamp
		}
		filled = append(filled, current)
	}
	return filled
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
		return nil, "", "", ErrProfitBoardNoBatch
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
			return nil, "", "", ErrProfitBoardBatchDuplicate
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

func profitBoardBatchSelectionSignature(batch ProfitBoardBatch) string {
	_, signature, err := normalizeProfitBoardSelection(ProfitBoardSelection{
		ScopeType:  batch.ScopeType,
		ChannelIDs: batch.ChannelIDs,
		Tags:       batch.Tags,
	})
	if err != nil {
		return ""
	}
	return signature
}

func remapProfitBoardComboConfigsByBatchSelection(currentBatches []ProfitBoardBatch, persistedBatches []ProfitBoardBatch, comboConfigs []ProfitBoardComboPricingConfig) []ProfitBoardComboPricingConfig {
	if len(currentBatches) == 0 || len(persistedBatches) == 0 || len(comboConfigs) == 0 {
		return comboConfigs
	}

	comboByID := make(map[string]ProfitBoardComboPricingConfig, len(comboConfigs))
	for _, config := range comboConfigs {
		comboID := strings.TrimSpace(config.ComboId)
		if comboID == "" {
			continue
		}
		comboByID[comboID] = config
	}

	configBySignature := make(map[string]ProfitBoardComboPricingConfig, len(persistedBatches))
	for _, batch := range persistedBatches {
		signature := profitBoardBatchSelectionSignature(batch)
		if signature == "" {
			continue
		}
		config, ok := comboByID[strings.TrimSpace(batch.Id)]
		if !ok {
			continue
		}
		configBySignature[signature] = config
	}

	remapped := make([]ProfitBoardComboPricingConfig, 0, len(currentBatches))
	for _, batch := range currentBatches {
		signature := profitBoardBatchSelectionSignature(batch)
		if signature == "" {
			continue
		}
		config, ok := configBySignature[signature]
		if !ok {
			continue
		}
		config.ComboId = batch.Id
		remapped = append(remapped, config)
	}

	if len(remapped) == 0 {
		return comboConfigs
	}
	return remapped
}

func normalizeProfitBoardPricingConfig(config ProfitBoardTokenPricingConfig, isSite bool) ProfitBoardTokenPricingConfig {
	if !isSite {
		switch strings.ToLower(strings.TrimSpace(config.UpstreamMode)) {
		case ProfitBoardUpstreamModeWallet:
			config.UpstreamMode = ProfitBoardUpstreamModeWallet
		default:
			config.UpstreamMode = ProfitBoardUpstreamModeManual
		}
		config.CostSource = normalizeProfitBoardCostSource(config.CostSource)
		if config.CostSource == "" {
			config.CostSource = ProfitBoardCostSourceManualOnly
		}
		if config.UpstreamMode != ProfitBoardUpstreamModeWallet {
			config.UpstreamAccountID = 0
		}
	}
	if isSite && config.PricingMode == "" {
		config.PricingMode = ProfitBoardSitePricingManual
	}
	if !isSite {
		config.PricingMode = ""
	}
	return config
}

func normalizeProfitBoardCostSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case ProfitBoardCostSourceReturnedFirst:
		return ProfitBoardCostSourceReturnedFirst
	case ProfitBoardCostSourceReturnedOnly:
		return ProfitBoardCostSourceReturnedOnly
	case "", ProfitBoardCostSourceManualOnly:
		return ProfitBoardCostSourceManualOnly
	default:
		return ""
	}
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
			return ErrProfitBoardPriceNonNegative
		}
	}
	switch config.CostSource {
	case "", ProfitBoardCostSourceReturnedFirst, ProfitBoardCostSourceReturnedOnly, ProfitBoardCostSourceManualOnly:
	default:
		return ErrProfitBoardInvalidCostSource
	}
	if !isSite {
		switch config.UpstreamMode {
		case "", ProfitBoardUpstreamModeManual, ProfitBoardUpstreamModeWallet:
		default:
			return ErrProfitBoardInvalidUpstreamMode
		}
		if config.UpstreamMode == ProfitBoardUpstreamModeWallet && config.UpstreamAccountID <= 0 {
			return ErrProfitBoardWalletRequireAccount
		}
		return nil
	}
	switch config.PricingMode {
	case "", ProfitBoardSitePricingManual, ProfitBoardSitePricingSiteModel:
		return nil
	default:
		return ErrProfitBoardInvalidSiteSource
	}
}

func GetProfitBoardConfig(batches []ProfitBoardBatch, selection ProfitBoardSelection) (*ProfitBoardConfigPayload, string, error) {
	normalized, signature, _, err := normalizeProfitBoardBatches(batches, selection)
	if err != nil {
		return nil, "", err
	}
	explicitBatches := len(batches) > 0

	defaultUpstream := normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
		CostSource: ProfitBoardCostSourceManualOnly,
	}, false)
	defaultSite := normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
		PricingMode: ProfitBoardSitePricingManual,
	}, true)

	config := &ProfitBoardConfig{}
	if err := DB.Where("selection_signature = ?", signature).First(config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			payload := &ProfitBoardConfigPayload{
				Batches:         normalized,
				SharedSite:      normalizeProfitBoardSharedSiteConfig(ProfitBoardSharedSitePricingConfig{}, defaultSite),
				ExcludedUserIDs: nil,
				ComboConfigs: normalizeProfitBoardComboConfigs(
					normalized,
					nil,
					normalizeProfitBoardSharedSiteConfig(ProfitBoardSharedSitePricingConfig{}, defaultSite),
					defaultSite,
					defaultUpstream,
				),
				Upstream: defaultUpstream,
				Site:     defaultSite,
			}
			if err := migrateProfitBoardLegacyWalletAccount(payload); err != nil {
				return nil, "", err
			}
			payload.ComboConfigs = stripProfitBoardRemoteObserverSecrets(payload.ComboConfigs)
			return payload, signature, nil
		}
		return nil, "", err
	}

	payload := &ProfitBoardConfigPayload{
		Batches:  normalized,
		Upstream: defaultUpstream,
		Site:     defaultSite,
	}
	persistedBatches := parseProfitBoardConfigBatches(config.SelectionValues)
	if !explicitBatches && len(persistedBatches) > 0 {
		payload.Batches = persistedBatches
	}
	_ = common.UnmarshalJsonStr(config.UpstreamConfig, &payload.Upstream)
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	persistedSite := profitBoardPersistedSiteConfig{}
	if err := common.UnmarshalJsonStr(config.SiteConfig, &persistedSite); err == nil &&
		(len(persistedSite.ComboConfigs) > 0 ||
			len(persistedSite.SharedSite.ModelNames) > 0 ||
			len(persistedSite.ExcludedUserIDs) > 0 ||
			persistedSite.LegacySite.PricingMode != "" ||
			persistedSite.LegacySite.Group != "" ||
			len(persistedSite.LegacySite.ModelNames) > 0) {
		payload.Site = normalizeProfitBoardPricingConfig(persistedSite.LegacySite, true)
		payload.SharedSite = normalizeProfitBoardSharedSiteConfig(persistedSite.SharedSite, payload.Site)
		payload.ExcludedUserIDs = normalizeProfitBoardExcludedUserIDs(persistedSite.ExcludedUserIDs)
		payload.ComboConfigs = normalizeProfitBoardComboConfigs(
			payload.Batches,
			remapProfitBoardComboConfigsByBatchSelection(
				payload.Batches,
				persistedBatches,
				persistedSite.ComboConfigs,
			),
			payload.SharedSite,
			payload.Site,
			payload.Upstream,
		)
	} else {
		_ = common.UnmarshalJsonStr(config.SiteConfig, &payload.Site)
		payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
		payload.SharedSite = normalizeProfitBoardSharedSiteConfig(ProfitBoardSharedSitePricingConfig{}, payload.Site)
		payload.ExcludedUserIDs = nil
		payload.ComboConfigs = normalizeProfitBoardComboConfigs(payload.Batches, nil, payload.SharedSite, payload.Site, payload.Upstream)
	}
	if err := migrateProfitBoardLegacyWalletAccount(payload); err != nil {
		return nil, "", err
	}
	payload.ComboConfigs = stripProfitBoardRemoteObserverSecrets(payload.ComboConfigs)
	return payload, signature, nil
}

func SaveProfitBoardConfig(payload ProfitBoardConfigPayload) (*ProfitBoardConfigPayload, string, error) {
	now := common.GetTimestamp()
	normalized, signature, selectionType, err := normalizeProfitBoardBatches(payload.Batches, payload.Selection)
	if err != nil {
		return nil, "", err
	}
	normalized = fillProfitBoardBatchCreatedAt(normalized, now)
	payload.Batches = normalized
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
	payload.SharedSite = normalizeProfitBoardSharedSiteConfig(payload.SharedSite, payload.Site)
	payload.ExcludedUserIDs = normalizeProfitBoardExcludedUserIDs(payload.ExcludedUserIDs)
	payload.ComboConfigs = normalizeProfitBoardComboConfigs(normalized, payload.ComboConfigs, payload.SharedSite, payload.Site, payload.Upstream)
	if err := migrateProfitBoardLegacyWalletAccount(&payload); err != nil {
		return nil, "", err
	}
	if err := validateProfitBoardPricingConfig(payload.Upstream, false); err != nil {
		return nil, "", err
	}
	if err := validateProfitBoardPricingConfig(payload.Site, true); err != nil {
		return nil, "", err
	}
	if err := validateProfitBoardComboConfigs(payload.ComboConfigs); err != nil {
		return nil, "", err
	}
	payload.ComboConfigs, err = prepareProfitBoardRemoteObserverConfigsForStorage(signature, payload.ComboConfigs)
	if err != nil {
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
	siteBytes, err := common.Marshal(profitBoardPersistedSiteConfig{
		LegacySite:      payload.Site,
		SharedSite:      payload.SharedSite,
		ComboConfigs:    payload.ComboConfigs,
		ExcludedUserIDs: payload.ExcludedUserIDs,
	})
	if err != nil {
		return nil, "", err
	}

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
		Batches:         normalized,
		SharedSite:      payload.SharedSite,
		ComboConfigs:    stripProfitBoardRemoteObserverSecrets(payload.ComboConfigs),
		ExcludedUserIDs: payload.ExcludedUserIDs,
		Upstream:        payload.Upstream,
		Site:            payload.Site,
	}, signature, nil
}

// GetLatestProfitBoardConfig 返回 updated_at 最新的那一条收益看板配置,
// 用作跨设备同步的唯一真相源。表里没有记录时返回 (nil, "", nil)。
func GetLatestProfitBoardConfig() (*ProfitBoardConfigPayload, string, error) {
	record := &ProfitBoardConfig{}
	if err := DB.Order("updated_at desc").First(record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", nil
		}
		return nil, "", err
	}
	batches := parseProfitBoardConfigBatches(record.SelectionValues)
	if len(batches) == 0 {
		return nil, "", nil
	}
	defaultUpstream := normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
		CostSource: ProfitBoardCostSourceManualOnly,
	}, false)
	defaultSite := normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
		PricingMode: ProfitBoardSitePricingManual,
	}, true)
	payload := &ProfitBoardConfigPayload{
		Batches:  batches,
		Upstream: defaultUpstream,
		Site:     defaultSite,
	}
	_ = common.UnmarshalJsonStr(record.UpstreamConfig, &payload.Upstream)
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	persistedSite := profitBoardPersistedSiteConfig{}
	if err := common.UnmarshalJsonStr(record.SiteConfig, &persistedSite); err == nil &&
		(len(persistedSite.ComboConfigs) > 0 ||
			len(persistedSite.SharedSite.ModelNames) > 0 ||
			len(persistedSite.ExcludedUserIDs) > 0 ||
			persistedSite.LegacySite.PricingMode != "" ||
			persistedSite.LegacySite.Group != "" ||
			len(persistedSite.LegacySite.ModelNames) > 0) {
		payload.Site = normalizeProfitBoardPricingConfig(persistedSite.LegacySite, true)
		payload.SharedSite = normalizeProfitBoardSharedSiteConfig(persistedSite.SharedSite, payload.Site)
		payload.ExcludedUserIDs = normalizeProfitBoardExcludedUserIDs(persistedSite.ExcludedUserIDs)
		payload.ComboConfigs = normalizeProfitBoardComboConfigs(
			payload.Batches,
			remapProfitBoardComboConfigsByBatchSelection(
				payload.Batches,
				batches,
				persistedSite.ComboConfigs,
			),
			payload.SharedSite,
			payload.Site,
			payload.Upstream,
		)
	} else {
		_ = common.UnmarshalJsonStr(record.SiteConfig, &payload.Site)
		payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
		payload.SharedSite = normalizeProfitBoardSharedSiteConfig(ProfitBoardSharedSitePricingConfig{}, payload.Site)
		payload.ExcludedUserIDs = nil
		payload.ComboConfigs = normalizeProfitBoardComboConfigs(payload.Batches, nil, payload.SharedSite, payload.Site, payload.Upstream)
	}
	if err := migrateProfitBoardLegacyWalletAccount(payload); err != nil {
		return nil, "", err
	}
	payload.ComboConfigs = stripProfitBoardRemoteObserverSecrets(payload.ComboConfigs)
	return payload, record.SelectionSignature, nil
}

func GetProfitBoardOptions() (*ProfitBoardOptions, error) {
	options := &ProfitBoardOptions{}

	channels := make([]ProfitBoardChannelOption, 0)
	if err := DB.Model(&Channel{}).
		Select("id, name, tag, status, models").
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
			ModelRatio:            item.ModelRatio,
			ModelPrice:            item.ModelPrice,
			CompletionRatio:       item.CompletionRatio,
			CacheRatio:            item.CacheRatio,
			CacheCreationRatio:    item.CacheCreationRatio,
		})
	}
	sort.Slice(options.LocalModels, func(i, j int) bool {
		return options.LocalModels[i].ModelName < options.LocalModels[j].ModelName
	})
	options.SiteModels = collectProfitBoardSiteModels()

	adminUsers := make([]ProfitBoardAdminUserOption, 0)
	if err := DB.Model(&User{}).
		Select("id, username, display_name, role").
		Where("role >= ?", common.RoleAdminUser).
		Order("role desc, id asc").
		Scan(&adminUsers).Error; err != nil {
		return nil, err
	}
	options.AdminUsers = adminUsers

	upstreamAccounts, err := GetProfitBoardUpstreamAccountOptions()
	if err != nil {
		return nil, err
	}
	options.UpstreamAccounts = upstreamAccounts

	return options, nil
}

func collectProfitBoardSiteModels() []string {
	seen := make(map[string]struct{})
	modelNames := make([]string, 0)
	appendName := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		lowerName := strings.ToLower(name)
		if _, ok := seen[lowerName]; ok {
			return
		}
		seen[lowerName] = struct{}{}
		modelNames = append(modelNames, name)
	}

	for _, name := range GetEnabledModels() {
		appendName(name)
	}
	for _, item := range GetPricing() {
		appendName(item.ModelName)
	}
	dbModels := make([]string, 0)
	if err := DB.Model(&Model{}).Where("status = ?", 1).Pluck("model_name", &dbModels).Error; err == nil {
		for _, name := range dbModels {
			appendName(name)
		}
	}
	sort.Strings(modelNames)
	return modelNames
}

func normalizeProfitBoardQuery(query ProfitBoardQuery) (ProfitBoardQuery, string, error) {
	normalizedBatches, signature, _, err := normalizeProfitBoardBatches(query.Batches, query.Selection)
	if err != nil {
		return ProfitBoardQuery{}, "", err
	}
	query.Upstream = normalizeProfitBoardPricingConfig(query.Upstream, false)
	query.Site = normalizeProfitBoardPricingConfig(query.Site, true)
	query.SharedSite = normalizeProfitBoardSharedSiteConfig(query.SharedSite, query.Site)
	query.ExcludedUserIDs = normalizeProfitBoardExcludedUserIDs(query.ExcludedUserIDs)
	query.ComboConfigs = normalizeProfitBoardComboConfigs(normalizedBatches, query.ComboConfigs, query.SharedSite, query.Site, query.Upstream)
	query.ComboConfigs = hydrateProfitBoardRemoteObserverSecrets(signature, query.ComboConfigs)
	if err := validateProfitBoardPricingConfig(query.Upstream, false); err != nil {
		return ProfitBoardQuery{}, "", err
	}
	if err := validateProfitBoardPricingConfig(query.Site, true); err != nil {
		return ProfitBoardQuery{}, "", err
	}
	if err := validateProfitBoardComboConfigs(query.ComboConfigs); err != nil {
		return ProfitBoardQuery{}, "", err
	}

	if query.StartTimestamp <= 0 {
		query.StartTimestamp = time.Now().Add(-7 * 24 * time.Hour).Unix()
	}
	if query.EndTimestamp <= 0 {
		query.EndTimestamp = time.Now().Unix()
	}
	if query.EndTimestamp < query.StartTimestamp {
		return ProfitBoardQuery{}, "", ErrProfitBoardEndBeforeStart
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
			return ProfitBoardQuery{}, "", ErrProfitBoardCustomGranularityMin
		}
		if query.CustomIntervalMinutes > 43200 {
			return ProfitBoardQuery{}, "", ErrProfitBoardCustomGranularityMax
		}
	default:
		return ProfitBoardQuery{}, "", ErrProfitBoardInvalidGranularity
	}
	if !query.IncludeDetails {
		query.DetailLimit = 0
	} else {
		if query.DetailLimit <= 0 {
			query.DetailLimit = 300
		}
		if query.DetailLimit > 2000 {
			query.DetailLimit = 2000
		}
	}
	query.Batches = normalizedBatches
	query.Selection = ProfitBoardSelection{}
	return query, signature, nil
}

func buildProfitBoardActivityWatermark(requestCount int, latestLogID int, latestCreatedAt int64) string {
	return fmt.Sprintf("%d:%d:%d", requestCount, latestLogID, latestCreatedAt)
}

func buildProfitBoardCombinedActivityWatermark(requestCount int, latestLogID int, latestCreatedAt int64, walletSnapshotWatermark string) string {
	base := buildProfitBoardActivityWatermark(requestCount, latestLogID, latestCreatedAt)
	walletSnapshotWatermark = strings.TrimSpace(walletSnapshotWatermark)
	if walletSnapshotWatermark == "" {
		return base
	}
	return base + "|" + walletSnapshotWatermark
}

func buildProfitBoardWalletSnapshotWatermark(comboPricingMap map[string]profitBoardResolvedComboPricing) (string, error) {
	accountCombos := profitBoardWalletObserverCombosByAccount(comboPricingMap)
	if len(accountCombos) == 0 {
		return "", nil
	}
	accountIDs := make([]int, 0, len(accountCombos))
	for accountID := range accountCombos {
		accountIDs = append(accountIDs, accountID)
	}
	sort.Ints(accountIDs)
	parts := make([]string, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		signature := profitBoardUpstreamAccountSnapshotSignature(accountID)
		latestSnapshot, err := getLatestProfitBoardRemoteSnapshot(signature, profitBoardUpstreamAccountSnapshotComboID)
		if err != nil {
			return "", err
		}
		if latestSnapshot == nil {
			parts = append(parts, fmt.Sprintf("%d:0:0", accountID))
			continue
		}
		parts = append(parts, fmt.Sprintf("%d:%d:%d", accountID, latestSnapshot.Id, latestSnapshot.SyncedAt))
	}
	return strings.Join(parts, ","), nil
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
			return nil, nil, ErrProfitBoardChannelNotExist
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
			return nil, nil, ErrProfitBoardTagNoChannel
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
		return nil, nil, ErrProfitBoardInvalidScopeType
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
		CreatedAt:        batch.CreatedAt,
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
				return nil, fmt.Errorf("%w: %s -> %s, %s", ErrProfitBoardChannelDuplicateBatch, name, owner, current.Name)
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
	comboPricingMap := resolveProfitBoardComboPricingMap(normalizedQuery, resolvedBatches)
	walletSnapshotWatermark, err := buildProfitBoardWalletSnapshotWatermark(comboPricingMap)
	if err != nil {
		return nil, err
	}

	channelIDs := collectProfitBoardChannelIDs(resolvedBatches)
	if len(channelIDs) == 0 {
		return &ProfitBoardActivity{
			Signature:         signature,
			GeneratedAt:       common.GetTimestamp(),
			ActivityWatermark: buildProfitBoardCombinedActivityWatermark(0, 0, 0, walletSnapshotWatermark),
		}, nil
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
		ActivityWatermark:  buildProfitBoardCombinedActivityWatermark(0, latestRow.Id, latestRow.CreatedAt, walletSnapshotWatermark),
		LatestLogId:        latestRow.Id,
		LatestLogCreatedAt: latestRow.CreatedAt,
		RequestCount:       0,
	}, nil
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
	var priceFactor float64
	source := "site_model_standard"
	if config.UseRechargePrice {
		priceFactor = profitBoardPriceFactor(true)
		source = "site_model_recharge"
	} else if config.PlanID > 0 {
		priceFactor = profitBoardPlanPriceFactor(config.PlanID)
		source = "site_model_package"
	} else {
		priceFactor = 1
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

func profitBoardFindManualRule(modelName string, rules []ProfitBoardModelPricingRule) (ProfitBoardModelPricingRule, bool, bool) {
	var defaultRule ProfitBoardModelPricingRule
	hasDefault := false
	for _, rule := range rules {
		if rule.IsDefault {
			defaultRule = rule
			hasDefault = true
			continue
		}
		if strings.EqualFold(strings.TrimSpace(rule.ModelName), strings.TrimSpace(modelName)) {
			return rule, false, true
		}
	}
	if hasDefault {
		return defaultRule, true, true
	}
	return ProfitBoardModelPricingRule{}, false, false
}

func profitBoardTokenMoneyUSDByRule(inputTokens, completionTokens, cacheReadTokens, cacheCreationTokens int, rule ProfitBoardModelPricingRule) float64 {
	return float64(inputTokens)*rule.InputPrice/1_000_000 +
		float64(completionTokens)*rule.OutputPrice/1_000_000 +
		float64(cacheReadTokens)*rule.CacheReadPrice/1_000_000 +
		float64(cacheCreationTokens)*rule.CacheCreationPrice/1_000_000
}

func profitBoardManualSiteRevenueUSD(
	row profitBoardLogRow,
	inputTokens int,
	cacheReadTokens int,
	cacheCreationTokens int,
	rules []ProfitBoardModelPricingRule,
) (float64, string, bool) {
	rule, usedDefault, ok := profitBoardFindManualRule(row.ModelName, rules)
	if !ok {
		return 0, "manual_missing", false
	}
	if usedDefault {
		return profitBoardTokenMoneyUSDByRule(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, rule), "manual_default", true
	}
	return profitBoardTokenMoneyUSDByRule(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, rule), "manual_rule", true
}

func profitBoardUpstreamCostUSD(
	row profitBoardLogRow,
	other profitBoardOtherInfo,
	inputTokens int,
	cacheReadTokens int,
	cacheCreationTokens int,
	costSource string,
	rules []ProfitBoardModelPricingRule,
	fixedTotalAmount float64,
) (float64, string, bool) {
	hasReturnedCost := other.UpstreamCostReported || other.UpstreamCost > 0
	if hasReturnedCost && other.UpstreamCost >= 0 {
		if costSource == ProfitBoardCostSourceReturnedFirst || costSource == ProfitBoardCostSourceReturnedOnly {
			return other.UpstreamCost, "returned_cost", true
		}
	}

	if costSource == ProfitBoardCostSourceReturnedOnly {
		return 0, "returned_cost_missing", false
	}
	if costSource == ProfitBoardCostSourceManualOnly && len(rules) == 0 && fixedTotalAmount > 0 {
		return 0, "manual_fixed_total_only", true
	}

	rule, usedDefault, ok := profitBoardFindManualRule(row.ModelName, rules)
	if !ok {
		return 0, "manual_missing", false
	}
	if usedDefault {
		return profitBoardTokenMoneyUSDByRule(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, rule), "manual_default", true
	}
	return profitBoardTokenMoneyUSDByRule(inputTokens, row.CompletionTokens, cacheReadTokens, cacheCreationTokens, rule), "manual_rule", true
}

type profitBoardPreparedRow struct {
	Batch               ProfitBoardBatchInfo
	ComboPricing        profitBoardResolvedComboPricing
	Row                 profitBoardLogRow
	Other               profitBoardOtherInfo
	InputTokens         int
	CacheReadTokens     int
	CacheCreationTokens int
}

type profitBoardSiteRevenueAllocation struct {
	EligibleBatchRequestCount   map[string]int
	EligibleChannelRequestCount map[string]int
	EligibleModelRequestCount   map[string]int
}

func newProfitBoardSiteRevenueAllocation() profitBoardSiteRevenueAllocation {
	return profitBoardSiteRevenueAllocation{
		EligibleBatchRequestCount:   make(map[string]int),
		EligibleChannelRequestCount: make(map[string]int),
		EligibleModelRequestCount:   make(map[string]int),
	}
}

func resolveProfitBoardComboPricingMap(query ProfitBoardQuery, batches []ProfitBoardBatchInfo) map[string]profitBoardResolvedComboPricing {
	configMap := make(map[string]profitBoardResolvedComboPricing, len(batches))
	for _, batch := range batches {
		configMap[batch.Id] = profitBoardResolvedComboPricing{
			ComboId:                  batch.Id,
			SiteMode:                 ProfitBoardComboSiteModeManual,
			UpstreamMode:             query.Upstream.UpstreamMode,
			CostSource:               normalizeProfitBoardCostSource(query.Upstream.CostSource),
			UpstreamAccountID:        query.Upstream.UpstreamAccountID,
			SiteExchangeRate:         1,
			UpstreamExchangeRate:     1,
			SharedSite:               normalizeProfitBoardSharedSiteConfig(query.SharedSite, query.Site),
			SiteRules:                normalizeProfitBoardModelPricingRules(nil, query.Site),
			UpstreamRules:            normalizeProfitBoardModelPricingRules(nil, query.Upstream),
			SiteFixedTotalAmount:     clampProfitBoardNumber(query.Site.FixedTotalAmount),
			UpstreamFixedTotalAmount: clampProfitBoardNumber(query.Upstream.FixedTotalAmount),
		}
	}
	for _, config := range query.ComboConfigs {
		current := configMap[config.ComboId]
		current.ComboId = config.ComboId
		if config.SiteMode != "" {
			current.SiteMode = config.SiteMode
		}
		if config.UpstreamMode != "" {
			current.UpstreamMode = config.UpstreamMode
		}
		if normalizeProfitBoardCostSource(config.CostSource) != "" {
			current.CostSource = normalizeProfitBoardCostSource(config.CostSource)
		}
		if strings.TrimSpace(current.UpstreamMode) != ProfitBoardUpstreamModeWallet {
			current.UpstreamMode = ProfitBoardUpstreamModeManual
			current.UpstreamAccountID = 0
		} else if config.UpstreamAccountID > 0 {
			current.UpstreamAccountID = config.UpstreamAccountID
		}
		current.SiteExchangeRate = normalizeProfitBoardExchangeRate(config.SiteExchangeRate)
		current.UpstreamExchangeRate = normalizeProfitBoardExchangeRate(config.UpstreamExchangeRate)
		if !profitBoardSharedSiteConfigEmpty(config.SharedSite) {
			current.SharedSite = normalizeProfitBoardSharedSiteConfig(
				config.SharedSite,
				query.Site,
			)
		}
		current.SiteRules = config.SiteRules
		current.UpstreamRules = config.UpstreamRules
		current.SiteFixedTotalAmount = config.SiteFixedTotalAmount
		current.UpstreamFixedTotalAmount = config.UpstreamFixedTotalAmount
		configMap[config.ComboId] = current
	}
	return configMap
}

func profitBoardHasSharedSiteMode(comboPricingMap map[string]profitBoardResolvedComboPricing) bool {
	for _, config := range comboPricingMap {
		if config.SiteMode == ProfitBoardComboSiteModeSharedSite {
			return true
		}
	}
	return false
}

func profitBoardSharedSiteMeta(comboPricingMap map[string]profitBoardResolvedComboPricing) (bool, float64, string) {
	sharedCount := 0
	useRechargeCount := 0
	usePlanCount := 0
	planID := 0
	samePlan := true
	for _, config := range comboPricingMap {
		if config.SiteMode != ProfitBoardComboSiteModeSharedSite {
			continue
		}
		sharedCount++
		if config.SharedSite.UseRechargePrice {
			useRechargeCount++
		} else if config.SharedSite.PlanID > 0 {
			usePlanCount++
			if planID == 0 {
				planID = config.SharedSite.PlanID
			} else if planID != config.SharedSite.PlanID {
				samePlan = false
			}
		}
	}
	if sharedCount == 0 {
		return false, 0, ""
	}
	if useRechargeCount == 0 && usePlanCount == 0 {
		factor, note := profitBoardPriceFactorMeta(false)
		return false, factor, note
	}
	if useRechargeCount == sharedCount {
		factor, note := profitBoardPriceFactorMeta(true)
		return true, factor, note
	}
	if usePlanCount == sharedCount && samePlan {
		factor, note := profitBoardPlanPriceFactorMeta(planID)
		return false, factor, note
	}
	return false, 0, "不同组合使用了不同的本站价格口径：部分按原价/套餐价/充值价"
}

func buildProfitBoardReportCacheKey(query ProfitBoardQuery) string {
	if profitBoardHasEnabledRemoteObserver(query.ComboConfigs) {
		return ""
	}
	if profitBoardHasWalletObserverCombo(resolveProfitBoardComboPricingMap(query, nil)) {
		return ""
	}
	payloadBytes, err := common.Marshal(struct {
		Batches               []ProfitBoardBatch                 `json:"batches"`
		SharedSite            ProfitBoardSharedSitePricingConfig `json:"shared_site"`
		ComboConfigs          []ProfitBoardComboPricingConfig    `json:"combo_configs"`
		StartTimestamp        int64                              `json:"start_timestamp"`
		EndTimestamp          int64                              `json:"end_timestamp"`
		Granularity           string                             `json:"granularity"`
		CustomIntervalMinutes int                                `json:"custom_interval_minutes"`
	}{
		Batches:               query.Batches,
		SharedSite:            query.SharedSite,
		ComboConfigs:          query.ComboConfigs,
		StartTimestamp:        query.StartTimestamp,
		EndTimestamp:          query.EndTimestamp,
		Granularity:           query.Granularity,
		CustomIntervalMinutes: query.CustomIntervalMinutes,
	})
	if err != nil {
		return ""
	}
	hash := sha1.Sum(payloadBytes)
	return hex.EncodeToString(hash[:])
}

func iterateProfitBoardRows(query ProfitBoardQuery, batches []ProfitBoardBatchInfo, callback func(prepared profitBoardPreparedRow) error) error {
	channelIDs := collectProfitBoardChannelIDs(batches)
	if len(channelIDs) == 0 {
		return nil
	}

	batchByChannelID := make(map[int]ProfitBoardBatchInfo, len(channelIDs))
	for _, batch := range batches {
		for _, channelID := range batch.ChannelIDs {
			batchByChannelID[channelID] = batch
		}
	}
	comboPricingMap := resolveProfitBoardComboPricingMap(query, batches)

	tx := LOG_DB.Table("logs").
		Select("id, user_id, created_at, request_id, channel_id, model_name, quota, prompt_tokens, completion_tokens, other").
		Where("type = ?", LogTypeConsume).
		Where("channel_id IN ?", channelIDs)
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
		batch, ok := batchByChannelID[row.ChannelId]
		if !ok {
			continue
		}
		if row.CreatedAt < profitBoardEffectiveStartTimestamp(batch, query.StartTimestamp) {
			continue
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
			ComboPricing:        comboPricingMap[batch.Id],
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
	return rowIterErr
}

func generateProfitBoardReport(query ProfitBoardQuery, applyDetailLimit bool) (*ProfitBoardReport, error) {
	normalizedQuery, signature, err := normalizeProfitBoardQuery(query)
	if err != nil {
		return nil, err
	}

	if !normalizedQuery.IncludeDetails {
		cacheKey := buildProfitBoardReportCacheKey(normalizedQuery)
		if cacheKey != "" {
			if cached, found, cacheErr := getProfitBoardReportCache().Get(cacheKey); cacheErr == nil && found {
				return &cached, nil
			}
		}
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
	comboPricingMap := resolveProfitBoardComboPricingMap(normalizedQuery, resolvedBatches)
	excludedUserSet := profitBoardExcludedUserSet(normalizedQuery.ExcludedUserIDs)
	siteUseRechargePrice, sitePriceFactor, sitePriceFactorNote := profitBoardSharedSiteMeta(comboPricingMap)
	siteRevenueAllocation := newProfitBoardSiteRevenueAllocation()

	report := &ProfitBoardReport{
		Signature:      signature,
		Batches:        resolvedBatches,
		BatchSummaries: make([]ProfitBoardBatchSummary, 0, len(resolvedBatches)),
		Meta: ProfitBoardMeta{
			SiteUseRechargePrice:      siteUseRechargePrice,
			SitePriceFactor:           roundProfitBoardAmount(sitePriceFactor),
			SitePriceFactorNote:       sitePriceFactorNote,
			GeneratedAt:               common.GetTimestamp(),
			FixedTotalAmountScope:     "created_at_once",
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
	accountWalletCombos := profitBoardWalletObserverCombosByAccount(comboPricingMap)
	accountWalletAggregates := make(map[int]*profitBoardUpstreamAccountObservedAggregate, len(accountWalletCombos))
	for _, batch := range resolvedBatches {
		batchSummaryMap[batch.Id] = &ProfitBoardBatchSummary{
			BatchId:   batch.Id,
			BatchName: batch.Name,
		}
		for _, channel := range batch.ResolvedChannels {
			channelNameMap[channel.Id] = channel.Name
		}
	}

	for accountID := range accountWalletCombos {
		aggregate, aggregateErr := collectProfitBoardUpstreamAccountObservedAggregate(
			accountID,
			normalizedQuery.StartTimestamp,
			normalizedQuery.EndTimestamp,
			normalizedQuery.Granularity,
			normalizedQuery.CustomIntervalMinutes,
			false,
		)
		if aggregateErr != nil {
			return nil, aggregateErr
		}
		accountWalletAggregates[accountID] = aggregate
	}

	remoteAggregate, remoteErr := collectProfitBoardRemoteObserverAggregate(
		signature,
		resolvedBatches,
		normalizedQuery.ComboConfigs,
		normalizedQuery.StartTimestamp,
		normalizedQuery.EndTimestamp,
		normalizedQuery.Granularity,
		normalizedQuery.CustomIntervalMinutes,
		false,
		true,
	)
	if remoteErr != nil {
		return nil, remoteErr
	}
	report.RemoteObserverStates = remoteAggregate.States
	report.Summary.RemoteObservedCostUSD += remoteAggregate.TotalCostUSD
	report.Warnings = append(report.Warnings, remoteAggregate.Warnings...)
	for batchID, observedCostUSD := range remoteAggregate.BatchCostUSD {
		if summary := batchSummaryMap[batchID]; summary != nil {
			summary.RemoteObservedCostUSD += observedCostUSD
		}
	}
	for _, remotePoint := range remoteAggregate.Timeseries {
		timeKey := fmt.Sprintf("%s:%d", remotePoint.BatchId, remotePoint.BucketTimestamp)
		point, ok := timeBuckets[timeKey]
		if !ok {
			current := remotePoint
			timeBuckets[timeKey] = &current
			continue
		}
		point.RemoteObservedCostUSD += remotePoint.RemoteObservedCostUSD
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
		_, excludedFromRevenue := excludedUserSet[row.UserId]
		comboPricing := prepared.ComboPricing
		configuredSiteRevenueUSD := 0.0
		sitePricingSource := ""
		sitePricingKnown := false
		if excludedFromRevenue {
			sitePricingSource = "excluded_user"
			sitePricingKnown = true
		} else if comboPricing.SiteMode == ProfitBoardComboSiteModeLogQuota {
			logQuotaModelNames := comboPricing.SharedSite.ModelNames
			logQuotaMatched := len(logQuotaModelNames) == 0
			if !logQuotaMatched {
				for _, mn := range logQuotaModelNames {
					if mn == row.ModelName {
						logQuotaMatched = true
						break
					}
				}
			}
			if logQuotaMatched {
				configuredSiteRevenueUSD = float64(row.Quota) / common.QuotaPerUnit
				sitePricingSource = "log_quota"
				sitePricingKnown = row.Quota > 0
			}
		} else if comboPricing.SiteMode == ProfitBoardComboSiteModeSharedSite {
			sharedSiteConfig := comboPricing.SharedSite
			configuredSiteRevenueUSD, sitePricingSource, sitePricingKnown = profitBoardSiteModelRevenueUSD(
				row,
				prepared.InputTokens,
				prepared.CacheReadTokens,
				prepared.CacheCreationTokens,
				ProfitBoardTokenPricingConfig{
					PricingMode:      ProfitBoardSitePricingSiteModel,
					ModelNames:       sharedSiteConfig.ModelNames,
					Group:            sharedSiteConfig.Group,
					UseRechargePrice: sharedSiteConfig.UseRechargePrice,
					PlanID:           sharedSiteConfig.PlanID,
				},
				pricingMap,
				groupRatios,
			)
			if !sitePricingKnown {
				if amount, source, ok := profitBoardManualSiteRevenueUSD(
					row,
					prepared.InputTokens,
					prepared.CacheReadTokens,
					prepared.CacheCreationTokens,
					comboPricing.SiteRules,
				); ok {
					configuredSiteRevenueUSD = amount
					sitePricingSource = source
					sitePricingKnown = true
				} else {
					sitePricingSource = "site_model_missing"
				}
			}
		} else {
			configuredSiteRevenueUSD, sitePricingSource, sitePricingKnown = profitBoardManualSiteRevenueUSD(
				row,
				prepared.InputTokens,
				prepared.CacheReadTokens,
				prepared.CacheCreationTokens,
				comboPricing.SiteRules,
			)
		}
		upstreamCostUSD := 0.0
		upstreamCostSource := ""
		upstreamCostKnown := false
		isWalletCombo := profitBoardComboUsesWalletObserver(comboPricing)
		if isWalletCombo {
			upstreamCostSource = "wallet_observer"
		} else {
			upstreamCostUSD, upstreamCostSource, upstreamCostKnown = profitBoardUpstreamCostUSD(
				row,
				prepared.Other,
				prepared.InputTokens,
				prepared.CacheReadTokens,
				prepared.CacheCreationTokens,
				comboPricing.CostSource,
				comboPricing.UpstreamRules,
				comboPricing.UpstreamFixedTotalAmount,
			)
		}
		configuredSiteRevenueCNY := 0.0
		if sitePricingKnown {
			configuredSiteRevenueCNY = profitBoardConfiguredSiteRevenueCNY(configuredSiteRevenueUSD, comboPricing)
		}
		upstreamCostCNY := 0.0
		if !isWalletCombo && upstreamCostKnown {
			upstreamCostCNY = profitBoardConfiguredUpstreamCostCNY(upstreamCostUSD, comboPricing)
		}

		report.Summary.RequestCount++
		batchSummary.RequestCount++
		if !excludedFromRevenue {
			siteRevenueAllocation.EligibleBatchRequestCount[batch.Id]++
		}
		if sitePricingKnown && strings.HasPrefix(sitePricingSource, "site_model_") {
			report.Summary.SiteModelMatchCount++
			batchSummary.SiteModelMatchCount++
		}
		if !sitePricingKnown {
			report.Summary.MissingSitePricingCount++
			batchSummary.MissingSitePricingCount++
		}
		if !isWalletCombo {
			if upstreamCostKnown {
				report.Summary.KnownUpstreamCostCount++
				batchSummary.KnownUpstreamCostCount++
				report.Summary.UpstreamCostUSD += upstreamCostUSD
				report.Summary.UpstreamCostCNY += upstreamCostCNY
				batchSummary.UpstreamCostUSD += upstreamCostUSD
				batchSummary.UpstreamCostCNY += upstreamCostCNY
				switch upstreamCostSource {
				case "returned_cost":
					report.Summary.ReturnedCostCount++
					batchSummary.ReturnedCostCount++
				case "manual_rule", "manual_default", "manual_fixed_total_only":
					report.Summary.ManualCostCount++
					batchSummary.ManualCostCount++
				}
			} else {
				report.Summary.MissingUpstreamCostCount++
				batchSummary.MissingUpstreamCostCount++
			}
		}

		report.Summary.ActualSiteRevenueUSD += actualSiteRevenueUSD
		batchSummary.ActualSiteRevenueUSD += actualSiteRevenueUSD
		if sitePricingKnown {
			report.Summary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
			report.Summary.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNY
			batchSummary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
			batchSummary.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNY
		}

		configuredProfitUSD := 0.0
		configuredProfitCNY := 0.0
		actualProfitUSD := 0.0
		if isWalletCombo && sitePricingKnown {
			configuredProfitUSD = configuredSiteRevenueUSD
			configuredProfitCNY = profitBoardConfiguredProfitCNY(configuredSiteRevenueUSD, 0, comboPricing)
			report.Summary.ConfiguredProfitUSD += configuredProfitUSD
			report.Summary.ConfiguredProfitCNY += configuredProfitCNY
			batchSummary.ConfiguredProfitUSD += configuredProfitUSD
			batchSummary.ConfiguredProfitCNY += configuredProfitCNY
		} else if upstreamCostKnown && sitePricingKnown {
			configuredProfitUSD = configuredSiteRevenueUSD - upstreamCostUSD
			configuredProfitCNY = profitBoardConfiguredProfitCNY(configuredSiteRevenueUSD, upstreamCostUSD, comboPricing)
			report.Summary.ConfiguredProfitUSD += configuredProfitUSD
			report.Summary.ConfiguredProfitCNY += configuredProfitCNY
			batchSummary.ConfiguredProfitUSD += configuredProfitUSD
			batchSummary.ConfiguredProfitCNY += configuredProfitCNY
		}
		if isWalletCombo {
			actualProfitUSD = actualSiteRevenueUSD
			report.Summary.ActualProfitUSD += actualProfitUSD
			batchSummary.ActualProfitUSD += actualProfitUSD
		} else if upstreamCostKnown {
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
		if !excludedFromRevenue {
			siteRevenueAllocation.EligibleChannelRequestCount[batch.Id+"|"+strconv.Itoa(row.ChannelId)]++
			siteRevenueAllocation.EligibleModelRequestCount[batch.Id+"|"+row.ModelName]++
		}
		if sitePricingKnown {
			point.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
			point.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNY
		}
		if isWalletCombo {
			point.ActualProfitUSD += actualProfitUSD
		} else if upstreamCostKnown {
			point.UpstreamCostUSD += upstreamCostUSD
			point.UpstreamCostCNY += upstreamCostCNY
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
		if isWalletCombo && sitePricingKnown {
			point.ConfiguredProfitUSD += configuredProfitUSD
			point.ConfiguredProfitCNY += configuredProfitCNY
		} else if upstreamCostKnown && sitePricingKnown {
			point.ConfiguredProfitUSD += configuredProfitUSD
			point.ConfiguredProfitCNY += configuredProfitCNY
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
			channelItem.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNY
		}
		if isWalletCombo {
			channelItem.ActualProfitUSD += actualProfitUSD
		} else if upstreamCostKnown {
			channelItem.UpstreamCostUSD += upstreamCostUSD
			channelItem.UpstreamCostCNY += upstreamCostCNY
			channelItem.KnownUpstreamCostCount++
			channelItem.ActualProfitUSD += actualProfitUSD
		} else {
			channelItem.MissingUpstreamCostCount++
		}
		if isWalletCombo && sitePricingKnown {
			channelItem.ConfiguredProfitUSD += configuredProfitUSD
			channelItem.ConfiguredProfitCNY += configuredProfitCNY
		} else if upstreamCostKnown && sitePricingKnown {
			channelItem.ConfiguredProfitUSD += configuredProfitUSD
			channelItem.ConfiguredProfitCNY += configuredProfitCNY
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
			modelItem.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNY
		}
		if isWalletCombo {
			modelItem.ActualProfitUSD += actualProfitUSD
		} else if upstreamCostKnown {
			modelItem.UpstreamCostUSD += upstreamCostUSD
			modelItem.UpstreamCostCNY += upstreamCostCNY
			modelItem.KnownUpstreamCostCount++
			modelItem.ActualProfitUSD += actualProfitUSD
		} else {
			modelItem.MissingUpstreamCostCount++
		}
		if isWalletCombo && sitePricingKnown {
			modelItem.ConfiguredProfitUSD += configuredProfitUSD
			modelItem.ConfiguredProfitCNY += configuredProfitCNY
		} else if upstreamCostKnown && sitePricingKnown {
			modelItem.ConfiguredProfitUSD += configuredProfitUSD
			modelItem.ConfiguredProfitCNY += configuredProfitCNY
		}

		if normalizedQuery.IncludeDetails {
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
				ConfiguredSiteRevenueCNY: roundProfitBoardAmount(configuredSiteRevenueCNY),
				UpstreamCostUSD:          roundProfitBoardAmount(upstreamCostUSD),
				UpstreamCostCNY:          roundProfitBoardAmount(upstreamCostCNY),
				ConfiguredProfitUSD:      roundProfitBoardAmount(configuredProfitUSD),
				ConfiguredProfitCNY:      roundProfitBoardAmount(configuredProfitCNY),
				ActualProfitUSD:          roundProfitBoardAmount(actualProfitUSD),
				ConfiguredActualDeltaUSD: roundProfitBoardAmount(configuredSiteRevenueUSD - actualSiteRevenueUSD),
				UpstreamCostKnown:        upstreamCostKnown,
				UpstreamCostSource:       upstreamCostSource,
				SitePricingSource:        sitePricingSource,
				SitePricingKnown:         sitePricingKnown,
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if report.Summary.RequestCount > 0 {
		knownOrWalletCount := report.Summary.KnownUpstreamCostCount
		for _, batch := range resolvedBatches {
			if profitBoardComboUsesWalletObserver(comboPricingMap[batch.Id]) {
				knownOrWalletCount += batchSummaryMap[batch.Id].RequestCount
			}
		}
		report.Summary.ConfiguredProfitCoverageRate = float64(knownOrWalletCount) / float64(report.Summary.RequestCount)
	}
	if report.Summary.MissingUpstreamCostCount > 0 {
		report.Warnings = append(report.Warnings, "部分日志未命中上游成本配置，已按可用规则回退，仍无法确定的记为未知")
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

	report.BatchSummaries = make([]ProfitBoardBatchSummary, 0, len(batchSummaryMap))
	for _, batch := range resolvedBatches {
		current := *batchSummaryMap[batch.Id]
		if current.RequestCount > 0 {
			if profitBoardComboUsesWalletObserver(comboPricingMap[batch.Id]) {
				current.ConfiguredProfitCoverageRate = 1
			} else {
				current.ConfiguredProfitCoverageRate = float64(current.KnownUpstreamCostCount) / float64(current.RequestCount)
			}
		}
		current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
		current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
		current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
		current.RemoteObservedCostUSD = roundProfitBoardAmount(current.RemoteObservedCostUSD)
		current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
		current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
		current.ConfiguredProfitCoverageRate = roundProfitBoardAmount(current.ConfiguredProfitCoverageRate)
		roundProfitBoardConfiguredMetrics(&current.ProfitBoardSummary)
		report.BatchSummaries = append(report.BatchSummaries, current)
	}

	report.Timeseries = make([]ProfitBoardTimeseriesPoint, 0, len(timeBuckets))
	for _, point := range timeBuckets {
		current := *point
		current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
		current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
		current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
		current.RemoteObservedCostUSD = roundProfitBoardAmount(current.RemoteObservedCostUSD)
		current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
		current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
		roundProfitBoardConfiguredTimeseriesMetrics(&current)
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
		roundProfitBoardConfiguredBreakdownMetrics(&current)
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
		roundProfitBoardConfiguredBreakdownMetrics(&current)
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

	applyProfitBoardComboFixedTotals(
		report,
		comboPricingMap,
		siteRevenueAllocation,
		resolvedBatches,
		normalizedQuery.StartTimestamp,
		normalizedQuery.EndTimestamp,
		normalizedQuery.Granularity,
		normalizedQuery.CustomIntervalMinutes,
	)
	for accountID, comboIDs := range accountWalletCombos {
		applyProfitBoardObservedWalletCost(
			report,
			accountWalletAggregates[accountID],
			comboPricingMap,
			resolvedBatches,
			comboIDs,
			normalizedQuery.Granularity,
			normalizedQuery.CustomIntervalMinutes,
		)
	}
	for index := range report.BatchSummaries {
		report.BatchSummaries[index].ActualSiteRevenueUSD = roundProfitBoardAmount(report.BatchSummaries[index].ActualSiteRevenueUSD)
		report.BatchSummaries[index].ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.BatchSummaries[index].ConfiguredSiteRevenueUSD)
		report.BatchSummaries[index].UpstreamCostUSD = roundProfitBoardAmount(report.BatchSummaries[index].UpstreamCostUSD)
		report.BatchSummaries[index].RemoteObservedCostUSD = roundProfitBoardAmount(report.BatchSummaries[index].RemoteObservedCostUSD)
		report.BatchSummaries[index].ConfiguredProfitUSD = roundProfitBoardAmount(report.BatchSummaries[index].ConfiguredProfitUSD)
		report.BatchSummaries[index].ActualProfitUSD = roundProfitBoardAmount(report.BatchSummaries[index].ActualProfitUSD)
		roundProfitBoardConfiguredMetrics(&report.BatchSummaries[index].ProfitBoardSummary)
	}
	for index := range report.Timeseries {
		report.Timeseries[index].ActualSiteRevenueUSD = roundProfitBoardAmount(report.Timeseries[index].ActualSiteRevenueUSD)
		report.Timeseries[index].ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.Timeseries[index].ConfiguredSiteRevenueUSD)
		report.Timeseries[index].UpstreamCostUSD = roundProfitBoardAmount(report.Timeseries[index].UpstreamCostUSD)
		report.Timeseries[index].RemoteObservedCostUSD = roundProfitBoardAmount(report.Timeseries[index].RemoteObservedCostUSD)
		report.Timeseries[index].ConfiguredProfitUSD = roundProfitBoardAmount(report.Timeseries[index].ConfiguredProfitUSD)
		report.Timeseries[index].ActualProfitUSD = roundProfitBoardAmount(report.Timeseries[index].ActualProfitUSD)
		roundProfitBoardConfiguredTimeseriesMetrics(&report.Timeseries[index])
	}
	for index := range report.ChannelBreakdown {
		report.ChannelBreakdown[index].ActualSiteRevenueUSD = roundProfitBoardAmount(report.ChannelBreakdown[index].ActualSiteRevenueUSD)
		report.ChannelBreakdown[index].ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.ChannelBreakdown[index].ConfiguredSiteRevenueUSD)
		report.ChannelBreakdown[index].UpstreamCostUSD = roundProfitBoardAmount(report.ChannelBreakdown[index].UpstreamCostUSD)
		report.ChannelBreakdown[index].ConfiguredProfitUSD = roundProfitBoardAmount(report.ChannelBreakdown[index].ConfiguredProfitUSD)
		report.ChannelBreakdown[index].ActualProfitUSD = roundProfitBoardAmount(report.ChannelBreakdown[index].ActualProfitUSD)
		roundProfitBoardConfiguredBreakdownMetrics(&report.ChannelBreakdown[index])
	}
	for index := range report.ModelBreakdown {
		report.ModelBreakdown[index].ActualSiteRevenueUSD = roundProfitBoardAmount(report.ModelBreakdown[index].ActualSiteRevenueUSD)
		report.ModelBreakdown[index].ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.ModelBreakdown[index].ConfiguredSiteRevenueUSD)
		report.ModelBreakdown[index].UpstreamCostUSD = roundProfitBoardAmount(report.ModelBreakdown[index].UpstreamCostUSD)
		report.ModelBreakdown[index].ConfiguredProfitUSD = roundProfitBoardAmount(report.ModelBreakdown[index].ConfiguredProfitUSD)
		report.ModelBreakdown[index].ActualProfitUSD = roundProfitBoardAmount(report.ModelBreakdown[index].ActualProfitUSD)
		roundProfitBoardConfiguredBreakdownMetrics(&report.ModelBreakdown[index])
	}

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
	report.Summary.RemoteObservedCostUSD = roundProfitBoardAmount(report.Summary.RemoteObservedCostUSD)
	report.Summary.ConfiguredProfitUSD = roundProfitBoardAmount(report.Summary.ConfiguredProfitUSD)
	report.Summary.ActualProfitUSD = roundProfitBoardAmount(report.Summary.ActualProfitUSD)
	report.Summary.ConfiguredProfitCoverageRate = roundProfitBoardAmount(report.Summary.ConfiguredProfitCoverageRate)
	roundProfitBoardConfiguredMetrics(&report.Summary)
	report.Meta.LatestLogId = latestLogId
	report.Meta.LatestLogCreatedAt = latestLogCreatedAt
	walletSnapshotWatermark, watermarkErr := buildProfitBoardWalletSnapshotWatermark(comboPricingMap)
	if watermarkErr != nil {
		return nil, watermarkErr
	}
	report.Meta.ActivityWatermark = buildProfitBoardCombinedActivityWatermark(
		report.Summary.RequestCount,
		latestLogId,
		latestLogCreatedAt,
		walletSnapshotWatermark,
	)
	report.Warnings = uniqueProfitBoardWarnings(report.Warnings)
	if !normalizedQuery.IncludeDetails {
		if cacheKey := buildProfitBoardReportCacheKey(normalizedQuery); cacheKey != "" {
			_ = getProfitBoardReportCache().SetWithTTL(cacheKey, *report, profitBoardReportCacheTTL())
		}
	}
	return report, nil
}

func GenerateProfitBoardOverview(payload ProfitBoardConfigPayload) (*ProfitBoardReport, error) {
	normalizedBatches, signature, _, err := normalizeProfitBoardBatches(payload.Batches, payload.Selection)
	if err != nil {
		return nil, err
	}
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
	payload.SharedSite = normalizeProfitBoardSharedSiteConfig(payload.SharedSite, payload.Site)
	payload.ExcludedUserIDs = normalizeProfitBoardExcludedUserIDs(payload.ExcludedUserIDs)
	payload.ComboConfigs = normalizeProfitBoardComboConfigs(normalizedBatches, payload.ComboConfigs, payload.SharedSite, payload.Site, payload.Upstream)
	if err := validateProfitBoardPricingConfig(payload.Upstream, false); err != nil {
		return nil, err
	}
	if err := validateProfitBoardPricingConfig(payload.Site, true); err != nil {
		return nil, err
	}
	if err := validateProfitBoardComboConfigs(payload.ComboConfigs); err != nil {
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
	query := ProfitBoardQuery{
		Batches:         normalizedBatches,
		SharedSite:      payload.SharedSite,
		ComboConfigs:    payload.ComboConfigs,
		ExcludedUserIDs: payload.ExcludedUserIDs,
		Upstream:        payload.Upstream,
		Site:            payload.Site,
	}
	comboPricingMap := resolveProfitBoardComboPricingMap(query, resolvedBatches)
	excludedUserSet := profitBoardExcludedUserSet(payload.ExcludedUserIDs)
	siteUseRechargePrice, sitePriceFactor, sitePriceFactorNote := profitBoardSharedSiteMeta(comboPricingMap)
	siteRevenueAllocation := newProfitBoardSiteRevenueAllocation()

	report := &ProfitBoardReport{
		Signature:      signature,
		Batches:        resolvedBatches,
		BatchSummaries: make([]ProfitBoardBatchSummary, 0, len(resolvedBatches)),
		Meta: ProfitBoardMeta{
			SiteUseRechargePrice:      siteUseRechargePrice,
			SitePriceFactor:           roundProfitBoardAmount(sitePriceFactor),
			SitePriceFactorNote:       sitePriceFactorNote,
			GeneratedAt:               common.GetTimestamp(),
			CumulativeScope:           "all_time",
			FixedTotalAmountScope:     "created_at_once",
			FixedAmountAllocationMode: "request_count",
			UpstreamFixedTotalAmount:  0,
			SiteFixedTotalAmount:      0,
			LegacyUpstreamFixedAmount: profitBoardLegacyFixedAmountEnabled(payload.Upstream),
			LegacySiteFixedAmount:     profitBoardLegacyFixedAmountEnabled(payload.Site),
		},
	}

	batchSummaryMap := make(map[string]*ProfitBoardBatchSummary, len(resolvedBatches))
	timeBuckets := make(map[string]*ProfitBoardTimeseriesPoint)
	latestLogId := 0
	latestLogCreatedAt := int64(0)
	accountWalletCombos := profitBoardWalletObserverCombosByAccount(comboPricingMap)
	accountWalletAggregates := make(map[int]*profitBoardUpstreamAccountObservedAggregate, len(accountWalletCombos))
	for _, batch := range resolvedBatches {
		batchSummaryMap[batch.Id] = &ProfitBoardBatchSummary{BatchId: batch.Id, BatchName: batch.Name}
	}

	for accountID := range accountWalletCombos {
		aggregate, aggregateErr := collectProfitBoardUpstreamAccountObservedAggregate(
			accountID,
			0,
			common.GetTimestamp(),
			"day",
			0,
			false,
		)
		if aggregateErr != nil {
			return nil, aggregateErr
		}
		accountWalletAggregates[accountID] = aggregate
	}

	remoteAggregate, remoteErr := collectProfitBoardRemoteObserverAggregate(
		signature,
		resolvedBatches,
		payload.ComboConfigs,
		0,
		common.GetTimestamp(),
		"day",
		0,
		false,
		false,
	)
	if remoteErr != nil {
		return nil, remoteErr
	}
	report.RemoteObserverStates = remoteAggregate.States
	report.Summary.RemoteObservedCostUSD += remoteAggregate.TotalCostUSD
	report.Warnings = append(report.Warnings, remoteAggregate.Warnings...)
	for batchID, observedCostUSD := range remoteAggregate.BatchCostUSD {
		if summary := batchSummaryMap[batchID]; summary != nil {
			summary.RemoteObservedCostUSD += observedCostUSD
		}
	}

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
		_, excludedFromRevenue := excludedUserSet[row.UserId]
		comboPricing := prepared.ComboPricing
		configuredSiteRevenueUSD := 0.0
		sitePricingSource := ""
		sitePricingKnown := false
		if excludedFromRevenue {
			sitePricingSource = "excluded_user"
			sitePricingKnown = true
		} else if comboPricing.SiteMode == ProfitBoardComboSiteModeLogQuota {
			logQuotaModelNames := comboPricing.SharedSite.ModelNames
			logQuotaMatched := len(logQuotaModelNames) == 0
			if !logQuotaMatched {
				for _, mn := range logQuotaModelNames {
					if mn == row.ModelName {
						logQuotaMatched = true
						break
					}
				}
			}
			if logQuotaMatched {
				configuredSiteRevenueUSD = float64(row.Quota) / common.QuotaPerUnit
				sitePricingSource = "log_quota"
				sitePricingKnown = row.Quota > 0
			}
		} else if comboPricing.SiteMode == ProfitBoardComboSiteModeSharedSite {
			sharedSiteConfig := comboPricing.SharedSite
			configuredSiteRevenueUSD, sitePricingSource, sitePricingKnown = profitBoardSiteModelRevenueUSD(
				row,
				prepared.InputTokens,
				prepared.CacheReadTokens,
				prepared.CacheCreationTokens,
				ProfitBoardTokenPricingConfig{
					PricingMode:      ProfitBoardSitePricingSiteModel,
					ModelNames:       sharedSiteConfig.ModelNames,
					Group:            sharedSiteConfig.Group,
					UseRechargePrice: sharedSiteConfig.UseRechargePrice,
					PlanID:           sharedSiteConfig.PlanID,
				},
				pricingMap,
				groupRatios,
			)
			if !sitePricingKnown {
				if amount, source, ok := profitBoardManualSiteRevenueUSD(
					row,
					prepared.InputTokens,
					prepared.CacheReadTokens,
					prepared.CacheCreationTokens,
					comboPricing.SiteRules,
				); ok {
					configuredSiteRevenueUSD = amount
					sitePricingSource = source
					sitePricingKnown = true
				} else {
					sitePricingSource = "site_model_missing"
				}
			}
		} else {
			configuredSiteRevenueUSD, sitePricingSource, sitePricingKnown = profitBoardManualSiteRevenueUSD(
				row,
				prepared.InputTokens,
				prepared.CacheReadTokens,
				prepared.CacheCreationTokens,
				comboPricing.SiteRules,
			)
		}
		upstreamCostUSD := 0.0
		upstreamCostSource := ""
		upstreamCostKnown := false
		isWalletCombo := profitBoardComboUsesWalletObserver(comboPricing)
		if isWalletCombo {
			upstreamCostSource = "wallet_observer"
		} else {
			upstreamCostUSD, upstreamCostSource, upstreamCostKnown = profitBoardUpstreamCostUSD(
				row,
				prepared.Other,
				prepared.InputTokens,
				prepared.CacheReadTokens,
				prepared.CacheCreationTokens,
				comboPricing.CostSource,
				comboPricing.UpstreamRules,
				comboPricing.UpstreamFixedTotalAmount,
			)
		}
		configuredSiteRevenueCNY := 0.0
		if sitePricingKnown {
			configuredSiteRevenueCNY = profitBoardConfiguredSiteRevenueCNY(configuredSiteRevenueUSD, comboPricing)
		}
		upstreamCostCNY := 0.0
		if !isWalletCombo && upstreamCostKnown {
			upstreamCostCNY = profitBoardConfiguredUpstreamCostCNY(upstreamCostUSD, comboPricing)
		}

		report.Summary.RequestCount++
		batchSummary.RequestCount++
		if !excludedFromRevenue {
			siteRevenueAllocation.EligibleBatchRequestCount[prepared.Batch.Id]++
		}
		if sitePricingKnown && strings.HasPrefix(sitePricingSource, "site_model_") {
			report.Summary.SiteModelMatchCount++
			batchSummary.SiteModelMatchCount++
		}
		if !sitePricingKnown {
			report.Summary.MissingSitePricingCount++
			batchSummary.MissingSitePricingCount++
		}
		if !isWalletCombo {
			if upstreamCostKnown {
				report.Summary.KnownUpstreamCostCount++
				batchSummary.KnownUpstreamCostCount++
				report.Summary.UpstreamCostUSD += upstreamCostUSD
				report.Summary.UpstreamCostCNY += upstreamCostCNY
				batchSummary.UpstreamCostUSD += upstreamCostUSD
				batchSummary.UpstreamCostCNY += upstreamCostCNY
				switch upstreamCostSource {
				case "returned_cost":
					report.Summary.ReturnedCostCount++
					batchSummary.ReturnedCostCount++
				case "manual_rule", "manual_default", "manual_fixed_total_only":
					report.Summary.ManualCostCount++
					batchSummary.ManualCostCount++
				}
			} else {
				report.Summary.MissingUpstreamCostCount++
				batchSummary.MissingUpstreamCostCount++
			}
		}

		report.Summary.ActualSiteRevenueUSD += actualSiteRevenueUSD
		batchSummary.ActualSiteRevenueUSD += actualSiteRevenueUSD
		if sitePricingKnown {
			report.Summary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
			report.Summary.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNY
			batchSummary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSD
			batchSummary.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNY
		}
		if isWalletCombo && sitePricingKnown {
			configuredProfitUSD := configuredSiteRevenueUSD
			configuredProfitCNY := profitBoardConfiguredProfitCNY(configuredSiteRevenueUSD, 0, comboPricing)
			report.Summary.ConfiguredProfitUSD += configuredProfitUSD
			report.Summary.ConfiguredProfitCNY += configuredProfitCNY
			batchSummary.ConfiguredProfitUSD += configuredProfitUSD
			batchSummary.ConfiguredProfitCNY += configuredProfitCNY
		} else if upstreamCostKnown && sitePricingKnown {
			configuredProfitUSD := configuredSiteRevenueUSD - upstreamCostUSD
			configuredProfitCNY := profitBoardConfiguredProfitCNY(configuredSiteRevenueUSD, upstreamCostUSD, comboPricing)
			report.Summary.ConfiguredProfitUSD += configuredProfitUSD
			report.Summary.ConfiguredProfitCNY += configuredProfitCNY
			batchSummary.ConfiguredProfitUSD += configuredProfitUSD
			batchSummary.ConfiguredProfitCNY += configuredProfitCNY
		}
		if isWalletCombo {
			actualProfitUSD := actualSiteRevenueUSD
			report.Summary.ActualProfitUSD += actualProfitUSD
			batchSummary.ActualProfitUSD += actualProfitUSD
		} else if upstreamCostKnown {
			actualProfitUSD := actualSiteRevenueUSD - upstreamCostUSD
			report.Summary.ActualProfitUSD += actualProfitUSD
			batchSummary.ActualProfitUSD += actualProfitUSD
		}

		bucketTimestamp, bucketLabel := buildProfitBoardBucket(
			row.CreatedAt,
			"day",
			0,
		)
		timeKey := fmt.Sprintf("%s:%d", prepared.Batch.Id, bucketTimestamp)
		point, ok := timeBuckets[timeKey]
		if !ok {
			point = &ProfitBoardTimeseriesPoint{
				BatchId:         prepared.Batch.Id,
				BatchName:       prepared.Batch.Name,
				Bucket:          bucketLabel,
				BucketTimestamp: bucketTimestamp,
			}
			timeBuckets[timeKey] = point
		}
		point.RequestCount++
		return nil
	}); err != nil {
		return nil, err
	}

	report.Timeseries = make([]ProfitBoardTimeseriesPoint, 0, len(timeBuckets))
	for _, point := range timeBuckets {
		report.Timeseries = append(report.Timeseries, *point)
	}

	for _, summary := range batchSummaryMap {
		summary.ActualSiteRevenueUSD = roundProfitBoardAmount(summary.ActualSiteRevenueUSD)
		summary.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(summary.ConfiguredSiteRevenueUSD)
		summary.UpstreamCostUSD = roundProfitBoardAmount(summary.UpstreamCostUSD)
		summary.RemoteObservedCostUSD = roundProfitBoardAmount(summary.RemoteObservedCostUSD)
		summary.ConfiguredProfitUSD = roundProfitBoardAmount(summary.ConfiguredProfitUSD)
		summary.ActualProfitUSD = roundProfitBoardAmount(summary.ActualProfitUSD)
		roundProfitBoardConfiguredMetrics(&summary.ProfitBoardSummary)
		report.BatchSummaries = append(report.BatchSummaries, *summary)
	}
	sort.Slice(report.BatchSummaries, func(i, j int) bool {
		return report.BatchSummaries[i].BatchName < report.BatchSummaries[j].BatchName
	})
	for accountID, comboIDs := range accountWalletCombos {
		applyProfitBoardObservedWalletCost(
			report,
			accountWalletAggregates[accountID],
			comboPricingMap,
			resolvedBatches,
			comboIDs,
			"day",
			0,
		)
	}
	applyProfitBoardComboFixedTotals(
		report,
		comboPricingMap,
		siteRevenueAllocation,
		resolvedBatches,
		0,
		common.GetTimestamp(),
		"day",
		0,
	)
	if report.Summary.RequestCount > 0 {
		knownOrWalletCount := report.Summary.KnownUpstreamCostCount
		for _, batch := range resolvedBatches {
			if profitBoardComboUsesWalletObserver(comboPricingMap[batch.Id]) {
				knownOrWalletCount += batchSummaryMap[batch.Id].RequestCount
			}
		}
		report.Summary.ConfiguredProfitCoverageRate = float64(knownOrWalletCount) / float64(report.Summary.RequestCount)
	}
	if report.Summary.MissingUpstreamCostCount > 0 {
		report.Warnings = append(report.Warnings, "累计总览中部分日志未命中上游成本配置，已按可用规则回退，仍无法确定的记为未知")
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
	report.Summary.ActualSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ActualSiteRevenueUSD)
	report.Summary.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ConfiguredSiteRevenueUSD)
	report.Summary.UpstreamCostUSD = roundProfitBoardAmount(report.Summary.UpstreamCostUSD)
	report.Summary.RemoteObservedCostUSD = roundProfitBoardAmount(report.Summary.RemoteObservedCostUSD)
	report.Summary.ConfiguredProfitUSD = roundProfitBoardAmount(report.Summary.ConfiguredProfitUSD)
	report.Summary.ActualProfitUSD = roundProfitBoardAmount(report.Summary.ActualProfitUSD)
	report.Summary.ConfiguredProfitCoverageRate = roundProfitBoardAmount(report.Summary.ConfiguredProfitCoverageRate)
	roundProfitBoardConfiguredMetrics(&report.Summary)
	report.Timeseries = nil
	report.Meta.LatestLogId = latestLogId
	report.Meta.LatestLogCreatedAt = latestLogCreatedAt
	walletSnapshotWatermark, watermarkErr := buildProfitBoardWalletSnapshotWatermark(comboPricingMap)
	if watermarkErr != nil {
		return nil, watermarkErr
	}
	report.Meta.ActivityWatermark = buildProfitBoardCombinedActivityWatermark(
		report.Summary.RequestCount,
		latestLogId,
		latestLogCreatedAt,
		walletSnapshotWatermark,
	)
	report.Warnings = uniqueProfitBoardWarnings(report.Warnings)
	return report, nil
}

func GenerateProfitBoardReport(query ProfitBoardQuery) (*ProfitBoardReport, error) {
	query.IncludeDetails = false
	return generateProfitBoardReport(query, false)
}

func profitBoardSitePricingSourceLabel(source string) string {
	switch source {
	case "excluded_user":
		return "排除收入"
	case "manual", "manual_rule":
		return "手动价格"
	case "manual_default":
		return "手动默认规则"
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
	case "manual_rule":
		return "手动价格回退"
	case "manual_default":
		return "手动默认规则"
	case "wallet_observer":
		return "上游钱包扣减"
	default:
		return source
	}
}

func QueryProfitBoardDetails(query ProfitBoardDetailQuery) (*ProfitBoardDetailPage, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 12
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query.IncludeDetails = true
	query.DetailLimit = 2000
	report, err := generateProfitBoardReport(query.ProfitBoardQuery, false)
	if err != nil {
		return nil, err
	}

	rows := report.DetailRows
	channelTagMap := make(map[int]string)
	for _, batch := range report.Batches {
		for _, channel := range batch.ResolvedChannels {
			channelTagMap[channel.Id] = channel.Tag
		}
	}
	if strings.TrimSpace(query.ViewBatchId) != "" && query.ViewBatchId != "all" {
		filtered := make([]ProfitBoardDetailRow, 0, len(rows))
		for _, row := range rows {
			if row.BatchId == query.ViewBatchId {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	if filterType := strings.TrimSpace(query.DetailFilter.Type); filterType != "" && strings.TrimSpace(query.DetailFilter.Value) != "" {
		filtered := make([]ProfitBoardDetailRow, 0, len(rows))
		for _, row := range rows {
			if query.DetailFilter.BatchId != "" && row.BatchId != query.DetailFilter.BatchId {
				continue
			}
			switch filterType {
			case "channel":
				if row.ChannelName == query.DetailFilter.Value {
					filtered = append(filtered, row)
				}
			case "tag":
				if channelTagMap[row.ChannelId] == query.DetailFilter.Value {
					filtered = append(filtered, row)
				}
			case "model":
				if row.ModelName == query.DetailFilter.Value {
					filtered = append(filtered, row)
				}
			case "trend":
				_, bucketLabel := buildProfitBoardBucket(row.CreatedAt, query.Granularity, query.CustomIntervalMinutes)
				if bucketLabel == query.DetailFilter.Value {
					filtered = append(filtered, row)
				}
			default:
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}

	total := len(rows)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return &ProfitBoardDetailPage{
		Rows:     rows[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
