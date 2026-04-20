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
	ProviderAmount        float64
	Currency              string
	Source                string
	ProviderPayload       string
	ProviderPaymentMethod string
}

type PaymentCallbackValidationResult struct {
	AlreadyCompleted bool
}

func ValidateTopUpCallback(input PaymentCallbackValidationInput) (PaymentCallbackValidationResult, error) {
	tradeNo := strings.TrimSpace(input.TradeNo)
	if tradeNo == "" {
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		_, _ = model.UpsertPaymentRiskCase(model.PaymentRiskCaseUpsertInput{
			RecordType:      model.PaymentRiskRecordTypeTopUp,
			TradeNo:         tradeNo,
			Source:          strings.TrimSpace(input.Source),
			Reason:          model.PaymentRiskReasonOrderNotFound,
			ReceivedMoney:   input.ProviderAmount,
			Currency:        strings.TrimSpace(input.Currency),
			ProviderPayload: strings.TrimSpace(input.ProviderPayload),
		})
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if topUp.Status == common.TopUpStatusSuccess {
		return PaymentCallbackValidationResult{AlreadyCompleted: true}, nil
	}
	if topUp.Status != common.TopUpStatusPending {
		recordTopUpRiskCase(topUp, input, model.PaymentRiskReasonOrderStatusInvalid, 0)
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if !paymentMethodMatches(topUp.PaymentMethod, input.PaymentMethod) {
		recordTopUpRiskCase(topUp, input, model.PaymentRiskReasonPaymentMethodMismatch, 0)
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}

	expectedMoney := topUp.Money
	if strings.EqualFold(topUp.PaymentMethod, "stripe") {
		if group, err := model.GetUserGroup(topUp.UserId, true); err == nil {
			expectedMoney = CalculateStripeTopUpPayMoney(float64(topUp.Amount), group)
		}
	}
	if input.ProviderAmount <= 0 || !paymentAmountsMatch(expectedMoney, input.ProviderAmount) {
		recordTopUpRiskCase(topUp, input, model.PaymentRiskReasonAmountMismatch, expectedMoney)
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	return PaymentCallbackValidationResult{}, nil
}

func ValidateSubscriptionCallback(input PaymentCallbackValidationInput) (PaymentCallbackValidationResult, error) {
	tradeNo := strings.TrimSpace(input.TradeNo)
	if tradeNo == "" {
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}

	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		_, _ = model.UpsertPaymentRiskCase(model.PaymentRiskCaseUpsertInput{
			RecordType:      model.PaymentRiskRecordTypeSubscription,
			TradeNo:         tradeNo,
			Source:          strings.TrimSpace(input.Source),
			Reason:          model.PaymentRiskReasonOrderNotFound,
			ReceivedMoney:   input.ProviderAmount,
			Currency:        strings.TrimSpace(input.Currency),
			ProviderPayload: strings.TrimSpace(input.ProviderPayload),
		})
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if order.Status == common.TopUpStatusSuccess {
		return PaymentCallbackValidationResult{AlreadyCompleted: true}, nil
	}
	if order.Status != common.TopUpStatusPending {
		recordSubscriptionRiskCase(order, input, model.PaymentRiskReasonOrderStatusInvalid)
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if !paymentMethodMatches(order.PaymentMethod, input.PaymentMethod) {
		recordSubscriptionRiskCase(order, input, model.PaymentRiskReasonPaymentMethodMismatch)
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	if input.ProviderAmount <= 0 || !paymentAmountsMatch(order.Money, input.ProviderAmount) {
		recordSubscriptionRiskCase(order, input, model.PaymentRiskReasonAmountMismatch)
		return PaymentCallbackValidationResult{}, ErrPaymentCallbackRejected
	}
	return PaymentCallbackValidationResult{}, nil
}

func paymentMethodMatches(expected string, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" || actual == "" {
		return true
	}
	return strings.EqualFold(expected, actual)
}

func paymentAmountsMatch(expected float64, actual float64) bool {
	if math.IsNaN(expected) || math.IsNaN(actual) {
		return false
	}
	left := decimal.NewFromFloat(expected).Round(2)
	right := decimal.NewFromFloat(actual).Round(2)
	return left.Equal(right)
}

func recordTopUpRiskCase(topUp *model.TopUp, input PaymentCallbackValidationInput, reason string, expectedMoney float64) {
	if topUp == nil {
		return
	}
	if expectedMoney <= 0 {
		expectedMoney = topUp.Money
	}
	_, _ = model.UpsertPaymentRiskCase(model.PaymentRiskCaseUpsertInput{
		RecordType:            model.PaymentRiskRecordTypeTopUp,
		TradeNo:               topUp.TradeNo,
		UserId:                topUp.UserId,
		PaymentMethod:         topUp.PaymentMethod,
		ProviderPaymentMethod: strings.TrimSpace(input.PaymentMethod),
		ExpectedAmount:        topUp.Amount,
		ExpectedMoney:         expectedMoney,
		ReceivedMoney:         input.ProviderAmount,
		Currency:              strings.TrimSpace(input.Currency),
		Source:                strings.TrimSpace(input.Source),
		Reason:                reason,
		OrderStatus:           topUp.Status,
		ProviderPayload:       strings.TrimSpace(input.ProviderPayload),
	})
}

func recordSubscriptionRiskCase(order *model.SubscriptionOrder, input PaymentCallbackValidationInput, reason string) {
	if order == nil {
		return
	}
	payload := strings.TrimSpace(input.ProviderPayload)
	if payload == "" {
		payload = strings.TrimSpace(order.ProviderPayload)
	}
	_, _ = model.UpsertPaymentRiskCase(model.PaymentRiskCaseUpsertInput{
		RecordType:            model.PaymentRiskRecordTypeSubscription,
		TradeNo:               order.TradeNo,
		UserId:                order.UserId,
		PaymentMethod:         order.PaymentMethod,
		ProviderPaymentMethod: strings.TrimSpace(input.PaymentMethod),
		ExpectedMoney:         order.Money,
		ReceivedMoney:         input.ProviderAmount,
		Currency:              strings.TrimSpace(input.Currency),
		Source:                strings.TrimSpace(input.Source),
		Reason:                reason,
		OrderStatus:           order.Status,
		ProviderPayload:       payload,
	})
}
