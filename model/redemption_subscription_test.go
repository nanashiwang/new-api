package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupRedemptionSubscriptionTest 在现有订阅返佣测试夹具基础上补齐兑换码表。
func setupRedemptionSubscriptionTest(t *testing.T) {
	t.Helper()
	setupInviteCommissionSubscriptionTest(t)
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Redemption{}).Error)
	_ = getSubscriptionPlanCache().Purge()
	_ = getSubscriptionPlanInfoCache().Purge()
}

// createSubscriptionRedemptionForTest 用于构造套餐兑换码测试数据。
func createSubscriptionRedemptionForTest(t *testing.T, planID int, mode string, quantity int) *Redemption {
	t.Helper()
	redemption := &Redemption{
		UserId:                       1,
		Key:                          common.GetUUID(),
		Status:                       common.RedemptionCodeStatusEnabled,
		Name:                         "套餐兑换码",
		BenefitType:                  RedemptionBenefitTypeSubscription,
		PlanId:                       planID,
		SubscriptionPurchaseMode:     mode,
		SubscriptionPurchaseQuantity: quantity,
		CreatedTime:                  common.GetTimestamp(),
	}
	require.NoError(t, redemption.Insert())
	return redemption
}

func TestRedeemWithResult_SubscriptionBenefitCreatesSubscriptionAndCommissionLedger(t *testing.T) {
	setupRedemptionSubscriptionTest(t)

	originEnabled := common.InviterCommissionEnabled
	originRate := common.InviterRechargeCommissionRate
	originQuotaPerUnit := common.QuotaPerUnit
	originPrice := operation_setting.Price
	t.Cleanup(func() {
		common.InviterCommissionEnabled = originEnabled
		common.InviterRechargeCommissionRate = originRate
		common.QuotaPerUnit = originQuotaPerUnit
		operation_setting.Price = originPrice
	})
	common.InviterCommissionEnabled = true
	common.InviterRechargeCommissionRate = 0.1
	common.QuotaPerUnit = 1000
	operation_setting.Price = 8

	inviter := createInviteCommissionTestUser(t, "inviter_redeem_subscription", 0)
	invitee := createInviteCommissionTestUser(t, "invitee_redeem_subscription", inviter.Id)
	plan := createSubscriptionPlanForInviteCommissionTest(t, "专业套餐", 88, 3000)
	redemption := createSubscriptionRedemptionForTest(t, plan.Id, SubscriptionPurchaseModeStack, 1)

	result, err := RedeemWithResult(redemption.Key, invitee.Id)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, RedemptionBenefitTypeSubscription, result.BenefitType)
	assert.Equal(t, plan.Id, result.PlanId)
	assert.Equal(t, plan.Title, result.PlanTitle)
	assert.Equal(t, SubscriptionPurchaseModeStack, result.PurchaseMode)
	assert.Equal(t, 1, result.PurchaseQuantity)

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", invitee.Id, plan.Id).First(&sub).Error)
	assert.Equal(t, "redemption", sub.Source)
	assert.EqualValues(t, plan.TotalAmount, sub.AmountTotal)
	assert.Equal(t, "active", sub.Status)

	var ledger InviteCommissionLedger
	require.NoError(t, DB.Where("topup_trade_no = ? AND inviter_user_id = ?", fmt.Sprintf("redeem:%d", redemption.Id), inviter.Id).First(&ledger).Error)
	assert.Equal(t, invitee.Id, ledger.InviteeUserId)
	assert.Equal(t, inviter.Id, ledger.InviterUserId)
	assert.Equal(t, 11000, ledger.BaseQuota)
	assert.Equal(t, 1100, ledger.CommissionQuota)
}

func TestRedeemWithResult_SubscriptionBenefitUsesCurrentPlanConfiguration(t *testing.T) {
	setupRedemptionSubscriptionTest(t)

	plan := createSubscriptionPlanForInviteCommissionTest(t, "月付套餐", 20, 1000)
	invitee := createInviteCommissionTestUser(t, "invitee_follow_current_plan", 0)
	redemption := createSubscriptionRedemptionForTest(t, plan.Id, SubscriptionPurchaseModeStack, 1)

	// 兑换前修改套餐，兑换码应跟随当前套餐配置。
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Updates(map[string]any{
		"title":        "月付套餐-新版",
		"total_amount": int64(4321),
	}).Error)
	InvalidateSubscriptionPlanCache(plan.Id)

	result, err := RedeemWithResult(redemption.Key, invitee.Id)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "月付套餐-新版", result.PlanTitle)

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", invitee.Id, plan.Id).First(&sub).Error)
	assert.EqualValues(t, 4321, sub.AmountTotal)
}

func TestRedeem_SubscriptionBenefitRejectedOnLegacyQuotaEndpoint(t *testing.T) {
	setupRedemptionSubscriptionTest(t)

	plan := createSubscriptionPlanForInviteCommissionTest(t, "兼容性套餐", 30, 2000)
	invitee := createInviteCommissionTestUser(t, "invitee_legacy_reject", 0)
	redemption := createSubscriptionRedemptionForTest(t, plan.Id, SubscriptionPurchaseModeStack, 1)

	quota, err := Redeem(redemption.Key, invitee.Id)
	require.Error(t, err)
	assert.Equal(t, 0, quota)
	assert.Contains(t, err.Error(), "套餐兑换码")

	var subCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", invitee.Id).Count(&subCount).Error)
	assert.EqualValues(t, 0, subCount)

	var refreshed Redemption
	require.NoError(t, DB.First(&refreshed, "id = ?", redemption.Id).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, refreshed.Status)
	assert.Equal(t, 0, refreshed.UsedUserId)
}

func TestRedeemWithOptions_SubscriptionRenewCreatesWhenNoActiveSubscription(t *testing.T) {
	setupRedemptionSubscriptionTest(t)

	plan := createSubscriptionPlanForInviteCommissionTest(t, "续费自动开通套餐", 50, 2500)
	invitee := createInviteCommissionTestUser(t, "invitee_redeem_auto_create", 0)
	redemption := createSubscriptionRedemptionForTest(t, plan.Id, SubscriptionPurchaseModeRenew, 1)

	result, err := RedeemWithOptions(redemption.Key, invitee.Id, 0)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, SubscriptionPurchaseModeRenew, result.PurchaseMode)

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", invitee.Id, plan.Id).First(&sub).Error)
	assert.Equal(t, "redemption", sub.Source)
	assert.Equal(t, "active", sub.Status)
}

func TestRedeemWithOptions_SubscriptionRenewRequiresTargetWhenMultipleActiveSubscriptions(t *testing.T) {
	setupRedemptionSubscriptionTest(t)

	plan := createSubscriptionPlanForInviteCommissionTest(t, "续费目标选择套餐", 60, 2600)
	invitee := createInviteCommissionTestUser(t, "invitee_redeem_choose_target", 0)
	redemption := createSubscriptionRedemptionForTest(t, plan.Id, SubscriptionPurchaseModeRenew, 1)

	now := common.GetTimestamp()
	first := &UserSubscription{UserId: invitee.Id, PlanId: plan.Id, AmountTotal: 2600, StartTime: now - 3600, EndTime: now + 86400, Status: "active", Source: "order"}
	second := &UserSubscription{UserId: invitee.Id, PlanId: plan.Id, AmountTotal: 2600, StartTime: now - 7200, EndTime: now + 172800, Status: "active", Source: "order"}
	require.NoError(t, DB.Create(first).Error)
	require.NoError(t, DB.Create(second).Error)

	_, err := RedeemWithOptions(redemption.Key, invitee.Id, 0)
	require.Error(t, err)
	needTargetErr, ok := err.(*RedeemNeedRenewTargetError)
	require.True(t, ok)
	assert.Equal(t, plan.Id, needTargetErr.PlanId)
	assert.Len(t, needTargetErr.Options, 2)

	result, err := RedeemWithOptions(redemption.Key, invitee.Id, first.Id)
	require.NoError(t, err)
	require.NotNil(t, result)

	var refreshedFirst UserSubscription
	require.NoError(t, DB.First(&refreshedFirst, "id = ?", first.Id).Error)
	assert.True(t, refreshedFirst.EndTime > first.EndTime)
}
