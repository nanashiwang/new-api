package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPaymentRiskCaseTestDB(t *testing.T) {
	t.Helper()

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
	originInviterCommissionEnabled := common.InviterCommissionEnabled

	DB = db
	LOG_DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.InviterCommissionEnabled = false

	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.RedisEnabled = originRedisEnabled
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
		common.InviterCommissionEnabled = originInviterCommissionEnabled
	})

	require.NoError(t, db.AutoMigrate(
		&User{},
		&TopUp{},
		&PaymentRiskCase{},
		&Log{},
		&SubscriptionPlan{},
		&SubscriptionOrder{},
		&SubscriptionIssuance{},
		&UserSubscription{},
		&BenefitChangeRecord{},
		&BenefitRollbackOperation{},
	))
}

func createPaymentRiskCaseTestUser(t *testing.T, username string) *User {
	t.Helper()

	user := &User{
		Username: username,
		Password: "password123",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		AffCode:  username + "-aff",
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func createPaymentRiskCaseTestTopUp(t *testing.T, userID int, tradeNo string, status string, amount int64, money float64, paymentMethod string) *TopUp {
	t.Helper()

	topup := &TopUp{
		UserId:        userID,
		Amount:        amount,
		Money:         money,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethod,
		CreateTime:    1_760_000_000,
		Status:        status,
	}
	if status == common.TopUpStatusSuccess {
		topup.CompleteTime = topup.CreateTime + 60
	}
	require.NoError(t, topup.Insert())
	return topup
}

func createPaymentRiskCaseRecord(t *testing.T, topUp *TopUp, reason string) *PaymentRiskCase {
	t.Helper()

	riskCase, err := UpsertPaymentRiskCase(PaymentRiskCaseUpsertInput{
		RecordType:            PaymentRiskRecordTypeTopUp,
		TradeNo:               topUp.TradeNo,
		UserId:                topUp.UserId,
		PaymentMethod:         topUp.PaymentMethod,
		ProviderPaymentMethod: topUp.PaymentMethod,
		ExpectedAmount:        topUp.Amount,
		ExpectedMoney:         topUp.Money,
		ReceivedMoney:         topUp.Money,
		Source:                "test",
		Reason:                reason,
		OrderStatus:           topUp.Status,
		ProviderPayload:       `{"source":"test"}`,
	})
	require.NoError(t, err)
	return riskCase
}

func createPaymentRiskCaseSubscriptionPlan(t *testing.T, title string, totalAmount int64, customSeconds int64) *SubscriptionPlan {
	t.Helper()

	plan := &SubscriptionPlan{
		Title:         title,
		PriceAmount:   88,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationCustom,
		CustomSeconds: customSeconds,
		Enabled:       true,
		TotalAmount:   totalAmount,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func createPaymentRiskCaseSubscriptionOrder(t *testing.T, userID int, planID int, tradeNo string, purchaseMode string, renewTargetSubscriptionId int) *SubscriptionOrder {
	t.Helper()

	order := &SubscriptionOrder{
		UserId:                    userID,
		PlanId:                    planID,
		Money:                     88,
		TradeNo:                   tradeNo,
		PaymentMethod:             "stripe",
		PurchaseMode:              purchaseMode,
		RenewTargetSubscriptionId: renewTargetSubscriptionId,
		Status:                    common.TopUpStatusPending,
		CreateTime:                common.GetTimestamp(),
	}
	require.NoError(t, order.Insert())
	return order
}

func createSubscriptionPaymentRiskCaseRecord(t *testing.T, order *SubscriptionOrder, reason string) *PaymentRiskCase {
	t.Helper()

	latest := GetSubscriptionOrderByTradeNo(order.TradeNo)
	require.NotNil(t, latest)

	riskCase, err := UpsertPaymentRiskCase(PaymentRiskCaseUpsertInput{
		RecordType:      PaymentRiskRecordTypeSubscription,
		TradeNo:         latest.TradeNo,
		UserId:          latest.UserId,
		PaymentMethod:   latest.PaymentMethod,
		ExpectedMoney:   latest.Money,
		ReceivedMoney:   latest.Money,
		Source:          "test",
		Reason:          reason,
		OrderStatus:     latest.Status,
		ProviderPayload: `{"source":"test"}`,
	})
	require.NoError(t, err)
	return riskCase
}

func TestResolvePaymentRiskCase_ConfirmCompletesPendingTopUp(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	user := createPaymentRiskCaseTestUser(t, "alice")
	topup := createPaymentRiskCaseTestTopUp(t, user.Id, "RISK-CONFIRM-001", common.TopUpStatusPending, 2, 16, "alipay")
	riskCase := createPaymentRiskCaseRecord(t, topup, PaymentRiskReasonAmountMismatch)

	grantedQuota, err := CalculateGrantedQuotaForTopUp(topup)
	require.NoError(t, err)

	require.NoError(t, ResolvePaymentRiskCase(riskCase.Id, 99, PaymentRiskActionConfirm, "manual confirm"))

	updatedCase, err := GetPaymentRiskCaseByID(riskCase.Id)
	require.NoError(t, err)
	require.Equal(t, PaymentRiskStatusConfirmed, updatedCase.Status)
	require.Equal(t, 99, updatedCase.HandlerAdminId)
	require.Equal(t, "manual confirm", updatedCase.HandlerNote)
	require.Equal(t, 0, updatedCase.AppliedQuotaDelta)
	require.NotZero(t, updatedCase.ResolvedAt)

	updatedTopUp := GetTopUpByTradeNo(topup.TradeNo)
	require.NotNil(t, updatedTopUp)
	require.Equal(t, common.TopUpStatusSuccess, updatedTopUp.Status)
	require.NotZero(t, updatedTopUp.CompleteTime)

	updatedUser, err := GetUserById(user.Id, false)
	require.NoError(t, err)
	require.Equal(t, grantedQuota, updatedUser.Quota)
}

func TestResolvePaymentRiskCase_ReverseDeductsGrantedQuota(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	user := createPaymentRiskCaseTestUser(t, "bob")
	topup := createPaymentRiskCaseTestTopUp(t, user.Id, "RISK-REVERSE-001", common.TopUpStatusSuccess, 3, 24, "alipay")
	riskCase := createPaymentRiskCaseRecord(t, topup, PaymentRiskReasonManualReview)

	grantedQuota, err := CalculateGrantedQuotaForTopUp(topup)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", grantedQuota).Error)

	require.NoError(t, ResolvePaymentRiskCase(riskCase.Id, 7, PaymentRiskActionReverse, "reverse suspicious credit"))

	updatedCase, err := GetPaymentRiskCaseByID(riskCase.Id)
	require.NoError(t, err)
	require.Equal(t, PaymentRiskStatusReversed, updatedCase.Status)
	require.Equal(t, -grantedQuota, updatedCase.AppliedQuotaDelta)
	require.Equal(t, 7, updatedCase.HandlerAdminId)

	updatedUser, err := GetUserById(user.Id, false)
	require.NoError(t, err)
	require.Equal(t, 0, updatedUser.Quota)
}

func TestResolvePaymentRiskCase_VoidExpiresPendingTopUp(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	user := createPaymentRiskCaseTestUser(t, "charlie")
	topup := createPaymentRiskCaseTestTopUp(t, user.Id, "RISK-VOID-001", common.TopUpStatusPending, 1, 8, "alipay")
	riskCase := createPaymentRiskCaseRecord(t, topup, PaymentRiskReasonManualReview)

	require.NoError(t, ResolvePaymentRiskCase(riskCase.Id, 12, PaymentRiskActionVoid, "void invalid callback"))

	updatedCase, err := GetPaymentRiskCaseByID(riskCase.Id)
	require.NoError(t, err)
	require.Equal(t, PaymentRiskStatusVoided, updatedCase.Status)
	require.Equal(t, "void invalid callback", updatedCase.HandlerNote)

	updatedTopUp := GetTopUpByTradeNo(topup.TradeNo)
	require.NotNil(t, updatedTopUp)
	require.Equal(t, common.TopUpStatusExpired, updatedTopUp.Status)
	require.NotZero(t, updatedTopUp.CompleteTime)
}

func TestResolvePaymentRiskCase_ReverseSubscriptionStackInvalidatesIssuedSubscription(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	user := createPaymentRiskCaseTestUser(t, "david")
	plan := createPaymentRiskCaseSubscriptionPlan(t, "stack plan", 1000, 3600)
	order := createPaymentRiskCaseSubscriptionOrder(t, user.Id, plan.Id, "RISK-SUB-STACK-001", SubscriptionPurchaseModeStack, 0)

	require.NoError(t, CompleteSubscriptionOrder(order.TradeNo, `{"status":"success"}`))

	riskCase := createSubscriptionPaymentRiskCaseRecord(t, order, PaymentRiskReasonManualReview)
	require.NoError(t, ResolvePaymentRiskCase(riskCase.Id, 18, PaymentRiskActionReverse, "reverse issued subscription"))

	updatedCase, err := GetPaymentRiskCaseByID(riskCase.Id)
	require.NoError(t, err)
	require.Equal(t, PaymentRiskStatusReversed, updatedCase.Status)

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).First(&sub).Error)
	require.Equal(t, "cancelled", sub.Status)
	require.LessOrEqual(t, sub.EndTime, common.GetTimestamp())
}

func TestResolvePaymentRiskCase_ReverseSubscriptionRenewRestoresPreviousEndTime(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	user := createPaymentRiskCaseTestUser(t, "eve")
	plan := createPaymentRiskCaseSubscriptionPlan(t, "renew plan", 1000, 3600)
	now := common.GetTimestamp()
	existing := &UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		AmountUsed:  0,
		StartTime:   now - 1800,
		EndTime:     now + 7200,
		Status:      "active",
		Source:      "order",
	}
	require.NoError(t, DB.Create(existing).Error)

	originalEndTime := existing.EndTime
	originalAmountTotal := existing.AmountTotal
	order := createPaymentRiskCaseSubscriptionOrder(t, user.Id, plan.Id, "RISK-SUB-RENEW-001", SubscriptionPurchaseModeRenew, existing.Id)

	require.NoError(t, CompleteSubscriptionOrder(order.TradeNo, `{"status":"success"}`))
	var grantRecords []BenefitChangeRecord
	require.NoError(t, DB.Where("source_type = ? AND source_ref = ? AND action = ?",
		BenefitSourceSubscriptionOrder, order.TradeNo, BenefitActionGrant).
		Order("id asc").
		Find(&grantRecords).Error)
	require.Len(t, grantRecords, 1)

	riskCase := createSubscriptionPaymentRiskCaseRecord(t, order, PaymentRiskReasonManualReview)
	require.NoError(t, ResolvePaymentRiskCase(riskCase.Id, 23, PaymentRiskActionReverse, "reverse renewed subscription"))

	updatedCase, err := GetPaymentRiskCaseByID(riskCase.Id)
	require.NoError(t, err)
	require.Equal(t, PaymentRiskStatusReversed, updatedCase.Status)

	var renewed UserSubscription
	require.NoError(t, DB.First(&renewed, existing.Id).Error)
	require.Equal(t, "active", renewed.Status)
	require.Equal(t, originalEndTime, renewed.EndTime)
	require.Equal(t, originalAmountTotal, renewed.AmountTotal)
}

func TestResolvePaymentRiskCase_ReverseSubscriptionPendingIssuanceCancelsPendingGrant(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	user := createPaymentRiskCaseTestUser(t, "frank")
	plan := createPaymentRiskCaseSubscriptionPlan(t, "pending issuance plan", 1000, 3600)
	now := common.GetTimestamp()
	subA := &UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		StartTime:   now - 1800,
		EndTime:     now + 7200,
		Status:      "active",
		Source:      "order",
	}
	subB := &UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		StartTime:   now - 1200,
		EndTime:     now + 10800,
		Status:      "active",
		Source:      "order",
	}
	require.NoError(t, DB.Create(subA).Error)
	require.NoError(t, DB.Create(subB).Error)

	order := createPaymentRiskCaseSubscriptionOrder(t, user.Id, plan.Id, "RISK-SUB-PENDING-001", SubscriptionPurchaseModeRenew, 0)
	require.NoError(t, CompleteSubscriptionOrder(order.TradeNo, `{"status":"success"}`))

	var issuance SubscriptionIssuance
	require.NoError(t, DB.Where("source_type = ? AND source_ref = ?", SubscriptionIssuanceSourceOrder, order.TradeNo).First(&issuance).Error)
	require.Equal(t, SubscriptionIssuanceStatusPending, issuance.Status)

	riskCase := createSubscriptionPaymentRiskCaseRecord(t, order, PaymentRiskReasonManualReview)
	require.NoError(t, ResolvePaymentRiskCase(riskCase.Id, 31, PaymentRiskActionReverse, "reverse pending issuance"))

	require.NoError(t, DB.First(&issuance, issuance.Id).Error)
	require.Equal(t, SubscriptionIssuanceStatusCancelled, issuance.Status)
}
