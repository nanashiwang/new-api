package claude

import (
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
	if usage.CacheReadInputTokens > 0 {
		state.Usage.PromptTokensDetails.CachedTokens = usage.CacheReadInputTokens
	}
	if created := usage.GetCacheCreationTotalTokens(); created > 0 {
		state.Usage.PromptTokensDetails.CachedCreationTokens = created
	}
	if tokens := usage.GetCacheCreation5mTokens(); tokens > 0 {
		state.Usage.ClaudeCacheCreation5mTokens = tokens
	}
	if tokens := usage.GetCacheCreation1hTokens(); tokens > 0 {
		state.Usage.ClaudeCacheCreation1hTokens = tokens
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
