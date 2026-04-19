package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	BenefitTypeQuota        = "quota"
	BenefitTypeSubscription = "subscription"

	BenefitActionGrant    = "grant"
	BenefitActionRollback = "rollback"

	BenefitTargetUserQuota            = "user_quota"
	BenefitTargetUserSubscription     = "user_subscription"
	BenefitTargetSubscriptionIssuance = "subscription_issuance"

	BenefitSourceTopUpOrder             = "topup_order"
	BenefitSourceSubscriptionOrder      = "subscription_order"
	BenefitSourceSubscriptionRedemption = "subscription_redemption"

	BenefitRollbackBusinessPaymentRiskCase = "payment_risk_case"

	BenefitRollbackStatusPending   = "pending"
	BenefitRollbackStatusSucceeded = "succeeded"
	BenefitRollbackStatusFailed    = "failed"

	SubscriptionBenefitOperationCreate        = "create"
	SubscriptionBenefitOperationRenew         = "renew"
	SubscriptionBenefitOperationCancelPending = "cancel_pending"
)

const (
	benefitContextSourceTypeKey   = "benefit_context_source_type"
	benefitContextSourceRefKey    = "benefit_context_source_ref"
	benefitContextPurchaseModeKey = "benefit_context_purchase_mode"
	benefitContextIssuanceIDKey   = "benefit_context_issuance_id"
)

// BenefitChangeRecord 的唯一索引 idx_benefit_change_dedup 覆盖
// (source_type, source_ref, action, target_type, target_id)，用于防止同一笔
// 来源订单在上游回调重放时重复发放或重复回退——grant 和 rollback 的 action
// 不同，天然不会互相冲突。
type BenefitChangeRecord struct {
	Id int `json:"id"`

	BenefitType string `json:"benefit_type" gorm:"type:varchar(32);not null;index"`
	Action      string `json:"action" gorm:"type:varchar(16);not null;index;uniqueIndex:idx_benefit_change_dedup,priority:3"`

	SourceType string `json:"source_type" gorm:"type:varchar(64);not null;index:idx_benefit_source;uniqueIndex:idx_benefit_change_dedup,priority:1"`
	SourceRef  string `json:"source_ref" gorm:"type:varchar(255);not null;index:idx_benefit_source;uniqueIndex:idx_benefit_change_dedup,priority:2"`

	UserId     int    `json:"user_id" gorm:"index"`
	TargetType string `json:"target_type" gorm:"type:varchar(64);not null;index;uniqueIndex:idx_benefit_change_dedup,priority:4"`
	TargetId   int    `json:"target_id" gorm:"index;uniqueIndex:idx_benefit_change_dedup,priority:5"`

	OriginRecordId      int    `json:"origin_record_id" gorm:"index;default:0"`
	RollbackOperationId int    `json:"rollback_operation_id" gorm:"index;default:0"`
	Detail              string `json:"detail" gorm:"type:text"`

	CreatedAt int64 `json:"created_at" gorm:"bigint;index"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint;index"`
}

type BenefitRollbackOperation struct {
	Id int `json:"id"`

	BusinessType string `json:"business_type" gorm:"type:varchar(64);not null;uniqueIndex:idx_benefit_rollback_business"`
	BusinessId   int    `json:"business_id" gorm:"not null;uniqueIndex:idx_benefit_rollback_business"`

	SourceType string `json:"source_type" gorm:"type:varchar(64);not null;index"`
	SourceRef  string `json:"source_ref" gorm:"type:varchar(255);not null;index"`

	Status        string `json:"status" gorm:"type:varchar(16);not null;default:'pending';index"`
	ResultSummary string `json:"result_summary" gorm:"type:text"`
	ErrorMessage  string `json:"error_message" gorm:"type:text"`

	CreatedAt int64 `json:"created_at" gorm:"bigint;index"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint;index"`
}

type QuotaBenefitDetail struct {
	QuotaDelta    int    `json:"quota_delta"`
	PaymentMethod string `json:"payment_method,omitempty"`
	Context       string `json:"context,omitempty"`
}

type SubscriptionBenefitDetail struct {
	Operation    string `json:"operation"`
	PurchaseMode string `json:"purchase_mode,omitempty"`
	IssuanceId   int    `json:"issuance_id,omitempty"`
	PlanId       int    `json:"plan_id,omitempty"`

	EndTimeBefore int64 `json:"end_time_before,omitempty"`
	EndTimeAfter  int64 `json:"end_time_after,omitempty"`
	EndTimeDelta  int64 `json:"end_time_delta,omitempty"`

	AmountTotalBefore int64 `json:"amount_total_before,omitempty"`
	AmountTotalAfter  int64 `json:"amount_total_after,omitempty"`
	AmountTotalDelta  int64 `json:"amount_total_delta,omitempty"`

	StatusBefore string `json:"status_before,omitempty"`
	StatusAfter  string `json:"status_after,omitempty"`
}

func (r *BenefitChangeRecord) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *BenefitChangeRecord) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func (r *BenefitRollbackOperation) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	if strings.TrimSpace(r.Status) == "" {
		r.Status = BenefitRollbackStatusPending
	}
	return nil
}

func (r *BenefitRollbackOperation) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

// withBenefitContextTx 把 benefit 发放上下文挂到 *gorm.DB 的 Statement.Settings 上，
// 供下游的 createUserSubscriptionFromPlanTx / renewUserSubscriptionByPlanTx 通过
// getBenefitContextTx 读取并写入 BenefitChangeRecord。
//
// 注意：GORM 的 Session(&gorm.Session{NewDB: true}) 会重建 Statement，从而丢失
// 这里挂的 Settings。任何在 benefit 发放/回退链路上创建 NewDB:true session 的
// 地方都必须显式调用 propagateBenefitContextTx 把上下文复制过去，否则 grant
// 记录会静默缺失，后续 RollbackBenefitsBySource 无法找到对应 grant 做回退。
func withBenefitContextTx(tx *gorm.DB, sourceType string, sourceRef string, purchaseMode string, issuanceId int) *gorm.DB {
	if tx == nil {
		return nil
	}
	return tx.
		Set(benefitContextSourceTypeKey, strings.TrimSpace(sourceType)).
		Set(benefitContextSourceRefKey, strings.TrimSpace(sourceRef)).
		Set(benefitContextPurchaseModeKey, strings.TrimSpace(purchaseMode)).
		Set(benefitContextIssuanceIDKey, issuanceId)
}

// propagateBenefitContextTx 从 src 读 benefit 上下文并写入 dst，用于绕过
// Session(NewDB: true) 丢 Settings 的问题。src/dst 为 nil 或 src 上没有上下文
// 时直接返回 dst，调用方不需要额外判空。
func propagateBenefitContextTx(src *gorm.DB, dst *gorm.DB) *gorm.DB {
	if dst == nil {
		return nil
	}
	sourceType, sourceRef, purchaseMode, issuanceId, ok := getBenefitContextTx(src)
	if !ok {
		return dst
	}
	return withBenefitContextTx(dst, sourceType, sourceRef, purchaseMode, issuanceId)
}

func getBenefitContextTx(tx *gorm.DB) (string, string, string, int, bool) {
	if tx == nil {
		return "", "", "", 0, false
	}
	rawSourceType, ok := tx.Get(benefitContextSourceTypeKey)
	if !ok {
		return "", "", "", 0, false
	}
	rawSourceRef, ok := tx.Get(benefitContextSourceRefKey)
	if !ok {
		return "", "", "", 0, false
	}
	rawPurchaseMode, _ := tx.Get(benefitContextPurchaseModeKey)
	rawIssuanceID, _ := tx.Get(benefitContextIssuanceIDKey)

	sourceType, _ := rawSourceType.(string)
	sourceRef, _ := rawSourceRef.(string)
	purchaseMode, _ := rawPurchaseMode.(string)
	issuanceId, _ := rawIssuanceID.(int)
	if strings.TrimSpace(sourceType) == "" || strings.TrimSpace(sourceRef) == "" {
		return "", "", "", 0, false
	}
	return strings.TrimSpace(sourceType), strings.TrimSpace(sourceRef), strings.TrimSpace(purchaseMode), issuanceId, true
}

func createBenefitChangeRecordTx(tx *gorm.DB, record *BenefitChangeRecord) error {
	if tx == nil {
		tx = DB
	}
	if record == nil {
		return nil
	}
	session := tx.Session(&gorm.Session{NewDB: true})
	record.BenefitType = strings.TrimSpace(record.BenefitType)
	record.Action = strings.TrimSpace(record.Action)
	record.SourceType = strings.TrimSpace(record.SourceType)
	record.SourceRef = strings.TrimSpace(record.SourceRef)
	record.TargetType = strings.TrimSpace(record.TargetType)
	err := session.Model(&BenefitChangeRecord{}).Create(record).Error
	if err != nil && isBenefitDuplicateKeyErr(err) {
		// 上游回调重放导致的重复发放/回退，被 idx_benefit_change_dedup 拦住属于
		// 预期情形，当作幂等成功处理，避免把数据库错误抛给业务方。
		return nil
	}
	return err
}

func isBenefitDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	// SQLite / MySQL / PostgreSQL 三种驱动返回的错误文案各不一样，这里做一次兜底匹配。
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "duplicate key")
}

func findBenefitGrantRecordsBySourceTx(tx *gorm.DB, sourceType string, sourceRef string) ([]*BenefitChangeRecord, error) {
	if tx == nil {
		tx = DB
	}
	session := tx.Session(&gorm.Session{NewDB: true})
	var records []*BenefitChangeRecord
	err := session.Model(&BenefitChangeRecord{}).Where("source_type = ? AND source_ref = ? AND action = ?",
		strings.TrimSpace(sourceType), strings.TrimSpace(sourceRef), BenefitActionGrant).
		Order("id desc").
		Find(&records).Error
	return records, err
}

func hasBenefitRollbackRecordForOriginTx(tx *gorm.DB, originRecordId int) (bool, error) {
	if tx == nil {
		tx = DB
	}
	session := tx.Session(&gorm.Session{NewDB: true})
	if originRecordId <= 0 {
		return false, nil
	}
	var count int64
	err := session.Model(&BenefitChangeRecord{}).
		Where("origin_record_id = ? AND action = ?", originRecordId, BenefitActionRollback).
		Count(&count).Error
	return count > 0, err
}

func ensureBenefitRollbackOperationTx(tx *gorm.DB, businessType string, businessId int, sourceType string, sourceRef string) (*BenefitRollbackOperation, error) {
	if tx == nil {
		tx = DB
	}
	session := tx.Session(&gorm.Session{NewDB: true})
	var op BenefitRollbackOperation
	result := session.Model(&BenefitRollbackOperation{}).Where("business_type = ? AND business_id = ?", strings.TrimSpace(businessType), businessId).Limit(1).Find(&op)
	if result.Error == nil && result.RowsAffected > 0 {
		return &op, nil
	}
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}
	op = BenefitRollbackOperation{
		BusinessType: strings.TrimSpace(businessType),
		BusinessId:   businessId,
		SourceType:   strings.TrimSpace(sourceType),
		SourceRef:    strings.TrimSpace(sourceRef),
		Status:       BenefitRollbackStatusPending,
	}
	if err := session.Model(&BenefitRollbackOperation{}).Create(&op).Error; err != nil {
		return nil, err
	}
	return &op, nil
}

func marshalBenefitDetail(detail any) string {
	if detail == nil {
		return ""
	}
	data, err := common.Marshal(detail)
	if err != nil {
		return ""
	}
	return string(data)
}

func unmarshalQuotaBenefitDetail(raw string) (*QuotaBenefitDetail, error) {
	if strings.TrimSpace(raw) == "" {
		return &QuotaBenefitDetail{}, nil
	}
	var detail QuotaBenefitDetail
	if err := common.UnmarshalJsonStr(raw, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func unmarshalSubscriptionBenefitDetail(raw string) (*SubscriptionBenefitDetail, error) {
	if strings.TrimSpace(raw) == "" {
		return &SubscriptionBenefitDetail{}, nil
	}
	var detail SubscriptionBenefitDetail
	if err := common.UnmarshalJsonStr(raw, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func normalizeSubscriptionBenefitSource(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case SubscriptionIssuanceSourceOrder:
		return BenefitSourceSubscriptionOrder
	case SubscriptionIssuanceSourceRedeem:
		return BenefitSourceSubscriptionRedemption
	default:
		raw := strings.TrimSpace(sourceType)
		if raw == "" {
			return ""
		}
		return "subscription_" + raw
	}
}

func subscriptionIssuanceSourceFromBenefitSource(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case BenefitSourceSubscriptionOrder:
		return SubscriptionIssuanceSourceOrder
	case BenefitSourceSubscriptionRedemption:
		return SubscriptionIssuanceSourceRedeem
	default:
		raw := strings.TrimSpace(sourceType)
		raw = strings.TrimPrefix(raw, "subscription_")
		return raw
	}
}
