package common

import (
	"math"

	"github.com/shopspring/decimal"
)

// CalculateAudioDurationQuota converts an hourly USD input-audio price into
// New API quota units. billedSeconds must already be rounded to the provider's
// accounting precision.
func CalculateAudioDurationQuota(pricePerHour float64, billedSeconds int64, groupRatio float64, timeRatio float64) int {
	if pricePerHour <= 0 || billedSeconds <= 0 || groupRatio <= 0 {
		return 0
	}
	if timeRatio <= 0 {
		timeRatio = 1
	}
	quota := decimal.NewFromFloat(pricePerHour).
		Mul(decimal.NewFromInt(billedSeconds)).
		Div(decimal.NewFromInt(3600)).
		Mul(decimal.NewFromFloat(QuotaPerUnit)).
		Mul(decimal.NewFromFloat(groupRatio)).
		Mul(decimal.NewFromFloat(timeRatio))
	quotaFloat := quota.InexactFloat64()
	if quota.IsNegative() || math.IsNaN(quotaFloat) || math.IsInf(quotaFloat, 0) {
		return 0
	}
	return int(quota.Round(0).IntPart())
}
