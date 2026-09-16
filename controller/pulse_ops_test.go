package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// The admin secret is what the browser must never hold. Verify the proxy signs
// with the admin role and the admin key, and forwards no browser credential.
func TestPulseOpsSignsWithAdminRoleAndAdminSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "pulse-admin-secret-at-least-32-chars"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, pulseOpsOverviewPath, r.URL.EscapedPath())
		require.Equal(t, pulseAdminRole, r.Header.Get(pulseHeaderRole))
		require.Equal(t, "7", r.Header.Get(pulseHeaderUserID))
		require.Empty(t, r.Header.Get("Cookie"))
		timestamp, err := strconv.ParseInt(r.Header.Get(pulseHeaderTimestamp), 10, 64)
		require.NoError(t, err)
		canonical := pulseBFFCanonicalPayload(r.Method, r.URL.EscapedPath(), "7", pulseAdminRole, timestamp, r.Header.Get(pulseHeaderNonce), nil)
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(canonical))
		require.Equal(t, hex.EncodeToString(mac.Sum(nil)), r.Header.Get(pulseHeaderSignature))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"periods":[]}`))
	}))
	defer upstream.Close()
	t.Setenv("PULSE_INTERNAL_URL", upstream.URL)
	t.Setenv("PULSE_ADMIN_HMAC_SECRET", secret)

	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/pulse/ops/overview", nil)
	c.Set("id", 7)
	c.Set("role", common.RoleAdminUser)
	GetPulseOperationsOverview(c)

	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"periods":[]}`, response.Body.String())
}

// The user-facing BFF secret must not grant operations access, so the two
// configurations resolve independently.
func TestPulseOpsRequiresItsOwnSecret(t *testing.T) {
	t.Setenv("PULSE_INTERNAL_URL", "http://pulse.internal")
	t.Setenv("PULSE_USER_BFF_HMAC_SECRET", "user-secret")
	t.Setenv("PULSE_ADMIN_HMAC_SECRET", "")

	_, _, err := pulseOpsConfig()
	require.Error(t, err)
	_, bffSecret, err := pulseBFFConfig()
	require.NoError(t, err)
	require.Equal(t, "user-secret", bffSecret)
}

func TestPulseOpsRejectsNonAdminAndQueryParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream must not be reached")
	}))
	defer upstream.Close()
	t.Setenv("PULSE_INTERNAL_URL", upstream.URL)
	t.Setenv("PULSE_ADMIN_HMAC_SECRET", "pulse-admin-secret-at-least-32-chars")

	cases := []struct {
		name   string
		role   int
		target string
		want   int
	}{
		{"普通用户", common.RoleCommonUser, "/api/pulse/ops/overview", http.StatusForbidden},
		{"未设置角色", 0, "/api/pulse/ops/overview", http.StatusForbidden},
		{"带查询参数", common.RoleAdminUser, "/api/pulse/ops/overview?period_id=1", http.StatusBadRequest},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(response)
			c.Request = httptest.NewRequest(http.MethodGet, testCase.target, nil)
			c.Set("id", 7)
			if testCase.role != 0 {
				c.Set("role", testCase.role)
			}
			GetPulseOperationsOverview(c)
			require.Equal(t, testCase.want, response.Code)
		})
	}
}

// A Pulse outage must degrade to 503 rather than surface upstream internals.
func TestPulseOpsDegradesWhenUpstreamFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "pulse_period table is missing", http.StatusInternalServerError)
	}))
	defer upstream.Close()
	t.Setenv("PULSE_INTERNAL_URL", upstream.URL)
	t.Setenv("PULSE_ADMIN_HMAC_SECRET", "pulse-admin-secret-at-least-32-chars")

	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/pulse/ops/overview", nil)
	c.Set("id", 7)
	c.Set("role", common.RoleAdminUser)
	GetPulseOperationsOverview(c)

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.NotContains(t, response.Body.String(), "pulse_period")
}
