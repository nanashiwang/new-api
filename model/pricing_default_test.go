package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDefaultVendorNameRecognizesMiMoBoundaries(t *testing.T) {
	for _, modelName := range []string{
		"mimo-v2.5",
		"MiMo-VL-7B",
		"xiaomi/mimo-v2.5-pro",
		"vendor.xiaomi-mimo",
	} {
		t.Run(modelName, func(t *testing.T) {
			require.Equal(t, defaultMiMoVendorName, getDefaultVendorName(modelName))
		})
	}
}

func TestGetDefaultVendorNameDoesNotMisclassifyMiMoSubstrings(t *testing.T) {
	for _, modelName := range []string{
		"mimosa",
		"notmimo-model",
		"xiaomish-model",
		"qwen-tts",
		"",
	} {
		t.Run(modelName, func(t *testing.T) {
			require.NotEqual(t, defaultMiMoVendorName, getDefaultVendorName(modelName))
		})
	}
}

func TestInitDefaultVendorMappingFillsUnassignedMiMoMetadata(t *testing.T) {
	originalMeta := &Model{
		ModelName: "mimo-v2.5",
		VendorID:  0,
		Status:    1,
		NameRule:  NameRuleExact,
	}
	metaMap := map[string]*Model{"mimo-v2.5": originalMeta}
	vendorMap := map[int]*Vendor{
		7: {Id: 7, Name: defaultMiMoVendorName},
	}

	initDefaultVendorMapping(metaMap, vendorMap, []AbilityWithChannel{{
		Ability: Ability{Model: "mimo-v2.5"},
	}})

	require.Equal(t, 7, metaMap["mimo-v2.5"].VendorID)
	require.Zero(t, originalMeta.VendorID, "fallback must not mutate shared rule metadata")
}

func TestInitDefaultVendorMappingPreservesExplicitVendor(t *testing.T) {
	metaMap := map[string]*Model{
		"mimo-v2.5": {
			ModelName: "mimo-v2.5",
			VendorID:  99,
			Status:    1,
			NameRule:  NameRuleExact,
		},
	}
	vendorMap := map[int]*Vendor{
		7:  {Id: 7, Name: defaultMiMoVendorName},
		99: {Id: 99, Name: "Custom Vendor"},
	}

	initDefaultVendorMapping(metaMap, vendorMap, []AbilityWithChannel{{
		Ability: Ability{Model: "mimo-v2.5"},
	}})

	require.Equal(t, 99, metaMap["mimo-v2.5"].VendorID)
}

func TestInitDefaultVendorMappingLeavesOtherUnassignedMetadataUntouched(t *testing.T) {
	originalMeta := &Model{
		ModelName: "qwen-custom",
		VendorID:  0,
		Status:    1,
		NameRule:  NameRuleExact,
	}
	metaMap := map[string]*Model{"qwen-custom": originalMeta}
	vendorMap := map[int]*Vendor{
		7: {Id: 7, Name: "阿里巴巴"},
	}

	initDefaultVendorMapping(metaMap, vendorMap, []AbilityWithChannel{{
		Ability: Ability{Model: "qwen-custom"},
	}})

	require.Same(t, originalMeta, metaMap["qwen-custom"])
	require.Zero(t, metaMap["qwen-custom"].VendorID)
}

func TestGetOrCreateVendorBackfillsEmptyMiMoIconInMemory(t *testing.T) {
	vendor := &Vendor{Id: 7, Name: defaultMiMoVendorName}
	vendorMap := map[int]*Vendor{7: vendor}

	vendorID := getOrCreateVendor(defaultMiMoVendorName, vendorMap)

	require.Equal(t, 7, vendorID)
	require.Equal(t, "Xiaomi.color='#FF6900'", vendor.Icon)
}

func TestGetOrCreateVendorPreservesExplicitMiMoIcon(t *testing.T) {
	vendor := &Vendor{Id: 7, Name: defaultMiMoVendorName, Icon: "CustomIcon"}
	vendorMap := map[int]*Vendor{7: vendor}

	vendorID := getOrCreateVendor(defaultMiMoVendorName, vendorMap)

	require.Equal(t, 7, vendorID)
	require.Equal(t, "CustomIcon", vendor.Icon)
}
