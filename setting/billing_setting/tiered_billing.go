package billing_setting

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/samber/lo"
)

const (
	BillingModeRatio      = "ratio"
	BillingModeTieredExpr = "tiered_expr"
	BillingModeField      = "billing_mode"
	BillingExprField      = "billing_expr"
	TieredBundleOptionKey = "billing_setting.tiered_bundle"
)

// BillingSetting is managed by config.GlobalConfig.Register.
// DB keys: billing_setting.billing_mode, billing_setting.billing_expr
type BillingSetting struct {
	BillingMode map[string]string `json:"billing_mode"`
	BillingExpr map[string]string `json:"billing_expr"`
}

var billingSetting = BillingSetting{
	BillingMode: make(map[string]string),
	BillingExpr: make(map[string]string),
}

var billingSettingMu sync.RWMutex

type TieredBundle struct {
	BillingMode map[string]string `json:"billing_mode"`
	BillingExpr map[string]string `json:"billing_expr"`
}

func init() {
	config.GlobalConfig.Register("billing_setting", &billingSetting)
}

// ---------------------------------------------------------------------------
// Read accessors (hot path, must be fast)
// ---------------------------------------------------------------------------

func GetBillingMode(model string) string {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	if mode, ok := billingSetting.BillingMode[model]; ok {
		return mode
	}
	return BillingModeRatio
}

func GetBillingExpr(model string) (string, bool) {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	expr, ok := billingSetting.BillingExpr[model]
	return expr, ok
}

func GetTieredExpr(model string) (string, bool) {
	mode, exprStr := GetBillingConfig(model)
	return exprStr, mode == BillingModeTieredExpr && exprStr != ""
}

func GetBillingConfig(model string) (string, string) {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	mode := billingSetting.BillingMode[model]
	if mode == "" {
		mode = BillingModeRatio
	}
	exprStr := strings.TrimSpace(billingSetting.BillingExpr[model])
	return mode, exprStr
}

func GetBillingModeCopy() map[string]string {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	return lo.Assign(billingSetting.BillingMode)
}

func GetBillingExprCopy() map[string]string {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	return lo.Assign(billingSetting.BillingExpr)
}

func ParseAndValidateBundle(raw string) (TieredBundle, error) {
	bundle := TieredBundle{}
	if err := common.UnmarshalJsonStr(raw, &bundle); err != nil {
		return TieredBundle{}, fmt.Errorf("阶梯计费配置必须是 JSON 对象: %w", err)
	}
	return validateBundle(bundle)
}

func ParseAndValidateFields(modeRaw, exprRaw string) (TieredBundle, error) {
	bundle := TieredBundle{}
	if err := common.UnmarshalJsonStr(modeRaw, &bundle.BillingMode); err != nil {
		return TieredBundle{}, fmt.Errorf("阶梯计费模式配置无效: %w", err)
	}
	if err := common.UnmarshalJsonStr(exprRaw, &bundle.BillingExpr); err != nil {
		return TieredBundle{}, fmt.Errorf("阶梯计费表达式配置无效: %w", err)
	}
	return validateBundle(bundle)
}

func validateBundle(bundle TieredBundle) (TieredBundle, error) {
	if bundle.BillingMode == nil {
		bundle.BillingMode = make(map[string]string)
	}
	if bundle.BillingExpr == nil {
		bundle.BillingExpr = make(map[string]string)
	}
	for model, mode := range bundle.BillingMode {
		if mode != BillingModeRatio && mode != BillingModeTieredExpr {
			return TieredBundle{}, fmt.Errorf("模型 %s 的计费模式无效: %s", model, mode)
		}
		if mode != BillingModeTieredExpr {
			continue
		}
		exprStr := strings.TrimSpace(bundle.BillingExpr[model])
		if exprStr == "" {
			return TieredBundle{}, fmt.Errorf("模型 %s 已启用阶梯计费但缺少表达式", model)
		}
		if err := SmokeTestExpr(exprStr); err != nil {
			return TieredBundle{}, fmt.Errorf("模型 %s 的阶梯计费表达式无效: %w", model, err)
		}
		bundle.BillingExpr[model] = exprStr
	}
	return bundle, nil
}

func ValidateFieldUpdate(field, raw string) error {
	bundle := TieredBundle{
		BillingMode: GetBillingModeCopy(),
		BillingExpr: GetBillingExprCopy(),
	}
	switch field {
	case BillingModeField:
		if err := common.UnmarshalJsonStr(raw, &bundle.BillingMode); err != nil {
			return fmt.Errorf("阶梯计费模式配置无效: %w", err)
		}
	case BillingExprField:
		if err := common.UnmarshalJsonStr(raw, &bundle.BillingExpr); err != nil {
			return fmt.Errorf("阶梯计费表达式配置无效: %w", err)
		}
	default:
		return fmt.Errorf("未知阶梯计费配置字段: %s", field)
	}
	_, err := validateBundle(bundle)
	return err
}

func ReplaceBundle(bundle TieredBundle) {
	billingSettingMu.Lock()
	billingSetting.BillingMode = lo.Assign(bundle.BillingMode)
	billingSetting.BillingExpr = lo.Assign(bundle.BillingExpr)
	billingSettingMu.Unlock()
	billingexpr.InvalidateCache()
}

func UpdateField(field, raw string) error {
	switch field {
	case BillingModeField:
		var modes map[string]string
		if err := common.UnmarshalJsonStr(raw, &modes); err != nil {
			return fmt.Errorf("阶梯计费模式配置无效: %w", err)
		}
		for model, mode := range modes {
			if mode != BillingModeRatio && mode != BillingModeTieredExpr {
				return fmt.Errorf("模型 %s 的计费模式无效: %s", model, mode)
			}
		}
		billingSettingMu.Lock()
		billingSetting.BillingMode = lo.Assign(modes)
		billingSettingMu.Unlock()
	case BillingExprField:
		var expressions map[string]string
		if err := common.UnmarshalJsonStr(raw, &expressions); err != nil {
			return fmt.Errorf("阶梯计费表达式配置无效: %w", err)
		}
		for model, exprStr := range expressions {
			exprStr = strings.TrimSpace(exprStr)
			if exprStr == "" {
				return fmt.Errorf("模型 %s 的阶梯计费表达式为空", model)
			}
			if err := SmokeTestExpr(exprStr); err != nil {
				return fmt.Errorf("模型 %s 的阶梯计费表达式无效: %w", model, err)
			}
			expressions[model] = exprStr
		}
		billingSettingMu.Lock()
		billingSetting.BillingExpr = lo.Assign(expressions)
		billingSettingMu.Unlock()
	default:
		return fmt.Errorf("未知阶梯计费配置字段: %s", field)
	}
	billingexpr.InvalidateCache()
	return nil
}

func GetPricingSyncData(base map[string]any) map[string]any {
	billingSettingMu.RLock()
	modes := lo.Assign(billingSetting.BillingMode)
	exprs := lo.Assign(billingSetting.BillingExpr)
	billingSettingMu.RUnlock()
	extra := make(map[string]any, 2)
	if len(modes) > 0 {
		extra[BillingModeField] = modes
	}
	if len(exprs) > 0 {
		extra[BillingExprField] = exprs
	}
	return lo.Assign(base, extra)
}

// ---------------------------------------------------------------------------
// Smoke test (called externally for validation before save)
// ---------------------------------------------------------------------------

func SmokeTestExpr(exprStr string) error {
	return smokeTestExpr(exprStr)
}

func smokeTestExpr(exprStr string) error {
	vectors := []billingexpr.TokenParams{
		{P: 0, C: 0, Len: 0},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
		{P: 1000000, C: 1000000, Len: 1000000},
	}
	requests := []billingexpr.RequestInput{
		{},
		{
			Headers: map[string]string{
				"anthropic-beta": "fast-mode-2026-02-01",
			},
			Body: []byte(`{"service_tier":"fast","stream_options":{"include_usage":true},"messages":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`),
		},
	}

	for _, v := range vectors {
		for _, request := range requests {
			result, _, err := billingexpr.RunExprWithRequest(exprStr, v, request)
			if err != nil {
				return fmt.Errorf("vector {p=%g, c=%g}: run failed: %w", v.P, v.C, err)
			}
			if math.IsNaN(result) || math.IsInf(result, 0) || result < 0 {
				return fmt.Errorf("vector {p=%g, c=%g}: result must be finite and non-negative, got %v", v.P, v.C, result)
			}
		}
	}
	return nil
}
