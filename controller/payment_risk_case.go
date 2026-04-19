package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type CreatePaymentRiskCaseRequest struct {
	RecordType string `json:"record_type"`
	TradeNo    string `json:"trade_no"`
	Note       string `json:"note"`
}

type ResolvePaymentRiskCaseRequest struct {
	Action string `json:"action"`
	Note   string `json:"note"`
}

func GetAllPaymentRiskCases(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	params := model.PaymentRiskCaseSearchParams{
		Keyword:    c.Query("keyword"),
		Username:   c.Query("username"),
		RecordType: c.Query("record_type"),
		Status:     c.Query("status"),
		Reason:     c.Query("reason"),
	}

	riskCases, total, err := model.ListPaymentRiskCasesByParams(params, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items := make([]gin.H, 0, len(riskCases))
	for _, riskCase := range riskCases {
		items = append(items, gin.H{
			"id":                      riskCase.Id,
			"record_type":             riskCase.RecordType,
			"trade_no":                riskCase.TradeNo,
			"user_id":                 riskCase.UserId,
			"username":                riskCase.Username,
			"display_name":            riskCase.DisplayName,
			"payment_method":          riskCase.PaymentMethod,
			"provider_payment_method": riskCase.ProviderPaymentMethod,
			"expected_amount":         riskCase.ExpectedAmount,
			"expected_money":          riskCase.ExpectedMoney,
			"received_money":          riskCase.ReceivedMoney,
			"currency":                riskCase.Currency,
			"source":                  riskCase.Source,
			"reason":                  riskCase.Reason,
			"order_status":            riskCase.OrderStatus,
			"status":                  riskCase.Status,
			"created_at":              riskCase.CreatedAt,
			"updated_at":              riskCase.UpdatedAt,
			"resolved_at":             riskCase.ResolvedAt,
			"handler_admin_id":        riskCase.HandlerAdminId,
			"handler_note":            riskCase.HandlerNote,
			"applied_quota_delta":     riskCase.AppliedQuotaDelta,
			"available_actions":       model.PaymentRiskAvailableActions(riskCase),
		})
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetPaymentRiskCaseDetail(c *gin.Context) {
	riskCaseID, _ := strconv.Atoi(c.Param("id"))
	if riskCaseID <= 0 {
		common.ApiErrorMsg(c, "无效的异常单 ID")
		return
	}

	riskCase, err := model.GetPaymentRiskCaseByID(riskCaseID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"risk_case":         riskCase,
		"available_actions": model.PaymentRiskAvailableActions(riskCase),
	})
}

func CreatePaymentRiskCase(c *gin.Context) {
	var req CreatePaymentRiskCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	riskCase, err := model.CreateManualPaymentRiskCase(req.RecordType, req.TradeNo, req.Note)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"risk_case":         riskCase,
		"available_actions": model.PaymentRiskAvailableActions(riskCase),
	})
}

func ResolvePaymentRiskCase(c *gin.Context) {
	riskCaseID, _ := strconv.Atoi(c.Param("id"))
	if riskCaseID <= 0 {
		common.ApiErrorMsg(c, "无效的异常单 ID")
		return
	}

	var req ResolvePaymentRiskCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Action == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if err := model.ResolvePaymentRiskCase(riskCaseID, c.GetInt("id"), req.Action, req.Note); err != nil {
		common.ApiError(c, err)
		return
	}
	updated, err := model.GetPaymentRiskCaseByID(riskCaseID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"risk_case":         updated,
		"available_actions": model.PaymentRiskAvailableActions(updated),
	})
}
