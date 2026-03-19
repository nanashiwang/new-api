package relay

import (
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// FinalizeTestConsumeQuota reuses the normal post-consume settlement path for
// token tests after the upstream request has succeeded.
func FinalizeTestConsumeQuota(c *gin.Context, info *relaycommon.RelayInfo, usage *dto.Usage) error {
	if info == nil || usage == nil {
		return nil
	}

	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		originModelName := info.OriginModelName
		originPriceData := info.PriceData

		_, err := helper.ModelPriceHelper(c, info, info.GetEstimatePromptTokens(), &types.TokenCountMeta{})
		if err != nil {
			info.OriginModelName = originModelName
			info.PriceData = originPriceData
			return err
		}

		postConsumeQuota(c, info, usage)
		info.OriginModelName = originModelName
		info.PriceData = originPriceData
		return nil
	}

	if info.RelayFormat == types.RelayFormatClaude {
		service.PostClaudeConsumeQuota(c, info, usage)
		return nil
	}

	containAudioTokens := usage.CompletionTokenDetails.AudioTokens > 0 || usage.PromptTokensDetails.AudioTokens > 0
	containsAudioRatios := ratio_setting.ContainsAudioRatio(info.OriginModelName) || ratio_setting.ContainsAudioCompletionRatio(info.OriginModelName)
	if containAudioTokens && containsAudioRatios {
		service.PostAudioConsumeQuota(c, info, usage, "")
		return nil
	}

	postConsumeQuota(c, info, usage)
	return nil
}
