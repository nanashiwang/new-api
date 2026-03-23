package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetUserPaymentRecords(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	params := model.PaymentRecordSearchParams{
		Keyword:       c.Query("keyword"),
		Status:        c.Query("status"),
		PaymentMethod: c.Query("payment_method"),
	}

	records, total, err := model.GetUserPaymentRecordsByParams(userId, params, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(records)
	common.ApiSuccess(c, pageInfo)
}

func GetAllPaymentRecords(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	params := model.PaymentRecordSearchParams{
		Keyword:       c.Query("keyword"),
		Username:      c.Query("username"),
		Status:        c.Query("status"),
		PaymentMethod: c.Query("payment_method"),
	}

	records, total, err := model.GetAllPaymentRecordsByParams(params, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(records)
	common.ApiSuccess(c, pageInfo)
}
