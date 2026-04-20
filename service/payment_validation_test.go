package service

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/require"
)

var (
	paymentValidationTestSchemaOnce sync.Once
	paymentValidationTestSchemaErr  error
)

func setupPaymentValidationTestDB(t *testing.T) {
	t.Helper()

	paymentValidationTestSchemaOnce.Do(func() {
		paymentValidationTestSchemaErr = model.DB.AutoMigrate(&model.User{}, &model.TopUp{}, &model.PaymentRiskCase{}, &model.SubscriptionOrder{})
	})
	require.NoError(t, paymentValidationTestSchemaErr)

	reset := func() {
		require.NoError(t, model.DB.Exec("DELETE FROM payment_risk_cases").Error)
		require.NoError(t, model.DB.Exec("DELETE FROM subscription_orders").Error)
		require.NoError(t, model.DB.Exec("DELETE FROM top_ups").Error)
		require.NoError(t, model.DB.Exec("DELETE FROM users").Error)
	}
	reset()
	t.Cleanup(reset)
}

func createPaymentValidationUser(t *testing.T, username string, group string) *model.User {
	t.Helper()

	user := &model.User{
		Username: username,
		Group:    group,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)
	return user
}

func createPaymentValidationTopUp(t *testing.T, userID int, tradeNo string, paymentMethod string, amount int64, money float64) *model.TopUp {
	t.Helper()

	topUp := &model.TopUp{
		UserId:        userID,
		Amount:        amount,
		Money:         money,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethod,
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, topUp.Insert())
	return topUp
}

func createPaymentValidationSubscriptionOrder(t *testing.T, userID int, tradeNo string, paymentMethod string, money float64) *model.SubscriptionOrder {
	t.Helper()

	order := &model.SubscriptionOrder{
		UserId:        userID,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethod,
		Money:         money,
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, order.Insert())
	return order
}

func withStripePriceConfig(t *testing.T) {
	t.Helper()

	originStripeUnitPrice := setting.StripeUnitPrice
	originQuotaPerUnit := common.QuotaPerUnit
	setting.StripeUnitPrice = 1
	common.QuotaPerUnit = 1
	t.Cleanup(func() {
		setting.StripeUnitPrice = originStripeUnitPrice
		common.QuotaPerUnit = originQuotaPerUnit
	})
}

func TestValidateTopUpCallback_RejectsPaymentMethodMismatch(t *testing.T) {
	setupPaymentValidationTestDB(t)

	user := createPaymentValidationUser(t, "payment-method-mismatch", "default")
	topUp := createPaymentValidationTopUp(t, user.Id, "TOPUP-MISMATCH-001", "alipay", 100, 16)

	result, err := ValidateTopUpCallback(PaymentCallbackValidationInput{
		TradeNo:        topUp.TradeNo,
		PaymentMethod:  "stripe",
		ProviderAmount: 16,
		Source:         "stripe_webhook",
		Currency:       "USD",
	})

	require.ErrorIs(t, err, ErrPaymentCallbackRejected)
	require.False(t, result.AlreadyCompleted)

	savedTopUp := model.GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, savedTopUp)
	require.Equal(t, common.TopUpStatusPending, savedTopUp.Status)

	riskCase, err := model.GetPaymentRiskCaseByRecord(model.PaymentRiskRecordTypeTopUp, topUp.TradeNo)
	require.NoError(t, err)
	require.Equal(t, model.PaymentRiskReasonPaymentMethodMismatch, riskCase.Reason)
	require.Equal(t, topUp.PaymentMethod, riskCase.PaymentMethod)
	require.Equal(t, "stripe", riskCase.ProviderPaymentMethod)
}

func TestValidateTopUpCallback_RejectsAmountMismatch(t *testing.T) {
	setupPaymentValidationTestDB(t)
	withStripePriceConfig(t)

	user := createPaymentValidationUser(t, "amount-mismatch", "default")
	expectedMoney := CalculateStripeTopUpPayMoney(100, user.Group)
	topUp := createPaymentValidationTopUp(t, user.Id, "TOPUP-AMOUNT-001", "stripe", 100, expectedMoney)

	result, err := ValidateTopUpCallback(PaymentCallbackValidationInput{
		TradeNo:        topUp.TradeNo,
		PaymentMethod:  "stripe",
		ProviderAmount: expectedMoney + 1,
		Source:         "stripe_webhook",
		Currency:       "USD",
	})

	require.ErrorIs(t, err, ErrPaymentCallbackRejected)
	require.False(t, result.AlreadyCompleted)

	riskCase, err := model.GetPaymentRiskCaseByRecord(model.PaymentRiskRecordTypeTopUp, topUp.TradeNo)
	require.NoError(t, err)
	require.Equal(t, model.PaymentRiskReasonAmountMismatch, riskCase.Reason)
	require.Equal(t, expectedMoney, riskCase.ExpectedMoney)
	require.Equal(t, expectedMoney+1, riskCase.ReceivedMoney)
}

func TestValidateTopUpCallback_RejectsNonPositiveAmount(t *testing.T) {
	setupPaymentValidationTestDB(t)
	withStripePriceConfig(t)

	user := createPaymentValidationUser(t, "amount-zero", "default")
	expectedMoney := CalculateStripeTopUpPayMoney(100, user.Group)
	topUp := createPaymentValidationTopUp(t, user.Id, "TOPUP-AMOUNT-000", "stripe", 100, expectedMoney)

	result, err := ValidateTopUpCallback(PaymentCallbackValidationInput{
		TradeNo:        topUp.TradeNo,
		PaymentMethod:  "stripe",
		ProviderAmount: 0,
		Source:         "stripe_webhook",
		Currency:       "USD",
	})

	require.ErrorIs(t, err, ErrPaymentCallbackRejected)
	require.False(t, result.AlreadyCompleted)

	savedTopUp := model.GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, savedTopUp)
	require.Equal(t, common.TopUpStatusPending, savedTopUp.Status)

	riskCase, err := model.GetPaymentRiskCaseByRecord(model.PaymentRiskRecordTypeTopUp, topUp.TradeNo)
	require.NoError(t, err)
	require.Equal(t, model.PaymentRiskReasonAmountMismatch, riskCase.Reason)
	require.Equal(t, 0.0, riskCase.ReceivedMoney)
}

func TestValidateTopUpCallback_AllowsMatchingStripeAmount(t *testing.T) {
	setupPaymentValidationTestDB(t)
	withStripePriceConfig(t)

	user := createPaymentValidationUser(t, "amount-match", "default")
	expectedMoney := CalculateStripeTopUpPayMoney(100, user.Group)
	topUp := createPaymentValidationTopUp(t, user.Id, "TOPUP-AMOUNT-OK", "stripe", 100, expectedMoney)

	result, err := ValidateTopUpCallback(PaymentCallbackValidationInput{
		TradeNo:        topUp.TradeNo,
		PaymentMethod:  "stripe",
		ProviderAmount: expectedMoney,
		Source:         "stripe_webhook",
		Currency:       "USD",
	})

	require.NoError(t, err)
	require.False(t, result.AlreadyCompleted)

	var count int64
	require.NoError(t, model.DB.Model(&model.PaymentRiskCase{}).
		Where("record_type = ? AND trade_no = ?", model.PaymentRiskRecordTypeTopUp, topUp.TradeNo).
		Count(&count).Error)
	require.Zero(t, count)
}

func TestValidateSubscriptionCallback_RejectsNonPositiveAmount(t *testing.T) {
	setupPaymentValidationTestDB(t)

	user := createPaymentValidationUser(t, "subscription-zero", "default")
	order := createPaymentValidationSubscriptionOrder(t, user.Id, "SUB-AMOUNT-000", "stripe", 88)

	result, err := ValidateSubscriptionCallback(PaymentCallbackValidationInput{
		TradeNo:        order.TradeNo,
		PaymentMethod:  "stripe",
		ProviderAmount: 0,
		Source:         "subscription_stripe_webhook",
		Currency:       "USD",
	})

	require.ErrorIs(t, err, ErrPaymentCallbackRejected)
	require.False(t, result.AlreadyCompleted)

	savedOrder := model.GetSubscriptionOrderByTradeNo(order.TradeNo)
	require.NotNil(t, savedOrder)
	require.Equal(t, common.TopUpStatusPending, savedOrder.Status)

	riskCase, err := model.GetPaymentRiskCaseByRecord(model.PaymentRiskRecordTypeSubscription, order.TradeNo)
	require.NoError(t, err)
	require.Equal(t, model.PaymentRiskReasonAmountMismatch, riskCase.Reason)
	require.Equal(t, 0.0, riskCase.ReceivedMoney)
}

func TestValidateSubscriptionCallback_AllowsMatchingAmount(t *testing.T) {
	setupPaymentValidationTestDB(t)

	user := createPaymentValidationUser(t, "subscription-match", "default")
	order := createPaymentValidationSubscriptionOrder(t, user.Id, "SUB-AMOUNT-OK", "stripe", 88)

	result, err := ValidateSubscriptionCallback(PaymentCallbackValidationInput{
		TradeNo:        order.TradeNo,
		PaymentMethod:  "stripe",
		ProviderAmount: 88,
		Source:         "subscription_stripe_webhook",
		Currency:       "USD",
	})

	require.NoError(t, err)
	require.False(t, result.AlreadyCompleted)

	var count int64
	require.NoError(t, model.DB.Model(&model.PaymentRiskCase{}).
		Where("record_type = ? AND trade_no = ?", model.PaymentRiskRecordTypeSubscription, order.TradeNo).
		Count(&count).Error)
	require.Zero(t, count)
}
