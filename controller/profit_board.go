package controller

import (
	"net/http"
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
	selection := model.ProfitBoardSelection{
		ScopeType:  c.DefaultQuery("scope_type", model.ProfitBoardScopeChannel),
		ChannelIDs: parseProfitBoardIntList(c.Query("channel_ids")),
		Tags:       parseProfitBoardStringList(c.Query("tags")),
	}
	config, signature, err := model.GetProfitBoardConfig(selection)
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

func ExportProfitBoardCSV(c *gin.Context) {
	query := model.ProfitBoardQuery{}
	if err := c.ShouldBindJSON(&query); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	data, filename, err := model.ExportProfitBoardCSV(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}
