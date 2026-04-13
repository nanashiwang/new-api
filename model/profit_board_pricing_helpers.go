package model

import (
	"fmt"
	"math"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func roundProfitBoardAmount(value float64) float64 {
	return math.Round(value*1000000) / 1000000
}

func normalizeProfitBoardExchangeRate(rate float64) float64 {
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 {
		return 1
	}
	return rate
}

func validateProfitBoardExchangeRate(rate float64) bool {
	return !math.IsNaN(rate) && !math.IsInf(rate, 0) && rate > 0
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

func profitBoardConfiguredSiteRevenueCNY(siteRevenueUSD float64, config profitBoardResolvedComboPricing) float64 {
	return siteRevenueUSD * normalizeProfitBoardExchangeRate(config.SiteExchangeRate)
}

func profitBoardConfiguredUpstreamCostCNY(upstreamCostUSD float64, config profitBoardResolvedComboPricing) float64 {
	return upstreamCostUSD * normalizeProfitBoardExchangeRate(config.UpstreamExchangeRate)
}

func profitBoardConfiguredProfitCNY(siteRevenueUSD, upstreamCostUSD float64, config profitBoardResolvedComboPricing) float64 {
	return profitBoardConfiguredSiteRevenueCNY(siteRevenueUSD, config) -
		profitBoardConfiguredUpstreamCostCNY(upstreamCostUSD, config)
}

func roundProfitBoardConfiguredMetrics(summary *ProfitBoardSummary) {
	if summary == nil {
		return
	}
	summary.ConfiguredSiteRevenueCNY = roundProfitBoardAmount(summary.ConfiguredSiteRevenueCNY)
	summary.UpstreamCostCNY = roundProfitBoardAmount(summary.UpstreamCostCNY)
	summary.ConfiguredProfitCNY = roundProfitBoardAmount(summary.ConfiguredProfitCNY)
}

func roundProfitBoardConfiguredTimeseriesMetrics(point *ProfitBoardTimeseriesPoint) {
	if point == nil {
		return
	}
	point.ConfiguredSiteRevenueCNY = roundProfitBoardAmount(point.ConfiguredSiteRevenueCNY)
	point.UpstreamCostCNY = roundProfitBoardAmount(point.UpstreamCostCNY)
	point.ConfiguredProfitCNY = roundProfitBoardAmount(point.ConfiguredProfitCNY)
}

func roundProfitBoardConfiguredBreakdownMetrics(item *ProfitBoardBreakdownItem) {
	if item == nil {
		return
	}
	item.ConfiguredSiteRevenueCNY = roundProfitBoardAmount(item.ConfiguredSiteRevenueCNY)
	item.UpstreamCostCNY = roundProfitBoardAmount(item.UpstreamCostCNY)
	item.ConfiguredProfitCNY = roundProfitBoardAmount(item.ConfiguredProfitCNY)
}

func roundProfitBoardConfiguredDetailMetrics(item *ProfitBoardDetailRow) {
	if item == nil {
		return
	}
	item.ConfiguredSiteRevenueCNY = roundProfitBoardAmount(item.ConfiguredSiteRevenueCNY)
	item.UpstreamCostCNY = roundProfitBoardAmount(item.UpstreamCostCNY)
	item.ConfiguredProfitCNY = roundProfitBoardAmount(item.ConfiguredProfitCNY)
}

func profitBoardPlanPriceFactor(planID int) float64 {
	if planID <= 0 {
		return 1
	}
	plan, err := GetSubscriptionPlanById(planID)
	if err != nil || plan == nil {
		return 1
	}
	totalAmount := float64(plan.TotalAmount)
	if totalAmount <= 0 {
		return 1
	}
	quotaPerResetUSD := totalAmount / common.QuotaPerUnit

	var durationSeconds int64
	switch plan.DurationUnit {
	case "year":
		durationSeconds = int64(plan.DurationValue) * 365 * 86400
	case "month":
		durationSeconds = int64(plan.DurationValue) * 30 * 86400
	case "day":
		durationSeconds = int64(plan.DurationValue) * 86400
	case "hour":
		durationSeconds = int64(plan.DurationValue) * 3600
	case "custom":
		durationSeconds = plan.CustomSeconds
	}

	var totalQuotaUSD float64
	if plan.QuotaResetPeriod == "never" || plan.QuotaResetPeriod == "" {
		totalQuotaUSD = quotaPerResetUSD
	} else {
		var resetSeconds int64
		switch plan.QuotaResetPeriod {
		case "daily":
			resetSeconds = 86400
		case "weekly":
			resetSeconds = 7 * 86400
		case "monthly":
			resetSeconds = 30 * 86400
		case "custom":
			resetSeconds = plan.QuotaResetCustomSeconds
			if resetSeconds <= 0 {
				resetSeconds = durationSeconds
			}
		default:
			resetSeconds = durationSeconds
		}
		numPeriods := int64(1)
		if resetSeconds > 0 && durationSeconds > 0 {
			numPeriods = durationSeconds / resetSeconds
			if numPeriods < 1 {
				numPeriods = 1
			}
		}
		totalQuotaUSD = quotaPerResetUSD * float64(numPeriods)
	}

	if totalQuotaUSD <= 0 {
		return 1
	}

	planPriceCNY := plan.PriceAmount
	if plan.Currency == "USD" || plan.Currency == "" {
		planPriceCNY *= operation_setting.USDExchangeRate
	}

	if operation_setting.USDExchangeRate <= 0 {
		return 1
	}
	effectiveRate := planPriceCNY / totalQuotaUSD
	return effectiveRate / operation_setting.USDExchangeRate
}

func profitBoardPlanPriceFactorMeta(planID int) (float64, string) {
	if planID <= 0 {
		return 1, "当前按本站模型原价重算"
	}
	factor := profitBoardPlanPriceFactor(planID)
	plan, err := GetSubscriptionPlanById(planID)
	if err != nil || plan == nil {
		return 1, fmt.Sprintf("已选择套餐 ID=%d，但套餐不存在，按原价重算", planID)
	}
	if math.Abs(factor-1) < 0.000001 {
		return factor, fmt.Sprintf("已选择套餐「%s」，当前套餐倍率为 1.000x，与原价一致", plan.Title)
	}
	return factor, fmt.Sprintf("已选择套餐「%s」，当前套餐倍率为 %.3fx", plan.Title, factor)
}
