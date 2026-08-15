package common

import "testing"

func TestCalculateAudioDurationQuotaOfficialMiMoBoundaries(t *testing.T) {
	originalQuotaPerUnit := QuotaPerUnit
	QuotaPerUnit = 500_000
	t.Cleanup(func() {
		QuotaPerUnit = originalQuotaPerUnit
	})

	tests := []struct {
		name          string
		billedSeconds int64
		wantQuota     int
	}{
		{name: "one second", billedSeconds: 1, wantQuota: 10},
		{name: "one second before an hour", billedSeconds: 3599, wantQuota: 36990},
		{name: "one hour", billedSeconds: 3600, wantQuota: 37000},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := CalculateAudioDurationQuota(0.074, test.billedSeconds, 1, 1)
			if got != test.wantQuota {
				t.Fatalf("CalculateAudioDurationQuota() = %d, want %d", got, test.wantQuota)
			}
		})
	}
}

func TestCalculateAudioDurationQuotaAppliesGroupAndTimeRatios(t *testing.T) {
	originalQuotaPerUnit := QuotaPerUnit
	QuotaPerUnit = 500_000
	t.Cleanup(func() {
		QuotaPerUnit = originalQuotaPerUnit
	})

	got := CalculateAudioDurationQuota(0.074, 3600, 2, 1.5)
	if got != 111000 {
		t.Fatalf("CalculateAudioDurationQuota() = %d, want 111000", got)
	}
}
