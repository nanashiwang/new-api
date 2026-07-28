package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func newCPATestSite(t *testing.T, serverURL string, key string) *model.CPASite {
	t.Helper()
	site := &model.CPASite{Host: serverURL, Scheme: "http"}
	require.NoError(t, site.SetManagementKeyPlain(key))
	return site
}

func TestFetchCPAAccountsUsesReadOnlyManagementEndpointAndSanitizesPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v0/management/auth-files", r.URL.Path)
		require.Equal(t, "Bearer management-secret", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"files":[{"id":"acct-1","auth_index":"idx-1","name":"codex.json","provider":"codex","status":"active","success":12,"failed":2,"email":"user@example.com","path":"/root/auth/codex.json","id_token":{"access_token":"must-not-survive"}}]}`))
	}))
	defer server.Close()

	accounts, err := FetchCPAAccounts(context.Background(), newCPATestSite(t, server.URL, "management-secret"))
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, "acct-1", accounts[0].ID)
	require.EqualValues(t, 12, accounts[0].Success)

	stored, err := common.Marshal(accounts)
	require.NoError(t, err)
	require.NotContains(t, string(stored), "/root/auth")
	require.NotContains(t, string(stored), "must-not-survive")
	require.NotContains(t, string(stored), "management-secret")
}

func TestFetchCPAAccountsDoesNotLeakAuthenticationErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key management-secret"}`))
	}))
	defer server.Close()

	_, err := FetchCPAAccounts(context.Background(), newCPATestSite(t, server.URL, "management-secret"))
	require.Error(t, err)
	require.Equal(t, "CPA management authentication failed", err.Error())
	require.NotContains(t, err.Error(), "management-secret")
}

func TestFetchCPAAccountsRejectsRedirectWithoutForwardingKey(t *testing.T) {
	redirectTargetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectTargetCalled = true
	}))
	defer target.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer server.Close()

	_, err := FetchCPAAccounts(context.Background(), newCPATestSite(t, server.URL, "management-secret"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP 302")
	require.False(t, redirectTargetCalled)
}

func TestFetchCPAAccountsRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"files":[]}` + strings.Repeat(" ", cpaManagementResponseLimit)))
	}))
	defer server.Close()

	_, err := FetchCPAAccounts(context.Background(), newCPATestSite(t, server.URL, "management-secret"))
	require.EqualError(t, err, "CPA management response is too large")
}
