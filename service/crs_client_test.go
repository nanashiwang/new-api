package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func TestValidateCRSURLAllowsCustomPortButKeepsSSRFGuards(t *testing.T) {
	original := *system_setting.GetFetchSetting()
	t.Cleanup(func() {
		*system_setting.GetFetchSetting() = original
	})

	setting := system_setting.GetFetchSetting()
	setting.EnableSSRFProtection = true
	setting.AllowPrivateIp = false
	setting.DomainFilterMode = false
	setting.IpFilterMode = false
	setting.DomainList = nil
	setting.IpList = nil
	setting.AllowedPorts = []string{"80", "443"}
	setting.ApplyIPFilterForDomain = false

	err := common.ValidateURLWithFetchSetting(
		"https://8.8.8.8:3000/admin/dashboard",
		setting.EnableSSRFProtection,
		setting.AllowPrivateIp,
		setting.DomainFilterMode,
		setting.IpFilterMode,
		setting.DomainList,
		setting.IpList,
		setting.AllowedPorts,
		setting.ApplyIPFilterForDomain,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "port 3000 is not allowed")

	require.NoError(t, validateCRSURL("https://8.8.8.8:3000/admin/dashboard"))

	err = validateCRSURL("http://127.0.0.1:3000/admin/dashboard")
	require.Error(t, err)
	require.Contains(t, err.Error(), "private IP address not allowed")
}
