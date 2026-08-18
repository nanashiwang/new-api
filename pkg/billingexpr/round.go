package billingexpr

import (
	"fmt"
	"math"
)

// QuotaRound converts a float64 quota value to int using half-away-from-zero
// rounding. Every tiered billing path (pre-consume, settlement, breakdown
// validation, log fields) MUST use this function to avoid +-1 discrepancies.
func QuotaRound(f float64) int {
	return int(math.Round(f))
}

func QuotaRoundChecked(f float64) (int, error) {
	// float64(math.MaxInt) rounds up to 2^63 on 64-bit platforms, so equality
	// is already outside the safely representable int range.
	if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 || f >= float64(math.MaxInt) {
		return 0, fmt.Errorf("quota must be finite, non-negative, and within int range, got %v", f)
	}
	return QuotaRound(f), nil
}
