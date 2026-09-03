package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestWritePaymentQuoteIncludesCurrencyAndKeepsLegacyAmount(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	writePaymentQuote(context, 0.2, "usd")

	require.Equal(t, 200, recorder.Code)
	var response struct {
		Message   string `json:"message"`
		Data      string `json:"data"`
		Currency  string `json:"currency"`
		Estimated bool   `json:"estimated"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "success", response.Message)
	require.Equal(t, "0.20", response.Data)
	require.Equal(t, "USD", response.Currency)
	require.True(t, response.Estimated)
}

func TestWritePaymentQuoteRejectsInvalidCurrency(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	writePaymentQuote(context, 0.2, "US")

	var response struct {
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "error", response.Message)
}
