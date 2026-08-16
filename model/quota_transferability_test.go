package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGrantUserQuotaTx_SeparatesPaidAndFreeQuota(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)
	user := createInviteCommissionTestUser(t, "quota_source_grants", 0)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"quota":              50,
		"transferable_quota": 100,
	}).Error)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return GrantUserQuotaTx(tx, user.Id, 30, 0)
	}))
	var afterFree User
	require.NoError(t, DB.First(&afterFree, user.Id).Error)
	assert.Equal(t, 80, afterFree.Quota)
	assert.Equal(t, 50, afterFree.TransferableQuota)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return GrantUserQuotaTx(tx, user.Id, 20, 20)
	}))
	var afterPaid User
	require.NoError(t, DB.First(&afterPaid, user.Id).Error)
	assert.Equal(t, 100, afterPaid.Quota)
	assert.Equal(t, 70, afterPaid.TransferableQuota)
}

func TestRevokeTransferableQuotaGrantTx_RemovesTransferEligibility(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)
	user := createInviteCommissionTestUser(t, "quota_source_revoke", 0)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"quota":              120,
		"transferable_quota": 80,
	}).Error)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return RevokeTransferableQuotaGrantTx(tx, user.Id, 50)
	}))
	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, 70, refreshed.Quota)
	assert.Equal(t, 30, refreshed.TransferableQuota)
}

func TestRevokeTransferableQuotaGrantTx_PartiallyShrinksActiveWalletCode(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "quota_revoke_active_code", 0)
	setWalletQuota(t, creator.Id, 300)
	created, err := CreateWalletFundedRedemption(creator.Id, 300, "quota-revoke-active-code-001")
	require.NoError(t, err)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return RevokeTransferableQuotaGrantTx(tx, creator.Id, 100)
	}))

	var refreshedCreator User
	var refreshedCode Redemption
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	require.NoError(t, DB.First(&refreshedCode, created.Redemption.Id).Error)
	assert.Zero(t, refreshedCreator.Quota)
	assert.Zero(t, refreshedCreator.TransferableQuota)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, refreshedCode.Status)
	assert.Equal(t, 200, refreshedCode.Quota)
	assert.Equal(t, 200, refreshedCode.TransferableQuota)
}

func TestRevokeTransferableQuotaGrantTx_DoesNotDoubleChargeSelfRedeem(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "quota_revoke_self_redeem", 0)
	setWalletQuota(t, creator.Id, 300)
	created, err := CreateWalletFundedRedemption(creator.Id, 100, "quota-revoke-self-code-001")
	require.NoError(t, err)
	_, err = RedeemWithResult(created.Redemption.Key, creator.Id)
	require.NoError(t, err)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return RevokeTransferableQuotaGrantTx(tx, creator.Id, 300)
	}))

	var refreshedCreator User
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	assert.Zero(t, refreshedCreator.Quota)
	assert.Zero(t, refreshedCreator.TransferableQuota)
}

func TestRevokeTransferableQuotaGrantTx_PreservesRecipientsOwnPaidQuota(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "quota_revoke_recipient_creator", 0)
	recipient := createInviteCommissionTestUser(t, "quota_revoke_paid_recipient", 0)
	setWalletQuota(t, creator.Id, 100)
	setWalletQuota(t, recipient.Id, 50)
	created, err := CreateWalletFundedRedemption(creator.Id, 100, "quota-revoke-recipient-code-001")
	require.NoError(t, err)
	_, err = RedeemWithResult(created.Redemption.Key, recipient.Id)
	require.NoError(t, err)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return RevokeTransferableQuotaGrantTx(tx, creator.Id, 100)
	}))

	var refreshedRecipient User
	require.NoError(t, DB.First(&refreshedRecipient, recipient.Id).Error)
	assert.Equal(t, 50, refreshedRecipient.Quota)
	assert.Equal(t, 50, refreshedRecipient.TransferableQuota)
}

func TestPaidTopUpEntryPointsGrantTransferableQuota(t *testing.T) {
	tests := []struct {
		name          string
		paymentMethod string
		provider      string
		amount        int64
		money         float64
		expectedQuota int
		complete      func(*TopUp) error
	}{
		{
			name:          "generic payment callback",
			paymentMethod: "alipay",
			provider:      PaymentProviderEpay,
			amount:        20,
			money:         16,
			expectedQuota: 200,
			complete: func(topUp *TopUp) error {
				return CompleteTopUpByTradeNoWithPayment(topUp.TradeNo, "test", "", topUp.PaymentMethod, topUp.PaymentProvider, nil)
			},
		},
		{
			name:          "stripe callback",
			paymentMethod: PaymentMethodStripe,
			provider:      PaymentProviderStripe,
			amount:        20,
			money:         20,
			expectedQuota: 200,
			complete: func(topUp *TopUp) error {
				return Recharge(topUp.TradeNo, "test_customer", "", 16)
			},
		},
		{
			name:          "creem callback",
			paymentMethod: PaymentMethodCreem,
			provider:      PaymentProviderCreem,
			amount:        250,
			money:         20,
			expectedQuota: 250,
			complete: func(topUp *TopUp) error {
				return RechargeCreem(topUp.TradeNo, "", "", "")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupInviteCommissionSubscriptionTest(t)
			originalQuotaPerUnit := common.QuotaPerUnit
			originalCommissionEnabled := common.InviterCommissionEnabled
			common.QuotaPerUnit = 10
			common.InviterCommissionEnabled = false
			t.Cleanup(func() {
				common.QuotaPerUnit = originalQuotaPerUnit
				common.InviterCommissionEnabled = originalCommissionEnabled
			})

			user := createInviteCommissionTestUser(t, "paid_entry_"+test.provider, 0)
			topUp := &TopUp{
				UserId:          user.Id,
				Amount:          test.amount,
				Money:           test.money,
				PaidMoney:       test.money,
				TradeNo:         "paid-entry-" + test.provider,
				PaymentMethod:   test.paymentMethod,
				PaymentProvider: test.provider,
				CreateTime:      common.GetTimestamp(),
				Status:          common.TopUpStatusPending,
			}
			require.NoError(t, DB.Create(topUp).Error)
			require.NoError(t, test.complete(topUp))
			// Payment providers retry callbacks. A replay must be an idempotent
			// success and must not grant quota a second time.
			require.NoError(t, test.complete(topUp))

			var refreshed User
			require.NoError(t, DB.First(&refreshed, user.Id).Error)
			assert.Equal(t, test.expectedQuota, refreshed.Quota)
			assert.Equal(t, test.expectedQuota, refreshed.TransferableQuota)
		})
	}
}

func TestFreeAndReferralQuotaRemainNonTransferable(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 10
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	user := createInviteCommissionTestUser(t, "non_transferable_rewards", 0)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"quota":              50,
		"transferable_quota": 50,
		"aff_quota":          100,
	}).Error)

	adminCode := &Redemption{
		UserId:        0,
		Key:           common.GetUUID(),
		Status:        common.RedemptionCodeStatusEnabled,
		Name:          "admin gift",
		BenefitType:   RedemptionBenefitTypeQuota,
		Quota:         100,
		CreatedTime:   common.GetTimestamp(),
		FundingSource: RedemptionFundingSourceAdmin,
	}
	require.NoError(t, DB.Create(adminCode).Error)
	_, err := RedeemWithResult(adminCode.Key, user.Id)
	require.NoError(t, err)

	var afterAdminGift User
	require.NoError(t, DB.First(&afterAdminGift, user.Id).Error)
	assert.Equal(t, 150, afterAdminGift.Quota)
	assert.Equal(t, 50, afterAdminGift.TransferableQuota)

	require.NoError(t, afterAdminGift.TransferAffQuotaToQuota(100))
	var afterReferralTransfer User
	require.NoError(t, DB.First(&afterReferralTransfer, user.Id).Error)
	assert.Equal(t, 250, afterReferralTransfer.Quota)
	assert.Equal(t, 50, afterReferralTransfer.TransferableQuota)
}
