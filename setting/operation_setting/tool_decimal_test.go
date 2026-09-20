package operation_setting

import (
	"github.com/QuantumNous/new-api/common"
	"testing"
)

func TestFourDecimalToolPrices(t *testing.T) {
	old := toolPriceSetting.Prices
	t.Cleanup(func() { toolPriceSetting.Prices = old; RebuildToolPriceIndex() })
	if err := common.UnmarshalJsonStr(`{"prices":{"web_search":0.0001,"web_search:decimal-*":0.1234,"file_search":0}}`, &toolPriceSetting); err != nil {
		t.Fatal(err)
	}
	RebuildToolPriceIndex()
	for _, tc := range []struct {
		tool, model string
		want        float64
	}{
		{"web_search", "other", 0.0001}, {"web_search", "decimal-model", 0.1234}, {"file_search", "other", 0},
	} {
		if got := GetToolPriceForModel(tc.tool, tc.model); got != tc.want {
			t.Fatalf("%s/%s: %v != %v", tc.tool, tc.model, got, tc.want)
		}
	}
}
