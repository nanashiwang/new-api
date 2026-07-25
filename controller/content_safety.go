package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type contentSafetyReviewResolutionRequest struct {
	Resolution string `json:"resolution"`
	Note       string `json:"note"`
}

func GetSelfContentSafetyState(c *gin.Context) {
	state, err := model.GetUserContentSafetyState(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data := gin.H{
		"level": state.Level, "window_count": state.WindowCount,
		"burst_count": state.BurstCount, "cooldown_count": state.CooldownCount,
		"cooldown_until": state.CooldownUntil, "has_unread_warning": state.HasUnreadWarning,
	}
	if latest := state.LatestViolation; latest != nil {
		data["latest_violation"] = gin.H{
			"error_type": latest.ErrorType, "error_code": latest.ErrorCode,
			"official_message": latest.OfficialMessage, "fine_category": latest.FineCategory,
			"reason_source": latest.ReasonSource, "reason_confidence": latest.ReasonConfidence,
			"reason_summary": latest.ReasonSummary, "classifier_version": latest.ClassifierVersion,
			"model_name": latest.ModelName, "is_stream": latest.IsStream,
			"created_at": latest.CreatedAt, "window_count": latest.WindowCount,
			"burst_count": latest.BurstCount, "cooldown_until": latest.CooldownUntil,
			"action": latest.Action,
		}
	}
	common.ApiSuccess(c, data)
}

func AcknowledgeSelfContentSafetyWarnings(c *gin.Context) {
	if err := model.AcknowledgeContentSafetyWarnings(c.GetInt("id"), time.Now().Unix()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"acknowledged": true})
}

func GetContentSafetyViolations(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userID, _ := strconv.Atoi(c.Query("user_id"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if startTimestamp == 0 {
		startTimestamp = time.Now().Add(-service.ContentSafetyPolicyWindow).Unix()
	}

	items, total, err := model.GetContentSafetyViolations(model.ContentSafetyViolationQuery{
		UserId:         userID,
		Username:       c.Query("username"),
		ErrorCode:      c.Query("error_code"),
		Action:         c.Query("action"),
		RequestId:      c.Query("request_id"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetContentSafetyReviewCases(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userID, _ := strconv.Atoi(c.Query("user_id"))
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if status != "" && status != model.ContentSafetyReviewPending && status != model.ContentSafetyReviewApprovedDisable &&
		status != model.ContentSafetyReviewDismissed && status != model.ContentSafetyReviewObserving {
		common.ApiError(c, errors.New("参数 status 无效"))
		return
	}
	items, total, err := model.GetContentSafetyReviewCases(model.ContentSafetyReviewCaseQuery{UserId: userID, Status: status}, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func ResolveContentSafetyReviewCase(c *gin.Context) {
	caseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || caseID <= 0 {
		common.ApiError(c, errors.New("无效的审核单 ID"))
		return
	}
	var request contentSafetyReviewResolutionRequest
	if err = common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return
	}
	resolved, userDisabled, err := model.ResolveContentSafetyReviewCase(caseID, c.GetInt("id"), request.Resolution, request.Note)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if userDisabled {
		if cacheErr := model.InvalidateUserAndTokenCaches(resolved.UserId); cacheErr != nil {
			common.SysError(fmt.Sprintf("content safety review cache invalidation failed: user_id=%d err=%v", resolved.UserId, cacheErr))
		}
		model.RecordLog(resolved.UserId, model.LogTypeSystem, "管理员审核了内容安全记录并决定停用账户。如需申诉，请联系管理员。")
	} else {
		model.RecordLog(resolved.UserId, model.LogTypeSystem, "管理员已完成人工内容安全复核，账户未被永久停用。")
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": resolved})
}
