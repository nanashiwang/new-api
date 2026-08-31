package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRedemptionBatchTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	originalDB := DB
	originalSQLite := common.UsingSQLite
	DB = db
	common.UsingSQLite = true
	require.NoError(t, db.AutoMigrate(&Redemption{}))
	t.Cleanup(func() {
		DB = originalDB
		common.UsingSQLite = originalSQLite
	})
}

func TestBatchManageRedemptionsProtectsWalletCodes(t *testing.T) {
	setupRedemptionBatchTestDB(t)
	adminUnused := &Redemption{Id: 1, UserId: 1, Key: "admin-unused", Status: common.RedemptionCodeStatusEnabled, Quota: 100, FundingSource: RedemptionFundingSourceAdmin}
	walletUnused := &Redemption{Id: 2, UserId: 2, Key: "wallet-unused", Status: common.RedemptionCodeStatusEnabled, Quota: 100, FundingSource: RedemptionFundingSourceWallet}
	adminUsed := &Redemption{Id: 3, UserId: 1, Key: "admin-used", Status: common.RedemptionCodeStatusUsed, Quota: 100, FundingSource: RedemptionFundingSourceAdmin}
	require.NoError(t, DB.Create(adminUnused).Error)
	require.NoError(t, DB.Create(walletUnused).Error)
	require.NoError(t, DB.Create(adminUsed).Error)

	result, err := BatchManageRedemptions([]int{1, 2, 99}, "disable")
	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Equal(t, 2, result.FailedCount)
	var disabled Redemption
	require.NoError(t, DB.First(&disabled, 1).Error)
	require.Equal(t, common.RedemptionCodeStatusDisabled, disabled.Status)
	var wallet Redemption
	require.NoError(t, DB.First(&wallet, 2).Error)
	require.Equal(t, common.RedemptionCodeStatusEnabled, wallet.Status)
}

func TestBatchManageRedemptionsDeletesSelectedAdminCodes(t *testing.T) {
	setupRedemptionBatchTestDB(t)
	require.NoError(t, DB.Create(&Redemption{Id: 1, Key: "admin-delete", Status: common.RedemptionCodeStatusDisabled, FundingSource: RedemptionFundingSourceAdmin}).Error)
	require.NoError(t, DB.Create(&Redemption{Id: 2, Key: "wallet-delete", Status: common.RedemptionCodeStatusEnabled, FundingSource: RedemptionFundingSourceWallet}).Error)
	require.NoError(t, DB.Create(&Redemption{Id: 3, Key: "admin-used-delete", Status: common.RedemptionCodeStatusUsed, FundingSource: RedemptionFundingSourceAdmin}).Error)

	result, err := BatchManageRedemptions([]int{1, 2, 3}, "delete")
	require.NoError(t, err)
	require.Equal(t, 2, result.SuccessCount)
	require.Equal(t, 1, result.FailedCount)
	var count int64
	require.NoError(t, DB.Model(&Redemption{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	var wallet Redemption
	require.NoError(t, DB.First(&wallet, 2).Error)
}
