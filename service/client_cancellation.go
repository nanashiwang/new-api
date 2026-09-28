package service

import (
	"context"
	"errors"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// NormalizeClientCancellation uses the downstream context because adapters may
// stringify transport errors. Keep confirmed upstream failures and completed
// responses intact, even if the client disconnects just after they arrive.
func NormalizeClientCancellation(c *gin.Context, err *types.NewAPIError) *types.NewAPIError {
	if err == nil || c == nil || c.Request == nil ||
		!errors.Is(c.Request.Context().Err(), context.Canceled) || err.UpstreamStatusCode != 0 {
		return err
	}
	switch err.GetErrorCode() {
	case types.ErrorCodeDoRequestFailed, types.ErrorCodeReadResponseBodyFailed, types.ErrorCodeReadRequestBodyFailed:
		return types.NewClientCanceledError()
	default:
		return err
	}
}
