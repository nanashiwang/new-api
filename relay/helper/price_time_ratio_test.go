package helper

import (
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func TestModelPriceHelperFreezesTimeRatioIntoPreConsume(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	originalModelPrice := ratio_setting.ModelPrice2JSONString()
	originalTimeRatio := ratio_setting.TimeRatioRules2JSONString()
	t.Cleanup(func() {
		if err := ratio_setting.UpdateModelRatioByJSONString(originalModelRatio); err != nil {
			t.Fatalf("restore model ratio: %v", err)
		}
		if err := ratio_setting.UpdateModelPriceByJSONString(originalModelPrice); err != nil {
			t.Fatalf("restore model price: %v", err)
		}
		if err := ratio_setting.UpdateTimeRatioRulesByJSONString(originalTimeRatio); err != nil {
			t.Fatalf("restore time ratio: %v", err)
		}
	})

	if err := ratio_setting.UpdateModelPriceByJSONString(`{}`); err != nil {
		t.Fatalf("clear model price: %v", err)
	}
	if err := ratio_setting.UpdateModelRatioByJSONString(`{"time-ratio-test-model":2}`); err != nil {
		t.Fatalf("set model ratio: %v", err)
	}
	if err := ratio_setting.UpdateTimeRatioRulesByJSONString(`[
		{"id":"peak","enabled":true,"timezone":"UTC","start":"10:00","end":"11:00","ratio":1.5,"models":["time-ratio-*"],"groups":["default"],"priority":1}
	]`); err != nil {
		t.Fatalf("set time ratio: %v", err)
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		OriginModelName: "time-ratio-test-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		StartTime:       time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
	}

	priceData, err := ModelPriceHelper(ctx, info, 100, &types.TokenCountMeta{MaxTokens: 50})
	if err != nil {
		t.Fatalf("ModelPriceHelper returned error: %v", err)
	}
	if priceData.TimeRatioInfo.RuleID != "peak" {
		t.Fatalf("expected peak time ratio rule, got %q", priceData.TimeRatioInfo.RuleID)
	}
	if priceData.QuotaToPreConsume != 1650 {
		t.Fatalf("expected quota pre-consume 1650, got %d", priceData.QuotaToPreConsume)
	}
	if info.PriceData.TimeRatioInfo.EffectiveRatio() != 1.5 {
		t.Fatalf("relay info did not freeze time ratio")
	}
}
