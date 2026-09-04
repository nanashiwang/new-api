package model

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrantPulseBenefitIsIdempotentAndConflictsOnPayloadChange(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-benefit-idempotent")
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"quota": 100, "transferable_quota": 100,
	}).Error)

	request := PulseBenefitGrantRequest{
		GrantID: "pulse-grant-1", UserID: user.Id, Amount: 50,
		SourceRef: "pulse-grant-1", RewardType: "newapi_quota",
	}
	first, err := GrantPulseBenefit(request)
	require.NoError(t, err)
	require.Equal(t, "applied", first.Status)

	second, err := GrantPulseBenefit(request)
	require.NoError(t, err)
	require.Equal(t, "already_applied", second.Status)

	conflicting := request
	conflicting.Amount = 51
	_, err = GrantPulseBenefit(conflicting)
	require.ErrorIs(t, err, ErrPulseBenefitConflict)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Equal(t, 150, refreshed.Quota)
	var count int64
	require.NoError(t, DB.Model(&BenefitChangeRecord{}).Where(
		"source_type = ? AND source_ref = ? AND action = ?",
		BenefitSourcePulseReward, request.SourceRef, BenefitActionGrant,
	).Count(&count).Error)
	require.EqualValues(t, 1, count)
	var record BenefitChangeRecord
	require.NoError(t, DB.Where("source_type = ? AND source_ref = ?", BenefitSourcePulseReward, request.SourceRef).First(&record).Error)
	require.Equal(t, user.Id, record.TargetId)
	var receiptCount int64
	require.NoError(t, DB.Model(&PulseBenefitReceipt{}).Where("source_ref = ?", request.SourceRef).Count(&receiptCount).Error)
	require.EqualValues(t, 1, receiptCount)
}

func TestRollbackPulseBenefitUsesOriginalSourceAndIsIdempotent(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-benefit-rollback")
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"quota": 100, "transferable_quota": 100,
	}).Error)
	request := PulseBenefitGrantRequest{
		GrantID: "pulse-grant-rollback", UserID: user.Id, Amount: 40,
		SourceRef: "pulse-grant-rollback", RewardType: "content",
	}
	require.NoError(t, func() error { _, err := GrantPulseBenefit(request); return err }())

	first, err := RollbackPulseBenefit(request.SourceRef, "人工撤销")
	require.NoError(t, err)
	require.Equal(t, "rolled_back", first.Status)
	second, err := RollbackPulseBenefit(request.SourceRef, "重复撤销")
	require.NoError(t, err)
	require.Equal(t, "rolled_back", second.Status)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Equal(t, 100, refreshed.Quota)
	require.Equal(t, 100, refreshed.TransferableQuota)
	var count int64
	require.NoError(t, DB.Model(&BenefitChangeRecord{}).Where(
		"source_type = ? AND source_ref = ? AND action = ?",
		BenefitSourcePulseReward, request.SourceRef, BenefitActionRollback,
	).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestGrantPulseBenefitReplayOneHundredTimesAddsQuotaOnce(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-benefit-replay-100")
	request := PulseBenefitGrantRequest{
		GrantID: "pulse-grant-replay-100", UserID: user.Id, Amount: 25,
		SourceRef: "pulse-grant-replay-100", RewardType: "period",
	}
	for i := 0; i < 100; i++ {
		result, err := GrantPulseBenefit(request)
		require.NoError(t, err)
		require.True(t, result.Applied)
	}

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Equal(t, 25, refreshed.Quota)
	var count int64
	require.NoError(t, DB.Model(&BenefitChangeRecord{}).Where(
		"source_type = ? AND source_ref = ? AND action = ?",
		BenefitSourcePulseReward, request.SourceRef, BenefitActionGrant,
	).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestGrantPulseBenefitRejectsInvalidPayloadHashAndIdentity(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-benefit-invalid")
	request := PulseBenefitGrantRequest{
		GrantID: "pulse-grant-invalid", UserID: user.Id, Amount: 10,
		SourceRef: "pulse-grant-invalid", RewardType: "period",
		PayloadHash: "not-the-server-fingerprint",
	}
	_, err := GrantPulseBenefit(request)
	require.ErrorIs(t, err, ErrPulseBenefitConflict)

	request.PayloadHash = ""
	request.GrantID = "different-grant-id"
	_, err = GrantPulseBenefit(request)
	require.Error(t, err)
}

func TestPulseBenefitFingerprintMatchesSettlementPayload(t *testing.T) {
	request := PulseBenefitGrantRequest{
		GrantID: "grant-fingerprint", UserID: 42, Amount: 99,
		SourceRef: "grant-fingerprint", RewardType: "period",
	}
	fingerprint, err := pulseBenefitFingerprint(request)
	require.NoError(t, err)
	digest := sha256.Sum256([]byte(`{"amount":99,"grant_id":"grant-fingerprint","reward_type":"period","source_ref":"grant-fingerprint","transferable_quota":false,"user_id":42}`))
	require.Equal(t, fmt.Sprintf("%x", digest[:]), fingerprint)
}

func TestCanonicalPulseBenefitJSONIgnoresObjectOrderWhitespaceAndPreservesNumbers(t *testing.T) {
	first, err := canonicalPulseBenefitJSON([]byte(` { "z": 9007199254740993, "a": { "b": 2, "a": 1 } } `))
	require.NoError(t, err)
	second, err := canonicalPulseBenefitJSON([]byte(`{"a":{"a":1,"b":2},"z":9007199254740993}`))
	require.NoError(t, err)
	require.Equal(t, second, first)
	require.Equal(t, `{"a":{"a":1,"b":2},"z":9007199254740993}`, string(first))
}

func TestCanonicalPulseBenefitJSONRejectsTrailingJSON(t *testing.T) {
	_, err := canonicalPulseBenefitJSON([]byte(`{"grant_id":"one"}{"grant_id":"two"}`))
	require.Error(t, err)
}
