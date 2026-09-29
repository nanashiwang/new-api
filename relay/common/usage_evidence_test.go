package common

import (
	"net/http/httptest"
	"testing"

	projectcommon "github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestKimiUsageEvidenceResetsBetweenChannelAttempts(t *testing.T) {
	info := &RelayInfo{KimiUsageEvidence: &KimiUsageEvidence{
		Cache: UsageCountEvidence{Status: "reported", Value: projectcommon.GetPointer(80)},
	}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info.InitChannelMeta(c)
	require.Nil(t, info.KimiUsageEvidence)
}
