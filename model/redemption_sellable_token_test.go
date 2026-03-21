package model

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	redemptionSellableTokenMigrateOnce sync.Once
	redemptionSellableTokenMigrateErr  error
)

func setupRedemptionSellableTokenTest(t *testing.T) {
	t.Helper()
	setupInviteCommissionSubscriptionTest(t)
	redemptionSellableTokenMigrateOnce.Do(func() {
		createTableIfNotExists := func(name string, table any) bool {
			if DB.Migrator().HasTable(table) {
				return true
			}
			if err := DB.Migrator().CreateTable(table); err != nil {
				redemptionSellableTokenMigrateErr = fmt.Errorf("create table %s failed: %w", name, err)
				return false
			}
			return true
		}

		if !createTableIfNotExists("redemptions", &Redemption{}) {
			return
		}
		if !createTableIfNotExists("sellable_token_products", &SellableTokenProduct{}) {
			return
		}
		if !createTableIfNotExists("sellable_token_issuances", &SellableTokenIssuance{}) {
			return
		}
	})
	require.NoError(t, redemptionSellableTokenMigrateErr)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&SellableTokenIssuance{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&SellableTokenProduct{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Redemption{}).Error)
}

func createSellableTokenProductForTest(t *testing.T, name string) *SellableTokenProduct {
	t.Helper()
	product := &SellableTokenProduct{
		Name:       name,
		Status:     SellableTokenProductStatusEnabled,
		PriceQuota: 0,
		TotalQuota: 1000,
	}
	require.NoError(t, CreateSellableTokenProduct(product))
	return product
}

func createSellableTokenRedemptionForTest(t *testing.T, productID int) *Redemption {
	t.Helper()
	redemption := &Redemption{
		UserId:                 1,
		Key:                    common.GetUUID(),
		Status:                 common.RedemptionCodeStatusEnabled,
		Name:                   "可售令牌兑换码",
		BenefitType:            RedemptionBenefitTypeSellableToken,
		SellableTokenProductId: productID,
		CreatedTime:            common.GetTimestamp(),
	}
	require.NoError(t, redemption.Insert())
	return redemption
}

func TestRedeemWithResult_SellableTokenBenefitCreatesPendingIssuance(t *testing.T) {
	setupRedemptionSellableTokenTest(t)

	product := createSellableTokenProductForTest(t, "测试令牌商品")
	user := createInviteCommissionTestUser(t, "invitee_redeem_sellable_token", 0)
	redemption := createSellableTokenRedemptionForTest(t, product.Id)

	result, err := RedeemWithResult(redemption.Key, user.Id)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, RedemptionBenefitTypeSellableToken, result.BenefitType)
	assert.Equal(t, product.Id, result.ProductId)
	assert.Equal(t, product.Name, result.ProductName)
	assert.NotZero(t, result.IssuanceId)

	var issuance SellableTokenIssuance
	require.NoError(t, DB.First(&issuance, "id = ?", result.IssuanceId).Error)
	assert.Equal(t, user.Id, issuance.UserId)
	assert.Equal(t, product.Id, issuance.ProductId)
	assert.Equal(t, SellableTokenSourceTypeRedeem, issuance.SourceType)
	assert.Equal(t, redemption.Id, issuance.SourceId)
	assert.Equal(t, SellableTokenIssuanceStatusPending, issuance.Status)

	var refreshed Redemption
	require.NoError(t, DB.First(&refreshed, "id = ?", redemption.Id).Error)
	assert.Equal(t, common.RedemptionCodeStatusUsed, refreshed.Status)
	assert.Equal(t, user.Id, refreshed.UsedUserId)
}

func TestRedeemWithResult_SellableTokenBenefitRejectsSecondUse(t *testing.T) {
	setupRedemptionSellableTokenTest(t)

	product := createSellableTokenProductForTest(t, "重复兑换商品")
	firstUser := createInviteCommissionTestUser(t, "first_redeem_sellable_token", 0)
	secondUser := createInviteCommissionTestUser(t, "second_redeem_sellable_token", 0)
	redemption := createSellableTokenRedemptionForTest(t, product.Id)

	_, err := RedeemWithResult(redemption.Key, firstUser.Id)
	require.NoError(t, err)

	_, err = RedeemWithResult(redemption.Key, secondUser.Id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRedemptionAlreadyUsed))
}

func TestGetAllRedemptions_FillsSellableTokenProductName(t *testing.T) {
	setupRedemptionSellableTokenTest(t)

	product := createSellableTokenProductForTest(t, "列表商品名")
	redemption := createSellableTokenRedemptionForTest(t, product.Id)

	redemptions, total, err := GetAllRedemptions(0, 10)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, redemptions, 1)
	assert.Equal(t, redemption.Id, redemptions[0].Id)
	assert.Equal(t, product.Name, redemptions[0].ProductName)
}

func TestSearchRedemptions_FillsSellableTokenProductName(t *testing.T) {
	setupRedemptionSellableTokenTest(t)

	product := createSellableTokenProductForTest(t, "搜索商品名")
	createSellableTokenRedemptionForTest(t, product.Id)

	redemptions, total, err := SearchRedemptions("可售令牌", 0, 10)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, redemptions, 1)
	assert.Equal(t, product.Name, redemptions[0].ProductName)
}
