package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	InviteCommissionStatusPending = "pending"
	InviteCommissionStatusSettled = "settled"
	InviteCommissionStatusSkipped = "skipped"
)

const inviteCommissionCapCasMaxRetry = 8

const (
	// 风控原因：邀请人与被邀请人相同（自邀）。
	InviteCommissionRiskReasonSelfInvite = "self_invite"
	// 风控原因：邀请人当日返佣已达上限。
	InviteCommissionRiskReasonDailyCapReached = "daily_cap_reached"
	// 风控原因：本单返佣被当日上限截断，仅部分发放。
	InviteCommissionRiskReasonDailyCapTruncated = "daily_cap_truncated"
)

var errInviteCommissionAlreadyProcessed = errors.New("invite commission ledger already processed")

// InviteCommissionLedger 记录邀请充值返佣的完整生命周期。
//
// 生命周期：
// 1. pending：充值成功后入池，等待日批结算。
// 2. settled：T+1 任务已将返佣发放到邀请人的 aff_quota / aff_history。
// 3. skipped：被风控跳过（例如自邀、单日上限已满）。
//
// 幂等策略：
// - 同一 trade_no + inviter_user_id 只允许一条台账（唯一索引）。
// - 结算时仅处理 status=pending 的记录，重复执行不会重复入账。
type InviteCommissionLedger struct {
	Id              int     `json:"id"`
	InviteeUserId   int     `json:"invitee_user_id" gorm:"index;not null"`
	InviterUserId   int     `json:"inviter_user_id" gorm:"index;not null;uniqueIndex:idx_invite_commission_trade_inviter"`
	TopupTradeNo    string  `json:"topup_trade_no" gorm:"type:varchar(255);not null;uniqueIndex:idx_invite_commission_trade_inviter"`
	BizDate         string  `json:"biz_date" gorm:"type:varchar(10);index;not null"` // 业务日期（YYYY-MM-DD）
	BaseQuota       int     `json:"base_quota" gorm:"type:int;not null;default:0"`
	CommissionRate  float64 `json:"commission_rate" gorm:"type:decimal(10,6);not null;default:0"`
	CommissionQuota int     `json:"commission_quota" gorm:"type:int;not null;default:0"`
	SettledQuota    int     `json:"settled_quota" gorm:"type:int;not null;default:0"`
	Status          string  `json:"status" gorm:"type:varchar(16);index;not null"`
	RiskReason      string  `json:"risk_reason" gorm:"type:varchar(64);default:''"`
	CreatedAt       int64   `json:"created_at" gorm:"index"`
	SettledAt       int64   `json:"settled_at"`
}

// InviteCommissionDailyCapState 记录 inviter + bizDate 的当日已结算返佣额度。
type InviteCommissionDailyCapState struct {
	Id            int    `json:"id"`
	InviterUserId int    `json:"inviter_user_id" gorm:"not null;uniqueIndex:idx_invite_commission_daily_cap_inviter_date"`
	BizDate       string `json:"biz_date" gorm:"type:varchar(10);not null;uniqueIndex:idx_invite_commission_daily_cap_inviter_date"`
	SettledQuota  int    `json:"settled_quota" gorm:"type:int;not null;default:0"`
	CreatedAt     int64  `json:"created_at" gorm:"index"`
	UpdatedAt     int64  `json:"updated_at" gorm:"index"`
}

func EnqueueInviteCommissionFromTopUp(topUp *TopUp, baseQuota int) error {
	// 基础防护：仅合法充值额度允许入池。
	if topUp == nil {
		return nil
	}
	return enqueueInviteCommission(topUp.UserId, topUp.TradeNo, topUp.CompleteTime, baseQuota)
}

// EnqueueInviteCommissionFromRedemption 将“兑换码充值”纳入邀请返佣口径。
// 管理员直接修改余额不会调用该入口，因此不计入返佣。
func EnqueueInviteCommissionFromRedemption(redemption *Redemption) error {
	return EnqueueInviteCommissionFromRedemptionTx(DB, redemption)
}

// EnqueueInviteCommissionFromSubscriptionOrderTx 将“订阅支付成功”纳入邀请返佣口径。
// 返佣基数按“实付金额折算额度”计算：
// 返佣基数公式：baseQuota = floor(order.money * QuotaPerUnit / Price)
// 其中 Price 为“充值价格（x元/美金）”。
func EnqueueInviteCommissionFromSubscriptionOrderTx(tx *gorm.DB, order *SubscriptionOrder) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if order == nil || order.Id <= 0 {
		return nil
	}
	if !common.InviterCommissionEnabled || common.InviterRechargeCommissionRate <= 0 {
		return nil
	}
	if operation_setting.Price <= 0 {
		common.SysError(fmt.Sprintf("skip invite commission for subscription order %d: invalid payment price setting", order.Id))
		return nil
	}

	// 仅信任 order.id；其余字段从 DB 读取，避免调用方传入被篡改数据。
	dbOrder := &SubscriptionOrder{}
	if err := tx.Select("id", "user_id", "trade_no", "money", "complete_time", "status").
		First(dbOrder, "id = ?", order.Id).Error; err != nil {
		return err
	}
	// 仅已支付成功订单允许入返佣池。
	if dbOrder.Status != common.TopUpStatusSuccess {
		return nil
	}

	dBaseQuota := decimal.NewFromFloat(dbOrder.Money).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Div(decimal.NewFromFloat(operation_setting.Price))
	baseQuota := int(dBaseQuota.IntPart())
	if baseQuota <= 0 {
		return nil
	}

	return enqueueInviteCommissionWithDB(tx, dbOrder.UserId, dbOrder.TradeNo, dbOrder.CompleteTime, baseQuota)
}

// EnqueueInviteCommissionFromRedemptionTx 在指定事务中入返佣台账，用于保证兑换与返佣原子一致。
func EnqueueInviteCommissionFromRedemptionTx(tx *gorm.DB, redemption *Redemption) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if redemption == nil || redemption.Id <= 0 {
		return nil
	}
	// 仅信任 redemption.id；其余字段统一从数据库读取，避免调用方传入伪造数据。
	dbRedemption := &Redemption{}
	if err := tx.Select("id", "used_user_id", "quota", "redeemed_time", "status", "benefit_type", "plan_id").
		First(dbRedemption, "id = ?", redemption.Id).Error; err != nil {
		return err
	}
	// 仅已兑换成功的兑换码允许参与返佣入池。
	if dbRedemption.Status != common.RedemptionCodeStatusUsed {
		return nil
	}

	// 返佣基数随兑换码权益类型变化：余额码按额度，套餐码按当前套餐售价折算。
	// 这样可以保证“买套餐”和“兑套餐”走同一套返佣口径。
	baseQuota, err := getRedemptionCommissionBaseQuotaTx(tx, dbRedemption)
	if err != nil {
		return err
	}
	// redemption.id 全局唯一，作为 trade_no 可保证幂等去重。
	tradeNo := fmt.Sprintf("redeem:%d", dbRedemption.Id)
	return enqueueInviteCommissionWithDB(tx, dbRedemption.UsedUserId, tradeNo, dbRedemption.RedeemedTime, baseQuota)
}

// getRedemptionCommissionBaseQuotaTx 统一计算兑换码返佣基数。
// 余额码直接使用兑换额度；套餐码按当前套餐售价折算额度。
func getRedemptionCommissionBaseQuotaTx(tx *gorm.DB, redemption *Redemption) (int, error) {
	if redemption == nil {
		return 0, nil
	}
	if NormalizeRedemptionBenefitType(redemption.BenefitType) != RedemptionBenefitTypeSubscription {
		// 余额码保持旧口径：返佣基数直接等于兑换到账额度。
		return redemption.Quota, nil
	}
	if redemption.PlanId <= 0 {
		return 0, nil
	}
	if operation_setting.Price <= 0 {
		return 0, errors.New("invalid payment price setting")
	}
	plan, err := getSubscriptionPlanByIdTx(tx, redemption.PlanId)
	if err != nil {
		return 0, err
	}
	if plan == nil || plan.PriceAmount <= 0 {
		// 当前套餐无有效售价时，不再给套餐兑换码产生返佣基数。
		// 这里选择“不给返佣”而不是报错，是为了避免后台临时调整展示价导致历史码无法兑换。
		return 0, nil
	}
	// 套餐码返佣和付费订阅保持同一套折算公式，避免不同入口口径不一致。
	dBaseQuota := decimal.NewFromFloat(plan.PriceAmount).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Div(decimal.NewFromFloat(operation_setting.Price))
	return int(dBaseQuota.IntPart()), nil
}

func enqueueInviteCommission(inviteeUserID int, tradeNo string, completeTime int64, baseQuota int) error {
	return enqueueInviteCommissionWithDB(DB, inviteeUserID, tradeNo, completeTime, baseQuota)
}

func enqueueInviteCommissionWithDB(db *gorm.DB, inviteeUserID int, tradeNo string, completeTime int64, baseQuota int) error {
	if db == nil {
		return errors.New("db is nil")
	}
	// 基础防护：仅合法充值额度允许入池。
	if inviteeUserID <= 0 || tradeNo == "" || baseQuota <= 0 {
		return nil
	}
	// 功能开关与比例检查。
	if !common.InviterCommissionEnabled || common.InviterRechargeCommissionRate <= 0 {
		return nil
	}

	// 在入池时快照 inviter_id，避免后续用户关系变更影响历史订单归属。
	invitee := &User{}
	if err := db.Select("id", "inviter_id").First(invitee, "id = ?", inviteeUserID).Error; err != nil {
		return err
	}
	if invitee.InviterId == 0 {
		return nil
	}
	if invitee.InviterId == invitee.Id {
		return nil
	}

	// 返佣额度取整向下，避免浮点误差导致超发。
	commissionQuota := int(decimal.NewFromInt(int64(baseQuota)).Mul(decimal.NewFromFloat(common.InviterRechargeCommissionRate)).IntPart())
	if commissionQuota <= 0 {
		return nil
	}

	if completeTime == 0 {
		completeTime = common.GetTimestamp()
	}

	ledger := &InviteCommissionLedger{
		InviteeUserId:   invitee.Id,
		InviterUserId:   invitee.InviterId,
		TopupTradeNo:    tradeNo,
		BizDate:         time.Unix(completeTime, 0).Format("2006-01-02"),
		BaseQuota:       baseQuota,
		CommissionRate:  common.InviterRechargeCommissionRate,
		CommissionQuota: commissionQuota,
		Status:          InviteCommissionStatusPending,
		CreatedAt:       common.GetTimestamp(),
	}

	return db.Clauses(clause.OnConflict{
		// 幂等保障：同一个订单对同一个邀请人只允许入池一次。
		Columns: []clause.Column{
			{Name: "topup_trade_no"},
			{Name: "inviter_user_id"},
		},
		DoNothing: true,
	}).Create(ledger).Error
}

func SettleInviteCommissionByBizDate(bizDate string, batchSize int) (settledCount int, skippedCount int, processedCount int, err error) {
	// 防御性检查：避免错误参数导致全表扫。
	if batchSize <= 0 {
		return 0, 0, 0, errors.New("batch size must be positive")
	}

	// 日批按 id 顺序处理，便于定位问题与保证处理顺序稳定。
	var ledgers []*InviteCommissionLedger
	if err = DB.Where("status = ? AND biz_date <= ?", InviteCommissionStatusPending, bizDate).
		Order("id asc").
		Limit(batchSize).
		Find(&ledgers).Error; err != nil {
		return 0, 0, 0, err
	}
	if len(ledgers) == 0 {
		return 0, 0, 0, nil
	}

	dailyCap := common.InviterCommissionDailyCap

	for _, ledger := range ledgers {
		if ledger == nil {
			continue
		}

		processed, settled, settleErr := settleSingleInviteCommissionLedger(ledger, dailyCap)
		if settleErr != nil {
			return settledCount, skippedCount, processedCount, settleErr
		}
		if !processed {
			continue
		}

		processedCount++
		if settled {
			settledCount++
		} else {
			skippedCount++
		}
	}

	return settledCount, skippedCount, processedCount, nil
}

func settleSingleInviteCommissionLedger(ledger *InviteCommissionLedger, dailyCap int) (processed bool, settled bool, err error) {
	now := common.GetTimestamp()
	allowedQuota := ledger.CommissionQuota
	riskReason := ""

	err = DB.Transaction(func(tx *gorm.DB) error {
		if ledger.InviterUserId == ledger.InviteeUserId {
			allowedQuota = 0
			riskReason = InviteCommissionRiskReasonSelfInvite
		}

		// 在事务内做日上限 CAS 预占，避免并发超发。
		if allowedQuota > 0 && dailyCap > 0 {
			var truncated bool
			allowedQuota, truncated, err = reserveInviteCommissionDailyCapTx(tx, ledger.InviterUserId, ledger.BizDate, allowedQuota, dailyCap)
			if err != nil {
				return err
			}
			if allowedQuota == 0 {
				riskReason = InviteCommissionRiskReasonDailyCapReached
			} else if truncated {
				riskReason = InviteCommissionRiskReasonDailyCapTruncated
			}
		}

		targetStatus := InviteCommissionStatusSkipped
		if allowedQuota > 0 {
			targetStatus = InviteCommissionStatusSettled
		}

		// CAS 风格状态流转：只允许 pending -> settled/skipped 一次。
		updateResult := tx.Model(&InviteCommissionLedger{}).
			Where("id = ? AND status = ?", ledger.Id, InviteCommissionStatusPending).
			Updates(map[string]interface{}{
				"status":        targetStatus,
				"settled_quota": allowedQuota,
				"risk_reason":   riskReason,
				"settled_at":    now,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			// 并发重跑命中时，回滚事务中的日上限预占。
			return errInviteCommissionAlreadyProcessed
		}

		processed = true
		if targetStatus != InviteCommissionStatusSettled {
			return nil
		}

		settled = true
		// 保持现有邀请体系体验：返佣先进邀请额度，用户再手动划转到余额。
		if err := tx.Model(&User{}).Where("id = ?", ledger.InviterUserId).Updates(map[string]interface{}{
			"aff_quota":   gorm.Expr("aff_quota + ?", allowedQuota),
			"aff_history": gorm.Expr("aff_history + ?", allowedQuota),
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errInviteCommissionAlreadyProcessed) {
			return false, false, nil
		}
		return false, false, err
	}

	if processed && settled {
		RecordLog(
			ledger.InviterUserId,
			LogTypeSystem,
			fmt.Sprintf("邀请返佣到账 %s（订单:%s）", logger.LogQuota(allowedQuota), maskTradeNoForLog(ledger.TopupTradeNo)),
		)
	}

	return processed, settled, nil
}

func reserveInviteCommissionDailyCapTx(tx *gorm.DB, inviterUserId int, bizDate string, requestQuota int, dailyCap int) (grantedQuota int, truncated bool, err error) {
	if requestQuota <= 0 || dailyCap <= 0 {
		return requestQuota, false, nil
	}
	if err := ensureInviteCommissionDailyCapStateTx(tx, inviterUserId, bizDate); err != nil {
		return 0, false, err
	}

	for i := 0; i < inviteCommissionCapCasMaxRetry; i++ {
		state := &InviteCommissionDailyCapState{}
		if err := tx.Select("inviter_user_id", "biz_date", "settled_quota").First(state, "inviter_user_id = ? AND biz_date = ?", inviterUserId, bizDate).Error; err != nil {
			return 0, false, err
		}
		if state.SettledQuota >= dailyCap {
			return 0, false, nil
		}

		remain := dailyCap - state.SettledQuota
		grantedQuota = requestQuota
		if grantedQuota > remain {
			grantedQuota = remain
			truncated = true
		}

		now := common.GetTimestamp()
		updateResult := tx.Model(&InviteCommissionDailyCapState{}).
			Where("inviter_user_id = ? AND biz_date = ? AND settled_quota = ?", inviterUserId, bizDate, state.SettledQuota).
			Updates(map[string]interface{}{
				"settled_quota": state.SettledQuota + grantedQuota,
				"updated_at":    now,
			})
		if updateResult.Error != nil {
			return 0, false, updateResult.Error
		}
		if updateResult.RowsAffected > 0 {
			return grantedQuota, truncated, nil
		}
	}
	return 0, false, fmt.Errorf("reserve invite commission daily cap retry exhausted: inviter=%d biz_date=%s", inviterUserId, bizDate)
}

func ensureInviteCommissionDailyCapStateTx(tx *gorm.DB, inviterUserId int, bizDate string) error {
	now := common.GetTimestamp()
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "inviter_user_id"},
			{Name: "biz_date"},
		},
		DoNothing: true,
	}).Create(&InviteCommissionDailyCapState{
		InviterUserId: inviterUserId,
		BizDate:       bizDate,
		SettledQuota:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error
}

func maskTradeNoForLog(tradeNo string) string {
	if tradeNo == "" {
		return "-"
	}
	if len(tradeNo) <= 10 {
		return "***"
	}
	return tradeNo[:6] + "***" + tradeNo[len(tradeNo)-4:]
}
