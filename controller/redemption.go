package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetAllRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.GetAllRedemptions(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.SearchRedemptions(keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemption, err := model.GetRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemption,
	})
	return
}

func AddRedemption(c *gin.Context) {
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(redemption.Name) == 0 || utf8.RuneCountInString(redemption.Name) > 20 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionNameLength)
		return
	}
	if redemption.Count <= 0 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountPositive)
		return
	}
	if redemption.Count > 100 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountMax)
		return
	}
	if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}
	if err := validateRedemptionPayload(&redemption); err != nil {
		common.ApiError(c, err)
		return
	}
	var keys []string
	for i := 0; i < redemption.Count; i++ {
		key := common.GetUUID()
		// 每个兑换码都展开成一条独立记录，后续禁用、删除、审计都更直接。
		cleanRedemption := model.Redemption{
			UserId:                       c.GetInt("id"),
			Name:                         redemption.Name,
			Key:                          key,
			CreatedTime:                  common.GetTimestamp(),
			BenefitType:                  model.NormalizeRedemptionBenefitType(redemption.BenefitType),
			Quota:                        redemption.Quota,
			PlanId:                       redemption.PlanId,
			SellableTokenProductId:       redemption.SellableTokenProductId,
			SubscriptionPurchaseMode:     model.NormalizeSubscriptionPurchaseMode(redemption.SubscriptionPurchaseMode),
			SubscriptionPurchaseQuantity: redemption.SubscriptionPurchaseQuantity,
			ExpiredTime:                  redemption.ExpiredTime,
			FundingSource:                model.RedemptionFundingSourceAdmin,
		}
		err = cleanRedemption.Insert()
		if err != nil {
			common.SysError("failed to insert redemption: " + err.Error())
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.T(c, i18n.MsgRedemptionCreateFailed),
				"data":    keys,
			})
			return
		}
		keys = append(keys, key)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
	return
}

func DeleteRedemption(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteRedemptionById(id)
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

func UpdateRedemption(c *gin.Context) {
	statusOnly := c.Query("status_only")
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanRedemption, err := model.GetRedemptionById(redemption.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if cleanRedemption.FundingSource == model.RedemptionFundingSourceWallet {
		common.ApiErrorMsg(c, "用户钱包创建的兑换码不允许由管理员编辑或禁用")
		return
	}
	if statusOnly == "" {
		if cleanRedemption.Status == common.RedemptionCodeStatusUsed {
			// 已使用兑换码禁止改权益，避免出现“账已经发出，后台又改成别的权益”的审计问题。
			common.ApiError(c, errors.New("已使用的兑换码不允许修改权益配置"))
			return
		}
		if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
		if err := validateRedemptionPayload(&redemption); err != nil {
			common.ApiError(c, err)
			return
		}
		// 如果新增可编辑字段，请同步更新 redemption.Update()。
		cleanRedemption.Name = redemption.Name
		cleanRedemption.BenefitType = model.NormalizeRedemptionBenefitType(redemption.BenefitType)
		cleanRedemption.Quota = redemption.Quota
		cleanRedemption.PlanId = redemption.PlanId
		cleanRedemption.SellableTokenProductId = redemption.SellableTokenProductId
		cleanRedemption.SubscriptionPurchaseMode = model.NormalizeSubscriptionPurchaseMode(redemption.SubscriptionPurchaseMode)
		cleanRedemption.SubscriptionPurchaseQuantity = redemption.SubscriptionPurchaseQuantity
		cleanRedemption.ExpiredTime = redemption.ExpiredTime
	}
	if statusOnly != "" {
		cleanRedemption.Status = redemption.Status
	}
	err = cleanRedemption.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
	return
}

type createWalletRedemptionRequest struct {
	Quota     int    `json:"quota"`
	RequestID string `json:"request_id"`
}

func CreateSelfRedemption(c *gin.Context) {
	userID := c.GetInt("id")
	lock := getTopUpLock(userID)
	if !lock.TryLock() {
		common.ApiErrorI18n(c, i18n.MsgUserTopUpProcessing)
		return
	}
	defer lock.Unlock()

	var req createWalletRedemptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	result, err := model.CreateWalletFundedRedemption(userID, req.Quota, req.RequestID)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrRedemptionInvalidQuota):
			common.ApiErrorMsg(c, "兑换码额度必须大于 0")
		case errors.Is(err, model.ErrRedemptionBelowMinimum):
			common.ApiErrorMsg(c, "普通用户创建兑换码的额度不能低于 10")
		case errors.Is(err, model.ErrRedemptionInsufficientQuota):
			common.ApiErrorMsg(c, "钱包余额不足")
		case errors.Is(err, model.ErrRedemptionInsufficientTransferableQuota):
			common.ApiErrorMsg(c, "可转赠额度不足；其他用户兑换码转入的额度不能再次创建兑换码")
		case errors.Is(err, model.ErrRedemptionBatchUpdateUnsafe):
			common.ApiErrorMsg(c, "当前余额采用批量更新模式，为避免余额不同步，暂不支持创建兑换码")
		case errors.Is(err, model.ErrRedemptionActiveLimit):
			common.ApiErrorMsg(c, "最多只能保留 100 个未使用兑换码")
		case errors.Is(err, model.ErrRedemptionDailyCreateLimit):
			common.ApiErrorMsg(c, fmt.Sprintf("普通用户单日最多创建 %d 个兑换码", common.WalletRedemptionDailyCreateLimit))
		case errors.Is(err, model.ErrRedemptionDailyQuotaLimit):
			common.ApiErrorMsg(c, fmt.Sprintf("普通用户单日最多转赠 %d 闪电", common.WalletRedemptionDailyQuotaLimit))
		case errors.Is(err, model.ErrRedemptionInvalidRequestID):
			common.ApiErrorMsg(c, "请求标识无效，请刷新页面后重试")
		default:
			common.SysError("failed to create wallet-funded redemption: " + err.Error())
			common.ApiErrorMsg(c, "创建兑换码失败，请稍后重试")
		}
		return
	}
	common.ApiSuccess(c, result)
}

func GetSelfRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.GetUserWalletRedemptions(c.GetInt("id"), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func DeleteInvalidRedemption(c *gin.Context) {
	rows, err := model.DeleteInvalidRedemptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
	return
}

func validateExpiredTime(c *gin.Context, expired int64) (bool, string) {
	if expired != 0 && expired < common.GetTimestamp() {
		return false, i18n.T(c, i18n.MsgRedemptionExpireTimeInvalid)
	}
	return true, ""
}

// validateRedemptionPayload 统一校验兑换码权益字段，避免控制器不同入口出现口径漂移。
func validateRedemptionPayload(redemption *model.Redemption) error {
	if redemption == nil {
		return errors.New("参数错误")
	}
	// 统一在控制器层做字段整理，这样模型层只处理“合法组合”的数据。
	benefitType := model.NormalizeRedemptionBenefitType(redemption.BenefitType)
	redemption.BenefitType = benefitType

	// 套餐码的购买模式固定在创建时写死，用户兑换时直接复用这里的配置。
	modeRaw := strings.TrimSpace(redemption.SubscriptionPurchaseMode)
	mode := model.NormalizeSubscriptionPurchaseMode(modeRaw)
	if modeRaw != "" && mode != modeRaw {
		return errors.New("无效的套餐购买方式")
	}
	// 兑换码场景只保留两个选项：叠加、续费。
	// 为兼容历史数据，这里把旧的 renew_extend 统一折叠到新的 renew 语义。
	if mode == model.SubscriptionPurchaseModeRenewExtend {
		mode = model.SubscriptionPurchaseModeRenew
	}
	redemption.SubscriptionPurchaseMode = mode
	if redemption.SubscriptionPurchaseQuantity <= 0 {
		redemption.SubscriptionPurchaseQuantity = 1
	}

	if benefitType == model.RedemptionBenefitTypeSubscription {
		// 套餐码不允许再配置余额，避免一个码混合多种权益导致发放口径复杂化。
		if redemption.PlanId <= 0 {
			return errors.New("套餐兑换码必须选择有效套餐")
		}
		if _, err := model.GetSubscriptionPlanById(redemption.PlanId); err != nil {
			return err
		}
		redemption.Quota = 0
		redemption.SellableTokenProductId = 0
		return nil
	}

	if benefitType == model.RedemptionBenefitTypeSellableToken {
		return errors.New("可售令牌兑换码功能已下线")
	}

	// 余额码沿用旧逻辑，统一清空套餐相关字段，避免脏数据落库。
	redemption.PlanId = 0
	redemption.SellableTokenProductId = 0
	redemption.SubscriptionPurchaseMode = model.SubscriptionPurchaseModeStack
	redemption.SubscriptionPurchaseQuantity = 1
	if redemption.Quota <= 0 {
		return errors.New("额度必须大于 0")
	}
	return nil
}
