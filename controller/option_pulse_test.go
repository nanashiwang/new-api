package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetPulseOptionsReportsConfiguredSecretsWithoutExposingValues(t *testing.T) {
	original := common.GetPulseConfigOverrides()
	t.Cleanup(func() { common.ReplacePulseConfig(original) })
	current, previous := strings.Repeat("a", 32), strings.Repeat("b", 32)
	common.ReplacePulseConfig(map[string]string{"PulseServiceHMACSecret": current, "PulseServiceHMACSecretPrevious": previous, "PulseRollbackHMACSecret": ""})
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	GetPulseOptions(ctx)
	require.Equal(t, http.StatusOK, response.Code)
	require.NotContains(t, response.Body.String(), current)
	require.NotContains(t, response.Body.String(), previous)
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Config       map[string]string `json:"config"`
			Secrets      map[string]bool   `json:"secrets"`
			QuotaPerUnit float64           `json:"quota_per_unit"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
	require.True(t, result.Success)
	require.True(t, result.Data.Secrets["PulseServiceHMACSecret"])
	require.True(t, result.Data.Secrets["PulseServiceHMACSecretPrevious"])
	require.False(t, result.Data.Secrets["PulseRollbackHMACSecret"])
	require.NotContains(t, result.Data.Config, "PulseServiceHMACSecret")
	require.Equal(t, common.QuotaPerUnit, result.Data.QuotaPerUnit)
}
