package model_setting

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

//var claudeHeadersSettings = map[string][]string{}
//
//var ClaudeThinkingAdapterEnabled = true
//var ClaudeThinkingAdapterMaxTokens = 8192
//var ClaudeThinkingAdapterBudgetTokensPercentage = 0.8

// ClaudeSettings 定义Claude模型的配置
type ClaudeSettings struct {
	HeadersSettings                       map[string]map[string][]string `json:"model_headers_settings"`
	DefaultMaxTokens                      map[string]int                 `json:"default_max_tokens"`
	ThinkingAdapterEnabled                bool                           `json:"thinking_adapter_enabled"`
	ThinkingAdapterBudgetTokensPercentage float64                        `json:"thinking_adapter_budget_tokens_percentage"`
}

// 默认配置
var defaultClaudeSettings = ClaudeSettings{
	HeadersSettings:        map[string]map[string][]string{},
	ThinkingAdapterEnabled: true,
	DefaultMaxTokens: map[string]int{
		"default": 8192,
	},
	ThinkingAdapterBudgetTokensPercentage: 0.8,
}

// 全局实例
var claudeSettings = defaultClaudeSettings

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("claude", &claudeSettings)
}

// GetClaudeSettings 获取Claude配置
func GetClaudeSettings() *ClaudeSettings {
	// check default max tokens must have default key
	if _, ok := claudeSettings.DefaultMaxTokens["default"]; !ok {
		claudeSettings.DefaultMaxTokens["default"] = 8192
	}
	return &claudeSettings
}

func (c *ClaudeSettings) WriteHeaders(originModel string, httpHeader *http.Header) {
	if headers, ok := c.HeadersSettings[originModel]; ok {
		for headerKey, headerValues := range headers {
			mergedValues := normalizeHeaderListValues(
				append(append([]string(nil), httpHeader.Values(headerKey)...), headerValues...),
			)
			if len(mergedValues) == 0 {
				continue
			}
			httpHeader.Set(headerKey, strings.Join(mergedValues, ","))
		}
	}
	sanitizeClaudeHeaders(originModel, httpHeader)
}

func normalizeHeaderListValues(values []string) []string {
	normalizedValues := make([]string, 0, len(values))
	seenValues := make(map[string]struct{}, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			normalizedItem := strings.TrimSpace(item)
			if normalizedItem == "" {
				continue
			}
			if _, exists := seenValues[normalizedItem]; exists {
				continue
			}
			seenValues[normalizedItem] = struct{}{}
			normalizedValues = append(normalizedValues, normalizedItem)
		}
	}
	return normalizedValues
}

func sanitizeClaudeHeaders(originModel string, httpHeader *http.Header) {
	if httpHeader == nil {
		return
	}

	sanitizedValue, ok := SanitizeClaudeHeaderValue(originModel, "anthropic-beta", strings.Join(httpHeader.Values("anthropic-beta"), ","))
	if !ok {
		httpHeader.Del("anthropic-beta")
		return
	}
	httpHeader.Set("anthropic-beta", sanitizedValue)
}

func shouldStripClaudeContext1MBeta(originModel string) bool {
	return strings.HasPrefix(originModel, "claude-sonnet-4-6") ||
		strings.HasPrefix(originModel, "claude-opus-4-6") ||
		strings.HasPrefix(originModel, "claude-opus-4-7")
}

func SanitizeClaudeHeaderValue(originModel string, headerKey string, headerValue string) (string, bool) {
	if !strings.EqualFold(strings.TrimSpace(headerKey), "anthropic-beta") {
		trimmed := strings.TrimSpace(headerValue)
		return trimmed, trimmed != ""
	}

	normalizedValues := normalizeHeaderListValues([]string{headerValue})
	if shouldStripClaudeContext1MBeta(originModel) {
		filteredValues := normalizedValues[:0]
		for _, value := range normalizedValues {
			if value == "context-1m-2025-08-07" {
				continue
			}
			filteredValues = append(filteredValues, value)
		}
		normalizedValues = filteredValues
	}

	if len(normalizedValues) == 0 {
		return "", false
	}
	return strings.Join(normalizedValues, ","), true
}

func (c *ClaudeSettings) GetDefaultMaxTokens(model string) int {
	if maxTokens, ok := c.DefaultMaxTokens[model]; ok {
		return maxTokens
	}
	return c.DefaultMaxTokens["default"]
}
