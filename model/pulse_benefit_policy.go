package model

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPulseBenefitPaused              = errors.New("pulse benefit grants are paused")
	ErrPulseBenefitPolicy              = errors.New("pulse benefit receiver policy is invalid")
	ErrPulseBenefitLimit               = errors.New("pulse benefit receiver quota limit exceeded")
	ErrPulseBenefitUserUnavailable     = errors.New("pulse benefit recipient is unavailable")
	ErrPulseBenefitInsufficientBalance = errors.New("pulse benefit rollback requires sufficient balance")
)

// These counters are receiver-side gross grant limits, independent of Pulse's
// budget. Reversals never replenish them. The singleton row also serializes
// grants across processes and all supported SQL databases; Redis is not used.
type PulseBenefitQuotaCounter struct {
	Scope string `gorm:"type:varchar(128);primaryKey"`
	Quota int64  `gorm:"not null;default:0"`
}

type pulseBenefitPolicy struct{ maxGrant, userDaily, daily int64 }

func loadPulseBenefitPolicy() (pulseBenefitPolicy, error) {
	var policy pulseBenefitPolicy
	switch os.Getenv("PULSE_BENEFIT_ENABLED") {
	case "true":
	case "", "false":
		return policy, ErrPulseBenefitPaused
	default:
		return policy, ErrPulseBenefitPolicy
	}
	for name, target := range map[string]*int64{
		"PULSE_BENEFIT_MAX_GRANT_QUOTA":  &policy.maxGrant,
		"PULSE_BENEFIT_USER_DAILY_QUOTA": &policy.userDaily,
		"PULSE_BENEFIT_DAILY_QUOTA":      &policy.daily,
	} {
		raw := os.Getenv(name)
		// Canonical decimal integers only: no signs, fractions or exponent notation.
		value, err := strconv.ParseInt(raw, 10, strconv.IntSize)
		if err != nil || value <= 0 || strconv.FormatInt(value, 10) != raw {
			return policy, fmt.Errorf("%w: %s", ErrPulseBenefitPolicy, name)
		}
		*target = value
	}
	return policy, nil
}

func lockPulseBenefitReceiverTx(tx *gorm.DB) error {
	row := PulseBenefitQuotaCounter{Scope: "receiver-lock"}
	// This write MUST precede any reads. Besides acquiring a row lock on MySQL
	// and PostgreSQL it reserves SQLite's single writer before taking a snapshot.
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return err
	}
	return tx.Model(&PulseBenefitQuotaCounter{}).Where("scope = ?", row.Scope).
		UpdateColumn("quota", gorm.Expr("quota")).Error
}

// Activity days consistently use UTC+08:00, including hosts configured in UTC.
func pulseBenefitDay(now time.Time) (string, int64, int64) {
	local := now.In(time.FixedZone("Asia/Shanghai", 8*60*60))
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	return start.Format("2006-01-02"), start.Unix(), start.Add(24 * time.Hour).Unix()
}

func reservePulseBenefitQuotaTx(tx *gorm.DB, req PulseBenefitGrantRequest, policy pulseBenefitPolicy) error {
	amount := int64(req.Amount)
	if amount > policy.maxGrant {
		return ErrPulseBenefitLimit
	}
	day, start, end := pulseBenefitDay(time.Now())
	for _, limit := range []struct {
		scope  string
		userID int
		quota  int64
	}{
		{"day:" + day + ":all", 0, policy.daily},
		{"day:" + day + ":user:" + strconv.Itoa(req.UserID), req.UserID, policy.userDaily},
	} {
		counter, err := loadPulseBenefitCounterTx(tx, limit.scope, limit.userID, start, end)
		if err != nil {
			return err
		}
		// Subtraction avoids overflowing either a quota balance or the counter.
		if amount > limit.quota || counter.Quota > limit.quota-amount {
			return ErrPulseBenefitLimit
		}
		if err := tx.Model(&PulseBenefitQuotaCounter{}).Where("scope = ?", limit.scope).
			UpdateColumn("quota", gorm.Expr("quota + ?", amount)).Error; err != nil {
			return err
		}
	}
	return nil
}

func loadPulseBenefitCounterTx(tx *gorm.DB, scope string, userID int, start, end int64) (*PulseBenefitQuotaCounter, error) {
	var counter PulseBenefitQuotaCounter
	if err := tx.Where("scope = ?", scope).First(&counter).Error; err == nil {
		return &counter, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// Include grants made by older releases before counters existed. Immutable
	// grant records also allow reconstructing a deleted counter without resetting
	// its limit. Rollback records intentionally do not reduce gross issued quota.
	query := tx.Model(&BenefitChangeRecord{}).Select("detail").Where(
		"source_type = ? AND action = ? AND created_at >= ? AND created_at < ?",
		BenefitSourcePulseReward, BenefitActionGrant, start, end)
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	var grants []BenefitChangeRecord
	if err := query.Find(&grants).Error; err != nil {
		return nil, err
	}
	counter.Scope = scope
	for _, grant := range grants {
		detail, err := unmarshalQuotaBenefitDetail(grant.Detail)
		if err != nil {
			return nil, err
		}
		if detail == nil || detail.QuotaDelta <= 0 {
			return nil, ErrPulseBenefitPolicy
		}
		amount := int64(detail.QuotaDelta)
		if counter.Quota > int64(^uint64(0)>>1)-amount {
			return nil, ErrPulseBenefitLimit
		}
		counter.Quota += amount
	}
	if err := tx.Create(&counter).Error; err != nil {
		return nil, err
	}
	return &counter, nil
}

func validatePulseBenefitRecipientTx(tx *gorm.DB, userID, amount int) error {
	query := tx.Select("id", "status", "quota", "pulse_reward_hold").Where("id = ?", userID)
	if !common.UsingSQLite {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var user User
	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPulseBenefitUserUnavailable
		}
		return err
	}
	if user.Status != common.UserStatusEnabled || user.PulseRewardHold {
		return ErrPulseBenefitUserUnavailable
	}
	if user.Quota > int(^uint(0)>>1)-amount {
		return ErrPulseBenefitLimit
	}
	return nil
}

// Ordinary activity corrections may never create debt. Payment fraud/chargeback
// reversals keep their independent recovery semantics in the generic engine.
func revokePulseQuotaGrantTx(tx *gorm.DB, userID, quota int) error {
	query := tx.Select("id", "quota").Where("id = ?", userID)
	if !common.UsingSQLite {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var user User
	if err := query.First(&user).Error; err != nil {
		return err
	}
	if quota <= 0 || user.Quota < quota {
		return ErrPulseBenefitInsufficientBalance
	}
	return RevokeQuotaGrantTx(tx, userID, quota)
}
