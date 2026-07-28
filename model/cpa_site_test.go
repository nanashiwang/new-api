package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCPASiteNormalizesAndValidatesHost(t *testing.T) {
	site := &CPASite{Host: " https://CPA.Example.com:8317/ ", Scheme: "HTTP", ManagementKeyEncrypted: "encrypted"}
	require.NoError(t, site.Validate(true))
	require.Equal(t, "cpa.example.com:8317", site.Host)
	require.Equal(t, "http", site.Scheme)
	require.Equal(t, "http://cpa.example.com:8317", site.BaseURL())
}

func TestCPASiteRejectsUnsafeOrMalformedHosts(t *testing.T) {
	tests := []string{
		"",
		"user@example.com",
		"example.com/path",
		"example.com?key=value",
		"example.com:0",
		"example.com:65536",
		"bad host.example.com",
		"-bad.example.com",
		strings.Repeat("a", 256),
	}
	for _, host := range tests {
		t.Run(host, func(t *testing.T) {
			site := &CPASite{Host: host, Scheme: "https", ManagementKeyEncrypted: "encrypted"}
			require.Error(t, site.Validate(true))
		})
	}
}

func TestCPASiteRejectsOverlongName(t *testing.T) {
	site := &CPASite{Host: "cpa.example.com", Name: strings.Repeat("名", 129), ManagementKeyEncrypted: "encrypted"}
	require.ErrorIs(t, site.Validate(true), ErrCPASiteNameTooLong)
}

func TestCPASecretEncryptionRoundTripAndMasking(t *testing.T) {
	encrypted, err := EncryptCPASecret("management-secret-value")
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)
	require.NotContains(t, encrypted, "management-secret-value")

	plain, err := DecryptCPASecret(encrypted)
	require.NoError(t, err)
	require.Equal(t, "management-secret-value", plain)
	require.Equal(t, "ma****ue", MaskCPASecret(plain))
}
