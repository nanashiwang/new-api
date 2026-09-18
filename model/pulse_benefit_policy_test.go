package model

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPulseBenefitTestDB(t *testing.T) {
	t.Helper()
	setupPaymentRiskCaseTestDB(t)
	require.NoError(t, DB.AutoMigrate(&PulseBenefitQuotaCounter{}))
	t.Setenv("PULSE_BENEFIT_ENABLED", "true")
	t.Setenv("PULSE_BENEFIT_MAX_GRANT_QUOTA", "1000000")
	t.Setenv("PULSE_BENEFIT_USER_DAILY_QUOTA", "1000000")
	t.Setenv("PULSE_BENEFIT_DAILY_QUOTA", "1000000")
}

func pulseTestRequest(ref string, userID, amount int) PulseBenefitGrantRequest {
	return PulseBenefitGrantRequest{GrantID: ref, SourceRef: ref, UserID: userID, Amount: amount, RewardType: "newapi_quota"}
}

func TestPulseBenefitPolicyFailsClosed(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "+1", "01", "1.0", "1e3", " 1", "9223372036854775808"} {
		t.Run(fmt.Sprintf("cap_%q", value), func(t *testing.T) {
			t.Setenv("PULSE_BENEFIT_ENABLED", "true")
			t.Setenv("PULSE_BENEFIT_MAX_GRANT_QUOTA", "1")
			t.Setenv("PULSE_BENEFIT_USER_DAILY_QUOTA", "1")
			t.Setenv("PULSE_BENEFIT_DAILY_QUOTA", value)
			_, err := loadPulseBenefitPolicy()
			require.ErrorIs(t, err, ErrPulseBenefitPolicy)
		})
	}
	for _, value := range []string{"", "false", "TRUE", "1", " true"} {
		t.Setenv("PULSE_BENEFIT_ENABLED", value)
		_, err := loadPulseBenefitPolicy()
		require.Error(t, err)
	}
}

func TestPulseBenefitPausedReplayAndQueryRemainAvailable(t *testing.T) {
	setupPulseBenefitTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-policy-paused")
	request := pulseTestRequest("before-pause", user.Id, 10)
	_, err := GrantPulseBenefit(request)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("status", common.UserStatusDisabled).Error)
	t.Setenv("PULSE_BENEFIT_ENABLED", "false")
	result, err := GrantPulseBenefit(request)
	require.NoError(t, err)
	require.Equal(t, PulseBenefitStatusAlreadyApplied, result.Status)
	result, err = QueryPulseBenefit(request.SourceRef)
	require.NoError(t, err)
	require.True(t, result.Applied)
	_, err = GrantPulseBenefit(pulseTestRequest("after-pause", user.Id, 10))
	require.ErrorIs(t, err, ErrPulseBenefitPaused)
	request.Amount++
	_, err = GrantPulseBenefit(request)
	require.ErrorIs(t, err, ErrPulseBenefitConflict)
}

func TestPulseBenefitRejectsDisabledDeletedMissingAndUnsupportedRecipients(t *testing.T) {
	setupPulseBenefitTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-policy-disabled")
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("status", common.UserStatusDisabled).Error)
	_, err := GrantPulseBenefit(pulseTestRequest("disabled", user.Id, 10))
	require.ErrorIs(t, err, ErrPulseBenefitUserUnavailable)
	require.NoError(t, DB.Delete(&User{}, user.Id).Error)
	_, err = GrantPulseBenefit(pulseTestRequest("deleted", user.Id, 10))
	require.ErrorIs(t, err, ErrPulseBenefitUserUnavailable)
	_, err = GrantPulseBenefit(pulseTestRequest("missing", user.Id+999, 10))
	require.ErrorIs(t, err, ErrPulseBenefitUserUnavailable)
	request := pulseTestRequest("unsupported", user.Id, 10)
	request.RewardType = "content"
	_, err = GrantPulseBenefit(request)
	require.ErrorContains(t, err, "reward_type")
	var count int64
	require.NoError(t, DB.Model(&PulseBenefitReceipt{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestPulseBenefitLimitsAreAtomicAndGrossAfterRollback(t *testing.T) {
	setupPulseBenefitTestDB(t)
	first := createPaymentRiskCaseTestUser(t, "pulse-policy-first")
	second := createPaymentRiskCaseTestUser(t, "pulse-policy-second")
	t.Setenv("PULSE_BENEFIT_MAX_GRANT_QUOTA", "10")
	t.Setenv("PULSE_BENEFIT_USER_DAILY_QUOTA", "15")
	t.Setenv("PULSE_BENEFIT_DAILY_QUOTA", "20")
	_, err := GrantPulseBenefit(pulseTestRequest("single-limit", first.Id, 11))
	require.ErrorIs(t, err, ErrPulseBenefitLimit)
	_, err = GrantPulseBenefit(pulseTestRequest("accepted-first", first.Id, 10))
	require.NoError(t, err)
	_, err = GrantPulseBenefit(pulseTestRequest("user-limit", first.Id, 6))
	require.ErrorIs(t, err, ErrPulseBenefitLimit)
	// Rejected per-user allocation must also roll back the global reservation.
	_, err = GrantPulseBenefit(pulseTestRequest("accepted-second", second.Id, 10))
	require.NoError(t, err)
	_, err = RollbackPulseBenefit("accepted-first", "活动纠正")
	require.NoError(t, err)
	_, err = GrantPulseBenefit(pulseTestRequest("global-limit-after-rollback", first.Id, 1))
	require.ErrorIs(t, err, ErrPulseBenefitLimit)
	var count int64
	require.NoError(t, DB.Model(&PulseBenefitReceipt{}).Count(&count).Error)
	require.EqualValues(t, 2, count)
}

func TestPulseBenefitConcurrentRequestsCannotExceedCaps(t *testing.T) {
	setupPulseBenefitTestDB(t)
	persistent, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "pulse.db")+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"), &gorm.Config{})
	require.NoError(t, err)
	pool, err := persistent.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, persistent.AutoMigrate(&User{}, &BenefitChangeRecord{}, &PulseBenefitReceipt{}, &PulseBenefitQuotaCounter{}))
	DB = persistent
	user := createPaymentRiskCaseTestUser(t, "pulse-policy-concurrent")
	t.Setenv("PULSE_BENEFIT_USER_DAILY_QUOTA", "10")
	t.Setenv("PULSE_BENEFIT_DAILY_QUOTA", "10")
	var wg sync.WaitGroup
	results := make(chan error, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := GrantPulseBenefit(pulseTestRequest(fmt.Sprintf("concurrent-%d", i), user.Id, 1))
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		} else {
			require.ErrorIs(t, err, ErrPulseBenefitLimit)
		}
	}
	require.Equal(t, 10, succeeded)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Equal(t, 10, refreshed.Quota)
}

func TestPulseBenefitConcurrentMixedPayloadForOneReference(t *testing.T) {
	setupPulseBenefitTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-policy-mixed")
	var wg sync.WaitGroup
	results := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := GrantPulseBenefit(pulseTestRequest("same-source", user.Id, 1+i%2))
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	conflicts := 0
	for err := range results {
		if errors.Is(err, ErrPulseBenefitConflict) {
			conflicts++
		} else {
			require.NoError(t, err)
		}
	}
	require.Equal(t, 50, conflicts)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Contains(t, []int{1, 2}, refreshed.Quota)
	var count int64
	require.NoError(t, DB.Model(&BenefitChangeRecord{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestPulseBenefitRollbackNeverCreatesDebt(t *testing.T) {
	setupPulseBenefitTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-policy-no-debt")
	_, err := GrantPulseBenefit(pulseTestRequest("spent-grant", user.Id, 10))
	require.NoError(t, err)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", 3).Error)
	_, err = RollbackPulseBenefit("spent-grant", "余额不足")
	require.ErrorIs(t, err, ErrPulseBenefitInsufficientBalance)
	result, err := QueryPulseBenefit("spent-grant")
	require.NoError(t, err)
	require.True(t, result.Applied)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Equal(t, 3, refreshed.Quota)
	var count int64
	require.NoError(t, DB.Model(&BenefitChangeRecord{}).Where("action = ?", BenefitActionRollback).Count(&count).Error)
	require.Zero(t, count)
}

func TestPulseBenefitCounterRebuildIncludesLegacyGrantsAndLegacyReplay(t *testing.T) {
	setupPulseBenefitTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-policy-legacy")
	legacy := pulseTestRequest("legacy", user.Id, 8)
	legacy.RewardType = "period"
	hash, err := pulseBenefitFingerprint(legacy)
	require.NoError(t, err)
	record := BenefitChangeRecord{BenefitType: BenefitTypeQuota, Action: BenefitActionGrant, SourceType: BenefitSourcePulseReward,
		SourceRef: legacy.SourceRef, UserId: user.Id, TargetType: BenefitTargetUserQuota, TargetId: user.Id, PayloadHash: hash,
		Detail: marshalBenefitDetail(&QuotaBenefitDetail{QuotaDelta: 8})}
	require.NoError(t, DB.Create(&record).Error)
	t.Setenv("PULSE_BENEFIT_DAILY_QUOTA", "10")
	result, err := GrantPulseBenefit(legacy)
	require.NoError(t, err)
	require.Equal(t, PulseBenefitStatusAlreadyApplied, result.Status)
	_, err = GrantPulseBenefit(pulseTestRequest("would-exceed-legacy-total", user.Id, 3))
	require.ErrorIs(t, err, ErrPulseBenefitLimit)
	_, err = GrantPulseBenefit(pulseTestRequest("fits-legacy-total", user.Id, 2))
	require.NoError(t, err)
}

func TestPulseBenefitDayUsesShanghaiMidnight(t *testing.T) {
	before := time.Date(2026, 9, 19, 15, 59, 59, 0, time.UTC)
	day, start, end := pulseBenefitDay(before)
	require.Equal(t, "2026-09-19", day)
	require.Equal(t, int64(24*60*60), end-start)
	next, nextStart, _ := pulseBenefitDay(before.Add(time.Second))
	require.Equal(t, "2026-09-20", next)
	require.Equal(t, end, nextStart)
}

func TestPulseBenefitPaymentRiskHoldStopsNewGrantsButAllowsReplay(t *testing.T) {
	setupPulseBenefitTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-policy-payment-risk")
	request := pulseTestRequest("before-payment-risk", user.Id, 10)
	_, err := GrantPulseBenefit(request)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("pulse_reward_hold", true).Error)
	_, err = GrantPulseBenefit(pulseTestRequest("after-payment-risk", user.Id, 10))
	require.ErrorIs(t, err, ErrPulseBenefitUserUnavailable)
	replay, err := GrantPulseBenefit(request)
	require.NoError(t, err)
	require.Equal(t, PulseBenefitStatusAlreadyApplied, replay.Status)
}

func TestPulseBenefitRejectsBalanceOverflowAtomically(t *testing.T) {
	setupPulseBenefitTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-policy-overflow")
	maximum := int(^uint(0) >> 1)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", maximum).Error)
	_, err := GrantPulseBenefit(pulseTestRequest("overflow", user.Id, 1))
	require.ErrorIs(t, err, ErrPulseBenefitLimit)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Equal(t, maximum, refreshed.Quota)
	var count int64
	require.NoError(t, DB.Model(&PulseBenefitReceipt{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestPulseBenefitSourceCannotMoveToAnotherRecipient(t *testing.T) {
	setupPulseBenefitTestDB(t)
	first := createPaymentRiskCaseTestUser(t, "pulse-first-owner")
	second := createPaymentRiskCaseTestUser(t, "pulse-second-owner")
	_, err := GrantPulseBenefit(pulseTestRequest("fixed-recipient", first.Id, 10))
	require.NoError(t, err)
	_, err = GrantPulseBenefit(pulseTestRequest("fixed-recipient", second.Id, 10))
	require.ErrorIs(t, err, ErrPulseBenefitConflict)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, second.Id).Error)
	require.Zero(t, refreshed.Quota)
}

func TestPulseBenefitGrantAndRollbackInvalidateRecipientCache(t *testing.T) {
	setupPulseBenefitTestDB(t)
	server := miniredis.RunT(t)
	previous := common.RDB
	common.RDB = redis.NewClient(&redis.Options{Addr: server.Addr()})
	common.RedisEnabled = true
	t.Cleanup(func() { _ = common.RDB.Close(); common.RDB = previous })
	user := createPaymentRiskCaseTestUser(t, "pulse-cache-consistency")
	server.HSet(getUserCacheKey(user.Id), "Quota", "999")
	_, err := GrantPulseBenefit(pulseTestRequest("cache-grant", user.Id, 10))
	require.NoError(t, err)
	require.False(t, server.Exists(getUserCacheKey(user.Id)))
	server.HSet(getUserCacheKey(user.Id), "Quota", "10")
	_, err = RollbackPulseBenefit("cache-grant", "cache test")
	require.NoError(t, err)
	require.False(t, server.Exists(getUserCacheKey(user.Id)))
	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Zero(t, refreshed.Quota)
}
