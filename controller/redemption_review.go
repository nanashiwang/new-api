package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetRedemptionReviewCases(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListRedemptionReviewCases(c.Query("status"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func ResolveRedemptionReviewCase(c *gin.Context) {
	caseID, err := parsePositivePathID(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	action := strings.ToLower(strings.TrimSpace(request.Action))
	switch action {
	case model.RedemptionReviewActionDismiss:
	case model.RedemptionReviewActionDisable:
		if strings.TrimSpace(request.Note) == "" {
			common.ApiErrorMsg(c, "封禁用户必须填写人工复查依据")
			return
		}
	default:
		common.ApiErrorMsg(c, "无效的人工复查操作")
		return
	}

	resolved, disabled, err := model.ResolveRedemptionReviewCaseAction(caseID, c.GetInt("id"), c.GetInt("role"), action, request.Note)
	if err != nil {
		if errors.Is(err, model.ErrRedemptionReviewResolved) {
			common.ApiErrorMsg(c, "该复查记录已处理")
			return
		}
		common.ApiError(c, err)
		return
	}
	if disabled {
		model.RecordLogWithAdminInfo(resolved.UserId, model.LogTypeManage,
			fmt.Sprintf("管理员因多来源小额兑换码人工复查停用账户：%s", strings.TrimSpace(request.Note)),
			map[string]interface{}{"admin_id": c.GetInt("id"), "admin_username": c.GetString("username"), "redemption_review_case_id": caseID},
		)
	}
	common.ApiSuccess(c, resolved)
}

func parsePositivePathID(raw string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || id <= 0 {
		return 0, errors.New("无效的记录 ID")
	}
	return id, nil
}
