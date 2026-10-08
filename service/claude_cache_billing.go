package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
)

// Adapters other than the native Claude parser may also return Anthropic
// usage. Check the partition at the handler boundary before settling it.
func ValidateClaudeCacheUsage(info *relaycommon.RelayInfo, usage *dto.Usage) *types.NewAPIError {
	if info == nil || usage == nil ||
		(info.GetFinalRequestRelayFormat() != types.RelayFormatClaude && !strings.EqualFold(usage.UsageSemantic, "anthropic")) {
		return nil
	}
	if _, err := usage.CacheCreationBreakdown(); err != nil {
		return types.NewError(err, types.ErrorCodeBadResponseBody, types.ErrOptionWithSkipRetry())
	}
	return nil
}

// ClaudeTokenQuotaAmount retains native Claude's integer-truncation policy.
// Both native and compatibility endpoints use this calculation; no prices
// are inferred from a model name and cache creation totals are not doubled.
func ClaudeTokenQuotaAmount(prompt, completion, read int, creation dto.CacheCreation, price types.PriceData) float64 {
	multiplier := price.GroupRatioInfo.GroupRatio * price.TimeRatioInfo.EffectiveRatio()
	if price.UsePrice {
		return price.ModelPrice * common.QuotaPerUnit * multiplier
	}
	tokens := float64(prompt) + float64(read)*price.CacheRatio +
		float64(creation.FiveMinute)*price.CacheCreation5mRatio +
		float64(creation.OneHour)*price.CacheCreation1hRatio +
		float64(creation.Unclassified)*price.CacheCreationRatio +
		float64(completion)*price.CompletionRatio
	amount := tokens * price.GroupRatioInfo.GroupRatio * price.ModelRatio * price.TimeRatioInfo.EffectiveRatio()
	if tokens > 0 && price.ModelRatio*multiplier > 0 && amount <= 0 {
		return 1
	}
	return amount
}

func AppendClaudeCacheCreationInfo(other map[string]interface{}, creation dto.CacheCreation, price types.PriceData) {
	other["claude"] = true
	other["usage_semantic"] = "anthropic"
	other["cache_creation_tokens"] = creation.Total
	other["cache_creation_ratio"] = price.CacheCreationRatio
	other["cache_creation_tokens_5m"] = creation.FiveMinute
	other["cache_creation_ratio_5m"] = price.CacheCreation5mRatio
	other["cache_creation_tokens_1h"] = creation.OneHour
	other["cache_creation_ratio_1h"] = price.CacheCreation1hRatio
	other["cache_creation_tokens_unclassified"] = creation.Unclassified
}
