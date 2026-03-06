package model

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	inviteCommissionSubscriptionMigrateOnce sync.Once
	inviteCommissionSubscriptionMigrateErr  error
)

func setupInviteCommissionSubscriptionTest(t *testing.T) {
	t.Helper()
	inviteCommissionSubscriptionMigrateOnce.Do(func() {
		migrateIfNotExists := func(name string, model any) bool {
			if DB.Migrator().HasTable(model) {
				return true
			}
			if err := DB.AutoMigrate(model); err != nil {
				inviteCommissionSubscriptionMigrateErr = fmt.Errorf("migrate %s failed: %w", name, err)
				return false
			}
			return true
		}

		if !migrateIfNotExists("users", &User{}) {
			return
		}
		if !migrateIfNotExists("invite_commission_ledgers", &InviteCommissionLedger{}) {
			return
		}
		if !migrateIfNotExists("invite_commission_daily_cap_states", &InviteCommissionDailyCapState{}) {
			return
		}
		if !migrateIfNotExists("subscription_plans", &SubscriptionPlan{}) {
			return
		}
		if !migrateIfNotExists("subscription_orders", &SubscriptionOrder{}) {
			return
		}
		if !migrateIfNotExists("user_subscriptions", &UserSubscription{}) {
			return
		}
		if !migrateIfNotExists("top_ups", &TopUp{}) {
			return
		}
	})
	require.NoError(t, inviteCommissionSubscriptionMigrateErr)

	clear := func(model any) {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error)
	}
	clear(&TopUp{})
	clear(&SubscriptionOrder{})
	clear(&UserSubscription{})
	clear(&SubscriptionPlan{})
	clear(&InviteCommissionLedger{})
	clear(&InviteCommissionDailyCapState{})
	clear(&User{})
}

func createSubscriptionPlanForInviteCommissionTest(t *testing.T, title string, priceAmount float64, totalAmount int64) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Title:         title,
		Subtitle:      "",
		PriceAmount:   priceAmount,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   totalAmount,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func createSubscriptionOrderForInviteCommissionTest(t *testing.T, userID, planID int, tradeNo string, money float64, purchaseMode string, renewTargetSubID int) *SubscriptionOrder {
	t.Helper()
	order := &SubscriptionOrder{
		UserId:                    userID,
		PlanId:                    planID,
		Money:                     money,
		TradeNo:                   tradeNo,
		PaymentMethod:             "epay",
		PurchaseMode:              purchaseMode,
		RenewTargetSubscriptionId: renewTargetSubID,
		CreateTime:                common.GetTimestamp(),
		Status:                    common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(order).Error)
	return order
}

func TestCompleteSubscriptionOrder_EnqueueInviteCommissionByPaidAmount_StackAndIdempotent(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)

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

	inviter := createInviteCommissionTestUser(t, "inviter_sub_stack", 0)
	invitee := createInviteCommissionTestUser(t, "invitee_sub_stack", inviter.Id)
	plan := createSubscriptionPlanForInviteCommissionTest(t, "大月卡", 88, 3000000000)

	tradeNo := "sub_invite_stack_001"
	createSubscriptionOrderForInviteCommissionTest(t, invitee.Id, plan.Id, tradeNo, 88, SubscriptionPurchaseModeStack, 0)

	require.NoError(t, CompleteSubscriptionOrder(tradeNo, `{"status":"success"}`))
	// 重复回调幂等：不应重复入返佣台账。
	require.NoError(t, CompleteSubscriptionOrder(tradeNo, `{"status":"success"}`))

	var ledger InviteCommissionLedger
	require.NoError(t, DB.Where("topup_trade_no = ? AND inviter_user_id = ?", tradeNo, inviter.Id).First(&ledger).Error)

	expectedBaseQuota := 11000 // floor(88 * 1000 / 8)
	assert.Equal(t, expectedBaseQuota, ledger.BaseQuota)
	assert.Equal(t, 1100, ledger.CommissionQuota)
	assert.Equal(t, InviteCommissionStatusPending, ledger.Status)
	assert.NotEqual(t, int(plan.TotalAmount), ledger.BaseQuota, "返佣基数必须来自实付金额折算，不能按套餐总额度")

	var count int64
	require.NoError(t, DB.Model(&InviteCommissionLedger{}).Where("topup_trade_no = ? AND inviter_user_id = ?", tradeNo, inviter.Id).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestCompleteSubscriptionOrder_EnqueueInviteCommission_Renew(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)

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

	inviter := createInviteCommissionTestUser(t, "inviter_sub_renew", 0)
	invitee := createInviteCommissionTestUser(t, "invitee_sub_renew", inviter.Id)
	plan := createSubscriptionPlanForInviteCommissionTest(t, "续费月卡", 88, 1000)

	now := time.Now().Unix()
	existingSub := &UserSubscription{
		UserId:      invitee.Id,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		AmountUsed:  0,
		StartTime:   now - 3600,
		EndTime:     now + 86400,
		Status:      "active",
		Source:      "order",
	}
	require.NoError(t, DB.Create(existingSub).Error)

	tradeNo := "sub_invite_renew_001"
	createSubscriptionOrderForInviteCommissionTest(t, invitee.Id, plan.Id, tradeNo, 88, SubscriptionPurchaseModeRenew, existingSub.Id)
	require.NoError(t, CompleteSubscriptionOrder(tradeNo, `{"status":"success"}`))

	var ledger InviteCommissionLedger
	require.NoError(t, DB.Where("topup_trade_no = ? AND inviter_user_id = ?", tradeNo, inviter.Id).First(&ledger).Error)
	assert.Equal(t, 11000, ledger.BaseQuota)
	assert.Equal(t, 1100, ledger.CommissionQuota)
	assert.Equal(t, InviteCommissionStatusPending, ledger.Status)
}

func TestCompleteSubscriptionOrder_InvalidPrice_RollbackWithoutLedger(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)

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
	operation_setting.Price = 0

	inviter := createInviteCommissionTestUser(t, "inviter_sub_invalid_price", 0)
	invitee := createInviteCommissionTestUser(t, "invitee_sub_invalid_price", inviter.Id)
	plan := createSubscriptionPlanForInviteCommissionTest(t, "异常价格套餐", 88, 1000)

	tradeNo := "sub_invite_invalid_price_001"
	createSubscriptionOrderForInviteCommissionTest(t, invitee.Id, plan.Id, tradeNo, 88, SubscriptionPurchaseModeStack, 0)

	err := CompleteSubscriptionOrder(tradeNo, `{"status":"success"}`)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid payment price setting"))

	order := GetSubscriptionOrderByTradeNo(tradeNo)
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusPending, order.Status)

	var ledgerCount int64
	require.NoError(t, DB.Model(&InviteCommissionLedger{}).Where("topup_trade_no = ?", tradeNo).Count(&ledgerCount).Error)
	assert.EqualValues(t, 0, ledgerCount)

	var topupCount int64
	require.NoError(t, DB.Model(&TopUp{}).Where("trade_no = ?", tradeNo).Count(&topupCount).Error)
	assert.EqualValues(t, 0, topupCount)

	var subCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ? AND plan_id = ?", invitee.Id, plan.Id).Count(&subCount).Error)
	assert.EqualValues(t, 0, subCount)
}

func TestCompleteSubscriptionOrder_RenewQuantityUseSameTargetSubscription(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)

	user := createInviteCommissionTestUser(t, "invitee_sub_renew_multi_target", 0)
	plan := createSubscriptionPlanForInviteCommissionTest(t, "多份续费定向月卡", 88, 1000)

	now := time.Now().Unix()
	earliestSub := &UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		AmountUsed:  0,
		StartTime:   now - 3600,
		EndTime:     now + 24*3600,
		Status:      "active",
		Source:      "order",
	}
	require.NoError(t, DB.Create(earliestSub).Error)
	targetSub := &UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		AmountUsed:  0,
		StartTime:   now - 1800,
		EndTime:     now + 2*24*3600,
		Status:      "active",
		Source:      "order",
	}
	require.NoError(t, DB.Create(targetSub).Error)

	targetOldEndTime := targetSub.EndTime
	earliestOldEndTime := earliestSub.EndTime

	tradeNo := "sub_renew_target_multi_quantity_001"
	order := &SubscriptionOrder{
		UserId:                    user.Id,
		PlanId:                    plan.Id,
		Money:                     176,
		PurchaseQuantity:          2,
		TradeNo:                   tradeNo,
		PaymentMethod:             "epay",
		PurchaseMode:              SubscriptionPurchaseModeRenew,
		RenewTargetSubscriptionId: targetSub.Id,
		CreateTime:                common.GetTimestamp(),
		Status:                    common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(order).Error)
	require.NoError(t, CompleteSubscriptionOrder(tradeNo, `{"status":"success"}`))

	// 两份续费应全部加到用户指定目标订阅上。
	require.NoError(t, DB.First(targetSub, "id = ?", targetSub.Id).Error)
	require.NoError(t, DB.First(earliestSub, "id = ?", earliestSub.Id).Error)

	firstEndTime, err := calcPlanEndTime(time.Unix(targetOldEndTime, 0), plan)
	require.NoError(t, err)
	secondEndTime, err := calcPlanEndTime(time.Unix(firstEndTime, 0), plan)
	require.NoError(t, err)
	assert.Equal(t, secondEndTime, targetSub.EndTime)
	assert.Equal(t, earliestOldEndTime, earliestSub.EndTime)
}

func TestCompleteSubscriptionOrder_RenewWithoutTargetFallbackToEarliest(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)

	user := createInviteCommissionTestUser(t, "invitee_sub_renew_fallback_earliest", 0)
	plan := createSubscriptionPlanForInviteCommissionTest(t, "默认最早到期续费月卡", 88, 1000)

	now := time.Now().Unix()
	earliestSub := &UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		AmountUsed:  0,
		StartTime:   now - 3600,
		EndTime:     now + 24*3600,
		Status:      "active",
		Source:      "order",
	}
	require.NoError(t, DB.Create(earliestSub).Error)
	laterSub := &UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		AmountUsed:  0,
		StartTime:   now - 1800,
		EndTime:     now + 3*24*3600,
		Status:      "active",
		Source:      "order",
	}
	require.NoError(t, DB.Create(laterSub).Error)

	earliestOldEndTime := earliestSub.EndTime
	laterOldEndTime := laterSub.EndTime

	tradeNo := "sub_renew_fallback_earliest_001"
	order := &SubscriptionOrder{
		UserId:                    user.Id,
		PlanId:                    plan.Id,
		Money:                     88,
		PurchaseQuantity:          1,
		TradeNo:                   tradeNo,
		PaymentMethod:             "epay",
		PurchaseMode:              SubscriptionPurchaseModeRenew,
		RenewTargetSubscriptionId: 0,
		CreateTime:                common.GetTimestamp(),
		Status:                    common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(order).Error)
	require.NoError(t, CompleteSubscriptionOrder(tradeNo, `{"status":"success"}`))

	// 旧客户端未传目标时，仍按“最早到期”兼容处理。
	require.NoError(t, DB.First(earliestSub, "id = ?", earliestSub.Id).Error)
	require.NoError(t, DB.First(laterSub, "id = ?", laterSub.Id).Error)
	expectedEarliestEndTime, err := calcPlanEndTime(time.Unix(earliestOldEndTime, 0), plan)
	require.NoError(t, err)
	assert.Equal(t, expectedEarliestEndTime, earliestSub.EndTime)
	assert.Equal(t, laterOldEndTime, laterSub.EndTime)
}

func TestCompleteSubscriptionOrder_StackQuantityWithoutActive_CreateMultipleSubscriptions(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)

	user := createInviteCommissionTestUser(t, "invitee_sub_stack_multi_create", 0)
	plan := createSubscriptionPlanForInviteCommissionTest(t, "新购叠加多条月卡", 88, 1000)

	tradeNo := "sub_stack_multi_create_001"
	order := &SubscriptionOrder{
		UserId:                    user.Id,
		PlanId:                    plan.Id,
		Money:                     176,
		PurchaseQuantity:          2,
		TradeNo:                   tradeNo,
		PaymentMethod:             "epay",
		PurchaseMode:              SubscriptionPurchaseModeStack,
		RenewTargetSubscriptionId: 0,
		CreateTime:                common.GetTimestamp(),
		Status:                    common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(order).Error)
	require.NoError(t, CompleteSubscriptionOrder(tradeNo, `{"status":"success"}`))

	// 新购并叠加：购买多份应创建多条订阅记录。
	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Find(&subs).Error)
	require.Len(t, subs, 2)
	for i := range subs {
		require.Equal(t, "active", subs[i].Status)
		require.NotZero(t, subs[i].StartTime)
		require.NotZero(t, subs[i].EndTime)
	}
}

func TestCompleteSubscriptionOrder_RenewExtendWithoutActive_ExtendSingleSubscription(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)

	user := createInviteCommissionTestUser(t, "invitee_sub_renew_extend_single", 0)
	plan := createSubscriptionPlanForInviteCommissionTest(t, "续费式顺延月卡", 88, 1000)

	tradeNo := "sub_renew_extend_single_001"
	order := &SubscriptionOrder{
		UserId:                    user.Id,
		PlanId:                    plan.Id,
		Money:                     176,
		PurchaseQuantity:          2,
		TradeNo:                   tradeNo,
		PaymentMethod:             "epay",
		PurchaseMode:              SubscriptionPurchaseModeRenewExtend,
		RenewTargetSubscriptionId: 0,
		CreateTime:                common.GetTimestamp(),
		Status:                    common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(order).Error)
	require.NoError(t, CompleteSubscriptionOrder(tradeNo, `{"status":"success"}`))

	// 续费式购买：无生效订阅时仅创建一条，然后按份数顺延。
	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Find(&subs).Error)
	require.Len(t, subs, 1)
	require.Equal(t, "active", subs[0].Status)

	firstEndTime, err := calcPlanEndTime(time.Unix(subs[0].StartTime, 0), plan)
	require.NoError(t, err)
	secondEndTime, err := calcPlanEndTime(time.Unix(firstEndTime, 0), plan)
	require.NoError(t, err)
	assert.Equal(t, secondEndTime, subs[0].EndTime)
}
