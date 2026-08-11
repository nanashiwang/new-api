package common

import (
	"fmt"
	"math"
)

const (
	InviteCommissionRateMin = 0.0
	InviteCommissionRateMax = 1.0
)

// ValidateInviteCommissionRates validates the complete payout configuration.
// Limiting the combined rate to 100% prevents an accidental configuration from
// paying more commission than the trusted payment base.
func ValidateInviteCommissionRates(firstLevelRate, secondLevelRate float64) error {
	rates := []struct {
		name  string
		value float64
	}{
		{name: "first-level", value: firstLevelRate},
		{name: "second-level", value: secondLevelRate},
	}
	for _, rate := range rates {
		if math.IsNaN(rate.value) || math.IsInf(rate.value, 0) ||
			rate.value < InviteCommissionRateMin || rate.value > InviteCommissionRateMax {
			return fmt.Errorf("%s invite commission rate must be between 0 and 1", rate.name)
		}
	}
	if firstLevelRate+secondLevelRate > InviteCommissionRateMax {
		return fmt.Errorf("combined invite commission rate must not exceed 1")
	}
	return nil
}

func InviteCommissionConfigured() bool {
	configured, _, _ := InviteCommissionConfigSnapshot()
	return configured
}

// InviteCommissionConfigSnapshot returns one internally consistent view of the
// enable switch and both rates while option synchronization may be running.
func InviteCommissionConfigSnapshot() (configured bool, firstLevelRate, secondLevelRate float64) {
	OptionMapRWMutex.RLock()
	defer OptionMapRWMutex.RUnlock()

	firstLevelRate = InviterRechargeCommissionRate
	secondLevelRate = InviterRechargeSecondLevelCommissionRate
	if !InviterCommissionEnabled || ValidateInviteCommissionRates(firstLevelRate, secondLevelRate) != nil {
		return false, firstLevelRate, secondLevelRate
	}
	return firstLevelRate > 0 || secondLevelRate > 0, firstLevelRate, secondLevelRate
}
