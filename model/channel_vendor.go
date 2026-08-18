package model

import "strings"

const (
	ChannelVendorAll  = "all"
	ChannelVendorMiMo = "mimo"
)

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

// ChannelMatchesVendor classifies a channel from its advertised model set.
// Channel.Type remains the transport protocol and is intentionally ignored.
func ChannelMatchesVendor(channel *Channel, vendor string) bool {
	if channel == nil {
		return false
	}

	switch NormalizeChannelVendorFilter(vendor) {
	case ChannelVendorAll:
		return true
	case ChannelVendorMiMo:
		for _, modelName := range channel.GetModels() {
			if getDefaultVendorName(strings.TrimSpace(modelName)) == defaultMiMoVendorName {
				return true
			}
		}
	}
	return false
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

// CountChannelDisplayTypes returns the mutually exclusive categories shown in
// the channel navigation. A recognized vendor override, currently Xiaomi MiMo,
// is displayed as its own top-level category instead of under its transport
// protocol (usually OpenAI-compatible).
func CountChannelDisplayTypes(channels []*Channel) map[int64]int64 {
	counts := make(map[int64]int64)
	for _, channel := range channels {
		if channel == nil || ChannelMatchesVendor(channel, ChannelVendorMiMo) {
			continue
		}
		counts[int64(channel.Type)]++
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

// FilterChannelsByDisplayType mirrors CountChannelDisplayTypes so vendor
// overrides do not also appear in their underlying protocol tab. It never
// changes Channel.Type, which remains the source of truth for relay behavior.
func FilterChannelsByDisplayType(channels []*Channel, typeFilter int) []*Channel {
	if typeFilter < 0 {
		return channels
	}

	filtered := make([]*Channel, 0, len(channels))
	for _, channel := range channels {
		if channel == nil || channel.Type != typeFilter {
			continue
		}
		if ChannelMatchesVendor(channel, ChannelVendorMiMo) {
			continue
		}
		filtered = append(filtered, channel)
	}
	return filtered
}
