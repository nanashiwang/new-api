package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestApplyPricingCacheSupport(t *testing.T) {
	originalCacheRatioJSON := ratio_setting.CacheRatio2JSONString()
	originalCreateCacheRatioJSON := ratio_setting.CreateCacheRatio2JSONString()
	defer func() {
		if err := ratio_setting.UpdateCacheRatioByJSONString(originalCacheRatioJSON); err != nil {
			t.Fatalf("restore cache ratio failed: %v", err)
		}
		if err := ratio_setting.UpdateCreateCacheRatioByJSONString(originalCreateCacheRatioJSON); err != nil {
			t.Fatalf("restore create cache ratio failed: %v", err)
		}
	}()

	cacheRatioJSON, err := common.Marshal(map[string]float64{
		"cache-supported-model": 0.5,
	})
	if err != nil {
		t.Fatalf("marshal cache ratio failed: %v", err)
	}
	createCacheRatioJSON, err := common.Marshal(map[string]float64{
		"cache-supported-model": 1.25,
	})
	if err != nil {
		t.Fatalf("marshal create cache ratio failed: %v", err)
	}

	if err = ratio_setting.UpdateCacheRatioByJSONString(string(cacheRatioJSON)); err != nil {
		t.Fatalf("update cache ratio failed: %v", err)
	}
	if err = ratio_setting.UpdateCreateCacheRatioByJSONString(string(createCacheRatioJSON)); err != nil {
		t.Fatalf("update create cache ratio failed: %v", err)
	}

	perTokenPricing := Pricing{QuotaType: 0}
	applyPricingCacheSupport(&perTokenPricing, "cache-supported-model")
	if !perTokenPricing.SupportsCacheRead || perTokenPricing.CacheRatio != 0.5 {
		t.Fatalf("unexpected cache read pricing: %+v", perTokenPricing)
	}
	if !perTokenPricing.SupportsCacheCreation || perTokenPricing.CacheCreationRatio != 1.25 {
		t.Fatalf("unexpected cache creation pricing: %+v", perTokenPricing)
	}

	unsupportedPricing := Pricing{QuotaType: 0}
	applyPricingCacheSupport(&unsupportedPricing, "cache-unsupported-model")
	if unsupportedPricing.SupportsCacheRead || unsupportedPricing.SupportsCacheCreation {
		t.Fatalf("unsupported model should not expose cache pricing: %+v", unsupportedPricing)
	}

	perCallPricing := Pricing{QuotaType: 1}
	applyPricingCacheSupport(&perCallPricing, "cache-supported-model")
	if perCallPricing.SupportsCacheRead || perCallPricing.SupportsCacheCreation {
		t.Fatalf("per-call pricing should ignore cache support: %+v", perCallPricing)
	}
}
