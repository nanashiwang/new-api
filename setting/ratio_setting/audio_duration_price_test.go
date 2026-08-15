package ratio_setting

import "testing"

func TestDefaultAudioDurationPriceMatchesMiMoOfficialPrice(t *testing.T) {
	if got := defaultAudioDurationPrice["mimo-v2.5-asr"]; got != 0.074 {
		t.Fatalf("default mimo-v2.5-asr price = %v, want 0.074", got)
	}
}

func TestUpdateAudioDurationPriceRejectsInvalidValueWithoutPublishingPartialState(t *testing.T) {
	original := AudioDurationPrice2JSONString()
	t.Cleanup(func() {
		if err := UpdateAudioDurationPriceByJSONString(original); err != nil {
			t.Fatalf("restore audio duration price: %v", err)
		}
	})

	if err := UpdateAudioDurationPriceByJSONString(`{"mimo-v2.5-asr":0.1}`); err != nil {
		t.Fatalf("set audio duration price: %v", err)
	}
	if err := UpdateAudioDurationPriceByJSONString(`{"mimo-v2.5-asr":-1}`); err == nil {
		t.Fatal("negative audio duration price should be rejected")
	}
	got, ok := GetAudioDurationPrice("mimo-v2.5-asr")
	if !ok || got != 0.1 {
		t.Fatalf("last valid price was not preserved: got %v, ok=%v", got, ok)
	}
}
