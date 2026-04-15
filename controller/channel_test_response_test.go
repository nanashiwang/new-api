package controller

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestBuildChannelTestFailureResponse_CompatibleFields(t *testing.T) {
	result := testResult{
		localErr:    errors.New("bad upstream response"),
		newAPIError: types.NewOpenAIError(errors.New("bad upstream response"), types.ErrorCodeModelPriceError, http.StatusBadRequest),
	}

	resp := buildChannelTestFailureResponse(result, 1.25)

	require.Equal(t, false, resp["success"])
	require.Equal(t, "bad upstream response", resp["message"])
	require.Equal(t, 1.25, resp["time"])
	require.Equal(t, http.StatusBadRequest, resp["status_code"])
	require.Equal(t, string(types.ErrorCodeModelPriceError), resp["error_code"])
}

func TestBuildChannelTestFailureResponse_LocalOnlyStillReturnsStatusCode(t *testing.T) {
	result := testResult{
		localErr: errors.New("unsupported channel"),
	}

	resp := buildChannelTestFailureResponse(result, 0)

	require.Equal(t, false, resp["success"])
	require.Equal(t, "unsupported channel", resp["message"])
	require.Equal(t, 0.0, resp["time"])
	require.Equal(t, http.StatusInternalServerError, resp["status_code"])
	_, hasErrorCode := resp["error_code"]
	require.False(t, hasErrorCode)
}
