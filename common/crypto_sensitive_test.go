package common

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSensitiveDataEncryptionIsAuthenticatedAndPurposeBound(t *testing.T) {
	originalSecret := CryptoSecret
	CryptoSecret = "unit-test-secret"
	t.Cleanup(func() { CryptoSecret = originalSecret })

	plaintext := []byte("high-risk prompt evidence")
	ciphertext, nonce, err := EncryptSensitiveData(plaintext, "content-safety-evidence")
	require.NoError(t, err)
	require.False(t, bytes.Contains(ciphertext, plaintext))

	decrypted, err := DecryptSensitiveData(ciphertext, nonce, "content-safety-evidence")
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)

	_, err = DecryptSensitiveData(ciphertext, nonce, "another-purpose")
	require.Error(t, err)
	ciphertext[0] ^= 1
	_, err = DecryptSensitiveData(ciphertext, nonce, "content-safety-evidence")
	require.Error(t, err)
}
