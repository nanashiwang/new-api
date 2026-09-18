package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	pulseUserHeader          = "X-Pulse-User-Id"
	pulseRoleHeader          = "X-Pulse-Role"
	pulseTimestampHeader     = "X-Pulse-Timestamp"
	pulseNonceHeader         = "X-Pulse-Nonce"
	pulseSignatureHeader     = "X-Pulse-Signature"
	pulseServiceUserIDKey    = "pulse_service_user_id"
	pulseMaxRequestBodyBytes = 64 << 10
	pulseMaxClockSkew        = 5 * time.Minute
	minimumPulseSecretLength = 32
)

var (
	errPulseSignatureInvalid = errors.New("invalid pulse service signature")
	pulseNonceMemory         = struct {
		sync.Mutex
		values map[string]time.Time
	}{values: make(map[string]time.Time)}
)

// PulseServiceAuth authenticates calls from the Pulse worker. The secret is
// deliberately read at request time so operators can rotate it without
// rebuilding the binary. Redis is the production nonce authority; an
// in-memory fallback is allowed only outside production.
func PulseServiceAuth() gin.HandlerFunc { return pulseServiceAuthForRoles("pulse-settlement") }

func PulseBenefitQueryAuth() gin.HandlerFunc {
	return pulseServiceAuthForRoles("pulse-settlement", "pulse-rollback")
}

func PulseRollbackAuth() gin.HandlerFunc { return pulseServiceAuthForRoles("pulse-rollback") }

func pulseServiceAuthForRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := strings.TrimSpace(c.GetHeader(pulseRoleHeader))
		allowed := false
		for _, candidate := range allowedRoles {
			if role == candidate {
				allowed = true
				break
			}
		}
		secrets := pulseServiceHMACSecrets()
		if role == "pulse-rollback" {
			secrets = pulseRollbackHMACSecrets()
		}
		if !allowed || !verifyPulseServiceRequestForRole(c.Request, secrets, role) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		// Downstream middleware must use the identity that was authenticated
		// here, not an unauthenticated client address or user-supplied value.
		userID, err := strconv.ParseUint(strings.TrimSpace(c.GetHeader(pulseUserHeader)), 10, 64)
		if err != nil || userID == 0 {
			// Verification already checked this; keep the second parse fail-closed
			// if the request is mutated by another middleware.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set(pulseServiceUserIDKey, userID)
		c.Set("pulse_service_role", strings.TrimSpace(c.GetHeader(pulseRoleHeader)))
		c.Next()
	}
}

func verifyPulseServiceRequest(req *http.Request, secret string) bool {
	return verifyPulseServiceRequestWithSecrets(req, []string{secret})
}

func pulseServiceHMACSecrets() []string {
	return pulseHMACSecrets(os.Getenv("PULSE_SERVICE_HMAC_SECRET"), os.Getenv("PULSE_SERVICE_HMAC_SECRET_PREVIOUS"))
}

// Rollback credentials are never usable by the settlement worker. Fail closed
// when operators accidentally reuse any current/previous settlement key.
func pulseRollbackHMACSecrets() []string {
	secrets := pulseHMACSecrets(os.Getenv("PULSE_ROLLBACK_HMAC_SECRET"), os.Getenv("PULSE_ROLLBACK_HMAC_SECRET_PREVIOUS"))
	for _, candidate := range secrets {
		for _, worker := range []string{os.Getenv("PULSE_SERVICE_HMAC_SECRET"), os.Getenv("PULSE_SERVICE_HMAC_SECRET_PREVIOUS")} {
			if candidate == strings.TrimSpace(worker) {
				return nil
			}
		}
	}
	return secrets
}

// pulseHMACSecrets returns the active key first and, during rotation, the
// previous key second. In production a malformed rotation configuration fails
// closed instead of silently weakening the authentication boundary.
func pulseHMACSecrets(currentValue, previousValue string) []string {
	current := strings.TrimSpace(currentValue)
	previous := strings.TrimSpace(previousValue)
	if !pulseSecretUsable(current) || (previous != "" && !pulseSecretUsable(previous)) || (previous != "" && previous == current) {
		return nil
	}
	secrets := []string{current}
	if previous != "" {
		secrets = append(secrets, previous)
	}
	return secrets
}

func pulseSecretUsable(secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PULSE_ENV")), "production") {
		return len(secret) >= minimumPulseSecretLength && secret != "replace-me"
	}
	return true
}

func verifyPulseServiceRequestWithSecrets(req *http.Request, secrets []string) bool {
	return verifyPulseServiceRequestForRole(req, secrets, "pulse-settlement")
}

func verifyPulseServiceRequestForRole(req *http.Request, secrets []string, expectedRole string) bool {
	configured := false
	for _, secret := range secrets {
		if strings.TrimSpace(secret) != "" {
			configured = true
			break
		}
	}
	if req == nil || req.URL == nil || !configured {
		return false
	}
	userID := strings.TrimSpace(req.Header.Get(pulseUserHeader))
	if parsed, err := strconv.ParseUint(userID, 10, 64); err != nil || parsed == 0 {
		return false
	}
	role := strings.TrimSpace(req.Header.Get(pulseRoleHeader))
	if role != expectedRole || (role != "pulse-settlement" && role != "pulse-rollback") {
		return false
	}
	nonce := strings.TrimSpace(req.Header.Get(pulseNonceHeader))
	if nonce == "" || len(nonce) > 128 {
		return false
	}
	timestamp, err := strconv.ParseInt(strings.TrimSpace(req.Header.Get(pulseTimestampHeader)), 10, 64)
	if err != nil {
		return false
	}
	requestTime := time.Unix(timestamp, 0)
	if delta := time.Since(requestTime); delta > pulseMaxClockSkew || delta < -pulseMaxClockSkew {
		return false
	}
	if req.Body == nil {
		req.Body = http.NoBody
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, pulseMaxRequestBodyBytes+1))
	if err != nil || len(body) > pulseMaxRequestBodyBytes {
		return false
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	canonical := pulseCanonicalPayload(req.Method, req.URL.EscapedPath(), userID, role, timestamp, nonce, body)
	provided, err := hex.DecodeString(strings.TrimSpace(req.Header.Get(pulseSignatureHeader)))
	if err != nil || len(provided) != sha256.Size {
		return false
	}
	matched := false
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(canonical))
		if hmac.Equal(provided, mac.Sum(nil)) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	return claimPulseNonce(req.Context(), role+":"+userID+":"+nonce, requestTime.Add(pulseMaxClockSkew))
}

func pulseCanonicalPayload(method, path, userID, role string, timestamp int64, nonce string, body []byte) string {
	digest := sha256.Sum256(body)
	return strings.Join([]string{
		strings.ToUpper(strings.TrimSpace(method)), path, userID, role,
		strconv.FormatInt(timestamp, 10), nonce, hex.EncodeToString(digest[:]),
	}, "\n")
}

func claimPulseNonce(ctx context.Context, key string, expiresAt time.Time) bool {
	if common.RedisEnabled && common.RDB != nil {
		ttl := time.Until(expiresAt)
		if ttl <= 0 {
			return false
		}
		claimed, err := common.RDB.SetNX(ctx, "newapi:pulse:nonce:"+key, "1", ttl).Result()
		return err == nil && claimed
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PULSE_ENV")), "production") {
		return false
	}
	pulseNonceMemory.Lock()
	defer pulseNonceMemory.Unlock()
	now := time.Now()
	for existing, expiry := range pulseNonceMemory.values {
		if !expiry.After(now) {
			delete(pulseNonceMemory.values, existing)
		}
	}
	if expiry, exists := pulseNonceMemory.values[key]; exists && expiry.After(now) {
		return false
	}
	pulseNonceMemory.values[key] = expiresAt
	return true
}
