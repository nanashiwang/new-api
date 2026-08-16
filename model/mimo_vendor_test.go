package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newMiMoVendorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Vendor{}, &Model{}))
	return db
}

func TestNormalizeMiMoVendorMetadataRenamesLegacyAndBackfillsUnassignedModels(t *testing.T) {
	db := newMiMoVendorTestDB(t)
	legacy := Vendor{Name: " mImO ", Status: 1}
	require.NoError(t, db.Create(&legacy).Error)
	require.NoError(t, db.Create(&Model{ModelName: "mimo-v2.5", VendorID: legacy.Id, Status: 1}).Error)
	require.NoError(t, db.Create(&Model{ModelName: "xiaomi/mimo-v2.5-asr", VendorID: 0, Status: 1}).Error)
	require.NoError(t, db.Create(&Model{ModelName: "mimosa", VendorID: 0, Status: 1}).Error)

	result, err := normalizeMiMoVendorMetadata(db)
	require.NoError(t, err)
	require.Equal(t, legacy.Id, result.VendorID)
	require.True(t, result.RenamedVendor)
	require.True(t, result.UpdatedIcon)
	require.EqualValues(t, 1, result.ReassignedModels)

	var normalized Vendor
	require.NoError(t, db.First(&normalized, legacy.Id).Error)
	require.Equal(t, defaultMiMoVendorName, normalized.Name)
	require.Equal(t, "Xiaomi.color='#FF6900'", normalized.Icon)

	var models []Model
	require.NoError(t, db.Order("model_name ASC").Find(&models).Error)
	assignments := make(map[string]int, len(models))
	for _, item := range models {
		assignments[item.ModelName] = item.VendorID
	}
	require.Equal(t, legacy.Id, assignments["mimo-v2.5"])
	require.Equal(t, legacy.Id, assignments["xiaomi/mimo-v2.5-asr"])
	require.Zero(t, assignments["mimosa"])

	second, err := normalizeMiMoVendorMetadata(db)
	require.NoError(t, err)
	require.False(t, second.changed(), "normalization must be idempotent")
}

func TestNormalizeMiMoVendorMetadataPreservesExplicitAndMixedLegacyAssignments(t *testing.T) {
	db := newMiMoVendorTestDB(t)
	canonical := Vendor{Name: defaultMiMoVendorName, Icon: "CustomXiaomi", Status: 1}
	legacy := Vendor{Name: legacyMiMoVendorName, Status: 1}
	custom := Vendor{Name: "Custom Vendor", Status: 1}
	require.NoError(t, db.Create(&canonical).Error)
	require.NoError(t, db.Create(&legacy).Error)
	require.NoError(t, db.Create(&custom).Error)
	require.NoError(t, db.Create(&Model{ModelName: "mimo-v2.5", VendorID: legacy.Id, Status: 1}).Error)
	require.NoError(t, db.Create(&Model{ModelName: "qwen-plus", VendorID: legacy.Id, Status: 1}).Error)
	require.NoError(t, db.Create(&Model{ModelName: "MiMo-VL-7B", VendorID: custom.Id, Status: 1}).Error)
	require.NoError(t, db.Create(&Model{ModelName: "xiaomi/mimo-v2.5-asr", VendorID: 0, Status: 1}).Error)

	result, err := normalizeMiMoVendorMetadata(db)
	require.NoError(t, err)
	require.Equal(t, canonical.Id, result.VendorID)
	require.EqualValues(t, 2, result.ReassignedModels)
	require.Zero(t, result.RemovedAliases, "legacy vendor still owns a non-MiMo model")

	var byName []Model
	require.NoError(t, db.Order("model_name ASC").Find(&byName).Error)
	assignments := make(map[string]int, len(byName))
	for _, item := range byName {
		assignments[item.ModelName] = item.VendorID
	}
	require.Equal(t, canonical.Id, assignments["mimo-v2.5"])
	require.Equal(t, canonical.Id, assignments["xiaomi/mimo-v2.5-asr"])
	require.Equal(t, custom.Id, assignments["MiMo-VL-7B"], "explicit non-legacy vendor must be preserved")
	require.Equal(t, legacy.Id, assignments["qwen-plus"])

	var persistedCanonical Vendor
	require.NoError(t, db.First(&persistedCanonical, canonical.Id).Error)
	require.Equal(t, "CustomXiaomi", persistedCanonical.Icon, "explicit icon must be preserved")
	require.NoError(t, db.First(&Vendor{}, legacy.Id).Error, "mixed legacy vendor must not be deleted")
}

func TestNormalizeMiMoVendorMetadataSplitsMixedLegacyVendorWithoutCanonical(t *testing.T) {
	db := newMiMoVendorTestDB(t)
	legacy := Vendor{Name: legacyMiMoVendorName, Status: 1}
	require.NoError(t, db.Create(&legacy).Error)
	require.NoError(t, db.Create(&Model{ModelName: "mimo-v2.5", VendorID: legacy.Id, Status: 1}).Error)
	require.NoError(t, db.Create(&Model{ModelName: "qwen-plus", VendorID: legacy.Id, Status: 1}).Error)

	result, err := normalizeMiMoVendorMetadata(db)
	require.NoError(t, err)
	require.True(t, result.CreatedVendor)
	require.NotEqual(t, legacy.Id, result.VendorID)
	require.EqualValues(t, 1, result.ReassignedModels)
	require.Zero(t, result.RemovedAliases)

	var models []Model
	require.NoError(t, db.Find(&models).Error)
	assignments := make(map[string]int, len(models))
	for _, item := range models {
		assignments[item.ModelName] = item.VendorID
	}
	require.Equal(t, result.VendorID, assignments["mimo-v2.5"])
	require.Equal(t, legacy.Id, assignments["qwen-plus"])

	var persistedLegacy Vendor
	require.NoError(t, db.First(&persistedLegacy, legacy.Id).Error)
	require.Equal(t, legacyMiMoVendorName, persistedLegacy.Name)
}

func TestNormalizeMiMoVendorMetadataRemovesOnlyUnusedLegacyAliases(t *testing.T) {
	db := newMiMoVendorTestDB(t)
	canonical := Vendor{Name: defaultMiMoVendorName, Status: 1}
	legacy := Vendor{Name: legacyXiaomiMiMoVendorName, Status: 1}
	require.NoError(t, db.Create(&canonical).Error)
	require.NoError(t, db.Create(&legacy).Error)

	result, err := normalizeMiMoVendorMetadata(db)
	require.NoError(t, err)
	require.Equal(t, 1, result.RemovedAliases)

	var count int64
	require.NoError(t, db.Model(&Vendor{}).Where("id = ?", legacy.Id).Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Unscoped().Model(&Vendor{}).Where("id = ?", legacy.Id).Count(&count).Error)
	require.EqualValues(t, 1, count, "legacy alias removal must remain recoverable through soft delete")
}

func TestNormalizeMiMoVendorMetadataDoesNothingWithoutMiMoData(t *testing.T) {
	db := newMiMoVendorTestDB(t)
	require.NoError(t, db.Create(&Model{ModelName: "qwen-plus", VendorID: 0, Status: 1}).Error)

	result, err := normalizeMiMoVendorMetadata(db)
	require.NoError(t, err)
	require.False(t, result.changed())

	var vendorCount int64
	require.NoError(t, db.Model(&Vendor{}).Count(&vendorCount).Error)
	require.Zero(t, vendorCount)
}

func TestNormalizeMiMoVendorMetadataBackfillsNullVendorID(t *testing.T) {
	db := newMiMoVendorTestDB(t)
	modelRecord := Model{ModelName: "mimo-v2.5", Status: 1}
	require.NoError(t, db.Create(&modelRecord).Error)
	require.NoError(t, db.Model(&Model{}).Where("id = ?", modelRecord.Id).UpdateColumn("vendor_id", nil).Error)

	result, err := normalizeMiMoVendorMetadata(db)
	require.NoError(t, err)
	require.True(t, result.CreatedVendor)
	require.EqualValues(t, 1, result.ReassignedModels)

	var persisted Model
	require.NoError(t, db.First(&persisted, modelRecord.Id).Error)
	require.Equal(t, result.VendorID, persisted.VendorID)
}

func TestFindVendorByCanonicalNamePrefersCanonicalAndSupportsLegacyAlias(t *testing.T) {
	db := newMiMoVendorTestDB(t)
	legacy := Vendor{Name: " mImO ", Status: 1}
	canonical := Vendor{Name: defaultMiMoVendorName, Status: 1}
	require.NoError(t, db.Create(&legacy).Error)
	require.NoError(t, db.Create(&canonical).Error)

	found, err := findVendorByCanonicalName(db, legacyXiaomiMiMoVendorName)
	require.NoError(t, err)
	require.Equal(t, canonical.Id, found.Id)

	require.NoError(t, db.Delete(&canonical).Error)
	found, err = findVendorByCanonicalName(db, defaultMiMoVendorName)
	require.NoError(t, err)
	require.Equal(t, legacy.Id, found.Id)
}

func TestResolveVendorNameForModelUsesCanonicalMiMoName(t *testing.T) {
	require.Equal(t, defaultMiMoVendorName, ResolveVendorNameForModel("mimo-v2.5", ""))
	require.Equal(t, defaultMiMoVendorName, ResolveVendorNameForModel("qwen-plus", legacyMiMoVendorName))
	require.Equal(t, "阿里通义千问", ResolveVendorNameForModel("qwen-plus", " 阿里通义千问 "))
}
