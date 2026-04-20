package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/setting"
)

func isStripeWebhookEnabled() bool {
	return strings.TrimSpace(setting.StripeApiSecret) != "" &&
		strings.TrimSpace(setting.StripeWebhookSecret) != "" &&
		strings.TrimSpace(setting.StripePriceId) != ""
}
