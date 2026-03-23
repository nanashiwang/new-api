package helper

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
)

func TestEstimateConservativePreConsumeQuota_TextUsesCompletionAndOtherRatios(t *testing.T) {
	originPreConsumedQuota := common.PreConsumedQuota
	t.Cleanup(func() {
		common.PreConsumedQuota = originPreConsumedQuota
	})
	common.PreConsumedQuota = 0

	priceData := types.PriceData{
		ModelRatio:      1.5,
		CompletionRatio: 4,
		GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 2},
		OtherRatios: map[string]float64{
			"seconds": 2,
		},
	}

	got := EstimateConservativePreConsumeQuota(&relaycommon.RelayInfo{}, 10, &types.TokenCountMeta{MaxTokens: 20}, priceData)
	if got != 540 {
		t.Fatalf("unexpected conservative quota: got=%d want=540", got)
	}
}

func TestEstimateConservativePreConsumeQuota_ImageRatioInflatesPromptSide(t *testing.T) {
	originPreConsumedQuota := common.PreConsumedQuota
	t.Cleanup(func() {
		common.PreConsumedQuota = originPreConsumedQuota
	})
	common.PreConsumedQuota = 0

	priceData := types.PriceData{
		ModelRatio:      1,
		CompletionRatio: 1,
		ImageRatio:      2,
		GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
	}

	got := EstimateConservativePreConsumeQuota(&relaycommon.RelayInfo{}, 10, &types.TokenCountMeta{}, priceData)
	if got != 20 {
		t.Fatalf("unexpected image conservative quota: got=%d want=20", got)
	}
}

func TestEstimateConservativePreConsumeQuota_DefaultsCompletionWhenMissing(t *testing.T) {
	originPreConsumedQuota := common.PreConsumedQuota
	t.Cleanup(func() {
		common.PreConsumedQuota = originPreConsumedQuota
	})
	common.PreConsumedQuota = 0

	priceData := types.PriceData{
		ModelRatio:      1,
		CompletionRatio: 2,
		GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
	}

	got := EstimateConservativePreConsumeQuota(&relaycommon.RelayInfo{}, 10, &types.TokenCountMeta{}, priceData)
	if got != 8202 {
		t.Fatalf("unexpected default conservative quota: got=%d want=8202", got)
	}
}

func TestEstimateConservativePreConsumeQuota_IncludesBuiltInToolCosts(t *testing.T) {
	originPreConsumedQuota := common.PreConsumedQuota
	originQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.PreConsumedQuota = originPreConsumedQuota
		common.QuotaPerUnit = originQuotaPerUnit
	})
	common.PreConsumedQuota = 0
	common.QuotaPerUnit = 1000

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5",
		Request:         &dto.OpenAIResponsesRequest{},
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				dto.BuildInToolWebSearchPreview: {
					ToolName:          dto.BuildInToolWebSearchPreview,
					SearchContextSize: "medium",
				},
				dto.BuildInToolFileSearch: {
					ToolName: dto.BuildInToolFileSearch,
				},
			},
		},
	}

	priceData := types.PriceData{
		UsePrice:       true,
		ModelPrice:     0,
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}

	got := EstimateConservativePreConsumeQuota(info, 0, &types.TokenCountMeta{}, priceData)
	if got != 13 {
		t.Fatalf("unexpected tool conservative quota: got=%d want=13", got)
	}
}
