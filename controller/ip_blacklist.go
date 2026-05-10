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

type manageIPBlacklistUsersRequest struct {
	Action  string `json:"action"`
	Scope   string `json:"scope"`
	UserIDs []int  `json:"user_ids"`
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

func GetIPBlacklistUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, model.ErrIPBlacklistNotFound)
		return
	}
	pageInfo := common.GetPageQuery(c)
	users, total, err := model.SearchUsersByIPBlacklistID(id, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(users)
	common.ApiSuccess(c, pageInfo)
}

func ManageIPBlacklistUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, model.ErrIPBlacklistNotFound)
		return
	}
	req := manageIPBlacklistUsersRequest{}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "disable" && action != "delete" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope == "" {
		scope = "selected"
	}
	if scope != "selected" && scope != "all" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	matchedIDs, err := model.GetUserIDsByIPBlacklistID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	allowedIDs := make(map[int]struct{}, len(matchedIDs))
	for _, matchedID := range matchedIDs {
		allowedIDs[matchedID] = struct{}{}
	}

	targetIDs := matchedIDs
	if scope == "selected" {
		targetIDs = deduplicatePositiveIDs(req.UserIDs)
		if len(targetIDs) == 0 {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
	}

	result := manageMatchedUsers(c, targetIDs, allowedIDs, action)
	recordIPBlacklistManageLog(c, c.GetInt("id"), "管理 IP 黑名单命中用户", gin.H{
		"rule_id":       id,
		"manage_action": action,
		"scope":         scope,
		"target_ids":    targetIDs,
		"success_count": result["success_count"],
		"failed_count":  result["failed_count"],
	})
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

func deduplicatePositiveIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func manageMatchedUsers(c *gin.Context, targetIDs []int, allowedIDs map[int]struct{}, action string) gin.H {
	myRole := c.GetInt("role")
	updatedUsers := make([]*model.User, 0, len(targetIDs))
	failed := make([]gin.H, 0)

	for _, id := range deduplicatePositiveIDs(targetIDs) {
		if _, ok := allowedIDs[id]; !ok {
			failed = append(failed, gin.H{
				"id":      id,
				"message": "用户不属于该 IP 黑名单规则",
			})
			continue
		}

		user := model.User{Id: id}
		model.DB.Unscoped().Where(&user).First(&user)
		if user.Id == 0 {
			failed = append(failed, gin.H{
				"id":      id,
				"message": "用户不存在",
			})
			continue
		}
		if user.DeletedAt.Valid {
			failed = append(failed, gin.H{
				"id":      id,
				"message": "用户已注销",
			})
			continue
		}
		if myRole <= user.Role && myRole != common.RoleRootUser {
			failed = append(failed, gin.H{
				"id":      id,
				"message": "权限不足",
			})
			continue
		}
		if err := applyManageAction(&user, action, myRole); err != nil {
			failed = append(failed, gin.H{
				"id":      id,
				"message": err.Error(),
			})
			continue
		}
		if shouldInvalidateManagedUserCaches(action) {
			if err := invalidateManagedUserCaches(user.Id); err != nil {
				failed = append(failed, gin.H{
					"id":      id,
					"message": err.Error(),
				})
				continue
			}
		}
		updatedUsers = append(updatedUsers, &user)
	}

	updated := make([]gin.H, 0, len(updatedUsers))
	for _, user := range updatedUsers {
		updated = append(updated, gin.H{
			"id":     user.Id,
			"role":   user.Role,
			"status": user.Status,
		})
	}
	return gin.H{
		"success_count": len(updated),
		"failed_count":  len(failed),
		"updated":       updated,
		"failed":        failed,
	}
}
