package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"gorm.io/gorm"
)

// PulseBenefitLogReceipt lives in LOG_DB. Its insertion and the visible log
// share a transaction, so a crash before the main DB acknowledgement cannot
// duplicate the log. Keep receipts when usage logs are explicitly purged.
type PulseBenefitLogReceipt struct {
	BenefitRecordId int   `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt       int64 `gorm:"bigint"`
}

var errPulseLogDelivered = errors.New("pulse benefit log already delivered")

// SyncPulseBenefitLogs projects committed grant/reversal audits into usage
// logs, including historical rewards. An empty sourceRef runs a bounded batch.
// No quota mutation or current reward-policy check belongs in this path.
func SyncPulseBenefitLogs(sourceRef string) error {
	query := DB.Where("source_type = ? AND pulse_log_synced = ? AND benefit_type = ? AND action IN ?",
		BenefitSourcePulseReward, false, BenefitTypeQuota, []string{BenefitActionGrant, BenefitActionRollback})
	if sourceRef != "" {
		query = query.Where("source_ref = ?", sourceRef)
	}
	var records []BenefitChangeRecord
	if err := query.Order("id ASC").Limit(100).Find(&records).Error; err != nil {
		return err
	}
	var failures []error
	for i := range records {
		if err := syncPulseBenefitLog(&records[i]); err != nil {
			failures = append(failures, fmt.Errorf("benefit record %d: %w", records[i].Id, err))
		}
	}
	return errors.Join(failures...)
}

func syncPulseBenefitLog(record *BenefitChangeRecord) error {
	detail, err := unmarshalQuotaBenefitDetail(record.Detail)
	if err != nil {
		return err
	}
	label, sign, amount := "Meta Pulse 奖励到账", "+", detail.QuotaDelta
	if record.Action == BenefitActionRollback {
		label, sign, amount = "Meta Pulse 奖励撤销", "-", -detail.QuotaDelta
	}
	if amount <= 0 {
		return errors.New("invalid Pulse log quota delta")
	}
	var user User
	// Deleted users still have an audit history; do not strand their records.
	if err := DB.Unscoped().Select("username").Where("id = ?", record.UserId).Take(&user).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	entry := &Log{
		UserId: record.UserId, Username: user.Username, CreatedAt: record.CreatedAt,
		Type: LogTypeSystem, Quota: detail.QuotaDelta,
		Content: fmt.Sprintf("%s：%s%s；奖励编号：%s", label, sign, logger.FormatQuota(amount), record.SourceRef),
		Other: common.MapToJsonStr(map[string]interface{}{
			"pulse_reward": map[string]interface{}{
				"source_ref": record.SourceRef, "action": record.Action, "quota_delta": detail.QuotaDelta,
			},
		}),
	}
	err = LOG_DB.Transaction(func(tx *gorm.DB) error {
		receipt := &PulseBenefitLogReceipt{BenefitRecordId: record.Id, CreatedAt: common.GetTimestamp()}
		if err := tx.Create(receipt).Error; err != nil {
			if isBenefitDuplicateKeyErr(err) {
				return errPulseLogDelivered
			}
			return err
		}
		return tx.Create(entry).Error
	})
	if err != nil && !errors.Is(err, errPulseLogDelivered) {
		return err
	}
	// UpdateColumn deliberately preserves the original audit UpdatedAt.
	return DB.Model(&BenefitChangeRecord{}).Where("id = ?", record.Id).UpdateColumn("pulse_log_synced", true).Error
}

func syncPulseBenefitLogsAfterCommit(sourceRef string) {
	if err := SyncPulseBenefitLogs(sourceRef); err != nil {
		common.SysError("failed to sync Pulse benefit usage logs; will retry: " + err.Error())
	}
}
