package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func writePaymentQuote(c *gin.Context, amount float64, currency string) {
	normalizedCurrency := model.NormalizePaymentCurrency(currency)
	if normalizedCurrency == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付币种配置无效"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "success",
		"data":      strconv.FormatFloat(amount, 'f', 2, 64),
		"currency":  normalizedCurrency,
		"estimated": true,
	})
}
