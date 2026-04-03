package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func parseProfitBoardIntList(raw string) []int {
	items := strings.Split(strings.TrimSpace(raw), ",")
	results := make([]int, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		value, err := strconv.Atoi(item)
		if err != nil || value <= 0 {
			continue
		}
		results = append(results, value)
	}
	return results
}

func parseProfitBoardStringList(raw string) []string {
	items := strings.Split(strings.TrimSpace(raw), ",")
	results := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			results = append(results, item)
		}
	}
	return results
}

func GetProfitBoardOptions(c *gin.Context) {
	options, err := model.GetProfitBoardOptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, options)
}

func GetProfitBoardConfig(c *gin.Context) {
	batches := make([]model.ProfitBoardBatch, 0)
	if raw := strings.TrimSpace(c.Query("batches")); raw != "" {
		if err := common.UnmarshalJsonStr(raw, &batches); err != nil {
			common.ApiErrorMsg(c, "批次参数格式错误")
			return
		}
	}
	selection := model.ProfitBoardSelection{
		ScopeType:  c.DefaultQuery("scope_type", model.ProfitBoardScopeChannel),
		ChannelIDs: parseProfitBoardIntList(c.Query("channel_ids")),
		Tags:       parseProfitBoardStringList(c.Query("tags")),
	}
	config, signature, err := model.GetProfitBoardConfig(batches, selection)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"signature": signature,
		"config":    config,
	})
}

func SaveProfitBoardConfig(c *gin.Context) {
	payload := model.ProfitBoardConfigPayload{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	config, signature, err := model.SaveProfitBoardConfig(payload)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"signature": signature,
		"config":    config,
	})
}

func GetProfitBoardOverview(c *gin.Context) {
	payload := model.ProfitBoardConfigPayload{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	report, err := model.GenerateProfitBoardOverview(payload)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, report)
}

func SyncProfitBoardRemote(c *gin.Context) {
	payload := model.ProfitBoardConfigPayload{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	states, err := model.SyncProfitBoardRemoteObservers(payload, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"states": states,
	})
}

func GetProfitBoardUpstreamAccounts(c *gin.Context) {
	accounts, err := model.GetProfitBoardUpstreamAccountOptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, accounts)
}

func SaveProfitBoardUpstreamAccount(c *gin.Context) {
	account := model.ProfitBoardUpstreamAccount{}
	if err := c.ShouldBindJSON(&account); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if rawID := strings.TrimSpace(c.Param("id")); rawID != "" {
		id, err := strconv.Atoi(rawID)
		if err != nil || id <= 0 {
			common.ApiErrorMsg(c, "无效的上游账户")
			return
		}
		account.Id = id
	}
	saved, err := model.SaveProfitBoardUpstreamAccount(account)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, saved)
}

func DeleteProfitBoardUpstreamAccount(c *gin.Context) {
	id, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的上游账户")
		return
	}
	if err := model.DeleteProfitBoardUpstreamAccount(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"deleted": true})
}

func SyncProfitBoardUpstreamAccount(c *gin.Context) {
	id, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的上游账户")
		return
	}
	account, syncErr := model.SyncProfitBoardUpstreamAccount(id, true)
	if syncErr != nil {
		common.ApiError(c, syncErr)
		return
	}
	common.ApiSuccess(c, account)
}

func QueryProfitBoard(c *gin.Context) {
	query := model.ProfitBoardQuery{}
	if err := c.ShouldBindJSON(&query); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	report, err := model.GenerateProfitBoardReport(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, report)
}

func QueryProfitBoardDetails(c *gin.Context) {
	query := model.ProfitBoardDetailQuery{}
	if err := c.ShouldBindJSON(&query); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	page, err := model.QueryProfitBoardDetails(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetProfitBoardActivity(c *gin.Context) {
	query := model.ProfitBoardQuery{}
	if err := c.ShouldBindJSON(&query); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	activity, err := model.GetProfitBoardActivity(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, activity)
}
