package common

import "testing"

func TestParseInviteBindingSettings(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    InviteBindingSettings
		wantErr bool
	}{
		{name: "valid", raw: `{"threshold":1000,"rate_after_threshold":20}`, want: InviteBindingSettings{Threshold: 1000, RateAfterThreshold: 20}},
		{name: "disabled", raw: `{"threshold":0,"rate_after_threshold":100}`, want: InviteBindingSettings{Threshold: 0, RateAfterThreshold: 100}},
		{name: "negative threshold", raw: `{"threshold":-1,"rate_after_threshold":20}`, wantErr: true},
		{name: "negative rate", raw: `{"threshold":1,"rate_after_threshold":-1}`, wantErr: true},
		{name: "rate too high", raw: `{"threshold":1,"rate_after_threshold":101}`, wantErr: true},
		{name: "missing threshold", raw: `{"rate_after_threshold":20}`, wantErr: true},
		{name: "missing rate", raw: `{"threshold":1}`, wantErr: true},
		{name: "fractional threshold", raw: `{"threshold":1.5,"rate_after_threshold":20}`, wantErr: true},
		{name: "invalid json", raw: `{`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInviteBindingSettings(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse settings: %v", err)
			}
			if got != tt.want {
				t.Fatalf("settings = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSetInviteBindingSettingsIsAtomicSnapshot(t *testing.T) {
	original := GetInviteBindingSettings()
	t.Cleanup(func() { _ = SetInviteBindingSettings(original) })

	want := InviteBindingSettings{Threshold: 1000, RateAfterThreshold: 20}
	if err := SetInviteBindingSettings(want); err != nil {
		t.Fatalf("set settings: %v", err)
	}
	if got := GetInviteBindingSettings(); got != want {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}
}
