package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func TestComputeToolCallQuotaCountsImageGenerationCalls(t *testing.T) {
	result := ComputeToolCallQuota(ToolCallUsage{
		ImageGenerationCall:    true,
		ImageGenerationCalls:   2,
		ImageGenerationQuality: "high",
		ImageGenerationSize:    "1536x1024",
	}, 1)

	if len(result.Items) != 1 {
		t.Fatalf("items length = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.CallCount != 2 {
		t.Fatalf("CallCount = %d, want 2", item.CallCount)
	}
	wantPrice := operation_setting.GPTImage1High1536x1024 * 2
	if item.TotalPrice != wantPrice {
		t.Fatalf("TotalPrice = %v, want %v", item.TotalPrice, wantPrice)
	}
	if result.TotalQuota <= 0 {
		t.Fatalf("TotalQuota = %d, want positive", result.TotalQuota)
	}
}

func TestComputeToolCallQuotaUsesMiMoOfficialWebSearchPrice(t *testing.T) {
	result := ComputeToolCallQuota(ToolCallUsage{
		ModelName:         "mimo-v2.5-pro",
		WebSearchCalls:    3,
		WebSearchToolName: "web_search",
	}, 1)
	if len(result.Items) != 1 {
		t.Fatalf("items length = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.PricePer1K != 5 {
		t.Fatalf("PricePer1K = %v, want 5", item.PricePer1K)
	}
	if item.CallCount != 3 {
		t.Fatalf("CallCount = %d, want 3", item.CallCount)
	}
	if item.TotalPrice != 0.015 {
		t.Fatalf("TotalPrice = %v, want 0.015", item.TotalPrice)
	}
	if result.TotalQuota != 7500 {
		t.Fatalf("TotalQuota = %d, want 7500", result.TotalQuota)
	}
}
