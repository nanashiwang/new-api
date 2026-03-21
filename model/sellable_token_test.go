package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateSellableTokenProduct_PersistsZeroValues(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&SellableTokenProduct{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM sellable_token_products")
	})

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
