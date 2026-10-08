package claude

import (
	"errors"
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func mergeClaudeUsage(state *ClaudeResponseInfo, usage *dto.ClaudeUsage, final bool) {
	if usage == nil {
		return
	}
	if state.UsageError != nil {
		return
	}
	state.Usage.UsageSemantic = "anthropic"
	if usage.InputTokens < 0 || usage.OutputTokens < 0 || usage.CacheReadInputTokens < 0 {
		state.UsageError = errors.New("negative Claude usage")
		return
	}
	if _, err := dto.ResolveCacheCreation(usage.CacheCreationInputTokens, usage.GetCacheCreation5mTokens(), usage.GetCacheCreation1hTokens(), usage.HasCacheCreationTokens()); err != nil {
		state.UsageError = err
		return
	}
	if usage.HasInputTokens() && usage.InputTokens >= 0 {
		state.Usage.PromptTokens = usage.InputTokens
		state.HasInputUsage = true
	}
	if usage.HasOutputTokens() && usage.OutputTokens >= 0 {
		state.Usage.CompletionTokens = usage.OutputTokens
		if final {
			state.HasFinalOutputUsage = true
		}
	}
	// Claude counters are cumulative snapshots, never per-event increments.
	// Output-only final frames must retain the earlier cache breakdown.
	if usage.HasCacheReadTokens() {
		state.Usage.PromptTokensDetails.CachedTokens = usage.CacheReadInputTokens
	}
	if usage.HasCacheCreationTokens() {
		if usage.CacheCreationInputTokens != state.Usage.PromptTokensDetails.CachedCreationTokens {
			// A changed total cannot inherit unreported TTLs from an older
			// snapshot. Keep the new total as unclassified if necessary.
			if !usage.CacheCreation.HasFiveMinute() {
				state.Usage.ClaudeCacheCreation5mTokens = 0
			}
			if !usage.CacheCreation.HasOneHour() {
				state.Usage.ClaudeCacheCreation1hTokens = 0
			}
		}
		state.Usage.PromptTokensDetails.CachedCreationTokens = usage.CacheCreationInputTokens
		state.Usage.CacheCreationTotalReported = true
	}
	if usage.CacheCreation != nil {
		if usage.CacheCreation.HasFiveMinute() {
			state.Usage.ClaudeCacheCreation5mTokens = usage.GetCacheCreation5mTokens()
		}
		if usage.CacheCreation.HasOneHour() {
			state.Usage.ClaudeCacheCreation1hTokens = usage.GetCacheCreation1hTokens()
		}
		if !state.Usage.CacheCreationTotalReported {
			// Derived totals follow the latest snapshot; they are not authority.
			state.Usage.PromptTokensDetails.CachedCreationTokens = 0
		}
	}
	breakdown, err := state.Usage.CacheCreationBreakdown()
	if err != nil {
		state.UsageError = err
		return
	}
	state.Usage.PromptTokensDetails.CachedCreationTokens = breakdown.Total
	total := 0
	for _, count := range []int{state.Usage.PromptTokens, state.Usage.CompletionTokens, state.Usage.PromptTokensDetails.CachedTokens, breakdown.Total} {
		if count < 0 || count > math.MaxInt-total {
			state.UsageError = errors.New("Claude usage total overflow")
			return
		}
		total += count
	}
	state.Usage.TotalTokens = state.Usage.PromptTokens + state.Usage.CompletionTokens
}

func completeClaudeUsage(c *gin.Context, info *relaycommon.RelayInfo, state *ClaudeResponseInfo) {
	usage := state.Usage
	estimated := false
	if !state.HasInputUsage && usage.PromptTokens == 0 && (state.Done || state.ResponseText.Len() > 0 || state.HasFinalOutputUsage) {
		// The request estimate includes cached input; do not charge it twice.
		usage.PromptTokens = max(0, info.GetEstimatePromptTokens()-usage.PromptTokensDetails.CachedTokens-usage.PromptTokensDetails.CachedCreationTokens)
		estimated = true
	}
	if !state.HasFinalOutputUsage {
		usage.CompletionTokens = max(usage.CompletionTokens, service.EstimateTokenByModel(info.UpstreamModelName, state.ResponseText.String()))
		estimated = true
	}
	if estimated {
		common.SetContextKey(c, constant.ContextKeyLocalCountTokens, true)
		usage.InputTokensEstimated = true
	}
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
}

func appendClaudeUsageText(state *ClaudeResponseInfo, block *dto.ClaudeMediaMessage) {
	if block == nil {
		return
	}
	if block.Text != nil {
		state.ResponseText.WriteString(*block.Text)
	}
	if block.Thinking != nil {
		state.ResponseText.WriteString(*block.Thinking)
	}
	if block.PartialJson != nil {
		state.ResponseText.WriteString(*block.PartialJson)
	}
	if block.Type == "tool_use" && block.Input != nil {
		if data, err := common.Marshal(block.Input); err == nil && string(data) != "{}" {
			state.ResponseText.Write(data)
		}
	}
}
