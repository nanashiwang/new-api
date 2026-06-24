package claude

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandleStreamResponseData_IgnoresHTTP200ForCapacityErrors(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{}
	claudeInfo := &ClaudeResponseInfo{}

	err := HandleStreamResponseData(
		ctx,
		info,
		claudeInfo,
		`{"type":"error","error":{"type":"api_error","message":"Selected model is at capacity.Please try a different model."}}`,
		200,
	)
	require.NotNil(t, err)
	require.Equal(t, 529, err.StatusCode)
	require.Equal(t, "api_error", err.ToOpenAIError().Type)
}
