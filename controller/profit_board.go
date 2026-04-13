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

// profitBoardErrorMap maps sentinel errors from the model layer to readable
// Chinese API messages.
var profitBoardErrorMap = map[error]string{
	model.ErrProfitBoardNoChannel:                 "请至少选择一个渠道",
	model.ErrProfitBoardNoTag:                     "请至少选择一个标签",
	model.ErrProfitBoardInvalidScopeType:          "无效的收益看板选择类型",
	model.ErrProfitBoardRuleMustSpecifyModel:      "手动价格规则必须指定模型，或者标记为默认规则",
	model.ErrProfitBoardRuleOnlyOneDefault:        "手动价格规则只允许存在一条默认规则",
	model.ErrProfitBoardRuleNonNegative:           "手动价格规则必须是非负数字",
	model.ErrProfitBoardComboMissingId:            "组合价格配置缺少组合标识",
	model.ErrProfitBoardComboDuplicate:            "组合价格配置重复，请刷新页面后重试",
	model.ErrProfitBoardInvalidSitePricingMode:    "无效的本站价格模式",
	model.ErrProfitBoardComboSiteNonNegative:      "组合固定本站收入必须是非负数字",
	model.ErrProfitBoardComboUpstreamNonNegative:  "组合固定上游费用必须是非负数字",
	model.ErrProfitBoardComboSiteExchangeRate:     "组合收入汇率必须大于 0",
	model.ErrProfitBoardComboUpstreamExchangeRate: "组合成本汇率必须大于 0",
	model.ErrProfitBoardNoBatch:                   "请至少添加一个批次",
	model.ErrProfitBoardBatchDuplicate:            "批次标识重复，请删除后重新添加批次",
	model.ErrProfitBoardPriceNonNegative:          "价格配置必须是非负数字",
	model.ErrProfitBoardInvalidCostSource:         "无效的上游费用来源配置",
	model.ErrProfitBoardInvalidUpstreamMode:       "无效的上游成本模式",
	model.ErrProfitBoardWalletRequireAccount:      "钱包扣减模式必须绑定上游账户",
	model.ErrProfitBoardInvalidSiteSource:         "无效的本站价格来源配置",
	model.ErrProfitBoardEndBeforeStart:            "结束时间不能早于开始时间",
	model.ErrProfitBoardCustomGranularityMin:      "自定义时间粒度必须大于 0 分钟",
	model.ErrProfitBoardCustomGranularityMax:      "自定义时间粒度不能超过 43200 分钟",
	model.ErrProfitBoardInvalidGranularity:        "无效的时间粒度",
	model.ErrProfitBoardChannelNotExist:           "所选渠道不存在",
	model.ErrProfitBoardTagNoChannel:              "所选标签下没有渠道",
	model.ErrProfitBoardChannelDuplicateBatch:     "组合中存在重复渠道，请调整后再统计",
	model.ErrProfitBoardRemoteMissingURL:          "远端额度观测已启用，但缺少远端地址",
	model.ErrProfitBoardRemoteMissingUID:          "远端额度观测已启用，但缺少远端用户 ID",
	model.ErrProfitBoardRemoteMissingToken:        "远端额度观测已启用，但缺少远端 access token",
	model.ErrProfitBoardRemoteTokenEmpty:          "远端 access token 为空",
	model.ErrProfitBoardRemoteNotNewAPI:           "远端不是受支持的 new-api 实例",
	model.ErrProfitBoardRemoteURLInvalid:          "远端额度地址不可用",
	model.ErrProfitBoardRemoteRequestFail:         "远端请求失败",
	model.ErrProfitBoardAccountTypeUnsupported:    "当前仅支持 new-api 上游账户",
	model.ErrProfitBoardAccountNameEmpty:          "上游账户名称不能为空",
	model.ErrProfitBoardAccountInvalid:            "无效的上游账户",
	model.ErrProfitBoardAccountTokenEmpty:         "上游 access token 不能为空",
	model.ErrProfitBoardAccountInUse:              "该上游账户仍被收益看板组合使用，请先改掉组合绑定后再删除",
}

func translateDuplicateBatchError(err error) string {
	raw := strings.TrimPrefix(
		err.Error(),
		model.ErrProfitBoardChannelDuplicateBatch.Error()+": ",
	)
	parts := strings.SplitN(raw, " -> ", 2)
	if len(parts) != 2 {
		return profitBoardErrorMap[model.ErrProfitBoardChannelDuplicateBatch]
	}
	owners := strings.SplitN(parts[1], ", ", 2)
	if len(owners) != 2 {
		return profitBoardErrorMap[model.ErrProfitBoardChannelDuplicateBatch]
	}
	return fmt.Sprintf(
		"%s 同时出现在组合\"%s\"和\"%s\"中，请拆开后再统计",
		parts[0],
		owners[0],
		owners[1],
	)
}

func profitBoardApiError(c *gin.Context, err error) {
	for sentinel, message := range profitBoardErrorMap {
		if errors.Is(err, sentinel) {
			if sentinel == model.ErrProfitBoardChannelDuplicateBatch {
				common.ApiErrorMsg(c, translateDuplicateBatchError(err))
				return
			}
			common.ApiErrorMsg(c, message)
			return
		}
	}
	common.ApiError(c, err)
}

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
		profitBoardApiError(c, err)
		return
	}
	common.ApiSuccess(c, options)
}

func GetLatestProfitBoardConfig(c *gin.Context) {
	config, signature, err := model.GetLatestProfitBoardConfig()
	if err != nil {
		profitBoardApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"signature": signature,
		"config":    config,
	})
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
		profitBoardApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"signature": signature,
		"config":    config,
	})
}

func LookupProfitBoardConfig(c *gin.Context) {
	var req struct {
		Batches   []model.ProfitBoardBatch   `json:"batches"`
		Selection model.ProfitBoardSelection `json:"selection"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.Selection.ScopeType == "" {
		req.Selection.ScopeType = model.ProfitBoardScopeChannel
	}
	config, signature, err := model.GetProfitBoardConfig(req.Batches, req.Selection)
	if err != nil {
		profitBoardApiError(c, err)
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
		profitBoardApiError(c, err)
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
		profitBoardApiError(c, err)
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
		profitBoardApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"states": states,
	})
}

func GetProfitBoardUpstreamAccounts(c *gin.Context) {
	accounts, err := model.GetProfitBoardUpstreamAccountOptions()
	if err != nil {
		profitBoardApiError(c, err)
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
		profitBoardApiError(c, err)
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
		profitBoardApiError(c, err)
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
		profitBoardApiError(c, syncErr)
		return
	}
	common.ApiSuccess(c, account)
}

func SyncAllProfitBoardUpstreamAccounts(c *gin.Context) {
	accounts, err := model.SyncAllProfitBoardUpstreamAccounts(true)
	if err != nil {
		profitBoardApiError(c, err)
		return
	}
	common.ApiSuccess(c, accounts)
}

func GetProfitBoardUpstreamAccountTrend(c *gin.Context) {
	id, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的上游账户")
		return
	}
	startTimestamp, _ := strconv.ParseInt(strings.TrimSpace(c.Query("start_timestamp")), 10, 64)
	endTimestamp, _ := strconv.ParseInt(strings.TrimSpace(c.Query("end_timestamp")), 10, 64)
	customIntervalMinutes, _ := strconv.Atoi(strings.TrimSpace(c.Query("custom_interval_minutes")))
	trend, trendErr := model.GetProfitBoardUpstreamAccountTrend(
		id,
		startTimestamp,
		endTimestamp,
		c.DefaultQuery("granularity", "day"),
		customIntervalMinutes,
	)
	if trendErr != nil {
		profitBoardApiError(c, trendErr)
		return
	}
	common.ApiSuccess(c, trend)
}

func QueryProfitBoard(c *gin.Context) {
	query := model.ProfitBoardQuery{}
	if err := c.ShouldBindJSON(&query); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	report, err := model.GenerateProfitBoardReport(query)
	if err != nil {
		profitBoardApiError(c, err)
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
		profitBoardApiError(c, err)
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
		profitBoardApiError(c, err)
		return
	}
	common.ApiSuccess(c, activity)
}
