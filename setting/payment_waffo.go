package setting

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

var (
	WaffoEnabled               bool
	WaffoApiKey                string
	WaffoPrivateKey            string
	WaffoPublicCert            string
	WaffoSandboxPublicCert     string
	WaffoSandboxApiKey         string
	WaffoSandboxPrivateKey     string
	WaffoSandbox               bool
	WaffoMerchantId            string
	WaffoNotifyUrl             string
	WaffoReturnUrl             string
	WaffoSubscriptionReturnUrl string
	WaffoCurrency              string
	WaffoUnitPrice             float64 = 1.0
	WaffoMinTopUp              int     = 1
)

// GetWaffoPayMethods 从 options 读取 Waffo 支付方式配置
func GetWaffoPayMethods() []constant.WaffoPayMethod {
	common.OptionMapRWMutex.RLock()
	jsonStr := common.OptionMap["WaffoPayMethods"]
	common.OptionMapRWMutex.RUnlock()

	if jsonStr == "" {
		return copyDefaultWaffoPayMethods()
	}
	var methods []constant.WaffoPayMethod
	if err := common.UnmarshalJsonStr(jsonStr, &methods); err != nil {
		return copyDefaultWaffoPayMethods()
	}
	return sanitizeWaffoPayMethods(methods)
}

// SetWaffoPayMethods 序列化 Waffo 支付方式配置并更新 OptionMap
func SetWaffoPayMethods(methods []constant.WaffoPayMethod) error {
	methods = sanitizeWaffoPayMethods(methods)
	jsonBytes, err := common.Marshal(methods)
	if err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	common.OptionMap["WaffoPayMethods"] = string(jsonBytes)
	common.OptionMapRWMutex.Unlock()
	return nil
}

func copyDefaultWaffoPayMethods() []constant.WaffoPayMethod {
	cp := make([]constant.WaffoPayMethod, len(constant.DefaultWaffoPayMethods))
	copy(cp, constant.DefaultWaffoPayMethods)
	return cp
}

// WaffoPayMethods2JsonString 将默认 WaffoPayMethods 序列化为 JSON 字符串（供 InitOptionMap 使用）
func WaffoPayMethods2JsonString() string {
	jsonBytes, err := common.Marshal(constant.DefaultWaffoPayMethods)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func sanitizeWaffoPayMethods(methods []constant.WaffoPayMethod) []constant.WaffoPayMethod {
	sanitized := make([]constant.WaffoPayMethod, 0, len(methods))
	for _, method := range methods {
		method.Icon = sanitizeWaffoPayMethodIcon(method.Icon)
		sanitized = append(sanitized, method)
	}
	return sanitized
}

func sanitizeWaffoPayMethodIcon(icon string) string {
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return ""
	}
	lower := strings.ToLower(icon)
	if strings.HasPrefix(lower, "data:image/png;base64,") ||
		strings.HasPrefix(lower, "data:image/jpeg;base64,") ||
		strings.HasPrefix(lower, "data:image/jpg;base64,") ||
		strings.HasPrefix(lower, "data:image/webp;base64,") ||
		strings.HasPrefix(lower, "data:image/gif;base64,") ||
		strings.HasPrefix(icon, "/") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "http://") {
		return icon
	}
	return ""
}
