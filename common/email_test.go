package common

import "testing"

func TestShouldUseSMTPLoginAuthRespectsForceSwitch(t *testing.T) {
	originForce := SMTPForceAuthLogin
	originAccount := SMTPAccount
	originServer := SMTPServer
	t.Cleanup(func() {
		SMTPForceAuthLogin = originForce
		SMTPAccount = originAccount
		SMTPServer = originServer
	})

	SMTPForceAuthLogin = true
	SMTPAccount = "user@example.com"
	SMTPServer = "smtp.example.com"

	if !shouldUseSMTPLoginAuth() {
		t.Fatal("expected AUTH LOGIN when SMTPForceAuthLogin is enabled")
	}
}

func TestShouldUseSMTPLoginAuthDetectsKnownServers(t *testing.T) {
	originForce := SMTPForceAuthLogin
	originAccount := SMTPAccount
	originServer := SMTPServer
	t.Cleanup(func() {
		SMTPForceAuthLogin = originForce
		SMTPAccount = originAccount
		SMTPServer = originServer
	})

	SMTPForceAuthLogin = false
	SMTPAccount = "user@example.com"
	SMTPServer = "smtp.sendcloud.net"

	if !shouldUseSMTPLoginAuth() {
		t.Fatal("expected AUTH LOGIN for configured LOGIN-only SMTP server")
	}
}
