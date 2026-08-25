package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	ChannelVendorAll      = "all"
	ChannelVendorAuto     = "auto"
	ChannelVendorProtocol = "protocol"
	ChannelVendorMiMo     = "mimo"

	ChannelCategoryAll          = "all"
	ChannelCategoryTypePrefix   = "type:"
	ChannelCategoryVendorPrefix = "vendor:"
)

type ChannelCategory struct {
	Key    string
	Type   int
	Vendor string
}

func (category ChannelCategory) IsAll() bool {
	return category.Key == ChannelCategoryAll
}

func NormalizeChannelVendorSetting(vendor string) string {
	vendor = strings.ToLower(strings.TrimSpace(vendor))
	switch vendor {
	case "", ChannelVendorAuto:
		return ChannelVendorAuto
	case ChannelVendorProtocol:
		return ChannelVendorProtocol
	case ChannelVendorMiMo, strings.ToLower(defaultMiMoVendorName), "xiaomi", "xiaomi mimo":
		return ChannelVendorMiMo
	default:
		return vendor
	}
}

func IsSupportedChannelVendorSetting(vendor string) bool {
	switch NormalizeChannelVendorSetting(vendor) {
	case ChannelVendorAuto, ChannelVendorProtocol, ChannelVendorMiMo:
		return true
	default:
		return false
	}
}

func (channel *Channel) GetChannelVendorSetting() string {
	if channel == nil || channel.ChannelVendor == nil {
		return ChannelVendorAuto
	}
	return NormalizeChannelVendorSetting(*channel.ChannelVendor)
}

// NormalizeChannelVendorFilter converts the public query value into a stable
// internal key. Unknown values are kept so they fail closed instead of
// accidentally returning every channel.
func NormalizeChannelVendorFilter(vendor string) string {
	vendor = strings.ToLower(strings.TrimSpace(vendor))
	switch vendor {
	case "", ChannelVendorAll:
		return ChannelVendorAll
	case ChannelVendorMiMo, strings.ToLower(defaultMiMoVendorName), "xiaomi", "xiaomi mimo":
		return ChannelVendorMiMo
	default:
		return vendor
	}
}

func ParseChannelCategory(value string) (ChannelCategory, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == ChannelCategoryAll {
		return ChannelCategory{Key: ChannelCategoryAll, Type: -1}, nil
	}
	if strings.HasPrefix(value, ChannelCategoryTypePrefix) {
		typeValue := strings.TrimSpace(strings.TrimPrefix(value, ChannelCategoryTypePrefix))
		channelType, err := strconv.Atoi(typeValue)
		if err != nil || channelType < 0 {
			return ChannelCategory{}, fmt.Errorf("invalid channel category: %s", value)
		}
		return ChannelCategory{Key: ChannelCategoryTypePrefix + strconv.Itoa(channelType), Type: channelType}, nil
	}
	if strings.HasPrefix(value, ChannelCategoryVendorPrefix) {
		vendor := NormalizeChannelVendorFilter(strings.TrimPrefix(value, ChannelCategoryVendorPrefix))
		if vendor == ChannelVendorAll || !IsSupportedChannelVendorSetting(vendor) || vendor == ChannelVendorAuto || vendor == ChannelVendorProtocol {
			return ChannelCategory{}, fmt.Errorf("invalid channel category: %s", value)
		}
		return ChannelCategory{Key: ChannelCategoryVendorPrefix + vendor, Type: -1, Vendor: vendor}, nil
	}
	return ChannelCategory{}, fmt.Errorf("invalid channel category: %s", value)
}

func resolveMappedChannelModel(channel *Channel, modelName string) string {
	if channel == nil || channel.ModelMapping == nil || strings.TrimSpace(*channel.ModelMapping) == "" {
		return modelName
	}
	mapping := make(map[string]string)
	if err := common.Unmarshal([]byte(*channel.ModelMapping), &mapping); err != nil {
		return modelName
	}
	if mapped := strings.TrimSpace(mapping[strings.TrimSpace(modelName)]); mapped != "" {
		return mapped
	}
	return modelName
}

func InferChannelVendor(channel *Channel) string {
	if channel == nil {
		return ""
	}
	modelCount := 0
	for _, modelName := range channel.GetModels() {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		modelCount++
		resolvedModel := resolveMappedChannelModel(channel, modelName)
		if getDefaultVendorName(resolvedModel) != defaultMiMoVendorName {
			return ""
		}
	}
	if modelCount > 0 {
		return ChannelVendorMiMo
	}
	return ""
}

func ResolveChannelVendor(channel *Channel) string {
	switch channel.GetChannelVendorSetting() {
	case ChannelVendorProtocol:
		return ""
	case ChannelVendorAuto:
		return InferChannelVendor(channel)
	default:
		return channel.GetChannelVendorSetting()
	}
}

// ChannelMatchesVendor uses the explicit channel vendor first and falls back
// to conservative model inference. Channel.Type remains the relay protocol.
func ChannelMatchesVendor(channel *Channel, vendor string) bool {
	if channel == nil {
		return false
	}

	switch NormalizeChannelVendorFilter(vendor) {
	case ChannelVendorAll:
		return true
	default:
		return ResolveChannelVendor(channel) == NormalizeChannelVendorFilter(vendor)
	}
}

func FilterChannelsByVendor(channels []*Channel, vendor string) []*Channel {
	vendor = NormalizeChannelVendorFilter(vendor)
	if vendor == ChannelVendorAll {
		return channels
	}

	filtered := make([]*Channel, 0, len(channels))
	for _, channel := range channels {
		if ChannelMatchesVendor(channel, vendor) {
			filtered = append(filtered, channel)
		}
	}
	return filtered
}

// CountChannelVendors returns independent facet counts. "all" is the number
// of distinct channels, while a multi-model channel is counted at most once
// for each supported vendor.
func CountChannelVendors(channels []*Channel) map[string]int64 {
	counts := map[string]int64{
		ChannelVendorAll:  0,
		ChannelVendorMiMo: 0,
	}
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		counts[ChannelVendorAll]++
		if ChannelMatchesVendor(channel, ChannelVendorMiMo) {
			counts[ChannelVendorMiMo]++
		}
	}
	return counts
}

func CountChannelTypes(channels []*Channel) map[int64]int64 {
	counts := make(map[int64]int64)
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		counts[int64(channel.Type)]++
	}
	return counts
}

func CountChannelCategories(channels []*Channel) map[string]int64 {
	counts := map[string]int64{ChannelCategoryAll: 0}
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		counts[ChannelCategoryAll]++
		if vendor := ResolveChannelVendor(channel); vendor != "" {
			counts[ChannelCategoryVendorPrefix+vendor]++
			continue
		}
		counts[ChannelCategoryTypePrefix+strconv.Itoa(channel.Type)]++
	}
	return counts
}

func FilterChannelsByType(channels []*Channel, typeFilter int) []*Channel {
	if typeFilter < 0 {
		return channels
	}
	filtered := make([]*Channel, 0, len(channels))
	for _, channel := range channels {
		if channel != nil && channel.Type == typeFilter {
			filtered = append(filtered, channel)
		}
	}
	return filtered
}

func FilterChannelsByCategory(channels []*Channel, category ChannelCategory) []*Channel {
	if category.IsAll() {
		return channels
	}

	filtered := make([]*Channel, 0, len(channels))
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		if category.Vendor != "" {
			if ChannelMatchesVendor(channel, category.Vendor) {
				filtered = append(filtered, channel)
			}
			continue
		}
		if channel.Type == category.Type && ResolveChannelVendor(channel) == "" {
			filtered = append(filtered, channel)
		}
	}
	return filtered
}
