package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func signedPulseRequest(t *testing.T, body string, timestamp time.Time, nonce string) *http.Request {
	t.Helper()
	const secret = "pulse-test-secret"
	userID := "42"
	role := "pulse-settlement"
	req := httptest.NewRequest(http.MethodPost, "http://example.test/api/internal/pulse/benefits/grant", bytes.NewBufferString(body))
	req.Header.Set(pulseUserHeader, userID)
	req.Header.Set(pulseRoleHeader, role)
	req.Header.Set(pulseTimestampHeader, strconv.FormatInt(timestamp.Unix(), 10))
	req.Header.Set(pulseNonceHeader, nonce)
	canonical := pulseCanonicalPayload(req.Method, req.URL.EscapedPath(), userID, role, timestamp.Unix(), nonce, []byte(body))
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write([]byte(canonical))
	require.NoError(t, err)
	req.Header.Set(pulseSignatureHeader, hex.EncodeToString(mac.Sum(nil)))
	return req
}

func TestVerifyPulseServiceRequestAcceptsValidRequestAndRejectsReplay(t *testing.T) {
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedisEnabled })
	t.Setenv("PULSE_ENV", "test")

	now := time.Now().Truncate(time.Second)
	req := signedPulseRequest(t, `{"source_ref":"grant-1"}`, now, "nonce-valid-1")
	require.True(t, verifyPulseServiceRequest(req, "pulse-test-secret"))
	require.False(t, verifyPulseServiceRequest(req, "pulse-test-secret"), "nonce must be single-use")
	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, `{"source_ref":"grant-1"}`, string(body))
}

func TestVerifyPulseServiceRequestRejectsMissingSecretAndTampering(t *testing.T) {
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedisEnabled })
	t.Setenv("PULSE_ENV", "test")
	now := time.Now().Truncate(time.Second)

	require.False(t, verifyPulseServiceRequest(signedPulseRequest(t, "body", now, "nonce-missing-secret"), ""))
	tampered := signedPulseRequest(t, "body", now, "nonce-tampered")
	tampered.Body = io.NopCloser(bytes.NewBufferString("changed"))
	require.False(t, verifyPulseServiceRequest(tampered, "pulse-test-secret"))
}

func TestVerifyPulseServiceRequestFailsClosedInProductionWithoutRedis(t *testing.T) {
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedisEnabled })
	t.Setenv("PULSE_ENV", "production")
	req := signedPulseRequest(t, "body", time.Now().Truncate(time.Second), "nonce-production-no-redis")
	require.False(t, verifyPulseServiceRequest(req, "pulse-test-secret"))
}

func TestVerifyPulseServiceRequestAcceptsPreviousSecretFromRotationSet(t *testing.T) {
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedisEnabled })
	t.Setenv("PULSE_ENV", "test")
	t.Setenv("PULSE_SERVICE_HMAC_SECRET", "pulse-current-secret")
	t.Setenv("PULSE_SERVICE_HMAC_SECRET_PREVIOUS", "pulse-test-secret")
	req := signedPulseRequest(t, `{"source_ref":"grant-rotation"}`, time.Now().Truncate(time.Second), "nonce-rotation")
	require.True(t, verifyPulseServiceRequestWithSecrets(req, pulseServiceHMACSecrets()))
}

func TestPulseServiceHMACSecretsRequireActiveKey(t *testing.T) {
	t.Setenv("PULSE_SERVICE_HMAC_SECRET", "")
	t.Setenv("PULSE_SERVICE_HMAC_SECRET_PREVIOUS", "old-secret")
	require.Empty(t, pulseServiceHMACSecrets())
}

func TestPulseHMACSecretsFailClosedForInvalidProductionRotation(t *testing.T) {
	t.Setenv("PULSE_ENV", "production")
	require.Empty(t, pulseHMACSecrets("short", ""))
	require.Empty(t, pulseHMACSecrets(strings.Repeat("c", 32), "short"))
	require.Empty(t, pulseHMACSecrets(strings.Repeat("c", 32), strings.Repeat("c", 32)))
}

func TestPulseServiceAuthStoresVerifiedServiceIdentity(t *testing.T) {
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedisEnabled })
	t.Setenv("PULSE_ENV", "test")
	t.Setenv("PULSE_SERVICE_HMAC_SECRET", "pulse-test-secret")
	t.Setenv("PULSE_SERVICE_HMAC_SECRET_PREVIOUS", "")

	req := signedPulseRequest(t, `{"source_ref":"grant-context"}`, time.Now().Truncate(time.Second), "nonce-context")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req

	PulseServiceAuth()(ctx)
	userID, exists := ctx.Get(pulseServiceUserIDKey)
	require.True(t, exists)
	require.Equal(t, uint64(42), userID)
	role, exists := ctx.Get("pulse_service_role")
	require.True(t, exists)
	require.Equal(t, "pulse-settlement", role)
	require.False(t, ctx.IsAborted())
}
