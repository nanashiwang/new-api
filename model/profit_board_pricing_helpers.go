package model

import (
	"fmt"
	"math"
	"strconv"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

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
