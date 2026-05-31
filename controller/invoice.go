package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type createInvoiceRequestPayload struct {
	TitleType string                  `json:"title_type"`
	Title     string                  `json:"title"`
	TaxNumber string                  `json:"tax_number"`
	Email     string                  `json:"email"`
	Phone     string                  `json:"phone"`
	Remark    string                  `json:"remark"`
	Orders    []model.InvoiceOrderRef `json:"orders"`
}

type reviewInvoiceRequestPayload struct {
	InvoiceNo   string `json:"invoice_no"`
	InvoiceUrl  string `json:"invoice_url"`
	AdminRemark string `json:"admin_remark"`
}

func GetEligibleInvoiceOrders(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	records, total, err := model.GetEligibleInvoiceOrders(c.GetInt("id"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(records)
	common.ApiSuccess(c, pageInfo)
}

func CreateInvoiceRequest(c *gin.Context) {
	var req createInvoiceRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	invoiceRequest, err := model.CreateInvoiceRequest(c.GetInt("id"), model.CreateInvoiceRequestInput{
		TitleType: req.TitleType,
		Title:     req.Title,
		TaxNumber: req.TaxNumber,
		Email:     req.Email,
		Phone:     req.Phone,
		Remark:    req.Remark,
		Orders:    req.Orders,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, invoiceRequest)
}

func GetUserInvoiceRequests(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	params := model.InvoiceRequestSearchParams{
		Status: c.Query("status"),
	}
	requests, total, err := model.GetUserInvoiceRequestsByParams(c.GetInt("id"), params, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(requests)
	common.ApiSuccess(c, pageInfo)
}

func GetUserInvoiceRequestDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	request, err := model.GetUserInvoiceRequestDetail(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, request)
}

func GetAllInvoiceRequests(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	params := model.InvoiceRequestSearchParams{
		Status:   c.Query("status"),
		Username: c.Query("username"),
	}
	requests, total, err := model.GetAllInvoiceRequestsByParams(params, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(requests)
	common.ApiSuccess(c, pageInfo)
}

func GetInvoiceRequestDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	request, err := model.GetInvoiceRequestDetail(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, request)
}

func ApproveInvoiceRequest(c *gin.Context) {
	reviewInvoiceRequest(c, "approve")
}

func RejectInvoiceRequest(c *gin.Context) {
	reviewInvoiceRequest(c, "reject")
}

func reviewInvoiceRequest(c *gin.Context, action string) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	var req reviewInvoiceRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	var request *model.InvoiceRequest
	if action == "approve" {
		request, err = model.ApproveInvoiceRequest(id, c.GetInt("id"), model.InvoiceReviewInput{
			InvoiceNo:   req.InvoiceNo,
			InvoiceUrl:  req.InvoiceUrl,
			AdminRemark: req.AdminRemark,
		})
	} else {
		request, err = model.RejectInvoiceRequest(id, c.GetInt("id"), req.AdminRemark)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, request)
}
