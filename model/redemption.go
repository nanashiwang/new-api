package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
)

// ErrRedeemFailed is returned when redemption fails due to database error
var ErrRedeemFailed = errors.New("redeem.failed")

var (
	ErrRedemptionAlreadyUsed = errors.New("redemption.already_used")
	ErrRedemptionDisabled    = errors.New("redemption.disabled")
	ErrRedemptionExpired     = errors.New("redemption.expired")
)

const (
	RedemptionBenefitTypeQuota         = "quota"
	RedemptionBenefitTypeSubscription  = "subscription"
	RedemptionBenefitTypeSellableToken = "sellable_token"
)

type RedemptionResult struct {
	// BenefitType 标识本次兑换最终发放的是哪类权益，前端据此决定提示文案和刷新动作。
	BenefitType string `json:"benefit_type"`
	// QuotaAdded 仅对余额码有值，表示本次实际增加到用户余额的额度。
	QuotaAdded int `json:"quota_added"`
	// PlanId / PlanTitle 描述套餐码最终对应的当前套餐。
	PlanId    int    `json:"plan_id"`
	PlanTitle string `json:"plan_title"`
	// PurchaseMode / PurchaseQuantity 保留套餐兑换时的执行方式，方便前端展示。
	PurchaseMode     string `json:"purchase_mode"`
	PurchaseQuantity int    `json:"purchase_quantity"`
	// ActionSummary 同时用于套餐待发放提示与令牌待发放提示。
	ActionSummary string `json:"action_summary"`
	// IssuanceId 统一表示“待发放记录 ID”：
	// - benefit_type=subscription: 套餐待发放记录
	// - benefit_type=sellable_token: 可售令牌待发放记录
	IssuanceId  int    `json:"issuance_id"`
	ProductId   int    `json:"product_id"`
	ProductName string `json:"product_name"`
}

// RedeemNeedRenewTargetError 表示套餐兑换码在“续费”模式下命中了多条可续费订阅。
// 此时后端不自动替用户决定目标，而是把候选项返回给前端让用户选择。
type RedeemNeedRenewTargetError struct {
	PlanId      int                   `json:"plan_id"`
	PlanTitle   string                `json:"plan_title"`
	Options     []SubscriptionSummary `json:"options"`
	MessageText string                `json:"message"`
}

func (e *RedeemNeedRenewTargetError) Error() string {
	if e == nil || strings.TrimSpace(e.MessageText) == "" {
		return "存在多条可续费订阅，请先选择续费目标"
	}
	return e.MessageText
}

// RedeemNeedSelectPurchaseModeError 表示套餐兑换码未指定购买模式。
// 前端收到后应弹出叠加/续费选择框，用户选择后带 purchase_mode 重新请求。
type RedeemNeedSelectPurchaseModeError struct {
	PlanId    int    `json:"plan_id"`
	PlanTitle string `json:"plan_title"`
}

func (e *RedeemNeedSelectPurchaseModeError) Error() string {
	return "请选择兑换方式"
}

type Redemption struct {
	Id                           int    `json:"id"`
	UserId                       int    `json:"user_id"`
	Key                          string `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status                       int    `json:"status" gorm:"default:1"`
	Name                         string `json:"name" gorm:"index"`
	BenefitType                  string `json:"benefit_type" gorm:"type:varchar(32);not null;default:'quota';index"`
	Quota                        int    `json:"quota" gorm:"default:100"`
	PlanId                       int    `json:"plan_id" gorm:"type:int;default:0;index"`
	SellableTokenProductId       int    `json:"sellable_token_product_id" gorm:"type:int;default:0;index"`
	SubscriptionPurchaseMode     string `json:"subscription_purchase_mode" gorm:"type:varchar(16);not null;default:'stack'"`
	SubscriptionPurchaseQuantity int    `json:"subscription_purchase_quantity" gorm:"type:int;not null;default:1"`
	CreatedTime                  int64  `json:"created_time" gorm:"bigint"`
	RedeemedTime                 int64  `json:"redeemed_time" gorm:"bigint"`
	Count                        int    `json:"count" gorm:"-:all"`
	UsedUserId                   int    `json:"used_user_id"`
	// PlanTitle 仅用于列表展示当前套餐标题，不落库，不保留历史快照。
	PlanTitle   string         `json:"plan_title" gorm:"-"`
	ProductName string         `json:"product_name" gorm:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	ExpiredTime int64          `json:"expired_time" gorm:"bigint"`
}

func NormalizeRedemptionBenefitType(benefitType string) string {
	switch strings.ToLower(strings.TrimSpace(benefitType)) {
	case RedemptionBenefitTypeSubscription:
		return RedemptionBenefitTypeSubscription
	case RedemptionBenefitTypeSellableToken:
		return RedemptionBenefitTypeSellableToken
	default:
		return RedemptionBenefitTypeQuota
	}
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	// 这里沿用事务包裹分页查询，保持旧逻辑的一致性，避免后续插入统计口径漂移。
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err = tx.Model(&Redemption{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	// 列表返回前补齐套餐标题，前端无需再额外查套餐详情。
	fillRedemptionPlanTitles(redemptions)
	return redemptions, total, nil
}

func SearchRedemptions(keyword string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&Redemption{})
	if id, convErr := strconv.Atoi(keyword); convErr == nil {
		query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
	} else {
		query = query.Where("name LIKE ?", keyword+"%")
	}

	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	// 搜索结果和列表页保持同一展示口径。
	fillRedemptionPlanTitles(redemptions)
	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	if err := DB.First(&redemption, "id = ?", id).Error; err != nil {
		return nil, err
	}
	fillSingleRedemptionPlanTitle(&redemption)
	return &redemption, nil
}

func Redeem(key string, userId int) (quota int, err error) {
	// 旧接口只兼容余额码，先做轻量预判，避免套餐码被旧接口误消费。
	benefitType, detectErr := getRedemptionBenefitTypeByKey(key)
	if detectErr == nil && benefitType != RedemptionBenefitTypeQuota {
		return 0, legacyRedeemUnsupportedError(benefitType)
	}
	result, err := RedeemWithResult(key, userId)
	if err != nil {
		return 0, err
	}
	if result.BenefitType != RedemptionBenefitTypeQuota {
		return 0, legacyRedeemUnsupportedError(result.BenefitType)
	}
	return result.QuotaAdded, nil
}

func RedeemWithResult(key string, userId int) (*RedemptionResult, error) {
	return RedeemWithOptions(key, userId, 0)
}

// RedeemWithOptions 为统一兑换入口提供可选参数。
// purchaseMode 由用户选择（stack/renew），为空时后端返回特殊响应让前端弹出选择。
func RedeemWithOptions(key string, userId int, renewTargetSubscriptionId int, purchaseMode ...string) (*RedemptionResult, error) {
	if key == "" {
		return nil, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return nil, errors.New("无效的 user id")
	}
	redemption := &Redemption{}
	result := &RedemptionResult{}
	selectedPurchaseMode := ""
	if len(purchaseMode) > 0 {
		selectedPurchaseMode = purchaseMode[0]
	}

	// key 是保留字，跨库场景下统一显式处理列名引用。
	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}
	common.RandomSleep()
	err := DB.Transaction(func(tx *gorm.DB) error {
		// 兑换码先锁定，保证并发下只有一个请求能消费成功。
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		// 兑换码最终按数据库里的权益类型执行，避免前端或调用方伪造字段。
		redemption.BenefitType = NormalizeRedemptionBenefitType(redemption.BenefitType)
		switch redemption.Status {
		case common.RedemptionCodeStatusUsed:
			return ErrRedemptionAlreadyUsed
		case common.RedemptionCodeStatusDisabled:
			return ErrRedemptionDisabled
		case common.RedemptionCodeStatusEnabled:
		default:
			return ErrRedeemFailed
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return ErrRedemptionExpired
		}

		// 返回对象在事务内一次性填满，避免事务提交后再查引入展示与真实发放不一致。
		result.BenefitType = redemption.BenefitType
		switch redemption.BenefitType {
		case RedemptionBenefitTypeSubscription:
			// 套餐码先生成待发放记录，用户稍后再确认叠加/续费目标。
			planTitle, issuanceId, actionSummary, redeemErr := redeemSubscriptionBenefitTx(tx, redemption, userId, renewTargetSubscriptionId, selectedPurchaseMode)
			if redeemErr != nil {
				return redeemErr
			}
			result.PlanId = redemption.PlanId
			result.PlanTitle = planTitle
			result.PurchaseMode = resolveRedemptionSubscriptionPurchaseMode(redemption, selectedPurchaseMode)
			result.PurchaseQuantity = normalizeRedemptionSubscriptionPurchaseQuantity(redemption.SubscriptionPurchaseQuantity)
			result.IssuanceId = issuanceId
			result.ActionSummary = actionSummary
		case RedemptionBenefitTypeSellableToken:
			product, err := MustGetSellableTokenProductAvailableTx(tx, redemption.SellableTokenProductId)
			if err != nil {
				return err
			}
			issuance := &SellableTokenIssuance{
				UserId:     userId,
				ProductId:  product.Id,
				SourceType: SellableTokenSourceTypeRedeem,
				SourceId:   redemption.Id,
			}
			if err := CreateSellableTokenIssuanceTx(tx, issuance); err != nil {
				return err
			}
			result.IssuanceId = issuance.Id
			result.ProductId = product.Id
			result.ProductName = product.Name
		default:
			// 余额码保持旧行为：直接把额度加到用户余额。
			if err := tx.Model(&User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", redemption.Quota)).Error; err != nil {
				return err
			}
			result.QuotaAdded = redemption.Quota
		}

		// 权益实际发放成功后，最后再把兑换码标记为已使用，保证状态与权益一致。
		redemption.RedeemedTime = common.GetTimestamp()
		redemption.Status = common.RedemptionCodeStatusUsed
		redemption.UsedUserId = userId
		if err := tx.Save(redemption).Error; err != nil {
			return err
		}
		// 返佣台账和权益发放放在同一事务里，保持“全成或全不成”的一致性。
		return EnqueueInviteCommissionFromRedemptionTx(tx, redemption)
	})
	if err != nil {
		if _, ok := err.(*RedeemNeedRenewTargetError); ok {
			return nil, err
		}
		if _, ok := err.(*RedeemNeedSelectPurchaseModeError); ok {
			return nil, err
		}
		if isRedeemBusinessError(err) {
			return nil, err
		}
		common.SysError("redemption failed: " + err.Error())
		return nil, ErrRedeemFailed
	}

	switch result.BenefitType {
	case RedemptionBenefitTypeSubscription:
		RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码领取套餐 %s，兑换码ID %d，待发放记录 %d", result.PlanTitle, redemption.Id, result.IssuanceId))
		return result, nil
	case RedemptionBenefitTypeSellableToken:
		RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码领取可售令牌 %s，兑换码ID %d，待发放记录 %d", result.ProductName, redemption.Id, result.IssuanceId))
		return result, nil
	}
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuota(redemption.Quota), redemption.Id))
	return result, nil
}

func (redemption *Redemption) Insert() error {
	// 创建前统一归一化，避免不同入口写出不同风格的数据。
	redemption.BenefitType = NormalizeRedemptionBenefitType(redemption.BenefitType)
	redemption.SubscriptionPurchaseMode = NormalizeSubscriptionPurchaseMode(redemption.SubscriptionPurchaseMode)
	redemption.SubscriptionPurchaseQuantity = normalizeRedemptionSubscriptionPurchaseQuantity(redemption.SubscriptionPurchaseQuantity)
	return DB.Create(redemption).Error
}

func (redemption *Redemption) SelectUpdate() error {
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

func (redemption *Redemption) Update() error {
	// 编辑时同样走归一化，保证管理端改完后字段组合仍然合法。
	redemption.BenefitType = NormalizeRedemptionBenefitType(redemption.BenefitType)
	redemption.SubscriptionPurchaseMode = NormalizeSubscriptionPurchaseMode(redemption.SubscriptionPurchaseMode)
	redemption.SubscriptionPurchaseQuantity = normalizeRedemptionSubscriptionPurchaseQuantity(redemption.SubscriptionPurchaseQuantity)
	return DB.Model(redemption).Select(
		"name",
		"status",
		"benefit_type",
		"quota",
		"plan_id",
		"sellable_token_product_id",
		"subscription_purchase_mode",
		"subscription_purchase_quantity",
		"redeemed_time",
		"expired_time",
	).Updates(redemption).Error
}

func (redemption *Redemption) Delete() error {
	return DB.Delete(redemption).Error
}

func DeleteRedemptionById(id int) error {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	if err := DB.Where(redemption).First(&redemption).Error; err != nil {
		return err
	}
	return redemption.Delete()
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	result := DB.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled}, common.RedemptionCodeStatusEnabled, now).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}

func normalizeRedemptionSubscriptionPurchaseQuantity(quantity int) int {
	// 套餐兑换份数对外不允许出现 0 或负数，统一兜底为 1。
	if quantity <= 0 {
		return 1
	}
	return quantity
}

// getRedemptionBenefitTypeByKey 仅用于兼容旧接口的预判，不承担最终业务校验。
func getRedemptionBenefitTypeByKey(key string) (string, error) {
	// 这里只做兼容旧接口的快速识别，不承担最终兑换校验。
	if strings.TrimSpace(key) == "" {
		return "", errors.New("empty redemption key")
	}
	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}
	var redemption Redemption
	if err := DB.Select("benefit_type").Where(keyCol+" = ?", key).First(&redemption).Error; err != nil {
		return "", err
	}
	return NormalizeRedemptionBenefitType(redemption.BenefitType), nil
}

func fillSingleRedemptionPlanTitle(redemption *Redemption) {
	// 列表/详情展示时按当前套餐标题回填，不做历史快照。
	if redemption == nil || redemption.PlanId <= 0 || NormalizeRedemptionBenefitType(redemption.BenefitType) != RedemptionBenefitTypeSubscription {
		if redemption == nil || redemption.SellableTokenProductId <= 0 || NormalizeRedemptionBenefitType(redemption.BenefitType) != RedemptionBenefitTypeSellableToken {
			return
		}
		product, err := GetSellableTokenProductById(redemption.SellableTokenProductId)
		if err == nil && product != nil {
			redemption.ProductName = product.Name
		}
		return
	}
	plan, err := GetSubscriptionPlanById(redemption.PlanId)
	if err == nil && plan != nil {
		redemption.PlanTitle = plan.Title
	}
}

func fillRedemptionPlanTitles(redemptions []*Redemption) {
	// 批量回填列表展示字段，避免套餐/可售令牌列表页出现 N+1 查询。
	planIDs := make([]int, 0)
	productIDs := make([]int, 0)
	seenPlanIDs := make(map[int]struct{})
	seenProductIDs := make(map[int]struct{})
	for _, redemption := range redemptions {
		if redemption == nil {
			continue
		}
		switch NormalizeRedemptionBenefitType(redemption.BenefitType) {
		case RedemptionBenefitTypeSubscription:
			if redemption.PlanId <= 0 {
				continue
			}
			if _, ok := seenPlanIDs[redemption.PlanId]; ok {
				continue
			}
			seenPlanIDs[redemption.PlanId] = struct{}{}
			planIDs = append(planIDs, redemption.PlanId)
		case RedemptionBenefitTypeSellableToken:
			if redemption.SellableTokenProductId <= 0 {
				continue
			}
			if _, ok := seenProductIDs[redemption.SellableTokenProductId]; ok {
				continue
			}
			seenProductIDs[redemption.SellableTokenProductId] = struct{}{}
			productIDs = append(productIDs, redemption.SellableTokenProductId)
		}
	}

	planTitleByID := make(map[int]string, len(planIDs))
	if len(planIDs) > 0 {
		var plans []SubscriptionPlan
		if err := DB.Select("id", "title").Where("id IN ?", planIDs).Find(&plans).Error; err == nil {
			for _, plan := range plans {
				planTitleByID[plan.Id] = plan.Title
			}
		}
	}

	productNameByID := make(map[int]string, len(productIDs))
	if len(productIDs) > 0 {
		var products []SellableTokenProduct
		if err := DB.Select("id", "name").Where("id IN ?", productIDs).Find(&products).Error; err == nil {
			for _, product := range products {
				productNameByID[product.Id] = product.Name
			}
		}
	}

	for _, redemption := range redemptions {
		if redemption == nil {
			continue
		}
		redemption.PlanTitle = planTitleByID[redemption.PlanId]
		redemption.ProductName = productNameByID[redemption.SellableTokenProductId]
	}
}

func redeemSubscriptionBenefitTx(tx *gorm.DB, redemption *Redemption, userId int, renewTargetSubscriptionId int, userPurchaseMode string) (string, int, string, error) {
	// 套餐码先生成待发放记录，再由用户确认叠加/续费，避免兑换时中断导致权益状态不清晰。
	if tx == nil {
		return "", 0, "", errors.New("tx is nil")
	}
	if redemption == nil || redemption.PlanId <= 0 {
		return "", 0, "", errors.New("兑换码未配置有效套餐")
	}
	plan, err := getSubscriptionPlanByIdTx(tx, redemption.PlanId)
	if err != nil {
		return "", 0, "", err
	}
	if plan == nil {
		return "", 0, "", errors.New("套餐不存在")
	}
	// 套餐码跟随当前套餐状态：套餐被停用后，旧码也不能继续兑换。
	if !plan.Enabled {
		return "", 0, "", errors.New("当前套餐不可兑换")
	}
	// 优先使用用户传入的 purchaseMode；为空时兼容沿用兑换码本身的默认兑换方式，
	// 避免老调用方/测试在未传 mode 时被破坏。
	modeInput := strings.TrimSpace(userPurchaseMode)
	if modeInput == "" {
		modeInput = strings.TrimSpace(redemption.SubscriptionPurchaseMode)
	}
	mode := normalizeSubscriptionIssuancePurchaseMode(modeInput)
	purchaseQuantity := normalizeRedemptionSubscriptionPurchaseQuantity(redemption.SubscriptionPurchaseQuantity)
	issuance := &SubscriptionIssuance{
		UserId:                    userId,
		PlanId:                    plan.Id,
		PlanTitle:                 plan.Title,
		SourceType:                SubscriptionIssuanceSourceRedeem,
		SourceRef:                 strconv.Itoa(redemption.Id),
		PurchaseMode:              mode,
		PurchaseQuantity:          purchaseQuantity,
		RenewTargetSubscriptionId: renewTargetSubscriptionId,
	}
	if err := CreateSubscriptionIssuanceTx(tx, issuance); err != nil {
		return "", 0, "", err
	}
	actionSummary := "已生成套餐待发放记录，请完成叠加或续费设置"
	return plan.Title, issuance.Id, actionSummary, nil
}

func validateRedemptionSubscriptionPurchaseQuantityTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, quantity int) error {
	if plan == nil {
		return errors.New("套餐不存在")
	}
	minQuantity := plan.PurchaseQuantityMin
	maxQuantity := plan.PurchaseQuantityMax
	if minQuantity < 1 {
		minQuantity = 1
	}
	if maxQuantity < 1 {
		maxQuantity = 12
	}
	if maxQuantity < minQuantity {
		maxQuantity = minQuantity
	}
	// 套餐码虽然是固定份数，但仍然要受当前套餐最小/最大购买份数约束。
	activeSubs, err := getActiveUserSubscriptionsByPlanTx(tx, userId, plan.Id, 0)
	if err != nil {
		return err
	}
	activeQuantity := 0
	nowUnix := GetDBTimestampTx(tx)
	for i := range activeSubs {
		activeQuantity += countRemainingSubscriptionQuantity(&activeSubs[i], plan, nowUnix)
	}
	// 动态最大值 = 套餐配置上限 - 当前仍在生效的份数。
	dynamicMaxQuantity := maxQuantity - activeQuantity
	if dynamicMaxQuantity <= 0 {
		return errors.New("当前可兑换数量为 0，请等待部分订阅到期后再试")
	}
	if dynamicMaxQuantity < minQuantity {
		return fmt.Errorf("当前最多可兑换 %d 份，低于最小购买数量 %d", dynamicMaxQuantity, minQuantity)
	}
	if quantity < minQuantity {
		return errors.New("兑换数量低于下限")
	}
	if quantity > dynamicMaxQuantity {
		return fmt.Errorf("兑换数量超出上限，当前最多可兑换 %d 份", dynamicMaxQuantity)
	}
	return nil
}

func legacyRedeemUnsupportedError(benefitType string) error {
	switch NormalizeRedemptionBenefitType(benefitType) {
	case RedemptionBenefitTypeSubscription:
		return errors.New("该兑换码为套餐兑换码，请使用统一兑换接口")
	case RedemptionBenefitTypeSellableToken:
		return errors.New("该兑换码为可售令牌兑换码，请使用统一兑换接口")
	default:
		return errors.New("该兑换码为非余额兑换码，请使用统一兑换接口")
	}
}

func isRedeemBusinessError(err error) bool {
	return errors.Is(err, ErrRedemptionAlreadyUsed) ||
		errors.Is(err, ErrRedemptionDisabled) ||
		errors.Is(err, ErrRedemptionExpired)
}

func validateRedemptionSubscriptionLimitsTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, purchaseMode string, purchaseQuantity int) error {
	if userId <= 0 || plan == nil {
		return errors.New("参数错误")
	}
	if purchaseQuantity <= 0 {
		purchaseQuantity = 1
	}
	if plan.MaxPurchasePerUser > 0 {
		// 历史购买上限继续生效，保证套餐码与付费购买口径一致。
		var totalCount int64
		if err := tx.Model(&UserSubscription{}).
			Where("user_id = ? AND plan_id = ?", userId, plan.Id).
			Count(&totalCount).Error; err != nil {
			return err
		}
		limit := int64(plan.MaxPurchasePerUser)
		if ((purchaseMode == SubscriptionPurchaseModeStack || purchaseMode == SubscriptionPurchaseModeRenewExtend) &&
			totalCount+int64(purchaseQuantity) > limit) ||
			(purchaseMode == SubscriptionPurchaseModeRenew && totalCount >= limit) {
			return errors.New("已达到该套餐购买上限")
		}
	}
	if (purchaseMode == SubscriptionPurchaseModeStack || purchaseMode == SubscriptionPurchaseModeRenewExtend) && plan.MaxStackPerUser > 0 {
		// 叠加上限只约束会新增有效订阅条数的模式。
		nowUnix := GetDBTimestampTx(tx)
		var activeCount int64
		if err := tx.Model(&UserSubscription{}).
			Where("user_id = ? AND plan_id = ? AND status = ? AND end_time > ?", userId, plan.Id, "active", nowUnix).
			Count(&activeCount).Error; err != nil {
			return err
		}
		addedActiveCount := int64(purchaseQuantity)
		if purchaseMode == SubscriptionPurchaseModeRenewExtend {
			addedActiveCount = 0
		}
		if purchaseMode == SubscriptionPurchaseModeRenewExtend && activeCount == 0 && purchaseQuantity > 0 {
			addedActiveCount = 1
		}
		if activeCount+addedActiveCount > int64(plan.MaxStackPerUser) {
			return errors.New("已达到该套餐叠加上限")
		}
	}
	return nil
}

func resolveRedemptionSubscriptionPurchaseMode(redemption *Redemption, userPurchaseMode string) string {
	modeInput := strings.TrimSpace(userPurchaseMode)
	if modeInput == "" && redemption != nil {
		modeInput = strings.TrimSpace(redemption.SubscriptionPurchaseMode)
	}
	return NormalizeSubscriptionPurchaseMode(modeInput)
}
