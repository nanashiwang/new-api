package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// profitBoardErrorMap maps sentinel errors from the model layer to i18n message keys.
var profitBoardErrorMap = map[error]string{
	// profit_board.go
	model.ErrProfitBoardNoChannel:                i18n.MsgProfitBoardNoChannel,
	model.ErrProfitBoardNoTag:                    i18n.MsgProfitBoardNoTag,
	model.ErrProfitBoardInvalidScopeType:         i18n.MsgProfitBoardInvalidScopeType,
	model.ErrProfitBoardRuleMustSpecifyModel:     i18n.MsgProfitBoardRuleMustSpecifyModel,
	model.ErrProfitBoardRuleOnlyOneDefault:       i18n.MsgProfitBoardRuleOnlyOneDefault,
	model.ErrProfitBoardRuleNonNegative:          i18n.MsgProfitBoardRuleNonNegative,
	model.ErrProfitBoardComboMissingId:           i18n.MsgProfitBoardComboMissingId,
	model.ErrProfitBoardComboDuplicate:           i18n.MsgProfitBoardComboDuplicate,
	model.ErrProfitBoardInvalidSitePricingMode:   i18n.MsgProfitBoardInvalidSitePricingMode,
	model.ErrProfitBoardComboSiteNonNegative:     i18n.MsgProfitBoardComboSiteNonNegative,
	model.ErrProfitBoardComboUpstreamNonNegative: i18n.MsgProfitBoardComboUpstreamNonNegative,
	model.ErrProfitBoardNoBatch:                  i18n.MsgProfitBoardNoBatch,
	model.ErrProfitBoardBatchDuplicate:           i18n.MsgProfitBoardBatchDuplicate,
	model.ErrProfitBoardPriceNonNegative:         i18n.MsgProfitBoardPriceNonNegative,
	model.ErrProfitBoardInvalidCostSource:        i18n.MsgProfitBoardInvalidCostSource,
	model.ErrProfitBoardInvalidUpstreamMode:      i18n.MsgProfitBoardInvalidUpstreamMode,
	model.ErrProfitBoardWalletRequireAccount:     i18n.MsgProfitBoardWalletRequireAccount,
	model.ErrProfitBoardInvalidSiteSource:        i18n.MsgProfitBoardInvalidSiteSource,
	model.ErrProfitBoardEndBeforeStart:           i18n.MsgProfitBoardEndBeforeStart,
	model.ErrProfitBoardCustomGranularityMin:     i18n.MsgProfitBoardCustomGranularityMin,
	model.ErrProfitBoardCustomGranularityMax:     i18n.MsgProfitBoardCustomGranularityMax,
	model.ErrProfitBoardInvalidGranularity:       i18n.MsgProfitBoardInvalidGranularity,
	model.ErrProfitBoardChannelNotExist:          i18n.MsgProfitBoardChannelNotExist,
	model.ErrProfitBoardTagNoChannel:             i18n.MsgProfitBoardTagNoChannel,
	model.ErrProfitBoardChannelDuplicateBatch:    i18n.MsgProfitBoardChannelDuplicateBatch,

	// profit_board_remote.go
	model.ErrProfitBoardRemoteMissingURL:   i18n.MsgProfitBoardRemoteMissingURL,
	model.ErrProfitBoardRemoteMissingUID:   i18n.MsgProfitBoardRemoteMissingUID,
	model.ErrProfitBoardRemoteMissingToken: i18n.MsgProfitBoardRemoteMissingToken,
	model.ErrProfitBoardRemoteTokenEmpty:   i18n.MsgProfitBoardRemoteTokenEmpty,
	model.ErrProfitBoardRemoteNotNewAPI:    i18n.MsgProfitBoardRemoteNotNewAPI,
	model.ErrProfitBoardRemoteURLInvalid:   i18n.MsgProfitBoardRemoteURLInvalid,
	model.ErrProfitBoardRemoteRequestFail:  i18n.MsgProfitBoardRemoteRequestFail,

	// profit_board_account.go
	model.ErrProfitBoardAccountTypeUnsupported: i18n.MsgProfitBoardAccountTypeUnsupported,
	model.ErrProfitBoardAccountNameEmpty:       i18n.MsgProfitBoardAccountNameEmpty,
	model.ErrProfitBoardAccountInvalid:         i18n.MsgProfitBoardAccountInvalid,
	model.ErrProfitBoardAccountTokenEmpty:      i18n.MsgProfitBoardAccountTokenEmpty,
}

// profitBoardApiError translates a sentinel error from the model layer into an
// i18n-aware API error response. If the error does not match any known sentinel,
// it falls back to returning the raw error message.
func profitBoardApiError(c *gin.Context, err error) {
	for sentinel, msgKey := range profitBoardErrorMap {
		if errors.Is(err, sentinel) {
			common.ApiErrorI18n(c, msgKey)
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

func GetProfitBoardConfig(c *gin.Context) {
	batches := make([]model.ProfitBoardBatch, 0)
	if raw := strings.TrimSpace(c.Query("batches")); raw != "" {
		if err := common.UnmarshalJsonStr(raw, &batches); err != nil {
			common.ApiErrorI18n(c, i18n.MsgProfitBoardBatchFormatError)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidParams)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidParams)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidParams)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidParams)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidParams)
		return
	}
	if rawID := strings.TrimSpace(c.Param("id")); rawID != "" {
		id, err := strconv.Atoi(rawID)
		if err != nil || id <= 0 {
			common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidUpstreamAccount)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidUpstreamAccount)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidUpstreamAccount)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidUpstreamAccount)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidParams)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidParams)
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
		common.ApiErrorI18n(c, i18n.MsgProfitBoardInvalidParams)
		return
	}
	activity, err := model.GetProfitBoardActivity(query)
	if err != nil {
		profitBoardApiError(c, err)
		return
	}
	common.ApiSuccess(c, activity)
}
