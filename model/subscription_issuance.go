package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	SubscriptionIssuanceStatusPending = "pending"
	SubscriptionIssuanceStatusIssued  = "issued"

	SubscriptionIssuanceSourceRedeem = "redeem"
	SubscriptionIssuanceSourceOrder  = "order"
)

type SubscriptionIssuance struct {
	Id                        int                   `json:"id"`
	UserId                    int                   `json:"user_id" gorm:"index"`
	PlanId                    int                   `json:"plan_id" gorm:"index"`
	PlanTitle                 string                `json:"plan_title" gorm:"type:varchar(128);default:''"`
	SourceType                string                `json:"source_type" gorm:"type:varchar(32);not null;default:'redeem'"`
	SourceRef                 string                `json:"-" gorm:"type:varchar(128);default:''"`
	Status                    string                `json:"status" gorm:"type:varchar(16);not null;default:'pending';index"`
	PurchaseMode              string                `json:"purchase_mode" gorm:"type:varchar(16);default:''"`
	PurchaseQuantity          int                   `json:"purchase_quantity" gorm:"type:int;not null;default:1"`
	RenewTargetSubscriptionId int                   `json:"renew_target_subscription_id" gorm:"type:int;default:0;index"`
	IssueSummary              string                `json:"issue_summary" gorm:"type:varchar(255);default:''"`
	CreatedTime               int64                 `json:"created_time" gorm:"bigint"`
	UpdatedTime               int64                 `json:"updated_time" gorm:"bigint"`
	IssuedTime                int64                 `json:"issued_time" gorm:"bigint;default:0"`
	Plan                      *SubscriptionPlan     `json:"plan,omitempty" gorm:"foreignKey:PlanId;-:migration"`
	RenewableTargets          []SubscriptionSummary `json:"renewable_targets,omitempty" gorm:"-"`
}

func (i *SubscriptionIssuance) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	i.CreatedTime = now
	i.UpdatedTime = now
	if i.Status == "" {
		i.Status = SubscriptionIssuanceStatusPending
	}
	return nil
}

func (i *SubscriptionIssuance) BeforeUpdate(tx *gorm.DB) error {
	i.UpdatedTime = common.GetTimestamp()
	return nil
}

func normalizeSubscriptionIssuancePurchaseMode(mode string) string {
	trimmed := strings.TrimSpace(mode)
	switch trimmed {
	case "":
		return ""
	case SubscriptionPurchaseModeStack, SubscriptionPurchaseModeRenew, SubscriptionPurchaseModeRenewExtend:
		return trimmed
	default:
		return ""
	}
}

func CreateSubscriptionIssuanceTx(tx *gorm.DB, issuance *SubscriptionIssuance) error {
	if tx == nil {
		tx = DB
	}
	if issuance == nil {
		return errors.New("issuance is nil")
	}
	if issuance.UserId <= 0 || issuance.PlanId <= 0 {
		return errors.New("待发放记录参数无效")
	}
	issuance.SourceType = strings.TrimSpace(issuance.SourceType)
	if issuance.SourceType == "" {
		issuance.SourceType = SubscriptionIssuanceSourceRedeem
	}
	issuance.PurchaseMode = normalizeSubscriptionIssuancePurchaseMode(issuance.PurchaseMode)
	if issuance.PurchaseQuantity <= 0 {
		issuance.PurchaseQuantity = 1
	}
	issuance.Status = SubscriptionIssuanceStatusPending
	return tx.Create(issuance).Error
}

func ListSubscriptionIssuancesByUser(userId int, status string) ([]*SubscriptionIssuance, error) {
	if userId <= 0 {
		return nil, errors.New("无效的用户")
	}
	var issuances []*SubscriptionIssuance
	query := DB.Preload("Plan").Where("user_id = ?", userId).Order("id desc")
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	if err := query.Find(&issuances).Error; err != nil {
		return nil, err
	}
	return issuances, nil
}

func GetSubscriptionIssuanceByIdForUser(id int, userId int) (*SubscriptionIssuance, error) {
	if id <= 0 || userId <= 0 {
		return nil, errors.New("无效的待发放记录")
	}
	var issuance SubscriptionIssuance
	if err := DB.Preload("Plan").First(&issuance, "id = ? AND user_id = ?", id, userId).Error; err != nil {
		return nil, err
	}
	return &issuance, nil
}

func ResolveSubscriptionIssuanceDetails(issuance *SubscriptionIssuance) error {
	if issuance == nil {
		return errors.New("issuance is nil")
	}
	if issuance.Plan == nil && issuance.PlanId > 0 {
		plan, err := GetSubscriptionPlanById(issuance.PlanId)
		if err == nil && plan != nil {
			issuance.Plan = plan
		}
	}
	activeSubs, err := GetActiveUserSubscriptionsByPlan(issuance.UserId, issuance.PlanId)
	if err != nil {
		return err
	}
	issuance.RenewableTargets = buildSubscriptionSummaries(activeSubs)
	return nil
}

func ConfirmSubscriptionIssuanceTx(tx *gorm.DB, issuanceId int, userId int, purchaseMode string, renewTargetSubscriptionId int) (*SubscriptionIssuance, string, error) {
	if tx == nil {
		tx = DB
	}
	if issuanceId <= 0 || userId <= 0 {
		return nil, "", errors.New("无效的待发放记录")
	}
	query := tx.Preload("Plan").Where("id = ? AND user_id = ?", issuanceId, userId)
	if !common.UsingSQLite {
		query = query.Set("gorm:query_option", "FOR UPDATE")
	}
	var issuance SubscriptionIssuance
	if err := query.First(&issuance).Error; err != nil {
		return nil, "", err
	}
	if issuance.Status != SubscriptionIssuanceStatusPending {
		return nil, "", errors.New("该待发放记录已处理")
	}
	plan := issuance.Plan
	if plan == nil {
		loadedPlan, err := getSubscriptionPlanByIdTx(tx, issuance.PlanId)
		if err != nil {
			return nil, "", err
		}
		plan = loadedPlan
	}
	mode := normalizeSubscriptionIssuancePurchaseMode(purchaseMode)
	if mode == "" {
		mode = normalizeSubscriptionIssuancePurchaseMode(issuance.PurchaseMode)
	}
	if mode == "" {
		return nil, "", errors.New("请选择发放方式")
	}
	targetId := renewTargetSubscriptionId
	if targetId <= 0 {
		targetId = issuance.RenewTargetSubscriptionId
	}
	if mode == SubscriptionPurchaseModeRenew {
		activeSubs, err := getActiveUserSubscriptionsByPlanTx(tx, userId, plan.Id, 0)
		if err != nil {
			return nil, "", err
		}
		if len(activeSubs) == 0 {
			return nil, "", errors.New("当前无可续费的同规格订阅")
		}
		if targetId > 0 {
			matched := false
			for i := range activeSubs {
				if activeSubs[i].Id == targetId {
					matched = true
					break
				}
			}
			if !matched {
				return nil, "", errors.New("续费目标订阅不存在或已失效")
			}
		} else if len(activeSubs) == 1 {
			targetId = activeSubs[0].Id
		} else {
			return nil, "", errors.New("请选择续费目标订阅")
		}
	}

	summary, err := bindSubscriptionWithOptionsTx(
		tx,
		userId,
		plan,
		mode,
		issuance.PurchaseQuantity,
		targetId,
		"subscription_issuance",
	)
	if err != nil {
		return nil, "", err
	}
	issuance.PurchaseMode = mode
	issuance.RenewTargetSubscriptionId = targetId
	issuance.IssueSummary = summary
	issuance.Status = SubscriptionIssuanceStatusIssued
	issuance.IssuedTime = common.GetTimestamp()
	if err := tx.Save(&issuance).Error; err != nil {
		return nil, "", err
	}
	return &issuance, summary, nil
}
