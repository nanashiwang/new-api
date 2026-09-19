package service

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestOutcomeDoesNotTreatHTTP200AsStreamSuccess(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	require.True(t, RequestSucceeded(c))
	c.Set(RequestOutcomeKey, false)
	require.False(t, RequestSucceeded(c))
	c.Set(RequestOutcomeKey, true)
	require.True(t, RequestSucceeded(c))
	MarkRequestFailure(c, types.NewError(errors.New("SSE failed"), types.ErrorCodeBadResponse))
	require.False(t, RequestSucceeded(c))
}
