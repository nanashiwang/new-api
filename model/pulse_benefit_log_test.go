package model

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func useSeparatePulseLogDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "logs.db")+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = pool.Close() })
	LOG_DB = db
	require.NoError(t, migrateLOGDB())
}

func TestPulseBenefitUsageLogs(t *testing.T) {
	for _, separate := range []bool{false, true} {
		t.Run(fmt.Sprintf("separate_log_db_%t", separate), func(t *testing.T) {
			setupPulseBenefitTestDB(t)
			if separate {
				useSeparatePulseLogDB(t)
			}
			user := createPaymentRiskCaseTestUser(t, "pulse-log-user")
			otherUser := createPaymentRiskCaseTestUser(t, "pulse-log-other")
			request := pulseTestRequest("pulse-visible-reward", user.Id, 500000)
			for i := 0; i < 100; i++ {
				_, err := GrantPulseBenefit(request)
				require.NoError(t, err)
			}
			changed := request
			changed.Amount++
			_, err := GrantPulseBenefit(changed)
			require.ErrorIs(t, err, ErrPulseBenefitConflict)
			logs, total, err := GetUserLogs(user.Id, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "")
			require.NoError(t, err)
			require.EqualValues(t, 1, total)
			require.Contains(t, logs[0].Content, "Meta Pulse 奖励到账：+")
			require.Contains(t, logs[0].Content, request.SourceRef)
			require.Equal(t, LogTypeSystem, logs[0].Type)
			require.Equal(t, request.Amount, logs[0].Quota)
			require.Equal(t, user.Username, logs[0].Username)
			var grant BenefitChangeRecord
			require.NoError(t, DB.Where("source_ref = ? AND action = ?", request.SourceRef, BenefitActionGrant).First(&grant).Error)
			require.Equal(t, grant.CreatedAt, logs[0].CreatedAt)
			require.True(t, grant.PulseLogSynced)
			require.Contains(t, logs[0].Other, `"source_ref":"pulse-visible-reward"`)
			_, total, err = GetUserLogs(otherUser.Id, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "")
			require.NoError(t, err)
			require.Zero(t, total)
			for i := 0; i < 3; i++ {
				_, err = RollbackPulseBenefit(request.SourceRef, "internal-only reason")
				require.NoError(t, err)
			}
			_, err = GrantPulseBenefit(request)
			require.NoError(t, err)
			logs, total, err = GetUserLogs(user.Id, LogTypeSystem, 0, 0, "", "", 0, 20, "", "")
			require.NoError(t, err)
			require.EqualValues(t, 2, total)
			require.Contains(t, logs[0].Content, "Meta Pulse 奖励撤销：-")
			require.Contains(t, logs[0].Content, request.SourceRef)
			require.NotContains(t, logs[0].Content+logs[0].Other, "internal-only")
			require.Equal(t, -request.Amount, logs[0].Quota)
			_, total, err = GetUserLogs(user.Id, LogTypeConsume, 0, 0, "", "", 0, 20, "", "")
			require.NoError(t, err)
			require.Zero(t, total)
			stat, err := SumUsedQuota(LogTypeUnknown, 0, 0, "", user.Username, "", 0, "", "", false)
			require.NoError(t, err)
			require.Zero(t, stat.Quota)
			var refreshed User
			require.NoError(t, DB.First(&refreshed, user.Id).Error)
			require.Zero(t, refreshed.Quota)
			require.Zero(t, refreshed.TransferableQuota)
		})
	}
}

func TestPulseBenefitLogFailureRetriesWithoutAffectingReward(t *testing.T) {
	setupPulseBenefitTestDB(t)
	useSeparatePulseLogDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-log-failure")
	request := pulseTestRequest("pulse-log-retry", user.Id, 10)
	injected := errors.New("log database unavailable")
	require.NoError(t, LOG_DB.Callback().Create().Before("gorm:create").Register("test:fail_log", func(tx *gorm.DB) {
		if tx.Statement.Table == "logs" {
			tx.AddError(injected)
		}
	}))
	t.Cleanup(func() { LOG_DB.Callback().Create().Remove("test:fail_log") })
	result, err := GrantPulseBenefit(request)
	require.NoError(t, err)
	require.True(t, result.Applied)
	result, err = QueryPulseBenefit(request.SourceRef)
	require.NoError(t, err)
	require.True(t, result.Applied)
	var count int64
	require.NoError(t, LOG_DB.Model(&PulseBenefitLogReceipt{}).Count(&count).Error)
	require.Zero(t, count, "receipt must roll back with failed log")
	var grant BenefitChangeRecord
	require.NoError(t, DB.Where("source_ref = ?", request.SourceRef).First(&grant).Error)
	require.False(t, grant.PulseLogSynced)
	_, err = RollbackPulseBenefit(request.SourceRef, "withdraw reward")
	require.NoError(t, err)
	require.ErrorIs(t, SyncPulseBenefitLogs(""), injected)
	require.NoError(t, LOG_DB.Callback().Create().Remove("test:fail_log"))
	t.Setenv("PULSE_BENEFIT_ENABLED", "false")
	require.NoError(t, SyncPulseBenefitLogs(""))
	require.NoError(t, SyncPulseBenefitLogs(""))
	var logs []Log
	require.NoError(t, LOG_DB.Order("id").Find(&logs).Error)
	require.Len(t, logs, 2)
	require.Equal(t, []int{10, -10}, []int{logs[0].Quota, logs[1].Quota})
	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Zero(t, refreshed.Quota)
}

func TestPulseBenefitLogAcknowledgementCrashAndConcurrentRetry(t *testing.T) {
	setupPulseBenefitTestDB(t)
	useSeparatePulseLogDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-log-crash")
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register("test:fail_ack", func(tx *gorm.DB) {
		if tx.Statement.Table == "benefit_change_records" {
			tx.AddError(errors.New("main DB acknowledgement interrupted"))
		}
	}))
	t.Cleanup(func() { DB.Callback().Update().Remove("test:fail_ack") })
	_, err := GrantPulseBenefit(pulseTestRequest("pulse-log-crash", user.Id, 10))
	require.NoError(t, err)
	var record BenefitChangeRecord
	require.NoError(t, DB.First(&record).Error)
	require.False(t, record.PulseLogSynced)
	require.NoError(t, DB.Callback().Update().Remove("test:fail_ack"))
	var workers sync.WaitGroup
	failures := make(chan error, 20)
	for i := 0; i < 20; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			failures <- syncPulseBenefitLog(&record)
		}()
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	var count int64
	require.NoError(t, LOG_DB.Model(&Log{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, DB.First(&record).Error)
	require.True(t, record.PulseLogSynced)
	// Explicit deletion/retention must not resurrect a delivered entry.
	require.NoError(t, LOG_DB.Where("user_id = ?", user.Id).Delete(&Log{}).Error)
	require.NoError(t, syncPulseBenefitLog(&record))
	require.NoError(t, LOG_DB.Model(&Log{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestPulseBenefitLogsBackfillPreservesHistoricalTime(t *testing.T) {
	setupPulseBenefitTestDB(t)
	user := createPaymentRiskCaseTestUser(t, "pulse-log-history")
	historical := time.Now().Add(-7 * 24 * time.Hour).Unix()
	for _, action := range []string{BenefitActionGrant, BenefitActionRollback} {
		delta := 100
		if action == BenefitActionRollback {
			delta = -delta
		}
		record := &BenefitChangeRecord{
			BenefitType: BenefitTypeQuota, Action: action, SourceType: BenefitSourcePulseReward,
			SourceRef: "old-reward", UserId: user.Id, TargetType: BenefitTargetUserQuota, TargetId: user.Id,
			Detail: marshalBenefitDetail(&QuotaBenefitDetail{QuotaDelta: delta}),
		}
		require.NoError(t, DB.Create(record).Error)
		require.NoError(t, DB.Model(record).UpdateColumns(map[string]interface{}{"created_at": historical, "updated_at": historical}).Error)
	}
	// Existing installations add the pending flag to already committed audits.
	require.NoError(t, DB.Migrator().DropIndex(&BenefitChangeRecord{}, "idx_benefit_pulse_log"))
	require.NoError(t, DB.Migrator().DropColumn(&BenefitChangeRecord{}, "PulseLogSynced"))
	require.NoError(t, DB.AutoMigrate(&BenefitChangeRecord{}))
	t.Setenv("PULSE_BENEFIT_ENABLED", "false")
	require.NoError(t, SyncPulseBenefitLogs(""))
	require.NoError(t, SyncPulseBenefitLogs(""))
	logs, total, err := GetUserLogs(user.Id, LogTypeUnknown, historical-1, historical+1, "", "", 0, 10, "", "")
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	for _, log := range logs {
		require.Equal(t, historical, log.CreatedAt)
	}
	var records []BenefitChangeRecord
	require.NoError(t, DB.Find(&records).Error)
	for _, record := range records {
		require.Equal(t, historical, record.UpdatedAt)
	}
	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	require.Zero(t, refreshed.Quota)
	require.Equal(t, common.UserStatusEnabled, refreshed.Status)
}
