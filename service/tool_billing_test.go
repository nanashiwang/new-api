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
