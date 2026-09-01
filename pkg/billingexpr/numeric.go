package billingexpr

import "math"

// Expression parameters are decoded from JSON and may arrive as int, int64,
// float64, or another numeric representation. The numeric helpers deliberately
// accept interface values so max/min/ceil/floor work for both literals and
// param() results without reflection type errors.
func numericFloat(value any) float64 {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case int8:
		return float64(typed)
	case int16:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint8:
		return float64(typed)
	case uint16:
		return float64(typed)
	case uint32:
		return float64(typed)
	case uint64:
		return float64(typed)
	case float32:
		return float64(typed)
	case float64:
		return typed
	default:
		return math.NaN()
	}
}

func numericMax(left, right any) float64 { return math.Max(numericFloat(left), numericFloat(right)) }
func numericMin(left, right any) float64 { return math.Min(numericFloat(left), numericFloat(right)) }
func numericAbs(value any) float64       { return math.Abs(numericFloat(value)) }
func numericCeil(value any) float64      { return math.Ceil(numericFloat(value)) }
func numericFloor(value any) float64     { return math.Floor(numericFloat(value)) }
