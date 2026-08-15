package model

import (
	"strings"
)

const defaultMiMoVendorName = "MiMo"

// 简化的供应商映射规则
var defaultVendorRules = map[string]string{
	"gpt":      "OpenAI",
	"dall-e":   "OpenAI",
	"whisper":  "OpenAI",
	"o1":       "OpenAI",
	"o3":       "OpenAI",
	"claude":   "Anthropic",
	"gemini":   "Google",
	"moonshot": "Moonshot",
	"kimi":     "Moonshot",
	"chatglm":  "智谱",
	"glm-":     "智谱",
	"qwen":     "阿里巴巴",
	"deepseek": "DeepSeek",
	"abab":     "MiniMax",
	"ernie":    "百度",
	"spark":    "讯飞",
	"hunyuan":  "腾讯",
	"command":  "Cohere",
	"@cf/":     "Cloudflare",
	"360":      "360",
	"yi":       "零一万物",
	"jina":     "Jina",
	"mistral":  "Mistral",
	"grok":     "xAI",
	"llama":    "Meta",
	"doubao":   "字节跳动",
	"kling":    "快手",
	"jimeng":   "即梦",
	"vidu":     "Vidu",
}

// 供应商默认图标映射
var defaultVendorIcons = map[string]string{
	"OpenAI":     "OpenAI",
	"Anthropic":  "Claude.Color",
	"Google":     "Gemini.Color",
	"Moonshot":   "Moonshot",
	"智谱":         "Zhipu.Color",
	"阿里巴巴":       "Qwen.Color",
	"DeepSeek":   "DeepSeek.Color",
	"MiniMax":    "Minimax.Color",
	"百度":         "Wenxin.Color",
	"讯飞":         "Spark.Color",
	"腾讯":         "Hunyuan.Color",
	"Cohere":     "Cohere.Color",
	"Cloudflare": "Cloudflare.Color",
	"360":        "Ai360.Color",
	"零一万物":       "Yi.Color",
	"Jina":       "Jina",
	"Mistral":    "Mistral.Color",
	"xAI":        "XAI",
	"Meta":       "Ollama",
	"字节跳动":       "Doubao.Color",
	"快手":         "Kling.Color",
	"即梦":         "Jimeng.Color",
	"Vidu":       "Vidu",
	"微软":         "AzureAI",
	"Microsoft":  "AzureAI",
	"Azure":      "AzureAI",
	"MiMo":       "Xiaomi.color='#FF6900'",
}

func hasModelNameSegment(modelName string, segments ...string) bool {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(modelName)), func(r rune) bool {
		switch r {
		case '/', ':', '.', '_', '-':
			return true
		default:
			return false
		}
	})
	for _, part := range parts {
		for _, segment := range segments {
			if part == segment {
				return true
			}
		}
	}
	return false
}

func getDefaultVendorName(modelName string) string {
	// MiMo models are commonly published as mimo-* or under a xiaomi/* namespace.
	// Segment matching avoids classifying unrelated names such as "mimosa".
	if hasModelNameSegment(modelName, "mimo", "xiaomi") {
		return defaultMiMoVendorName
	}

	modelLower := strings.ToLower(modelName)
	for pattern, vendorName := range defaultVendorRules {
		if strings.Contains(modelLower, pattern) {
			return vendorName
		}
	}
	return ""
}

func getDefaultVendorID(modelName string, vendorMap map[int]*Vendor) int {
	vendorName := getDefaultVendorName(modelName)
	if vendorName == "" {
		return 0
	}
	return getOrCreateVendor(vendorName, vendorMap)
}

// initDefaultVendorMapping 简化的默认供应商映射
func initDefaultVendorMapping(metaMap map[string]*Model, vendorMap map[int]*Vendor, enableAbilities []AbilityWithChannel) {
	for _, ability := range enableAbilities {
		modelName := ability.Model
		meta, exists := metaMap[modelName]
		if exists && meta != nil {
			if meta.VendorID != 0 || getDefaultVendorName(modelName) != defaultMiMoVendorName {
				continue
			}
			vendorID := getOrCreateVendor(defaultMiMoVendorName, vendorMap)
			if vendorID == 0 {
				continue
			}
			// Non-exact rules can share the same metadata pointer across several
			// concrete models. Copy before applying a display-only fallback so one
			// inferred vendor cannot leak into another matched model.
			resolvedMeta := *meta
			resolvedMeta.VendorID = vendorID
			metaMap[modelName] = &resolvedMeta
			continue
		}

		// 创建模型元数据
		metaMap[modelName] = &Model{
			ModelName: modelName,
			VendorID:  getDefaultVendorID(modelName, vendorMap),
			Status:    1,
			NameRule:  NameRuleExact,
		}
	}
}

// 查找或创建供应商
func getOrCreateVendor(vendorName string, vendorMap map[int]*Vendor) int {
	// 查找现有供应商
	for id, vendor := range vendorMap {
		if vendor.Name == vendorName {
			if vendor.Name == defaultMiMoVendorName && strings.TrimSpace(vendor.Icon) == "" {
				vendor.Icon = getDefaultVendorIcon(vendorName)
			}
			return id
		}
	}

	// 创建新供应商
	newVendor := &Vendor{
		Name:   vendorName,
		Status: 1,
		Icon:   getDefaultVendorIcon(vendorName),
	}

	if err := newVendor.Insert(); err != nil {
		return 0
	}

	vendorMap[newVendor.Id] = newVendor
	return newVendor.Id
}

// 获取供应商默认图标
func getDefaultVendorIcon(vendorName string) string {
	if icon, exists := defaultVendorIcons[vendorName]; exists {
		return icon
	}
	return ""
}
