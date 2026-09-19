package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func parseAdminLogScope(c *gin.Context) (model.LogScope, error) {
	scope := model.LogScope{Vendor: strings.TrimSpace(c.Query("group_vendor"))}
	if len(scope.Vendor) > 128 {
		return scope, errors.New("厂商名称过长")
	}
	raw, supplied := c.GetQuery("channel_ids")
	if !supplied || strings.TrimSpace(raw) == "" {
		return scope, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) > 200 {
		return scope, errors.New("最多选择 200 个渠道")
	}
	scope.ChannelIDs = []int{}
	seen := map[int]bool{}
	for _, part := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			return scope, errors.New("渠道 ID 必须是正整数")
		}
		if !seen[id] {
			scope.ChannelIDs = append(scope.ChannelIDs, id)
			seen[id] = true
		}
	}
	return scope, nil
}

func GetLogChannelOptions(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if len(keyword) > 128 {
		common.ApiError(c, errors.New("搜索关键词过长"))
		return
	}
	rows, err := model.FindLogChannels(keyword)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	truncated := len(rows) > 200
	if truncated {
		rows = rows[:200]
	}
	common.ApiSuccess(c, gin.H{"items": rows, "truncated": truncated})
}

func GetLogGroupSummary(c *gin.Context) {
	scope, err := parseAdminLogScope(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	start, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channel, _ := strconv.Atoi(c.Query("channel"))
	rows, err := model.GetLogGroupSummary(model.AdminLogQueryFilters{
		Scope: scope, StartTimestamp: start, EndTimestamp: end,
		ModelName: c.Query("model_name"), Username: c.Query("username"),
		TokenName: c.Query("token_name"), Channel: channel,
		Group: c.Query("group"), RequestID: c.Query("request_id"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rows)
}
