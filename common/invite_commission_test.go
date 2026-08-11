package common

import (
	"math"
	"testing"
)

func TestValidateInviteCommissionRates(t *testing.T) {
	tests := []struct {
		name    string
		first   float64
		second  float64
		wantErr bool
	}{
		{name: "disabled", first: 0, second: 0},
		{name: "valid two level", first: 0.1, second: 0.05},
		{name: "exact combined maximum", first: 0.8, second: 0.2},
		{name: "negative first", first: -0.01, second: 0, wantErr: true},
		{name: "negative second", first: 0, second: -0.01, wantErr: true},
		{name: "first above maximum", first: 1.01, second: 0, wantErr: true},
		{name: "second above maximum", first: 0, second: 1.01, wantErr: true},
		{name: "combined above maximum", first: 0.8, second: 0.21, wantErr: true},
		{name: "nan", first: math.NaN(), second: 0, wantErr: true},
		{name: "infinity", first: 0, second: math.Inf(1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInviteCommissionRates(tt.first, tt.second)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateInviteCommissionRates(%v, %v) error = %v, wantErr %v", tt.first, tt.second, err, tt.wantErr)
			}
		})
	}
}
