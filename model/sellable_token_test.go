package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpdateSellableTokenProduct_PersistsZeroValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	originDB := DB
	originLogDB := LOG_DB
	originRedisEnabled := common.RedisEnabled
	originUsingSQLite := common.UsingSQLite
	originUsingMySQL := common.UsingMySQL
	originUsingPostgreSQL := common.UsingPostgreSQL
	DB = db
	LOG_DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	initCol()
	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.RedisEnabled = originRedisEnabled
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
		initCol()
	})

	require.NoError(t, db.AutoMigrate(&SellableTokenProduct{}))

	product := &SellableTokenProduct{
		Name:                 "runtime-pack",
		Subtitle:             "initial subtitle",
		Status:               SellableTokenProductStatusEnabled,
		SortOrder:            9,
		PriceQuota:           100,
		PriceAmount:          9.9,
		TotalQuota:           1000,
		ModelLimitsEnabled:   true,
		ModelLimits:          "gpt-5",
		AllowedGroups:        "vip",
		MaxConcurrency:       3,
		WindowRequestLimit:   12,
		WindowSeconds:        60,
		PackageEnabled:       true,
		PackageLimitQuota:    200,
		PackagePeriod:        TokenPackagePeriodMonthly,
		PackagePeriodMode:    TokenPackagePeriodModeNatural,
		PackageCustomSeconds: 0,
	}
	require.NoError(t, CreateSellableTokenProduct(product))

	update := &SellableTokenProduct{
		Name:                 "runtime-pack-updated",
		Subtitle:             "",
		Status:               SellableTokenProductStatusDisabled,
		SortOrder:            0,
		PriceQuota:           0,
		PriceAmount:          0,
		TotalQuota:           1000,
		ModelLimitsEnabled:   false,
		ModelLimits:          "",
		AllowedGroups:        "",
		MaxConcurrency:       0,
		WindowRequestLimit:   0,
		WindowSeconds:        0,
		PackageEnabled:       false,
		PackageLimitQuota:    0,
		PackagePeriod:        TokenPackagePeriodNone,
		PackagePeriodMode:    TokenPackagePeriodModeRelative,
		PackageCustomSeconds: 0,
	}
	require.NoError(t, UpdateSellableTokenProduct(product.Id, update))

	reloaded, err := GetSellableTokenProductById(product.Id)
	require.NoError(t, err)
	require.Equal(t, "runtime-pack-updated", reloaded.Name)
	require.Empty(t, reloaded.Subtitle)
	require.Equal(t, SellableTokenProductStatusDisabled, reloaded.Status)
	require.Zero(t, reloaded.SortOrder)
	require.Zero(t, reloaded.PriceQuota)
	require.Zero(t, reloaded.PriceAmount)
	require.False(t, reloaded.ModelLimitsEnabled)
	require.Empty(t, reloaded.ModelLimits)
	require.Empty(t, reloaded.AllowedGroups)
	require.Zero(t, reloaded.MaxConcurrency)
	require.Zero(t, reloaded.WindowRequestLimit)
	require.Zero(t, reloaded.WindowSeconds)
	require.False(t, reloaded.PackageEnabled)
	require.Zero(t, reloaded.PackageLimitQuota)
	require.Equal(t, TokenPackagePeriodNone, reloaded.PackagePeriod)
	require.Equal(t, TokenPackagePeriodModeRelative, reloaded.PackagePeriodMode)
	require.Zero(t, reloaded.PackageCustomSeconds)
}
