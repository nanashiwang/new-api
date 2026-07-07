package service

import "testing"

func TestCalculateAudioQuotaAppliesTimeRatio(t *testing.T) {
	quota := calculateAudioQuota(QuotaInfo{
		InputDetails: TokenDetails{
			TextTokens: 100,
		},
		ModelName:  "time-ratio-audio-model",
		ModelRatio: 2,
		GroupRatio: 1,
		TimeRatio:  1.5,
	})
	if quota != 300 {
		t.Fatalf("expected quota 300, got %d", quota)
	}
}
