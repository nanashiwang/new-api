package controller

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
)

func TestStripePaymentDetailsFromCheckout_AdaptivePricing(t *testing.T) {
	payload := []byte(`{
		"client_reference_id":"ref_123",
		"status":"complete",
		"payment_status":"paid",
		"amount_total":300,
		"currency":"usd",
		"presentment_details":{"presentment_amount":2000,"presentment_currency":"cny"}
	}`)
	checkout := stripeCheckoutSessionEvent{}
	require.NoError(t, common.Unmarshal(payload, &checkout))

	details, err := stripePaymentDetailsFromCheckout(checkout)
	require.NoError(t, err)
	require.InDelta(t, 3.0, details.PaidMoney, 0.0001)
	require.Equal(t, "USD", details.PaidCurrency)
	require.InDelta(t, 20.0, details.PresentmentMoney, 0.0001)
	require.Equal(t, "CNY", details.PresentmentCurrency)
}

func TestStripePaymentDetailsFromCheckout_LegacyAdaptivePricingWithPromotion(t *testing.T) {
	payload := []byte(`{
		"amount_subtotal":2000,
		"amount_total":1600,
		"currency":"cny",
		"currency_conversion":{"amount_subtotal":300,"amount_total":240,"source_currency":"usd"}
	}`)
	checkout := stripeCheckoutSessionEvent{}
	require.NoError(t, common.Unmarshal(payload, &checkout))

	details, err := stripePaymentDetailsFromCheckout(checkout)
	require.NoError(t, err)
	require.InDelta(t, 2.4, details.PaidMoney, 0.0001)
	require.Equal(t, "USD", details.PaidCurrency)
	require.InDelta(t, 16.0, details.PresentmentMoney, 0.0001)
	require.Equal(t, "CNY", details.PresentmentCurrency)
	require.InDelta(t, 3.0, stripeSettlementSubtotalFromCheckout(checkout, details.PaidCurrency), 0.0001)
}

func TestStripePaymentDetailsFromCheckout_PromotionUsesActualPaidAmount(t *testing.T) {
	payload := []byte(`{
		"amount_subtotal":300,
		"amount_total":240,
		"currency":"usd",
		"total_details":{"amount_discount":60},
		"presentment_details":{"presentment_amount":1600,"presentment_currency":"cny"}
	}`)
	checkout := stripeCheckoutSessionEvent{}
	require.NoError(t, common.Unmarshal(payload, &checkout))

	details, err := stripePaymentDetailsFromCheckout(checkout)
	require.NoError(t, err)
	require.InDelta(t, 2.4, details.PaidMoney, 0.0001)
	require.InDelta(t, 3.0, stripeAmountFromMinorUnits(checkout.AmountSubtotal, details.PaidCurrency), 0.0001)
	require.InDelta(t, 0.6, stripeAmountFromMinorUnits(checkout.TotalDetails.AmountDiscount, details.PaidCurrency), 0.0001)
	require.InDelta(t, 16.0, details.PresentmentMoney, 0.0001)
}

func TestStripePaymentDetailsFromCheckout_WithoutAdaptivePricing(t *testing.T) {
	details, err := stripePaymentDetailsFromCheckout(stripeCheckoutSessionEvent{
		AmountTotal: 300,
		Currency:    "USD",
	})
	require.NoError(t, err)
	require.InDelta(t, 3.0, details.PaidMoney, 0.0001)
	require.Zero(t, details.PresentmentMoney)
	require.Empty(t, details.PresentmentCurrency)
}

func TestStripeAmountFromMinorUnits_CurrencyExponents(t *testing.T) {
	require.InDelta(t, 20.0, stripeAmountFromMinorUnits(2000, "CNY"), 0.0001)
	require.InDelta(t, 300.0, stripeAmountFromMinorUnits(300, "JPY"), 0.0001)
	require.InDelta(t, 1.234, stripeAmountFromMinorUnits(1234, "KWD"), 0.0001)
	require.Zero(t, stripeAmountFromMinorUnits(100, "invalid"))
}

func TestStripeCheckoutCustomerID_AcceptsExpandedCustomer(t *testing.T) {
	require.Equal(t, "cus_123", stripeCheckoutCustomerID(" cus_123 "))
	require.Equal(t, "cus_456", stripeCheckoutCustomerID(map[string]any{"id": "cus_456"}))
	require.Empty(t, stripeCheckoutCustomerID(nil))
}

func TestStripeSessionCompleted_CompletedSubscriptionDoesNotRequireTopUp(t *testing.T) {
	setupTopupCallbackTestDB(t)

	user := createTopupCallbackTestUser(t, "stripe-subscription-replay")
	order := &model.SubscriptionOrder{
		UserId:          user.Id,
		Money:           3,
		TradeNo:         "sub_ref_completed_replay",
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusSuccess,
	}
	require.NoError(t, order.Insert())

	event := stripe.Event{
		Type: stripe.EventTypeCheckoutSessionCompleted,
		Data: &stripe.EventData{Raw: []byte(`{
			"client_reference_id":"sub_ref_completed_replay",
			"status":"complete",
			"payment_status":"paid",
			"amount_subtotal":300,
			"amount_total":300,
			"currency":"usd"
		}`)},
	}

	require.NoError(t, sessionCompleted(event, "127.0.0.1"))
	require.Nil(t, model.GetTopUpByTradeNo(order.TradeNo))
	require.Equal(t, common.TopUpStatusSuccess, model.GetSubscriptionOrderByTradeNo(order.TradeNo).Status)
}

func TestStripeSessionCompleted_AdaptivePricingGrantsBenefitOnlyOnce(t *testing.T) {
	setupTopupCallbackTestDB(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })

	user := createTopupCallbackTestUser(t, "stripe-adaptive-topup")
	topUp := &model.TopUp{
		UserId:          user.Id,
		Amount:          100,
		Money:           100,
		PaidMoney:       3,
		PaidCurrency:    "USD",
		TradeNo:         "ref_adaptive_topup",
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, topUp.Insert())

	event := stripe.Event{
		Type: stripe.EventTypeCheckoutSessionCompleted,
		Data: &stripe.EventData{Raw: []byte(`{
			"customer":"cus_adaptive",
			"client_reference_id":"ref_adaptive_topup",
			"status":"complete",
			"payment_status":"paid",
			"amount_subtotal":300,
			"amount_total":300,
			"currency":"usd",
			"presentment_details":{"presentment_amount":2000,"presentment_currency":"cny"}
		}`)},
	}

	require.NoError(t, sessionCompleted(event, "127.0.0.1"))
	event.Type = stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded
	require.NoError(t, sessionCompleted(event, "127.0.0.1"))

	savedTopUp := model.GetTopUpByTradeNo(topUp.TradeNo)
	require.Equal(t, common.TopUpStatusSuccess, savedTopUp.Status)
	require.InDelta(t, 3, savedTopUp.PaidMoney, 0.0001)
	require.Equal(t, "USD", savedTopUp.PaidCurrency)
	require.InDelta(t, 20, savedTopUp.PresentmentMoney, 0.0001)
	require.Equal(t, "CNY", savedTopUp.PresentmentCurrency)

	savedUser, err := model.GetUserById(user.Id, false)
	require.NoError(t, err)
	require.Equal(t, 100, savedUser.Quota)
	require.Equal(t, "cus_adaptive", savedUser.StripeCustomer)
}

func TestStripeSessionCompleted_LegacyAdaptiveReplayRepairsPaymentWithoutQuota(t *testing.T) {
	setupTopupCallbackTestDB(t)

	user := createTopupCallbackTestUser(t, "stripe-legacy-adaptive-replay")
	require.NoError(t, model.DB.Model(user).Updates(map[string]any{
		"quota":              100,
		"transferable_quota": 100,
	}).Error)
	topUp := &model.TopUp{
		UserId:          user.Id,
		Amount:          100,
		Money:           100,
		PaidMoney:       20,
		TradeNo:         "ref_legacy_adaptive_replay",
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusSuccess,
	}
	require.NoError(t, topUp.Insert())

	event := stripe.Event{
		Type: stripe.EventTypeCheckoutSessionCompleted,
		Data: &stripe.EventData{Raw: []byte(`{
			"customer":"cus_legacy",
			"client_reference_id":"ref_legacy_adaptive_replay",
			"status":"complete",
			"payment_status":"paid",
			"amount_subtotal":2000,
			"amount_total":2000,
			"currency":"cny",
			"currency_conversion":{"amount_subtotal":300,"amount_total":300,"source_currency":"usd"}
		}`)},
	}

	require.NoError(t, sessionCompleted(event, "127.0.0.1"))

	savedTopUp := model.GetTopUpByTradeNo(topUp.TradeNo)
	require.InDelta(t, 3, savedTopUp.PaidMoney, 0.0001)
	require.Equal(t, "USD", savedTopUp.PaidCurrency)
	require.InDelta(t, 20, savedTopUp.PresentmentMoney, 0.0001)
	require.Equal(t, "CNY", savedTopUp.PresentmentCurrency)

	savedUser, err := model.GetUserById(user.Id, false)
	require.NoError(t, err)
	require.Equal(t, 100, savedUser.Quota)
}

func TestResolveStripePriceCurrencyUsesConfiguredPriceAndCache(t *testing.T) {
	originalSecret := setting.StripeApiSecret
	originalPriceID := setting.StripePriceId
	originalLoader := stripePriceCurrencyLoader
	t.Cleanup(func() {
		setting.StripeApiSecret = originalSecret
		setting.StripePriceId = originalPriceID
		stripePriceCurrencyLoader = originalLoader
		stripePriceCurrencyCacheMu.Lock()
		stripePriceCurrencyCache = make(map[string]string)
		stripePriceCurrencyCacheMu.Unlock()
	})

	stripePriceCurrencyCacheMu.Lock()
	stripePriceCurrencyCache = make(map[string]string)
	stripePriceCurrencyCacheMu.Unlock()
	setting.StripeApiSecret = "sk_test_currency"
	setting.StripePriceId = "price_usd"
	calls := 0
	stripePriceCurrencyLoader = func(_ context.Context, apiSecret string, priceID string) (string, error) {
		calls++
		require.Equal(t, setting.StripeApiSecret, apiSecret)
		if priceID == "price_usd" {
			return "usd", nil
		}
		return "eur", nil
	}

	currency, err := resolveStripePriceCurrency(context.Background())
	require.NoError(t, err)
	require.Equal(t, "USD", currency)

	currency, err = resolveStripePriceCurrency(context.Background())
	require.NoError(t, err)
	require.Equal(t, "USD", currency)
	require.Equal(t, 1, calls)

	setting.StripePriceId = "price_eur"
	currency, err = resolveStripePriceCurrency(context.Background())
	require.NoError(t, err)
	require.Equal(t, "EUR", currency)
	require.Equal(t, 2, calls)
}

func TestResolveStripePriceCurrencyRejectsInvalidProviderCurrency(t *testing.T) {
	originalSecret := setting.StripeApiSecret
	originalPriceID := setting.StripePriceId
	originalLoader := stripePriceCurrencyLoader
	t.Cleanup(func() {
		setting.StripeApiSecret = originalSecret
		setting.StripePriceId = originalPriceID
		stripePriceCurrencyLoader = originalLoader
		stripePriceCurrencyCacheMu.Lock()
		stripePriceCurrencyCache = make(map[string]string)
		stripePriceCurrencyCacheMu.Unlock()
	})

	stripePriceCurrencyCacheMu.Lock()
	stripePriceCurrencyCache = make(map[string]string)
	stripePriceCurrencyCacheMu.Unlock()
	setting.StripeApiSecret = "rk_test_currency"
	setting.StripePriceId = "price_invalid"
	stripePriceCurrencyLoader = func(_ context.Context, _ string, _ string) (string, error) {
		return "US", nil
	}

	_, err := resolveStripePriceCurrency(context.Background())
	require.Error(t, err)
}
