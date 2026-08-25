package service

import (
	"errors"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"
)

var ErrPaymentCallbackRejected = errors.New("payment callback rejected")

type PaymentCallbackValidationInput struct {
	TradeNo               string
	PaymentMethod         string
	PaymentProvider       string
	ProviderAmount        float64
	ProviderSubtotal      float64
	Currency              string
	Source                string
	ProviderPayload       string
	ProviderPaymentMethod string
	// AllowCompletedAmountBackfill is only for already verified provider events.
	AllowCompletedAmountBackfill bool
}

type PaymentCallbackValidationResult struct {
	AlreadyCompleted bool
}

func ValidateTopUpCallback(input PaymentCallbackValidationInput) (PaymentCallbackValidationResult, error) {
	tradeNo := strings.TrimSpace(input.TradeNo)
	if tradeNo == "" {
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}

	topUp, lookupErr := model.GetTopUpByTradeNoWithError(tradeNo)
	if errors.Is(lookupErr, model.ErrTopUpNotFound) {
		_, riskErr := model.UpsertPaymentRiskCase(model.PaymentRiskCaseUpsertInput{
			RecordType:      model.PaymentRiskRecordTypeTopUp,
			TradeNo:         tradeNo,
			Source:          strings.TrimSpace(input.Source),
			Reason:          model.PaymentRiskReasonOrderNotFound,
			ReceivedMoney:   input.ProviderAmount,
			Currency:        strings.TrimSpace(input.Currency),
			ProviderPayload: strings.TrimSpace(input.ProviderPayload),
		})
		if riskErr != nil {
			return PaymentCallbackValidationResult{}, riskErr
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if lookupErr != nil {
		return PaymentCallbackValidationResult{}, lookupErr
	}
	alreadyCompleted := topUp.Status == common.TopUpStatusSuccess
	if !alreadyCompleted && topUp.Status != common.TopUpStatusPending {
		if err := recordTopUpRiskCase(topUp, input, model.PaymentRiskReasonOrderStatusInvalid, 0); err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if !model.PaymentProviderMatches(topUp.PaymentProvider, topUp.PaymentMethod, input.PaymentProvider, input.PaymentMethod) {
		if err := recordTopUpRiskCase(topUp, input, model.PaymentRiskReasonPaymentMethodMismatch, 0); err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	expectedProvider := model.InferPaymentProvider(topUp.PaymentProvider, topUp.PaymentMethod)
	if !paymentMethodMatchesForProvider(expectedProvider, topUp.PaymentMethod, input.PaymentMethod) {
		if err := recordTopUpRiskCase(topUp, input, model.PaymentRiskReasonPaymentMethodMismatch, 0); err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}

	expectedMoney := topUp.PaidMoney
	if alreadyCompleted && expectedProvider == model.PaymentProviderStripe && input.AllowCompletedAmountBackfill {
		// A signed replay is provider truth and can repair stale legacy snapshots;
		// the completed-order path cannot grant quota again.
		expectedMoney = input.ProviderAmount
	} else if expectedMoney <= 0 && expectedProvider == model.PaymentProviderStripe {
		group, err := model.GetUserGroup(topUp.UserId, true)
		if err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		expectedMoney = CalculateStripeTopUpPayMoney(float64(topUp.Amount), group)
	}
	if expectedMoney <= 0 {
		expectedMoney = topUp.Money
	}
	comparisonMoney := input.ProviderAmount
	if !alreadyCompleted && input.ProviderSubtotal > 0 {
		comparisonMoney = input.ProviderSubtotal
	}
	if input.ProviderAmount <= 0 || comparisonMoney <= 0 || !paymentAmountsMatch(expectedMoney, comparisonMoney) {
		if err := recordTopUpRiskCase(topUp, input, model.PaymentRiskReasonAmountMismatch, expectedMoney); err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	expectedCurrency := model.NormalizePaymentCurrency(topUp.PaidCurrency)
	receivedCurrency := model.NormalizePaymentCurrency(input.Currency)
	if expectedCurrency != "" && receivedCurrency != "" && expectedCurrency != receivedCurrency {
		if err := recordTopUpRiskCase(topUp, input, model.PaymentRiskReasonAmountMismatch, expectedMoney); err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	return PaymentCallbackValidationResult{AlreadyCompleted: alreadyCompleted}, nil
}

func ValidateSubscriptionCallback(input PaymentCallbackValidationInput) (PaymentCallbackValidationResult, error) {
	tradeNo := strings.TrimSpace(input.TradeNo)
	if tradeNo == "" {
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}

	order, lookupErr := model.GetSubscriptionOrderByTradeNoWithError(tradeNo)
	if errors.Is(lookupErr, model.ErrSubscriptionOrderNotFound) {
		_, riskErr := model.UpsertPaymentRiskCase(model.PaymentRiskCaseUpsertInput{
			RecordType:      model.PaymentRiskRecordTypeSubscription,
			TradeNo:         tradeNo,
			Source:          strings.TrimSpace(input.Source),
			Reason:          model.PaymentRiskReasonOrderNotFound,
			ReceivedMoney:   input.ProviderAmount,
			Currency:        strings.TrimSpace(input.Currency),
			ProviderPayload: strings.TrimSpace(input.ProviderPayload),
		})
		if riskErr != nil {
			return PaymentCallbackValidationResult{}, riskErr
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if lookupErr != nil {
		return PaymentCallbackValidationResult{}, lookupErr
	}
	alreadyCompleted := order.Status == common.TopUpStatusSuccess
	if !alreadyCompleted && order.Status != common.TopUpStatusPending {
		if err := recordSubscriptionRiskCase(order, input, model.PaymentRiskReasonOrderStatusInvalid); err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if !model.PaymentProviderMatches(order.PaymentProvider, order.PaymentMethod, input.PaymentProvider, input.PaymentMethod) {
		if err := recordSubscriptionRiskCase(order, input, model.PaymentRiskReasonPaymentMethodMismatch); err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	expectedProvider := model.InferPaymentProvider(order.PaymentProvider, order.PaymentMethod)
	if !paymentMethodMatchesForProvider(expectedProvider, order.PaymentMethod, input.PaymentMethod) {
		if err := recordSubscriptionRiskCase(order, input, model.PaymentRiskReasonPaymentMethodMismatch); err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if input.ProviderAmount <= 0 || !paymentAmountsMatch(order.Money, input.ProviderAmount) {
		if err := recordSubscriptionRiskCase(order, input, model.PaymentRiskReasonAmountMismatch); err != nil {
			return PaymentCallbackValidationResult{}, err
		}
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	return PaymentCallbackValidationResult{AlreadyCompleted: alreadyCompleted}, nil
}

func paymentMethodMatches(expected string, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" || actual == "" {
		return true
	}
	return strings.EqualFold(expected, actual)
}

func paymentMethodMatchesForProvider(provider string, expected string, actual string) bool {
	if model.NormalizePaymentProvider(provider) == model.PaymentProviderEpay {
		return true
	}
	return paymentMethodMatches(expected, actual)
}

func providerPaymentMethod(input PaymentCallbackValidationInput) string {
	if strings.TrimSpace(input.ProviderPaymentMethod) != "" {
		return strings.TrimSpace(input.ProviderPaymentMethod)
	}
	return strings.TrimSpace(input.PaymentMethod)
}

func paymentAmountsMatch(expected float64, actual float64) bool {
	if math.IsNaN(expected) || math.IsNaN(actual) {
		return false
	}
	left := decimal.NewFromFloat(expected).Round(2)
	right := decimal.NewFromFloat(actual).Round(2)
	return left.Equal(right)
}

func recordTopUpRiskCase(topUp *model.TopUp, input PaymentCallbackValidationInput, reason string, expectedMoney float64) error {
	if topUp == nil {
		return nil
	}
	currency := model.NormalizePaymentCurrency(input.Currency)
	settlementMoney, settlementCurrency, settlementKnown := topUp.SettlementPaymentAmount()
	if expectedMoney <= 0 {
		if settlementKnown {
			expectedMoney = settlementMoney
		}
	}
	if currency == "" && settlementKnown {
		currency = settlementCurrency
	}
	_, err := model.UpsertPaymentRiskCase(model.PaymentRiskCaseUpsertInput{
		RecordType:            model.PaymentRiskRecordTypeTopUp,
		TradeNo:               topUp.TradeNo,
		UserId:                topUp.UserId,
		PaymentMethod:         topUp.PaymentMethod,
		ProviderPaymentMethod: providerPaymentMethod(input),
		ExpectedAmount:        topUp.Amount,
		ExpectedMoney:         expectedMoney,
		ReceivedMoney:         input.ProviderAmount,
		Currency:              currency,
		Source:                strings.TrimSpace(input.Source),
		Reason:                reason,
		OrderStatus:           topUp.Status,
		ProviderPayload:       strings.TrimSpace(input.ProviderPayload),
	})
	return err
}

func recordSubscriptionRiskCase(order *model.SubscriptionOrder, input PaymentCallbackValidationInput, reason string) error {
	if order == nil {
		return nil
	}
	payload := strings.TrimSpace(input.ProviderPayload)
	if payload == "" {
		payload = strings.TrimSpace(order.ProviderPayload)
	}
	_, err := model.UpsertPaymentRiskCase(model.PaymentRiskCaseUpsertInput{
		RecordType:            model.PaymentRiskRecordTypeSubscription,
		TradeNo:               order.TradeNo,
		UserId:                order.UserId,
		PaymentMethod:         order.PaymentMethod,
		ProviderPaymentMethod: providerPaymentMethod(input),
		ExpectedMoney:         order.Money,
		ReceivedMoney:         input.ProviderAmount,
		Currency:              strings.TrimSpace(input.Currency),
		Source:                strings.TrimSpace(input.Source),
		Reason:                reason,
		OrderStatus:           order.Status,
		ProviderPayload:       payload,
	})
	return err
}

func RecordSubscriptionProcessingRiskCase(input PaymentCallbackValidationInput, processingErr error) {
	tradeNo := strings.TrimSpace(input.TradeNo)
	if tradeNo == "" {
		return
	}
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		return
	}
	payload := strings.TrimSpace(input.ProviderPayload)
	if payload == "" {
		payload = strings.TrimSpace(order.ProviderPayload)
	}
	handlerNote := ""
	if processingErr != nil {
		handlerNote = strings.TrimSpace(processingErr.Error())
	}
	_, _ = model.UpsertPaymentRiskCase(model.PaymentRiskCaseUpsertInput{
		RecordType:            model.PaymentRiskRecordTypeSubscription,
		TradeNo:               order.TradeNo,
		UserId:                order.UserId,
		PaymentMethod:         order.PaymentMethod,
		ProviderPaymentMethod: providerPaymentMethod(input),
		ExpectedMoney:         order.Money,
		ReceivedMoney:         input.ProviderAmount,
		Currency:              strings.TrimSpace(input.Currency),
		Source:                strings.TrimSpace(input.Source),
		Reason:                model.PaymentRiskReasonManualReview,
		OrderStatus:           order.Status,
		ProviderPayload:       payload,
		HandlerNote:           handlerNote,
	})
}
