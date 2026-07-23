package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldWriteRelayErrorResponseSkipsErrorAlreadySentInResponsesStream(t *testing.T) {
	t.Parallel()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	require.True(t, shouldWriteRelayErrorResponse(c))

	common.SetContextKey(c, constant.ContextKeyResponsesStreamErrorWritten, true)
	require.False(t, shouldWriteRelayErrorResponse(c))
}
