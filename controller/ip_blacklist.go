package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type createIPBlacklistRequest struct {
	IP               string `json:"ip"`
	Reason           string `json:"reason"`
	SourceUserID     int    `json:"source_user_id"`
	ConfirmCurrentIP bool   `json:"confirm_current_ip"`
}

type batchCreateIPBlacklistRequest struct {
	UserIDs          []int  `json:"user_ids"`
	Reason           string `json:"reason"`
	ConfirmCurrentIP bool   `json:"confirm_current_ip"`
}

func GetIPBlacklists(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.SearchIPBlacklists(c.Query("keyword"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func CreateIPBlacklist(c *gin.Context) {
	req := createIPBlacklistRequest{}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	req.IP = strings.TrimSpace(req.IP)
	if req.IP == "" {
		common.ApiError(c, model.ErrInvalidIPBlacklistRule)
		return
	}
	if !confirmCurrentIPIfNeeded(c, req.IP, req.ConfirmCurrentIP) {
		return
	}

	item, created, err := model.CreateIPBlacklist(req.IP, req.Reason, req.SourceUserID, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if created {
		recordIPBlacklistManageLog(c, item.SourceUserId, "添加 IP 黑名单", gin.H{
			"rule_id":        item.Id,
			"ip":             item.IP,
			"cidr":           item.CIDR,
			"ip_version":     item.IPVersion,
			"source_user_id": item.SourceUserId,
			"reason":         item.Reason,
		})
	}
	common.ApiSuccess(c, gin.H{
		"item":    item,
		"created": created,
	})
}

func BatchCreateIPBlacklist(c *gin.Context) {
	req := batchCreateIPBlacklistRequest{}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil || len(req.UserIDs) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if !req.ConfirmCurrentIP {
		match, err := model.FindUserRegisterIPMatch(req.UserIDs, c.ClientIP())
		if err != nil && !errors.Is(err, model.ErrInvalidIPBlacklistRule) {
			common.ApiError(c, err)
			return
		}
		if match != "" {
			writeCurrentIPConfirmationRequired(c, match)
			return
		}
	}

	result, err := model.BatchCreateIPBlacklistFromUsers(req.UserIDs, req.Reason, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if result.CreatedCount > 0 || result.ExistingCount > 0 || result.SkippedCount > 0 || result.FailedCount > 0 {
		recordIPBlacklistManageLog(c, c.GetInt("id"), "批量拉黑注册 IP", gin.H{
			"user_ids":       req.UserIDs,
			"created_count":  result.CreatedCount,
			"existing_count": result.ExistingCount,
			"skipped_count":  result.SkippedCount,
			"failed_count":   result.FailedCount,
			"failed":         result.Failed,
			"rule_ids":       collectIPBlacklistIDs(result.Items),
		})
	}
	common.ApiSuccess(c, result)
}

func DeleteIPBlacklist(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, model.ErrIPBlacklistNotFound)
		return
	}
	item, err := model.GetIPBlacklistByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteIPBlacklistByID(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordIPBlacklistManageLog(c, item.SourceUserId, "解除 IP 拉黑", gin.H{
		"rule_id":        item.Id,
		"ip":             item.IP,
		"cidr":           item.CIDR,
		"ip_version":     item.IPVersion,
		"source_user_id": item.SourceUserId,
		"reason":         item.Reason,
	})
	common.ApiSuccess(c, nil)
}

func confirmCurrentIPIfNeeded(c *gin.Context, rule string, confirmed bool) bool {
	match, err := model.DoesIPBlacklistRuleMatchIP(rule, c.ClientIP())
	if err != nil {
		if errors.Is(err, model.ErrInvalidIPBlacklistRule) {
			common.ApiError(c, err)
			return false
		}
		common.ApiError(c, err)
		return false
	}
	if match && !confirmed {
		writeCurrentIPConfirmationRequired(c, rule)
		return false
	}
	return true
}

func writeCurrentIPConfirmationRequired(c *gin.Context, rule string) {
	c.JSON(200, gin.H{
		"success": false,
		"message": "该规则会拉黑你当前访问 IP，请确认后再提交",
		"data": gin.H{
			"code":      "current_ip_requires_confirmation",
			"client_ip": c.ClientIP(),
			"rule":      rule,
		},
	})
}

func recordIPBlacklistManageLog(c *gin.Context, targetUserID int, action string, extra gin.H) {
	if model.LOG_DB == nil {
		return
	}
	adminID := c.GetInt("id")
	if targetUserID <= 0 {
		targetUserID = adminID
	}
	adminInfo := map[string]interface{}{
		"admin_id":  adminID,
		"client_ip": c.ClientIP(),
		"action":    action,
	}
	for key, value := range extra {
		adminInfo[key] = value
	}
	content := fmt.Sprintf("管理员(ID:%d) %s", adminID, action)
	if cidr, ok := extra["cidr"].(string); ok && cidr != "" {
		content = fmt.Sprintf("%s: %s", content, cidr)
	}
	model.RecordLogWithAdminInfo(targetUserID, model.LogTypeManage, content, adminInfo)
}

func collectIPBlacklistIDs(items []*model.IPBlacklist) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		if item != nil && item.Id > 0 {
			ids = append(ids, item.Id)
		}
	}
	return ids
}
