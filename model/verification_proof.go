package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// VerificationProof is server-side authority for single-use assertions. Cookie
// deletion alone cannot prevent replay, and a process-local cache is insufficient
// when requests can reach different instances. Store only the token digest.
type VerificationProof struct {
	Digest    string `gorm:"primaryKey;type:varchar(64)"`
	UserID    int
	Purpose   string `gorm:"type:varchar(128)"`
	ExpiresAt int64  `gorm:"index"`
}

func verificationProofDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func IssueVerificationProof(userID int, purpose string, ttl time.Duration) (string, error) {
	if userID < 0 || purpose == "" || len(purpose) > 128 || ttl < time.Second || ttl > 5*time.Minute {
		return "", errors.New("invalid verification proof scope")
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(secret[:])
	now := time.Now()
	// Expiry is indexed; cleanup does not participate in authorization.
	DB.Where("expires_at <= ?", now.Unix()).Delete(&VerificationProof{})
	proof := VerificationProof{Digest: verificationProofDigest(token), UserID: userID, Purpose: purpose, ExpiresAt: now.Add(ttl).Unix()}
	if err := DB.Create(&proof).Error; err != nil {
		return "", err
	}
	return token, nil
}

func ConsumeVerificationProof(token string, userID int, purpose string) (bool, error) {
	if len(token) != 64 || purpose == "" {
		return false, nil
	}
	result := DB.Where("digest = ? AND user_id = ? AND purpose = ? AND expires_at > ?",
		verificationProofDigest(token), userID, purpose, time.Now().Unix()).Delete(&VerificationProof{})
	return result.RowsAffected == 1 && result.Error == nil, result.Error
}
