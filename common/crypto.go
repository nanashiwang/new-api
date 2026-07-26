package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const EncryptedDataVersion = "aes-gcm-v1"

func HasPersistentCryptoSecret() bool {
	return os.Getenv("CRYPTO_SECRET") != "" || os.Getenv("SESSION_SECRET") != ""
}

func deriveEncryptionKey(purpose string) []byte {
	digest := hmac.New(sha256.New, []byte(CryptoSecret))
	digest.Write([]byte("new-api/encryption/" + purpose))
	return digest.Sum(nil)
}

// EncryptSensitiveData uses a purpose-derived key so ciphertext from one
// subsystem cannot be replayed in another encryption context.
func EncryptSensitiveData(plaintext []byte, purpose string) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(deriveEncryptionKey(purpose))
	if err != nil {
		return nil, nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	return aead.Seal(nil, nonce, plaintext, []byte(EncryptedDataVersion+":"+purpose)), nonce, nil
}

func DecryptSensitiveData(ciphertext, nonce []byte, purpose string) ([]byte, error) {
	block, err := aes.NewCipher(deriveEncryptionKey(purpose))
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("invalid encrypted data nonce")
	}
	return aead.Open(nil, nonce, ciphertext, []byte(EncryptedDataVersion+":"+purpose))
}

func GenerateHMACWithKey(key []byte, data string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateHMAC(data string) string {
	h := hmac.New(sha256.New, []byte(CryptoSecret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func Password2Hash(password string) (string, error) {
	passwordBytes := []byte(password)
	hashedPassword, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)
	return string(hashedPassword), err
}

func ValidatePasswordAndHash(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
