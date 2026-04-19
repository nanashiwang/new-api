package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBenefitChangeRecordTx_SwallowsDuplicateGrant(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	record := &BenefitChangeRecord{
		BenefitType: BenefitTypeQuota,
		Action:      BenefitActionGrant,
		SourceType:  BenefitSourceTopUpOrder,
		SourceRef:   "DUP-TRADE-001",
		UserId:      101,
		TargetType:  BenefitTargetUserQuota,
		TargetId:    101,
		Detail:      `{"quota_delta":100}`,
	}
	require.NoError(t, createBenefitChangeRecordTx(DB, record))

	dup := &BenefitChangeRecord{
		BenefitType: BenefitTypeQuota,
		Action:      BenefitActionGrant,
		SourceType:  BenefitSourceTopUpOrder,
		SourceRef:   "DUP-TRADE-001",
		UserId:      101,
		TargetType:  BenefitTargetUserQuota,
		TargetId:    101,
		Detail:      `{"quota_delta":999}`,
	}
	require.NoError(t, createBenefitChangeRecordTx(DB, dup), "duplicate should be swallowed as idempotent success")

	var count int64
	require.NoError(t, DB.Model(&BenefitChangeRecord{}).
		Where("source_type = ? AND source_ref = ? AND action = ?",
			BenefitSourceTopUpOrder, "DUP-TRADE-001", BenefitActionGrant).
		Count(&count).Error)
	assert.EqualValues(t, 1, count)

	// 不同 action（rollback）不应被唯一索引拦下
	rollback := &BenefitChangeRecord{
		BenefitType:    BenefitTypeQuota,
		Action:         BenefitActionRollback,
		SourceType:     BenefitSourceTopUpOrder,
		SourceRef:      "DUP-TRADE-001",
		UserId:         101,
		TargetType:     BenefitTargetUserQuota,
		TargetId:       101,
		OriginRecordId: record.Id,
		Detail:         `{"quota_delta":-100}`,
	}
	require.NoError(t, createBenefitChangeRecordTx(DB, rollback))

	var totalCount int64
	require.NoError(t, DB.Model(&BenefitChangeRecord{}).
		Where("source_type = ? AND source_ref = ?",
			BenefitSourceTopUpOrder, "DUP-TRADE-001").
		Count(&totalCount).Error)
	assert.EqualValues(t, 2, totalCount)
}
