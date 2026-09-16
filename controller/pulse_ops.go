package controller

import (
	"net/http"
	"net/url"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	pulseAdminRole       = "admin"
	pulseOpsOverviewPath = "/v1/internal/admin/operations/overview"
)

// GetPulseOperationsOverview proxies Pulse's read-only operations projection to
// the new-api admin console.
//
// The console cannot call Pulse directly: that route requires an HMAC signature
// under the admin role, and putting the admin secret in a browser would hand
// every visitor the privilege. new-api already authenticates administrators via
// session, so it signs here instead and the secret stays server-side.
func GetPulseOperationsOverview(c *gin.Context) {
	if c.GetInt("role") < common.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权访问"})
		return
	}
	if len(c.Request.URL.Query()) != 0 {
		writePulseBFFBadRequest(c, "运营概览不支持查询参数")
		return
	}
	proxyPulseSignedRead(c, pulseAdminRole, pulseOpsConfig, pulseOpsOverviewPath, nil, pulseBFFHTTPClient)
}

func pulseOpsConfig() (*url.URL, string, error) {
	return pulseInternalConfig("PULSE_ADMIN_HMAC_SECRET")
}
