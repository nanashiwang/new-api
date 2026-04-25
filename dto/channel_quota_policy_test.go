package dto

import "testing"

func TestQuotaPolicyValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  QuotaPolicy
		wantErr bool
	}{
		{"zero inactive", QuotaPolicy{}, false},
		{"active day quota", QuotaPolicy{Enabled: true, Period: "day", QuotaLimit: 1}, false},
		{"active week count", QuotaPolicy{Enabled: true, Period: "week", CountLimit: 1}, false},
		{"active month both", QuotaPolicy{Enabled: true, Period: "month", QuotaLimit: 1, CountLimit: 1}, false},
		{"invalid period", QuotaPolicy{Enabled: true, Period: "year", QuotaLimit: 1}, true},
		{"missing active period", QuotaPolicy{Enabled: true, QuotaLimit: 1}, true},
		{"negative quota", QuotaPolicy{Period: "day", QuotaLimit: -1}, true},
		{"negative count", QuotaPolicy{Period: "day", CountLimit: -1}, true},
		{"period allowed inactive", QuotaPolicy{Period: "day"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestQuotaPolicyIsActive(t *testing.T) {
	if (QuotaPolicy{Enabled: true}).IsActive() {
		t.Fatal("enabled without limits should be inactive")
	}
	if (QuotaPolicy{Enabled: true, QuotaLimit: 1}).IsActive() == false {
		t.Fatal("enabled quota limit should be active")
	}
	if (QuotaPolicy{Enabled: true, CountLimit: 1}).IsActive() == false {
		t.Fatal("enabled count limit should be active")
	}
	if (QuotaPolicy{Enabled: false, CountLimit: 1}).IsActive() {
		t.Fatal("disabled policy should be inactive")
	}
}
