package controller

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type sellableTokenProductRequest struct {
	Product model.SellableTokenProduct `json:"product"`
}

type sellableTokenProductStatusRequest struct {
	Status int `json:"status"`
}

type sellableTokenPurchaseRequest struct {
	ProductId int `json:"product_id"`
}

type sellableTokenIssueConfirmRequest struct {
	Mode          string `json:"mode"`
	TargetTokenId int    `json:"target_token_id"`
	Name          string `json:"name"`
	Group         string `json:"group"`
}

type sellableTokenAdminIssueRequest struct {
	ProductId int `json:"product_id"`
	sellableTokenIssueConfirmRequest
}

type sellableTokenSummaryItem struct {
	Id                 int    `json:"id"`
	Name               string `json:"name"`
	Group              string `json:"group"`
	Status             int    `json:"status"`
	ProductId          int    `json:"product_id"`
	RemainQuota        int    `json:"remain_quota"`
	UsedQuota          int    `json:"used_quota"`
	MaxConcurrency     int    `json:"max_concurrency"`
	WindowRequestLimit int    `json:"window_request_limit"`
	WindowSeconds      int64  `json:"window_seconds"`
	PackageEnabled     bool   `json:"package_enabled"`
	PackageLimitQuota  int    `json:"package_limit_quota"`
	PackagePeriod      string `json:"package_period"`
	ExpiredTime        int64  `json:"expired_time"`
	CreatedTime        int64  `json:"created_time"`
}

type sellableTokenIssuanceSummaryItem struct {
	Id             int    `json:"id"`
	SourceType     string `json:"source_type"`
	Status         string `json:"status"`
	IssueMode      string `json:"issue_mode"`
	RequestedName  string `json:"requested_name"`
	RequestedGroup string `json:"requested_group"`
	TokenId        int    `json:"token_id"`
	CreatedTime    int64  `json:"created_time"`
	IssuedTime     int64  `json:"issued_time"`
	Product        *gin.H `json:"product,omitempty"`
}

type adminManageUserSellableTokenRequest struct {
	Id     int    `json:"id"`
	Action string `json:"action"`
}

type adminManageUserSellableTokenBatchRequest struct {
	Ids    []int  `json:"ids"`
	Action string `json:"action"`
}

func AdminListSellableTokenProducts(c *gin.Context) {
	products, err := model.ListSellableTokenProducts(true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, products)
}

func AdminCreateSellableTokenProduct(c *gin.Context) {
	var req sellableTokenProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.CreateSellableTokenProduct(&req.Product); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, req.Product)
}

func AdminUpdateSellableTokenProductStatus(c *gin.Context) {
	id := common.String2Int(c.Param("id"))
	var req sellableTokenProductStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.UpdateSellableTokenProductStatus(id, req.Status); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminDeleteSellableTokenProduct(c *gin.Context) {
	id := common.String2Int(c.Param("id"))
	if err := model.DeleteSellableTokenProduct(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminUpdateSellableTokenProduct(c *gin.Context) {
	id := common.String2Int(c.Param("id"))
	var req sellableTokenProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.UpdateSellableTokenProduct(id, &req.Product); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, req.Product)
}

func ListSellableTokenProducts(c *gin.Context) {
	products, err := model.ListSellableTokenProducts(false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	data := make([]gin.H, 0, len(products))
	for _, product := range products {
		allowedGroups, _ := resolveAllowedSellableTokenGroups(userId, product)
		groupOptions, _ := getUserAllowedGroupOptions(userId, allowedGroups)
		data = append(data, gin.H{
			"product":        product,
			"allowed_groups": allowedGroups,
			"user_groups":    groupOptions,
		})
	}
	common.ApiSuccess(c, data)
}

func PurchaseSellableToken(c *gin.Context) {
	userId := c.GetInt("id")
	lock := getTopUpLock(userId)
	if !lock.TryLock() {
		common.ApiErrorMsg(c, "当前有待处理的购买或兑换请求，请稍后再试")
		return
	}
	defer lock.Unlock()

	var req sellableTokenPurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	product, err := model.GetSellableTokenProductById(req.ProductId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.ValidateSellableTokenProductAvailability(product); err != nil {
		common.ApiError(c, err)
		return
	}
	priceQuota := product.PriceQuota
	if priceQuota <= 0 {
		common.ApiErrorMsg(c, "该可售令牌暂不支持钱包购买")
		return
	}
	if err := model.DecreaseUserQuota(userId, priceQuota); err != nil {
		common.ApiError(c, err)
		return
	}

	var order model.SellableTokenOrder
	var issuance model.SellableTokenIssuance
	txErr := model.DB.Transaction(func(tx *gorm.DB) error {
		order = model.SellableTokenOrder{
			UserId:     userId,
			ProductId:  product.Id,
			PriceQuota: priceQuota,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		issuance = model.SellableTokenIssuance{
			UserId:     userId,
			ProductId:  product.Id,
			SourceType: model.SellableTokenSourceTypeWallet,
			SourceId:   order.Id,
		}
		return model.CreateSellableTokenIssuanceTx(tx, &issuance)
	})
	if txErr != nil {
		_ = model.IncreaseUserQuota(userId, priceQuota, true)
		common.ApiError(c, txErr)
		return
	}
	common.ApiSuccess(c, gin.H{
		"issuance_id":  issuance.Id,
		"product_id":   product.Id,
		"product_name": product.Name,
		"price_quota":  priceQuota,
		"benefit_type": model.RedemptionBenefitTypeSellableToken,
		"source_type":  model.SellableTokenSourceTypeWallet,
	})
}

func ListSellableTokenIssuances(c *gin.Context) {
	userId := c.GetInt("id")
	status := strings.TrimSpace(c.Query("status"))
	issuances, err := model.ListSellableTokenIssuancesByUser(userId, status)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data := make([]gin.H, 0, len(issuances))
	for _, issuance := range issuances {
		item, err := buildSellableTokenIssuancePayload(userId, issuance)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		data = append(data, item)
	}
	common.ApiSuccess(c, data)
}

func GetSellableTokenIssuance(c *gin.Context) {
	userId := c.GetInt("id")
	issuanceId := common.String2Int(c.Param("id"))
	issuance, err := model.GetSellableTokenIssuanceByIdForUser(issuanceId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data, err := buildSellableTokenIssuancePayload(userId, issuance)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func ConfirmSellableTokenIssuance(c *gin.Context) {
	userId := c.GetInt("id")
	issuanceId := common.String2Int(c.Param("id"))
	var req sellableTokenIssueConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Mode = normalizeSellableTokenIssueMode(req.Mode)
	if req.Mode == "" {
		common.ApiErrorMsg(c, "请选择发放方式")
		return
	}

	var resultToken *model.Token
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Preload("Product").Where("id = ? AND user_id = ?", issuanceId, userId)
		if !common.UsingSQLite {
			query = query.Set("gorm:query_option", "FOR UPDATE")
		}
		var issuance model.SellableTokenIssuance
		if err := query.First(&issuance).Error; err != nil {
			return err
		}
		token, err := issueSellableTokenIssuanceTx(tx, &issuance, req)
		resultToken = token
		return err
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"token": resultToken,
	})
}

func CancelSellableTokenIssuance(c *gin.Context) {
	userId := c.GetInt("id")
	issuanceId := common.String2Int(c.Param("id"))
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		refundQuota, err := model.CancelSellableTokenIssuanceTx(tx, issuanceId, userId)
		if err != nil {
			return err
		}
		if refundQuota > 0 {
			if err := model.IncreaseUserQuota(userId, refundQuota, false); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": issuanceId})
}

func CancelAllPendingSellableTokenIssuances(c *gin.Context) {
	userId := c.GetInt("id")
	var cancelledCount int
	var totalRefund int
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		cancelledCount, totalRefund, err = model.CancelAllPendingSellableTokenIssuancesTx(tx, userId)
		if err != nil {
			return err
		}
		if totalRefund > 0 {
			if err := model.IncreaseUserQuota(userId, totalRefund, false); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"cancelled_count": cancelledCount,
		"total_refund":    totalRefund,
	})
}

func AdminCancelUserSellableTokenIssuance(c *gin.Context) {
	userId := common.String2Int(c.Param("id"))
	issuanceId := common.String2Int(c.Param("issuanceId"))
	if userId <= 0 || issuanceId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		refundQuota, err := model.CancelSellableTokenIssuanceTx(tx, issuanceId, userId)
		if err != nil {
			return err
		}
		if refundQuota > 0 {
			if err := model.IncreaseUserQuota(userId, refundQuota, false); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": issuanceId})
}

func AdminConfirmUserSellableTokenIssuance(c *gin.Context) {
	userId := common.String2Int(c.Param("id"))
	issuanceId := common.String2Int(c.Param("issuanceId"))
	if userId <= 0 || issuanceId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	var req sellableTokenIssueConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Mode = normalizeSellableTokenIssueMode(req.Mode)
	if req.Mode == "" {
		common.ApiErrorMsg(c, "请选择发放方式")
		return
	}
	var resultToken *model.Token
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Preload("Product").Where("id = ? AND user_id = ?", issuanceId, userId)
		if !common.UsingSQLite {
			query = query.Set("gorm:query_option", "FOR UPDATE")
		}
		var issuance model.SellableTokenIssuance
		if err := query.First(&issuance).Error; err != nil {
			return err
		}
		token, err := issueSellableTokenIssuanceTx(tx, &issuance, req)
		resultToken = token
		return err
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"token": resultToken,
	})
}

type adminBatchCancelIssuancesRequest struct {
	Ids []int `json:"ids"`
}

func AdminBatchCancelUserSellableTokenIssuances(c *gin.Context) {
	userId := common.String2Int(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}
	var req adminBatchCancelIssuancesRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	seen := make(map[int]struct{}, len(req.Ids))
	cancelled := make([]gin.H, 0, len(req.Ids))
	failed := make([]gin.H, 0)

	for _, issuanceId := range req.Ids {
		if issuanceId <= 0 {
			failed = append(failed, gin.H{"id": issuanceId, "message": "ID 无效"})
			continue
		}
		if _, ok := seen[issuanceId]; ok {
			continue
		}
		seen[issuanceId] = struct{}{}

		err := model.DB.Transaction(func(tx *gorm.DB) error {
			refundQuota, err := model.CancelSellableTokenIssuanceTx(tx, issuanceId, userId)
			if err != nil {
				return err
			}
			if refundQuota > 0 {
				if err := model.IncreaseUserQuota(userId, refundQuota, false); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			failed = append(failed, gin.H{"id": issuanceId, "message": err.Error()})
			continue
		}
		cancelled = append(cancelled, gin.H{"id": issuanceId})
	}

	common.ApiSuccess(c, gin.H{
		"success_count": len(cancelled),
		"failed_count":  len(failed),
		"cancelled":     cancelled,
		"failed":        failed,
	})
}

func AdminGetUserSellableTokenSummary(c *gin.Context) {
	userId := common.String2Int(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}
	var tokens []*model.Token
	if err := model.DB.Where("user_id = ? AND source_type = ?", userId, model.TokenSourceTypeSellableToken).Order("id desc").Find(&tokens).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	issuances, err := model.ListSellableTokenIssuancesByUser(userId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"tokens":    sanitizeSellableTokenSummaryItems(tokens),
		"issuances": sanitizeSellableTokenIssuanceSummaryItems(issuances),
	})
}

func AdminGetUserSellableTokenProductContext(c *gin.Context) {
	userId := common.String2Int(c.Param("id"))
	productId := common.String2Int(c.Param("productId"))
	if userId <= 0 || productId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	product, err := model.GetSellableTokenProductById(productId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.ValidateSellableTokenProductAvailability(product); err != nil {
		common.ApiError(c, err)
		return
	}
	data, err := buildSellableTokenIssueContextPayload(userId, product, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func AdminIssueSellableToken(c *gin.Context) {
	userId := common.String2Int(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}
	var req sellableTokenAdminIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Mode = normalizeSellableTokenIssueMode(req.Mode)
	if req.Mode == "" {
		common.ApiErrorMsg(c, "请选择发放方式")
		return
	}

	var (
		resultToken *model.Token
		issuanceId  int
	)
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		product, err := model.MustGetSellableTokenProductAvailableTx(tx, req.ProductId)
		if err != nil {
			return err
		}
		issuance := &model.SellableTokenIssuance{
			UserId:     userId,
			ProductId:  product.Id,
			SourceType: model.SellableTokenSourceTypeAdmin,
			Product:    product,
		}
		if err := model.CreateSellableTokenIssuanceTx(tx, issuance); err != nil {
			return err
		}
		resultToken, err = issueSellableTokenIssuanceTx(tx, issuance, req.sellableTokenIssueConfirmRequest)
		if err != nil {
			return err
		}
		issuanceId = issuance.Id
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"issuance_id": issuanceId,
		"token":       resultToken,
	})
}

func getAdminManageUserSellableTokenAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "enable":
		return "enable"
	case "disable":
		return "disable"
	case "delete":
		return "delete"
	default:
		return ""
	}
}

func manageUserSellableToken(userId int, tokenId int, action string) (string, error) {
	if userId <= 0 || tokenId <= 0 {
		return "", errors.New("参数错误")
	}
	action = getAdminManageUserSellableTokenAction(action)
	if action == "" {
		return "", errors.New("参数错误")
	}

	token, err := model.GetTokenById(tokenId)
	if err != nil {
		return "", err
	}
	if token == nil || token.UserId != userId {
		return "", errors.New("令牌不存在")
	}
	if token.SourceType != model.TokenSourceTypeSellableToken {
		return "", errors.New("仅支持管理可售令牌")
	}

	switch action {
	case "enable":
		if token.Status == common.TokenStatusExpired && token.ExpiredTime <= common.GetTimestamp() && token.ExpiredTime != -1 {
			return "", errors.New("已过期令牌不可启用")
		}
		if token.Status == common.TokenStatusExhausted && token.RemainQuota <= 0 && !token.UnlimitedQuota {
			return "", errors.New("已耗尽令牌不可启用")
		}
		token.Status = common.TokenStatusEnabled
		if err := token.Update(); err != nil {
			return "", err
		}
		return "已启用令牌", nil
	case "disable":
		token.Status = common.TokenStatusDisabled
		if err := token.Update(); err != nil {
			return "", err
		}
		return "已禁用令牌", nil
	case "delete":
		if err := token.Delete(); err != nil {
			return "", err
		}
		return "已删除令牌", nil
	default:
		return "", errors.New("参数错误")
	}
}

func AdminManageUserSellableToken(c *gin.Context) {
	userId := common.String2Int(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}

	var req adminManageUserSellableTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	action := getAdminManageUserSellableTokenAction(req.Action)
	if action == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	msg, err := manageUserSellableToken(userId, req.Id, action)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":      req.Id,
		"action":  action,
		"message": msg,
	})
}

func AdminManageUserSellableTokenBatch(c *gin.Context) {
	userId := common.String2Int(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}

	var req adminManageUserSellableTokenBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	action := getAdminManageUserSellableTokenAction(req.Action)
	if action == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	seen := make(map[int]struct{}, len(req.Ids))
	updated := make([]gin.H, 0, len(req.Ids))
	failed := make([]gin.H, 0)

	for _, id := range req.Ids {
		if id <= 0 {
			failed = append(failed, gin.H{
				"id":      id,
				"message": "令牌 ID 无效",
			})
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		msg, err := manageUserSellableToken(userId, id, action)
		if err != nil {
			failed = append(failed, gin.H{
				"id":      id,
				"message": err.Error(),
			})
			continue
		}
		updated = append(updated, gin.H{
			"id":      id,
			"action":  action,
			"message": msg,
		})
	}

	common.ApiSuccess(c, gin.H{
		"success_count": len(updated),
		"failed_count":  len(failed),
		"updated":       updated,
		"failed":        failed,
	})
}

func buildSellableTokenFromProduct(product *model.SellableTokenProduct, userId int, issuanceId int, name string, group string, key string) *model.Token {
	token := &model.Token{
		UserId:                  userId,
		Key:                     key,
		Name:                    name,
		Group:                   group,
		CreatedTime:             common.GetTimestamp(),
		AccessedTime:            common.GetTimestamp(),
		SourceType:              model.TokenSourceTypeSellableToken,
		BillingMode:             model.TokenBillingModeTokenOnly,
		SellableTokenProductId:  product.Id,
		SellableTokenIssuanceId: issuanceId,
		RemainQuota:             product.TotalQuota,
		UnlimitedQuota:          product.UnlimitedQuota,
		ModelLimitsEnabled:      product.ModelLimitsEnabled,
		ModelLimits:             product.ModelLimits,
		MaxConcurrency:          product.MaxConcurrency,
		WindowRequestLimit:      product.WindowRequestLimit,
		WindowSeconds:           product.WindowSeconds,
		PackageEnabled:          product.PackageEnabled,
		PackageLimitQuota:       product.PackageLimitQuota,
		PackagePeriod:           product.PackagePeriod,
		PackageCustomSeconds:    product.PackageCustomSeconds,
		PackagePeriodMode:       product.PackagePeriodMode,
		Status:                  common.TokenStatusEnabled,
	}
	token.ExpiredTime = model.CalcSellableTokenExpiry(common.GetTimestamp(), product.ValiditySeconds)
	return token
}

func getSellableTokenProductWithFallbackTx(tx *gorm.DB, issuance *model.SellableTokenIssuance) (*model.SellableTokenProduct, error) {
	if issuance == nil {
		return nil, errors.New("待发放记录不存在")
	}
	if issuance.Product != nil {
		return issuance.Product, nil
	}
	var product model.SellableTokenProduct
	if err := tx.First(&product, "id = ?", issuance.ProductId).Error; err != nil {
		var unscoped model.SellableTokenProduct
		if loadErr := tx.Unscoped().First(&unscoped, "id = ?", issuance.ProductId).Error; loadErr != nil {
			return nil, err
		}
		product = unscoped
	}
	issuance.Product = &product
	return issuance.Product, nil
}

func buildSellableTokenIssueContextPayload(userId int, product *model.SellableTokenProduct, requestedGroup string) (gin.H, error) {
	if product == nil {
		return gin.H{}, errors.New("可售令牌商品不存在")
	}
	allowedGroups, err := resolveAllowedSellableTokenGroups(userId, product)
	if err != nil {
		return gin.H{}, err
	}
	groupOptions, err := getUserAllowedGroupOptions(userId, allowedGroups)
	if err != nil {
		return gin.H{}, err
	}
	renewableTargets, err := model.ListRenewableSellableTokens(userId, product.Id)
	if err != nil {
		return gin.H{}, err
	}
	return gin.H{
		"product":           product,
		"allowed_groups":    allowedGroups,
		"group_options":     groupOptions,
		"renewable_targets": sanitizeSellableTokenTargets(renewableTargets),
		"requested_group":   requestedGroup,
	}, nil
}

func buildSellableTokenIssuancePayload(userId int, issuance *model.SellableTokenIssuance) (gin.H, error) {
	if issuance == nil {
		return gin.H{}, errors.New("待发放记录不存在")
	}
	if err := model.ResolveSellableTokenIssuanceDetails(issuance); err != nil {
		return gin.H{}, err
	}
	contextPayload, err := buildSellableTokenIssueContextPayload(
		userId,
		issuance.Product,
		issuance.RequestedGroup,
	)
	if err != nil {
		return gin.H{}, err
	}
	return gin.H{
		"issuance":          issuance,
		"product":           issuance.Product,
		"allowed_groups":    contextPayload["allowed_groups"],
		"group_options":     contextPayload["group_options"],
		"renewable_targets": contextPayload["renewable_targets"],
		"requested_group":   issuance.RequestedGroup,
	}, nil
}

func issueSellableTokenIssuanceTx(tx *gorm.DB, issuance *model.SellableTokenIssuance, req sellableTokenIssueConfirmRequest) (*model.Token, error) {
	if tx == nil {
		tx = model.DB
	}
	if issuance == nil {
		return nil, errors.New("待发放记录不存在")
	}
	if issuance.Status != model.SellableTokenIssuanceStatusPending {
		return nil, errors.New("该待发放记录已处理")
	}
	product, err := getSellableTokenProductWithFallbackTx(tx, issuance)
	if err != nil {
		return nil, err
	}
	allowedGroups, err := resolveAllowedSellableTokenGroups(issuance.UserId, product)
	if err != nil {
		return nil, err
	}
	if err := model.ValidateSellableTokenGroupChoice(req.Group, allowedGroups); err != nil {
		return nil, err
	}
	tokenName := model.BuildSellableTokenName(product, req.Name)
	if len([]rune(tokenName)) > 50 {
		return nil, errors.New("令牌名称不能超过 50 个字符")
	}

	renewableTargets, err := model.ListRenewableSellableTokens(issuance.UserId, product.Id)
	if err != nil {
		return nil, err
	}

	var resultToken *model.Token
	if req.Mode == "renew" {
		targetToken, err := selectRenewTargetToken(renewableTargets, req.TargetTokenId)
		if err != nil {
			return nil, err
		}
		if err := model.RenewSellableTokenTx(tx, targetToken, product); err != nil {
			return nil, err
		}
		if err := tx.Model(&model.Token{}).Where("id = ?", targetToken.Id).Updates(map[string]any{
			"name":          tokenName,
			"group":         req.Group,
			"accessed_time": common.GetTimestamp(),
		}).Error; err != nil {
			return nil, err
		}
		if err := tx.First(&resultToken, "id = ?", targetToken.Id).Error; err != nil {
			return nil, err
		}
		targetTokenId := targetToken.Id
		issuance.TokenId = &targetTokenId
		issuance.TargetTokenId = &targetTokenId
	} else {
		key, err := common.GenerateKey()
		if err != nil {
			return nil, err
		}
		tokenTemplate := buildSellableTokenFromProduct(product, issuance.UserId, issuance.Id, tokenName, req.Group, key)
		if err := model.ValidateTokenPackageConfig(tokenTemplate); err != nil {
			return nil, err
		}
		if err := model.ValidateTokenQuotaPackageRelation(tokenTemplate); err != nil {
			return nil, err
		}
		if err := model.CreateSellableTokenTx(tx, tokenTemplate); err != nil {
			return nil, err
		}
		resultToken = tokenTemplate
		issuedTokenId := tokenTemplate.Id
		issuance.TokenId = &issuedTokenId
	}

	issuance.Status = model.SellableTokenIssuanceStatusIssued
	issuance.IssueMode = req.Mode
	issuance.RequestedName = tokenName
	issuance.RequestedGroup = req.Group
	issuance.IssuedTime = common.GetTimestamp()
	if err := tx.Save(&issuance).Error; err != nil {
		return nil, err
	}
	if issuance.TokenId != nil && *issuance.TokenId > 0 {
		if err := tx.Model(&model.Token{}).Where("id = ?", *issuance.TokenId).Update("sellable_token_issuance_id", issuance.Id).Error; err != nil {
			return nil, err
		}
	}
	return resultToken, nil
}

func resolveAllowedSellableTokenGroups(userId int, product *model.SellableTokenProduct) ([]string, error) {
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		return nil, err
	}
	userUsableGroups := service.GetUserUsableGroups(userCache.Group)
	result := make([]string, 0)
	if product == nil || len(product.GetAllowedGroups()) == 0 {
		for group := range userUsableGroups {
			if group == "auto" || ratio_setting.ContainsGroupRatio(group) {
				result = append(result, group)
			}
		}
	} else {
		for _, group := range product.GetAllowedGroups() {
			if _, ok := userUsableGroups[group]; !ok {
				continue
			}
			if group != "auto" && !ratio_setting.ContainsGroupRatio(group) {
				continue
			}
			result = append(result, group)
		}
	}
	sort.Strings(result)
	return result, nil
}

func getUserAllowedGroupOptions(userId int, restrictTo []string) ([]gin.H, error) {
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		return nil, err
	}
	userUsableGroups := service.GetUserUsableGroups(userCache.Group)
	allowed := make(map[string]struct{})
	if len(restrictTo) > 0 {
		for _, group := range restrictTo {
			allowed[group] = struct{}{}
		}
	}
	keys := make([]string, 0)
	for group := range userUsableGroups {
		if len(allowed) > 0 {
			if _, ok := allowed[group]; !ok {
				continue
			}
		}
		if group != "auto" && !ratio_setting.ContainsGroupRatio(group) {
			continue
		}
		keys = append(keys, group)
	}
	sort.Strings(keys)
	options := make([]gin.H, 0, len(keys))
	for _, group := range keys {
		options = append(options, gin.H{
			"value": group,
			"label": userUsableGroups[group],
		})
	}
	return options, nil
}

func normalizeSellableTokenIssueMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "renew":
		return "renew"
	case "stack":
		fallthrough
	default:
		return "stack"
	}
}

func selectRenewTargetToken(targets []*model.Token, targetTokenId int) (*model.Token, error) {
	if len(targets) == 0 {
		return nil, errors.New("当前没有可续费的令牌")
	}
	if targetTokenId > 0 {
		for _, token := range targets {
			if token != nil && token.Id == targetTokenId {
				return token, nil
			}
		}
		return nil, errors.New("续费目标令牌不存在")
	}
	if len(targets) == 1 {
		return targets[0], nil
	}
	return nil, fmt.Errorf("存在多条可续费令牌，请先选择续费目标")
}

func sanitizeSellableTokenTargets(targets []*model.Token) []gin.H {
	result := make([]gin.H, 0, len(targets))
	for _, token := range targets {
		if token == nil {
			continue
		}
		result = append(result, gin.H{
			"id":            token.Id,
			"name":          token.Name,
			"group":         token.Group,
			"status":        token.Status,
			"expired_time":  token.ExpiredTime,
			"remain_quota":  token.RemainQuota,
			"used_quota":    token.UsedQuota,
			"created_time":  token.CreatedTime,
			"accessed_time": token.AccessedTime,
		})
	}
	return result
}

func intPointerValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func sanitizeSellableTokenSummaryItems(tokens []*model.Token) []sellableTokenSummaryItem {
	result := make([]sellableTokenSummaryItem, 0, len(tokens))
	for _, token := range tokens {
		if token == nil {
			continue
		}
		result = append(result, sellableTokenSummaryItem{
			Id:                 token.Id,
			Name:               token.Name,
			Group:              token.Group,
			Status:             token.Status,
			ProductId:          token.SellableTokenProductId,
			RemainQuota:        token.RemainQuota,
			UsedQuota:          token.UsedQuota,
			MaxConcurrency:     token.MaxConcurrency,
			WindowRequestLimit: token.WindowRequestLimit,
			WindowSeconds:      token.WindowSeconds,
			PackageEnabled:     token.PackageEnabled,
			PackageLimitQuota:  token.PackageLimitQuota,
			PackagePeriod:      token.PackagePeriod,
			ExpiredTime:        token.ExpiredTime,
			CreatedTime:        token.CreatedTime,
		})
	}
	return result
}

func sanitizeSellableTokenIssuanceSummaryItems(issuances []*model.SellableTokenIssuance) []sellableTokenIssuanceSummaryItem {
	result := make([]sellableTokenIssuanceSummaryItem, 0, len(issuances))
	for _, issuance := range issuances {
		if issuance == nil {
			continue
		}
		var product *gin.H
		if issuance.Product != nil {
			product = &gin.H{
				"id":   issuance.Product.Id,
				"name": issuance.Product.Name,
			}
		}
		result = append(result, sellableTokenIssuanceSummaryItem{
			Id:             issuance.Id,
			SourceType:     issuance.SourceType,
			Status:         issuance.Status,
			IssueMode:      issuance.IssueMode,
			RequestedName:  issuance.RequestedName,
			RequestedGroup: issuance.RequestedGroup,
			TokenId:        intPointerValue(issuance.TokenId),
			CreatedTime:    issuance.CreatedTime,
			IssuedTime:     issuance.IssuedTime,
			Product:        product,
		})
	}
	return result
}
