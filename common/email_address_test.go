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
