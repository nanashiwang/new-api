package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func buildPaymentRecordSearchParams(c *gin.Context, includeUsername bool) model.PaymentRecordSearchParams {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	params := model.PaymentRecordSearchParams{
		Keyword:        c.Query("keyword"),
		Status:         c.Query("status"),
		PaymentMethod:  c.Query("payment_method"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}
	if includeUsername {
		params.Username = c.Query("username")
	}
	return params
}

func GetUserPaymentRecords(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	params := buildPaymentRecordSearchParams(c, false)

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
	params := buildPaymentRecordSearchParams(c, true)

	records, total, err := model.GetAllPaymentRecordsByParams(params, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(records)
	common.ApiSuccess(c, pageInfo)
}

func GetPaymentRecordStats(c *gin.Context) {
	params := buildPaymentRecordSearchParams(c, true)

	stats, err := model.GetPaymentRecordStats(params)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, stats)
}

func GetPaymentRecordRankings(c *gin.Context) {
	params := buildPaymentRecordSearchParams(c, true)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	rankings, err := model.GetPaymentRecordRankings(params, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"items": rankings,
		"limit": limit,
	})
}
