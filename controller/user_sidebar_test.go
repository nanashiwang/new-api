package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestDefaultSidebarConfigIncludesPulse(t *testing.T) {
	var config map[string]map[string]bool
	require.NoError(t, common.Unmarshal([]byte(generateDefaultSidebarConfig(common.RoleCommonUser)), &config))
	require.True(t, config["personal"]["enabled"])
	require.True(t, config["personal"]["pulse"])
}
