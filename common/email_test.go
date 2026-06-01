package common

import (
	"strings"
	"testing"
)

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

func TestBuildEmailMessageWithAttachment(t *testing.T) {
	originFrom := SMTPFrom
	originAccount := SMTPAccount
	originServer := SMTPServer
	originSystemName := SystemName
	t.Cleanup(func() {
		SMTPFrom = originFrom
		SMTPAccount = originAccount
		SMTPServer = originServer
		SystemName = originSystemName
	})

	SMTPFrom = "noreply@example.com"
	SMTPAccount = "noreply@example.com"
	SMTPServer = "smtp.example.com"
	SystemName = "NAN"

	message, err := buildEmailMessage("发票已开具", "user@example.com", "<p>ok</p>", []EmailAttachment{{
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Data:        []byte("%PDF-1.4\n"),
	}})
	if err != nil {
		t.Fatalf("buildEmailMessage() error = %v", err)
	}
	raw := string(message)
	for _, want := range []string{
		"Content-Type: multipart/mixed;",
		"Content-Type: text/html; charset=UTF-8",
		"Content-Type: application/pdf; name=invoice.pdf",
		"Content-Disposition: attachment; filename=invoice.pdf",
		"JVBERi0xLjQK",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("message missing %q:\n%s", want, raw)
		}
	}
}
