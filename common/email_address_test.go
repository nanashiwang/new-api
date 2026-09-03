package common

import "testing"

func TestNormalizeEmailAddressLowercasesDomainOnly(t *testing.T) {
	got := NormalizeEmailAddress("  User.Name+Tag@Gmail.COM  ")
	want := "User.Name+Tag@gmail.com"
	if got != want {
		t.Fatalf("NormalizeEmailAddress() = %q, want %q", got, want)
	}
}

func TestEmailDomainInWhitelistIgnoresCaseAndSpace(t *testing.T) {
	if !EmailDomainInWhitelist(" Gmail.COM ", []string{"qq.com", " gmail.com "}) {
		t.Fatal("expected mixed-case domain to match normalized whitelist")
	}
	if EmailDomainInWhitelist("example.com", []string{"qq.com", "gmail.com"}) {
		t.Fatal("expected unrelated domain to be rejected")
	}
}

func TestValidateEmailAddress(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{name: "normal", email: "User.Name+Tag@Gmail.COM", valid: true},
		{name: "trimmed", email: "  user@example.com  ", valid: true},
		{name: "comma in domain", email: "2370708759@qq,com", valid: false},
		{name: "multiple at signs", email: "user@@example.com", valid: false},
		{name: "display name", email: "User <user@example.com>", valid: false},
		{name: "embedded whitespace", email: "user @example.com", valid: false},
		{name: "newline injection", email: "user@example.com\nBcc:evil@example.com", valid: false},
		{name: "missing local part", email: "@example.com", valid: false},
		{name: "missing domain", email: "user@", valid: false},
		{name: "consecutive domain dots", email: "user@example..com", valid: false},
		{name: "invalid domain label", email: "user@-example.com", valid: false},
		{name: "underscore in domain", email: "user@exam_ple.com", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmailAddress(tt.email)
			if (err == nil) != tt.valid {
				t.Fatalf("ValidateEmailAddress(%q) error = %v, valid = %v", tt.email, err, tt.valid)
			}
		})
	}
}
