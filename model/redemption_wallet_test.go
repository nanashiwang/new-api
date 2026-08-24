package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestWalletRedemptionLocalDayRangeUsesBeijingTime(t *testing.T) {
	bizDate, start, end := walletRedemptionLocalDayRange(time.Date(2026, time.August, 24, 15, 59, 0, 0, time.UTC))
	assert.Equal(t, "2026-08-24", bizDate)
	assert.Equal(t, int64(24*60*60), end-start)
	assert.Equal(t, "2026-08-24 00:00:00", time.Unix(start, 0).In(walletRedemptionLocation).Format("2006-01-02 15:04:05"))
}

func TestEnsureRedemptionColumnsSQLite_AddsWalletFundingColumnsWithLegacyDefault(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE redemptions (id integer primary key, `key` char(32), status integer)").Error)
	require.NoError(t, db.Exec("INSERT INTO redemptions (id, `key`, status) VALUES (1, 'legacy-code', 1)").Error)

	originalDB := DB
	originalUsingSQLite := common.UsingSQLite
	DB = db
	common.UsingSQLite = true
	t.Cleanup(func() {
		DB = originalDB
		common.UsingSQLite = originalUsingSQLite
	})

	require.NoError(t, ensureRedemptionColumnsSQLite())
	assert.True(t, db.Migrator().HasColumn("redemptions", "funding_source"))
	assert.True(t, db.Migrator().HasColumn("redemptions", "transferable_quota"))
	assert.True(t, db.Migrator().HasColumn("redemptions", "create_request_id"))
	var fundingSource string
	require.NoError(t, db.Raw("SELECT funding_source FROM redemptions WHERE id = 1").Scan(&fundingSource).Error)
	assert.Equal(t, RedemptionFundingSourceAdmin, fundingSource)
}

func setupWalletRedemptionTest(t *testing.T) {
	t.Helper()
	setupInviteCommissionSubscriptionTest(t)
	originalQuotaPerUnit := common.QuotaPerUnit
	originalBindingSettings := common.GetInviteBindingSettings()
	originalDailyCreateLimit := common.WalletRedemptionDailyCreateLimit
	originalMinimumQuota := common.WalletRedemptionMinimumQuota
	originalActiveLimit := common.WalletRedemptionActiveLimit
	originalDailyQuotaLimit := common.WalletRedemptionDailyQuotaLimit
	originalReviewThreshold := common.WalletRedemptionReviewDistinctCreatorThreshold
	originalReviewSmallLimit := common.WalletRedemptionReviewSmallQuotaLimit
	common.QuotaPerUnit = 10
	common.WalletRedemptionDailyCreateLimit = 100
	common.WalletRedemptionMinimumQuota = 10
	common.WalletRedemptionActiveLimit = 100
	common.WalletRedemptionDailyQuotaLimit = 5000
	common.WalletRedemptionReviewDistinctCreatorThreshold = 3
	common.WalletRedemptionReviewSmallQuotaLimit = 100
	require.NoError(t, common.SetInviteBindingSettings(common.InviteBindingSettings{
		Threshold:          0,
		RateAfterThreshold: 100,
	}))
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
		common.WalletRedemptionDailyCreateLimit = originalDailyCreateLimit
		common.WalletRedemptionMinimumQuota = originalMinimumQuota
		common.WalletRedemptionActiveLimit = originalActiveLimit
		common.WalletRedemptionDailyQuotaLimit = originalDailyQuotaLimit
		common.WalletRedemptionReviewDistinctCreatorThreshold = originalReviewThreshold
		common.WalletRedemptionReviewSmallQuotaLimit = originalReviewSmallLimit
		_ = common.SetInviteBindingSettings(originalBindingSettings)
	})
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Redemption{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RedemptionReviewCase{}).Error)
}

func setWalletQuota(t *testing.T, userID int, quota int) {
	t.Helper()
	require.NoError(t, DB.Model(&User{}).Where("id = ?", userID).Updates(map[string]any{
		"quota":              quota,
		"transferable_quota": quota,
	}).Error)
}

func setNonTransferableWalletQuota(t *testing.T, userID int, quota int) {
	t.Helper()
	require.NoError(t, DB.Model(&User{}).Where("id = ?", userID).Updates(map[string]any{
		"quota":              quota,
		"transferable_quota": 0,
	}).Error)
}

func countInviteCommissionLedgers(t *testing.T) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&InviteCommissionLedger{}).Count(&count).Error)
	return count
}

func TestCreateWalletFundedRedemption_DeductsQuotaAndReplaysIdempotently(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_create", 0)
	setWalletQuota(t, creator.Id, 1000)

	first, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-create-request-001")
	require.NoError(t, err)
	require.NotNil(t, first.Redemption)
	assert.False(t, first.Replayed)
	assert.Equal(t, 700, first.RemainingQuota)
	assert.Equal(t, 300, first.Redemption.Quota)

	second, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-create-request-001")
	require.NoError(t, err)
	assert.True(t, second.Replayed)
	assert.Equal(t, first.Redemption.Id, second.Redemption.Id)
	assert.Equal(t, first.Redemption.Key, second.Redemption.Key)
	assert.Equal(t, 700, second.RemainingQuota)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 700, refreshed.Quota)
	assert.Equal(t, 700, refreshed.TransferableQuota)
	var redemptions []Redemption
	require.NoError(t, DB.Find(&redemptions).Error)
	require.Len(t, redemptions, 1)
	assert.Equal(t, RedemptionFundingSourceWallet, redemptions[0].FundingSource)
	assert.NotNil(t, redemptions[0].CreateRequestId)
}

func TestCreateWalletFundedRedemption_RejectsNonTransferableAndTinyQuota(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_non_transferable", 0)
	setNonTransferableWalletQuota(t, creator.Id, 1000)

	_, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-free-request-001")
	assert.ErrorIs(t, err, ErrRedemptionInsufficientTransferableQuota)

	setWalletQuota(t, creator.Id, 1000)
	_, err = CreateWalletFundedRedemption(creator.Id, MinimumWalletRedemptionQuota()-1, "wallet-tiny-request-001")
	assert.ErrorIs(t, err, ErrRedemptionBelowMinimum)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 1000, refreshed.Quota)
	assert.Equal(t, 1000, refreshed.TransferableQuota)
}

func TestCreateWalletFundedRedemption_EnforcesDailyLimits(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_daily_limits", 0)
	setWalletQuota(t, creator.Id, 10_000)
	common.WalletRedemptionDailyCreateLimit = 2
	common.WalletRedemptionDailyQuotaLimit = 0

	_, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-daily-count-001")
	require.NoError(t, err)
	_, err = CreateWalletFundedRedemption(creator.Id, 100, "wallet-daily-count-002")
	require.NoError(t, err)
	_, err = CreateWalletFundedRedemption(creator.Id, 100, "wallet-daily-count-003")
	assert.ErrorIs(t, err, ErrRedemptionDailyCreateLimit)

	common.WalletRedemptionDailyCreateLimit = 100
	common.WalletRedemptionDailyQuotaLimit = 15
	other := createInviteCommissionTestUser(t, "wallet_daily_quota", 0)
	setWalletQuota(t, other.Id, 10_000)
	_, err = CreateWalletFundedRedemption(other.Id, 100, "wallet-daily-quota-001")
	require.NoError(t, err)
	_, err = CreateWalletFundedRedemption(other.Id, 100, "wallet-daily-quota-002")
	assert.ErrorIs(t, err, ErrRedemptionDailyQuotaLimit)
}

func TestCreateWalletFundedRedemption_DailyLimitIsAtomic(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_daily_atomic", 0)
	setWalletQuota(t, creator.Id, 1000)
	common.WalletRedemptionDailyCreateLimit = 1
	common.WalletRedemptionDailyQuotaLimit = 0

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := CreateWalletFundedRedemption(creator.Id, 100, fmt.Sprintf("wallet-daily-atomic-%03d", index))
			errs <- err
		}(index)
	}
	wg.Wait()
	close(errs)
	succeeded, limited := 0, 0
	for err := range errs {
		if err == nil {
			succeeded++
		} else if errors.Is(err, ErrRedemptionDailyCreateLimit) {
			limited++
		} else {
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	assert.Equal(t, 1, succeeded)
	assert.Equal(t, 1, limited)
}

func TestWalletRedemption_OtherUserQuotaCannotBeTransferredAgain(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_one_hop_creator", 0)
	recipient := createInviteCommissionTestUser(t, "wallet_one_hop_recipient", 0)
	setWalletQuota(t, creator.Id, 1000)
	setWalletQuota(t, recipient.Id, 0)

	created, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-one-hop-001")
	require.NoError(t, err)
	_, err = RedeemWithResult(created.Redemption.Key, recipient.Id)
	require.NoError(t, err)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, recipient.Id).Error)
	assert.Equal(t, 300, refreshed.Quota)
	assert.Zero(t, EffectiveTransferableQuota(refreshed.Quota, refreshed.TransferableQuota))
	_, err = CreateWalletFundedRedemption(recipient.Id, 100, "wallet-one-hop-recreate-001")
	assert.ErrorIs(t, err, ErrRedemptionInsufficientTransferableQuota)
}

func TestWalletRedemption_MultipleSmallCreatorsCreateReviewCase(t *testing.T) {
	setupWalletRedemptionTest(t)
	recipient := createInviteCommissionTestUser(t, "wallet_review_recipient", 0)
	setWalletQuota(t, recipient.Id, 0)
	creatorIDs := make([]int, 0, 3)
	for index := 0; index < 3; index++ {
		creator := createInviteCommissionTestUser(t, fmt.Sprintf("wallet_review_creator_%d", index), 0)
		creatorIDs = append(creatorIDs, creator.Id)
		setWalletQuota(t, creator.Id, 1000)
		created, err := CreateWalletFundedRedemption(creator.Id, 100, fmt.Sprintf("wallet-review-%03d", index))
		require.NoError(t, err)
		_, err = RedeemWithResult(created.Redemption.Key, recipient.Id)
		require.NoError(t, err)
	}

	var review RedemptionReviewCase
	require.NoError(t, DB.Where("user_id = ?", recipient.Id).First(&review).Error)
	assert.Equal(t, RedemptionReviewStatusPending, review.Status)
	assert.Equal(t, 3, review.DistinctCreatorCount)
	assert.Equal(t, 3, review.SmallCodeCount)
	assert.Equal(t, 300, review.TotalQuota)
	for _, creatorID := range creatorIDs {
		assert.Contains(t, review.CreatorIds, strconv.Itoa(creatorID))
	}
	pendingCount, err := CountPendingRedemptionReviewCases()
	require.NoError(t, err)
	assert.EqualValues(t, 1, pendingCount)
	resolved, err := ResolveRedemptionReviewCase(review.Id, 99, RedemptionReviewStatusDismissed, "normal gifts")
	require.NoError(t, err)
	assert.Equal(t, RedemptionReviewStatusDismissed, resolved.Status)

	creator := createInviteCommissionTestUser(t, "wallet_review_creator_reopen", 0)
	setWalletQuota(t, creator.Id, 1000)
	created, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-review-reopen-001")
	require.NoError(t, err)
	_, err = RedeemWithResult(created.Redemption.Key, recipient.Id)
	require.NoError(t, err)
	disabledReview, disabled, err := ResolveRedemptionReviewCaseAction(
		review.Id, 99, common.RoleRootUser, RedemptionReviewActionDisable, "confirmed abuse",
	)
	require.NoError(t, err)
	assert.True(t, disabled)
	assert.Equal(t, RedemptionReviewStatusDisabled, disabledReview.Status)
	var disabledRecipient User
	require.NoError(t, DB.First(&disabledRecipient, recipient.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, disabledRecipient.Status)
}

func TestCreateWalletFundedRedemption_AdminCanCreateBelowUserMinimum(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_admin_small_code", 0)
	creator.Role = common.RoleAdminUser
	require.NoError(t, DB.Model(&User{}).Where("id = ?", creator.Id).Update("role", creator.Role).Error)
	setWalletQuota(t, creator.Id, 1000)

	quota := MinimumWalletRedemptionQuota() - 1
	result, err := CreateWalletFundedRedemption(creator.Id, quota, "wallet-admin-small-001")
	require.NoError(t, err)
	assert.Equal(t, quota, result.Redemption.Quota)
	assert.Equal(t, 1000-quota, result.RemainingQuota)
	assert.Equal(t, 1000-quota, result.RemainingTransferableQuota)
}

func TestGetWalletRedemptionUsageSummary(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_usage_summary", 0)
	setWalletQuota(t, creator.Id, 5000)
	_, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-summary-001")
	require.NoError(t, err)
	_, err = CreateWalletFundedRedemption(creator.Id, 200, "wallet-summary-002")
	require.NoError(t, err)

	summary, err := GetWalletRedemptionUsageSummary(creator.Id)
	require.NoError(t, err)
	assert.Equal(t, 2, summary.DailyCreatedCount)
	assert.Equal(t, 300, summary.DailyCreatedQuota)
	assert.Equal(t, 2, summary.ActiveCount)
	assert.Equal(t, 100, summary.DailyCreateLimit)
	assert.Equal(t, 50_000, summary.DailyQuotaLimit)
	assert.Equal(t, 100, summary.ActiveLimit)
	assert.Equal(t, 100, summary.MinimumQuota)
	assert.Greater(t, summary.ResetAt, common.GetTimestamp())
}

func TestCreateWalletFundedRedemption_UsesConfigurableMinimumAndActiveLimit(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_configurable_limits", 0)
	setWalletQuota(t, creator.Id, 5000)
	common.WalletRedemptionMinimumQuota = 20
	common.WalletRedemptionActiveLimit = 1

	_, err := CreateWalletFundedRedemption(creator.Id, 199, "wallet-config-min-001")
	assert.ErrorIs(t, err, ErrRedemptionBelowMinimum)
	_, err = CreateWalletFundedRedemption(creator.Id, 200, "wallet-config-active-001")
	require.NoError(t, err)
	_, err = CreateWalletFundedRedemption(creator.Id, 200, "wallet-config-active-002")
	assert.ErrorIs(t, err, ErrRedemptionActiveLimit)
}

func TestCreateWalletFundedRedemption_RejectsBatchUpdatedBalance(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_batch_update_unsafe", 0)
	setWalletQuota(t, creator.Id, 1000)
	originalBatchUpdateEnabled := common.BatchUpdateEnabled
	common.BatchUpdateEnabled = true
	t.Cleanup(func() {
		common.BatchUpdateEnabled = originalBatchUpdateEnabled
	})

	_, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-batch-update-001")
	assert.ErrorIs(t, err, ErrRedemptionBatchUpdateUnsafe)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 1000, refreshed.Quota)
	assert.Equal(t, 1000, refreshed.TransferableQuota)
}

func TestCreateWalletFundedRedemption_RejectsInsufficientQuotaWithoutMutation(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_insufficient", 0)
	setWalletQuota(t, creator.Id, 200)

	_, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-insufficient-001")
	assert.ErrorIs(t, err, ErrRedemptionInsufficientQuota)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 200, refreshed.Quota)
	var count int64
	require.NoError(t, DB.Model(&Redemption{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCreateWalletFundedRedemption_RequestIDIsScopedPerUser(t *testing.T) {
	setupWalletRedemptionTest(t)
	firstCreator := createInviteCommissionTestUser(t, "wallet_request_scope_a", 0)
	secondCreator := createInviteCommissionTestUser(t, "wallet_request_scope_b", 0)
	setWalletQuota(t, firstCreator.Id, 100)
	setWalletQuota(t, secondCreator.Id, 100)

	first, err := CreateWalletFundedRedemption(firstCreator.Id, 100, "shared-wallet-request-001")
	require.NoError(t, err)
	second, err := CreateWalletFundedRedemption(secondCreator.Id, 100, "shared-wallet-request-001")
	require.NoError(t, err)
	assert.NotEqual(t, first.Redemption.Id, second.Redemption.Id)
	assert.NotEqual(t, first.Redemption.Key, second.Redemption.Key)
}

func TestCreateWalletFundedRedemption_EnforcesActiveCodeLimitWithoutDeduction(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_active_limit", 0)
	setWalletQuota(t, creator.Id, 1000)
	for index := 0; index < common.WalletRedemptionActiveLimit; index++ {
		requestID := fmt.Sprintf("wallet-active-existing-%03d", index)
		redemption := &Redemption{
			UserId:          creator.Id,
			Key:             common.GetUUID(),
			Status:          common.RedemptionCodeStatusEnabled,
			Name:            "用户钱包兑换码",
			BenefitType:     RedemptionBenefitTypeQuota,
			Quota:           1,
			CreatedTime:     common.GetTimestamp(),
			FundingSource:   RedemptionFundingSourceWallet,
			CreateRequestId: &requestID,
		}
		require.NoError(t, redemption.Insert())
	}

	_, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-active-limit-new-001")
	assert.ErrorIs(t, err, ErrRedemptionActiveLimit)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 1000, refreshed.Quota)
	assert.Equal(t, 1000, refreshed.TransferableQuota)
}

func TestWalletRedemption_SelfRedeemOnlyRestoresQuota(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_self", 0)
	setWalletQuota(t, creator.Id, 1000)
	created, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-self-request-001")
	require.NoError(t, err)

	result, err := RedeemWithResult(created.Redemption.Key, creator.Id)
	require.NoError(t, err)
	assert.Equal(t, 300, result.QuotaAdded)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 1000, refreshed.Quota)
	assert.Zero(t, refreshed.InviterId)
	assert.Zero(t, refreshed.AffCount)
	assert.Zero(t, refreshed.AffQuota)
	assert.Zero(t, refreshed.AffHistoryQuota)
	assert.Zero(t, countInviteCommissionLedgers(t))
}

func TestWalletRedemption_BindsUnboundRedeemerWithoutRewards(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_creator", 0)
	redeemer := createInviteCommissionTestUser(t, "wallet_redeemer", 0)
	setWalletQuota(t, creator.Id, 1000)
	setWalletQuota(t, redeemer.Id, 50)
	created, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-bind-request-001")
	require.NoError(t, err)

	_, err = RedeemWithResult(created.Redemption.Key, redeemer.Id)
	require.NoError(t, err)

	var refreshedCreator User
	var refreshedRedeemer User
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	require.NoError(t, DB.First(&refreshedRedeemer, redeemer.Id).Error)
	assert.Equal(t, creator.Id, refreshedRedeemer.InviterId)
	assert.Equal(t, 350, refreshedRedeemer.Quota)
	assert.Equal(t, 50, refreshedRedeemer.TransferableQuota)
	assert.Equal(t, 1, refreshedCreator.AffCount)
	assert.Zero(t, refreshedCreator.AffQuota)
	assert.Zero(t, refreshedCreator.AffHistoryQuota)
	assert.Zero(t, countInviteCommissionLedgers(t))
}

func TestWalletRedemption_LegacyCodeCannotTransferButCreatorCanRecover(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_legacy_creator", 0)
	redeemer := createInviteCommissionTestUser(t, "wallet_legacy_redeemer", 0)
	legacy := &Redemption{
		UserId:            creator.Id,
		Key:               common.GetUUID(),
		Status:            common.RedemptionCodeStatusEnabled,
		Name:              "历史钱包兑换码",
		BenefitType:       RedemptionBenefitTypeQuota,
		Quota:             100,
		CreatedTime:       common.GetTimestamp(),
		FundingSource:     RedemptionFundingSourceWallet,
		TransferableQuota: 0,
	}
	require.NoError(t, legacy.Insert())

	_, err := RedeemWithResult(legacy.Key, redeemer.Id)
	assert.ErrorIs(t, err, ErrLegacyWalletRedemptionRestricted)

	result, err := RedeemWithResult(legacy.Key, creator.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, result.QuotaAdded)

	var refreshedCreator User
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	assert.Equal(t, 100, refreshedCreator.Quota)
	assert.Zero(t, refreshedCreator.TransferableQuota)
}

func TestWalletRedemption_BindingHonorsThresholdProbabilityWithoutRewards(t *testing.T) {
	setupWalletRedemptionTest(t)
	originalInviterReward := common.QuotaForInviter
	originalInviteeReward := common.QuotaForInvitee
	common.QuotaForInviter = 500
	common.QuotaForInvitee = 300
	require.NoError(t, common.SetInviteBindingSettings(common.InviteBindingSettings{
		Threshold:          1,
		RateAfterThreshold: 0,
	}))
	t.Cleanup(func() {
		common.QuotaForInviter = originalInviterReward
		common.QuotaForInvitee = originalInviteeReward
	})

	creator := createInviteCommissionTestUser(t, "wallet_threshold_creator", 0)
	firstRedeemer := createInviteCommissionTestUser(t, "wallet_threshold_first", 0)
	secondRedeemer := createInviteCommissionTestUser(t, "wallet_threshold_second", 0)
	setWalletQuota(t, creator.Id, 1000)
	firstCode, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-threshold-first-001")
	require.NoError(t, err)
	secondCode, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-threshold-second-001")
	require.NoError(t, err)

	_, err = RedeemWithResult(firstCode.Redemption.Key, firstRedeemer.Id)
	require.NoError(t, err)
	_, err = RedeemWithResult(secondCode.Redemption.Key, secondRedeemer.Id)
	require.NoError(t, err)

	var refreshedCreator User
	var refreshedFirst User
	var refreshedSecond User
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	require.NoError(t, DB.First(&refreshedFirst, firstRedeemer.Id).Error)
	require.NoError(t, DB.First(&refreshedSecond, secondRedeemer.Id).Error)
	assert.Equal(t, 1, refreshedCreator.AffCount)
	assert.Zero(t, refreshedCreator.AffQuota)
	assert.Zero(t, refreshedCreator.AffHistoryQuota)
	assert.Equal(t, creator.Id, refreshedFirst.InviterId)
	assert.Equal(t, 100, refreshedFirst.Quota, "wallet binding must not add invitee reward")
	assert.Zero(t, refreshedSecond.InviterId)
	assert.Equal(t, 100, refreshedSecond.Quota)
}

func TestWalletRedemption_DoesNotOverrideExistingInviter(t *testing.T) {
	setupWalletRedemptionTest(t)
	existingInviter := createInviteCommissionTestUser(t, "wallet_existing_inviter", 0)
	creator := createInviteCommissionTestUser(t, "wallet_other_creator", 0)
	redeemer := createInviteCommissionTestUser(t, "wallet_already_bound", existingInviter.Id)
	setWalletQuota(t, creator.Id, 1000)
	created, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-existing-request-001")
	require.NoError(t, err)

	_, err = RedeemWithResult(created.Redemption.Key, redeemer.Id)
	require.NoError(t, err)

	var refreshedCreator User
	var refreshedRedeemer User
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	require.NoError(t, DB.First(&refreshedRedeemer, redeemer.Id).Error)
	assert.Equal(t, existingInviter.Id, refreshedRedeemer.InviterId)
	assert.Zero(t, refreshedCreator.AffCount)
}

func TestWalletRedemption_DoesNotCreateInviteCycle(t *testing.T) {
	setupWalletRedemptionTest(t)
	redeemer := createInviteCommissionTestUser(t, "wallet_cycle_parent", 0)
	creator := createInviteCommissionTestUser(t, "wallet_cycle_child", redeemer.Id)
	setWalletQuota(t, creator.Id, 1000)
	created, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-cycle-request-001")
	require.NoError(t, err)

	_, err = RedeemWithResult(created.Redemption.Key, redeemer.Id)
	require.NoError(t, err)

	var refreshedRedeemer User
	var refreshedCreator User
	require.NoError(t, DB.First(&refreshedRedeemer, redeemer.Id).Error)
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	assert.Zero(t, refreshedRedeemer.InviterId)
	assert.Equal(t, redeemer.Id, refreshedCreator.InviterId)
	assert.Zero(t, refreshedCreator.AffCount)
}

func TestWalletRedemption_SkipsBindingForUnavailableCreator(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(t *testing.T, creator *User)
		requestID string
	}{
		{
			name: "disabled",
			mutate: func(t *testing.T, creator *User) {
				require.NoError(t, DB.Model(&User{}).Where("id = ?", creator.Id).Update("status", common.UserStatusDisabled).Error)
			},
			requestID: "wallet-disabled-request-001",
		},
		{
			name: "deleted",
			mutate: func(t *testing.T, creator *User) {
				require.NoError(t, DB.Delete(&User{}, creator.Id).Error)
			},
			requestID: "wallet-deleted-request-001",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupWalletRedemptionTest(t)
			creator := createInviteCommissionTestUser(t, "wallet_unavailable_creator_"+test.name, 0)
			redeemer := createInviteCommissionTestUser(t, "wallet_unavailable_redeemer_"+test.name, 0)
			setWalletQuota(t, creator.Id, 1000)
			created, err := CreateWalletFundedRedemption(creator.Id, 100, test.requestID)
			require.NoError(t, err)
			test.mutate(t, creator)

			_, err = RedeemWithResult(created.Redemption.Key, redeemer.Id)
			require.NoError(t, err)

			var refreshedRedeemer User
			require.NoError(t, DB.First(&refreshedRedeemer, redeemer.Id).Error)
			assert.Equal(t, 100, refreshedRedeemer.Quota)
			assert.Zero(t, refreshedRedeemer.InviterId)
		})
	}
}

func TestAdminRedemption_DoesNotBindRedeemer(t *testing.T) {
	setupWalletRedemptionTest(t)
	admin := createInviteCommissionTestUser(t, "redemption_admin_source", 0)
	redeemer := createInviteCommissionTestUser(t, "redemption_admin_redeemer", 0)
	redemption := &Redemption{
		UserId:        admin.Id,
		Key:           common.GetUUID(),
		Status:        common.RedemptionCodeStatusEnabled,
		Name:          "管理员余额兑换码",
		BenefitType:   RedemptionBenefitTypeQuota,
		Quota:         100,
		CreatedTime:   common.GetTimestamp(),
		FundingSource: RedemptionFundingSourceAdmin,
	}
	require.NoError(t, redemption.Insert())

	_, err := RedeemWithResult(redemption.Key, redeemer.Id)
	require.NoError(t, err)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, redeemer.Id).Error)
	assert.Zero(t, refreshed.InviterId)
}

func TestWalletRedemption_AdminMutationAndCleanupAreBlocked(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_immutable", 0)
	setWalletQuota(t, creator.Id, 1000)
	created, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-immutable-request-001")
	require.NoError(t, err)

	redemption, err := GetRedemptionById(created.Redemption.Id)
	require.NoError(t, err)
	redemption.Name = "tampered"
	assert.ErrorIs(t, redemption.Update(), ErrWalletFundedRedemptionImmutable)
	assert.ErrorIs(t, (&Redemption{Id: redemption.Id, FundingSource: RedemptionFundingSourceAdmin}).Update(), ErrWalletFundedRedemptionImmutable)
	assert.ErrorIs(t, DeleteRedemptionById(redemption.Id), ErrWalletFundedRedemptionImmutable)

	redemption.Status = common.RedemptionCodeStatusUsed
	require.NoError(t, DB.Model(&Redemption{}).Where("id = ?", redemption.Id).Updates(map[string]any{
		"status":        redemption.Status,
		"redeemed_time": common.GetTimestamp(),
	}).Error)
	adminCode := &Redemption{
		UserId:        creator.Id,
		Key:           common.GetUUID(),
		Status:        common.RedemptionCodeStatusUsed,
		Name:          "admin cleanup code",
		BenefitType:   RedemptionBenefitTypeQuota,
		CreatedTime:   common.GetTimestamp(),
		FundingSource: RedemptionFundingSourceAdmin,
	}
	require.NoError(t, adminCode.Insert())

	deleted, err := DeleteInvalidRedemptions()
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)
	require.NoError(t, DB.First(&Redemption{}, redemption.Id).Error)
	assert.ErrorIs(t, DB.First(&Redemption{}, adminCode.Id).Error, gorm.ErrRecordNotFound)
}

func TestGetUserWalletRedemptions_IsOwnerScopedAndPrivacySafe(t *testing.T) {
	setupWalletRedemptionTest(t)
	creatorA := createInviteCommissionTestUser(t, "wallet_list_a", 0)
	creatorB := createInviteCommissionTestUser(t, "wallet_list_b", 0)
	redeemer := createInviteCommissionTestUser(t, "wallet_list_redeemer", 0)
	setWalletQuota(t, creatorA.Id, 1000)
	setWalletQuota(t, creatorB.Id, 1000)
	createdA, err := CreateWalletFundedRedemption(creatorA.Id, 100, "wallet-list-a-request-001")
	require.NoError(t, err)
	_, err = CreateWalletFundedRedemption(creatorB.Id, 200, "wallet-list-b-request-001")
	require.NoError(t, err)
	_, err = RedeemWithResult(createdA.Redemption.Key, redeemer.Id)
	require.NoError(t, err)

	items, total, err := GetUserWalletRedemptions(creatorA.Id, 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, createdA.Redemption.Key, items[0].Key)
	payload, err := common.Marshal(items)
	require.NoError(t, err)
	jsonText := string(payload)
	assert.False(t, strings.Contains(jsonText, "used_user_id"), jsonText)
	assert.False(t, strings.Contains(jsonText, redeemer.Username), jsonText)
}

func TestWalletRedemption_ConcurrentUseAndCreationDoNotDuplicateOrOverdraw(t *testing.T) {
	t.Run("same code can only be redeemed once", func(t *testing.T) {
		setupWalletRedemptionTest(t)
		creator := createInviteCommissionTestUser(t, "wallet_concurrent_creator", 0)
		firstRedeemer := createInviteCommissionTestUser(t, "wallet_concurrent_first", 0)
		secondRedeemer := createInviteCommissionTestUser(t, "wallet_concurrent_second", 0)
		setWalletQuota(t, creator.Id, 1000)
		created, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-concurrent-code-001")
		require.NoError(t, err)

		errs := make(chan error, 2)
		var wg sync.WaitGroup
		for _, userID := range []int{firstRedeemer.Id, secondRedeemer.Id} {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, redeemErr := RedeemWithResult(created.Redemption.Key, id)
				errs <- redeemErr
			}(userID)
		}
		wg.Wait()
		close(errs)
		successes := 0
		alreadyUsed := 0
		for redeemErr := range errs {
			switch {
			case redeemErr == nil:
				successes++
			case errors.Is(redeemErr, ErrRedemptionAlreadyUsed):
				alreadyUsed++
			default:
				require.NoError(t, redeemErr)
			}
		}
		assert.Equal(t, 1, successes)
		assert.Equal(t, 1, alreadyUsed)
	})

	t.Run("wallet cannot be overdrawn", func(t *testing.T) {
		setupWalletRedemptionTest(t)
		creator := createInviteCommissionTestUser(t, "wallet_concurrent_balance", 0)
		setWalletQuota(t, creator.Id, 100)

		errs := make(chan error, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, createErr := CreateWalletFundedRedemption(creator.Id, 100, fmt.Sprintf("wallet-overdraw-request-%03d", index))
				errs <- createErr
			}(i)
		}
		wg.Wait()
		close(errs)
		successes := 0
		insufficient := 0
		for createErr := range errs {
			switch {
			case createErr == nil:
				successes++
			case errors.Is(createErr, ErrRedemptionInsufficientQuota):
				insufficient++
			default:
				require.NoError(t, createErr)
			}
		}
		assert.Equal(t, 1, successes)
		assert.Equal(t, 1, insufficient)
		var refreshed User
		require.NoError(t, DB.First(&refreshed, creator.Id).Error)
		assert.Zero(t, refreshed.Quota)
	})
}
