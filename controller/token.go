package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

func GetAllTokens(c *gin.Context) {
	userId := c.GetInt("id")
	group := strings.TrimSpace(c.Query("group"))
	balanceMin, balanceMax, usedBalanceMin, usedBalanceMax, packageMode, sortBy, sortOrder, err := parseTokenListQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo := common.GetPageQuery(c)
	tokens, total, err := model.GetAllUserTokens(
		userId,
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
		group,
		balanceMin,
		balanceMax,
		usedBalanceMin,
		usedBalanceMax,
		packageMode,
		sortBy,
		sortOrder,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tokens)
	common.ApiSuccess(c, pageInfo)
	return
}

func SearchTokens(c *gin.Context) {
	userId := c.GetInt("id")
	keyword := c.Query("keyword")
	token := c.Query("token")
	group := strings.TrimSpace(c.Query("group"))
	balanceMin, balanceMax, usedBalanceMin, usedBalanceMax, packageMode, sortBy, sortOrder, err := parseTokenListQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo := common.GetPageQuery(c)

	tokens, total, err := model.SearchUserTokens(
		userId,
		keyword,
		token,
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
		group,
		balanceMin,
		balanceMax,
		usedBalanceMin,
		usedBalanceMax,
		packageMode,
		sortBy,
		sortOrder,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tokens)
	common.ApiSuccess(c, pageInfo)
	return
}

func parseTokenListQuery(c *gin.Context) (*int, *int, *int, *int, string, string, string, error) {
	// 统一解析令牌列表筛选：剩余余额区间 + 已使用余额区间 + 排序（白名单）。
	// 该解析同时给 /api/token 和 /api/token/search 复用，避免两处行为不一致。
	balanceMinRaw := strings.TrimSpace(c.Query("balance_min"))
	balanceMaxRaw := strings.TrimSpace(c.Query("balance_max"))
	usedBalanceMinRaw := strings.TrimSpace(c.Query("used_balance_min"))
	usedBalanceMaxRaw := strings.TrimSpace(c.Query("used_balance_max"))

	var balanceMin *int
	if balanceMinRaw != "" {
		value, err := strconv.Atoi(balanceMinRaw)
		if err != nil {
			return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 balance_min 不是有效整数")
		}
		if value < 0 {
			return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 balance_min 不能小于 0")
		}
		balanceMin = &value
	}

	var balanceMax *int
	if balanceMaxRaw != "" {
		value, err := strconv.Atoi(balanceMaxRaw)
		if err != nil {
			return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 balance_max 不是有效整数")
		}
		if value < 0 {
			return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 balance_max 不能小于 0")
		}
		balanceMax = &value
	}

	if balanceMin != nil && balanceMax != nil && *balanceMin > *balanceMax {
		return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 balance_min 不能大于 balance_max")
	}

	var usedBalanceMin *int
	if usedBalanceMinRaw != "" {
		value, err := strconv.Atoi(usedBalanceMinRaw)
		if err != nil {
			return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 used_balance_min 不是有效整数")
		}
		if value < 0 {
			return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 used_balance_min 不能小于 0")
		}
		usedBalanceMin = &value
	}

	var usedBalanceMax *int
	if usedBalanceMaxRaw != "" {
		value, err := strconv.Atoi(usedBalanceMaxRaw)
		if err != nil {
			return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 used_balance_max 不是有效整数")
		}
		if value < 0 {
			return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 used_balance_max 不能小于 0")
		}
		usedBalanceMax = &value
	}

	if usedBalanceMin != nil && usedBalanceMax != nil && *usedBalanceMin > *usedBalanceMax {
		return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 used_balance_min 不能大于 used_balance_max")
	}

	packageMode := strings.ToLower(strings.TrimSpace(c.Query("package_mode")))
	switch packageMode {
	case "", "package", "standard":
	default:
		return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 package_mode 仅支持 package 或 standard")
	}

	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by")))
	if sortBy == "" {
		sortBy = "id"
	}
	switch sortBy {
	case "id", "remain_quota":
	default:
		return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 sort_by 仅支持 id 或 remain_quota")
	}

	sortOrder := strings.ToLower(strings.TrimSpace(c.Query("sort_order")))
	if sortOrder == "" {
		sortOrder = "desc"
	}
	switch sortOrder {
	case "asc", "desc":
	default:
		return nil, nil, nil, nil, "", "", "", fmt.Errorf("参数 sort_order 仅支持 asc 或 desc")
	}

	return balanceMin, balanceMax, usedBalanceMin, usedBalanceMax, packageMode, sortBy, sortOrder, nil
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    token,
	})
	return
}

func GetTokenStatus(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	userId := c.GetInt("id")
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"object":          "credit_summary",
		"total_granted":   token.RemainQuota,
		"total_used":      0, // not supported currently
		"total_available": token.RemainQuota,
		"expires_at":      expiredAt * 1000,
	})
}

func GetTokenUsage(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "No Authorization header",
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Invalid Bearer token",
		})
		return
	}
	tokenKey := parts[1]

	token, err := model.GetTokenByKey(strings.TrimPrefix(tokenKey, "sk-"), false)
	if err != nil {
		common.SysError("failed to get token by key: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgTokenGetInfoFailed)
		return
	}

	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    true,
		"message": "ok",
		"data": gin.H{
			"object":                  "token_usage",
			"name":                    token.Name,
			"total_granted":           token.RemainQuota + token.UsedQuota,
			"total_used":              token.UsedQuota,
			"total_available":         token.RemainQuota,
			"unlimited_quota":         token.UnlimitedQuota,
			"model_limits":            token.GetModelLimitsMap(),
			"model_limits_enabled":    token.ModelLimitsEnabled,
			"package_enabled":         token.PackageEnabled,
			"package_limit_quota":     token.PackageLimitQuota,
			"package_period":          token.PackagePeriod,
			"package_custom_seconds":  token.PackageCustomSeconds,
			"package_period_mode":     token.PackagePeriodMode,
			"package_used_quota":      token.PackageUsedQuota,
			"package_next_reset_time": token.PackageNextResetTime,
			"expires_at":              expiredAt,
		},
	})
}

func AddToken(c *gin.Context) {
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	if err := model.ValidateTokenPackageConfig(&token); err != nil {
		common.ApiError(c, err)
		return
	}
	// 非无限额度时，检查额度值是否超出有效范围
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	}
	// 检查用户令牌数量是否已达上限
	maxTokens := operation_setting.GetMaxUserTokens()
	count, err := model.CountUserTokens(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if int(count) >= maxTokens {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("已达到最大令牌数量限制 (%d)", maxTokens),
		})
		return
	}
	key, err := common.GenerateKey()
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTokenGenerateFailed)
		common.SysLog("failed to generate token key: " + err.Error())
		return
	}
	cleanToken := model.Token{
		UserId:               c.GetInt("id"),
		Name:                 token.Name,
		Key:                  key,
		CreatedTime:          common.GetTimestamp(),
		AccessedTime:         common.GetTimestamp(),
		ExpiredTime:          token.ExpiredTime,
		RemainQuota:          token.RemainQuota,
		UnlimitedQuota:       token.UnlimitedQuota,
		ModelLimitsEnabled:   token.ModelLimitsEnabled,
		ModelLimits:          token.ModelLimits,
		AllowIps:             token.AllowIps,
		Group:                token.Group,
		CrossGroupRetry:      token.CrossGroupRetry,
		PackageEnabled:       token.PackageEnabled,
		PackageLimitQuota:    token.PackageLimitQuota,
		PackagePeriod:        token.PackagePeriod,
		PackageCustomSeconds: token.PackageCustomSeconds,
		PackagePeriodMode:    token.PackagePeriodMode,
		PackageUsedQuota:     token.PackageUsedQuota,
		PackageNextResetTime: token.PackageNextResetTime,
	}
	if err := model.ValidateTokenQuotaPackageRelation(&cleanToken); err != nil {
		common.ApiError(c, err)
		return
	}
	err = cleanToken.Insert()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	err := model.DeleteTokenById(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateToken(c *gin.Context) {
	userId := c.GetInt("id")
	statusOnly := c.Query("status_only")
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	if err := model.ValidateTokenPackageConfig(&token); err != nil {
		common.ApiError(c, err)
		return
	}
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	}
	cleanToken, err := model.GetTokenByIds(token.Id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if token.Status == common.TokenStatusEnabled {
		if cleanToken.Status == common.TokenStatusExpired && cleanToken.ExpiredTime <= common.GetTimestamp() && cleanToken.ExpiredTime != -1 {
			common.ApiErrorI18n(c, i18n.MsgTokenExpiredCannotEnable)
			return
		}
		if cleanToken.Status == common.TokenStatusExhausted && cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
			common.ApiErrorI18n(c, i18n.MsgTokenExhaustedCannotEable)
			return
		}
	}
	if statusOnly != "" {
		cleanToken.Status = token.Status
	} else {
		// If you add more fields, please also update token.Update()
		cleanToken.Name = token.Name
		cleanToken.ExpiredTime = token.ExpiredTime
		cleanToken.RemainQuota = token.RemainQuota
		cleanToken.UnlimitedQuota = token.UnlimitedQuota
		cleanToken.ModelLimitsEnabled = token.ModelLimitsEnabled
		cleanToken.ModelLimits = token.ModelLimits
		cleanToken.AllowIps = token.AllowIps
		cleanToken.Group = token.Group
		cleanToken.CrossGroupRetry = token.CrossGroupRetry
		cleanToken.PackageEnabled = token.PackageEnabled
		cleanToken.PackageLimitQuota = token.PackageLimitQuota
		cleanToken.PackagePeriod = token.PackagePeriod
		cleanToken.PackageCustomSeconds = token.PackageCustomSeconds
		cleanToken.PackagePeriodMode = token.PackagePeriodMode
		// 允许管理员/用户在编辑时手动归零该周期已用额度，便于紧急解锁
		cleanToken.PackageUsedQuota = token.PackageUsedQuota
		cleanToken.PackageNextResetTime = token.PackageNextResetTime
		if err := model.ValidateTokenQuotaPackageRelation(cleanToken); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	err = cleanToken.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanToken,
	})
}

type TokenBatch struct {
	Ids []int `json:"ids"`
}

type TokenBatchManageRequest struct {
	Ids    []int  `json:"ids"`
	Action string `json:"action"`
}

func DeleteTokenBatch(c *gin.Context) {
	tokenBatch := TokenBatch{}
	if err := c.ShouldBindJSON(&tokenBatch); err != nil || len(tokenBatch.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	userId := c.GetInt("id")
	count, err := model.BatchDeleteTokens(tokenBatch.Ids, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
}

// ManageTokenBatch 统一处理令牌批量管理动作。
//
// 设计要点：
// 1. 一个接口同时支持 enable / disable / delete，便于前端复用；
// 2. 按“部分成功”模型执行：单条失败不影响其他令牌；
// 3. 返回成功数、失败数、失败明细，便于前端给出明确反馈；
// 4. 严格限定只能操作当前用户自己的令牌（通过 userId + tokenId 查询约束）。
func ManageTokenBatch(c *gin.Context) {
	req := TokenBatchManageRequest{}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "enable", "disable", "delete":
	default:
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	userId := c.GetInt("id")
	seen := make(map[int]struct{}, len(req.Ids))
	updated := make([]gin.H, 0, len(req.Ids))
	failed := make([]gin.H, 0)

	for _, id := range req.Ids {
		// 输入兜底：非法 ID 直接记失败，避免无效查询。
		if id <= 0 {
			failed = append(failed, gin.H{
				"id":      id,
				"message": "令牌 ID 无效",
			})
			continue
		}
		// 同一请求内去重，确保一个令牌最多处理一次。
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		// 只能读取到“当前用户自己的令牌”，从根源避免跨用户越权操作。
		token, err := model.GetTokenByIds(id, userId)
		if err != nil || token == nil || token.Id == 0 {
			failed = append(failed, gin.H{
				"id":      id,
				"message": "令牌不存在",
			})
			continue
		}

		switch action {
		case "enable":
			// 规则与单条编辑保持一致：已过期令牌不允许重新启用。
			if token.Status == common.TokenStatusExpired && token.ExpiredTime <= common.GetTimestamp() && token.ExpiredTime != -1 {
				failed = append(failed, gin.H{
					"id":      id,
					"message": "已过期令牌不可启用",
				})
				continue
			}
			// 规则与单条编辑保持一致：已耗尽且非无限额令牌不可启用。
			if token.Status == common.TokenStatusExhausted && token.RemainQuota <= 0 && !token.UnlimitedQuota {
				failed = append(failed, gin.H{
					"id":      id,
					"message": "已耗尽令牌不可启用",
				})
				continue
			}
			token.Status = common.TokenStatusEnabled
			if err := token.Update(); err != nil {
				failed = append(failed, gin.H{
					"id":      id,
					"message": err.Error(),
				})
				continue
			}
		case "disable":
			token.Status = common.TokenStatusDisabled
			if err := token.Update(); err != nil {
				failed = append(failed, gin.H{
					"id":      id,
					"message": err.Error(),
				})
				continue
			}
		case "delete":
			if err := token.Delete(); err != nil {
				failed = append(failed, gin.H{
					"id":      id,
					"message": err.Error(),
				})
				continue
			}
		}

		// 成功项返回最小必要字段，便于前端做局部刷新或统计。
		updated = append(updated, gin.H{
			"id":     token.Id,
			"status": token.Status,
		})
	}

	common.ApiSuccess(c, gin.H{
		"success_count": len(updated),
		"failed_count":  len(failed),
		"updated":       updated,
		"failed":        failed,
	})
}
