package model

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupContentSafetyViolationTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	originalDB, originalLogDB := DB, LOG_DB
	originalRedisEnabled := common.RedisEnabled
	DB, LOG_DB = db, db
	common.RedisEnabled = false
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.RedisEnabled = originalRedisEnabled
	})
	require.NoError(t, db.AutoMigrate(&User{}, &Log{}, &ContentSafetyViolation{}))
}

func createContentSafetyTestUser(t *testing.T, username string, role int) *User {
	t.Helper()
	user := &User{
		Username: username,
		Password: "password123",
		Role:     role,
		Status:   common.UserStatusEnabled,
		AffCode:  username + "-aff",
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func contentSafetyTestParams(userID int, sequence int, now int64) RecordContentSafetyViolationParams {
	return RecordContentSafetyViolationParams{
		UserId:       userID,
		TokenId:      100 + sequence,
		ChannelId:    200 + sequence,
		RequestId:    fmt.Sprintf("req-%d", sequence),
		EventKey:     fmt.Sprintf("event-%d-%d", userID, sequence),
		ModelName:    "gpt-5.6-sol",
		ErrorType:    "invalid_request",
		ErrorCode:    "cyber_policy",
		InputHash:    fmt.Sprintf("hash-%d", sequence),
		IsStream:     true,
		CreatedAt:    now,
		WindowStart:  now - int64((30 * 24 * time.Hour).Seconds()),
		DisableAfter: 4,
	}
}

func TestRecordContentSafetyViolationWarnsThenDisablesCommonUser(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-user", common.RoleCommonUser)
	now := time.Now().Unix()

	first, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 1, now))
	require.NoError(t, err)
	require.False(t, first.Duplicate)
	require.Equal(t, 1, first.Violation.WindowCount)
	require.Equal(t, ContentSafetyActionWarning, first.Violation.Action)

	duplicate, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 1, now+1))
	require.NoError(t, err)
	require.True(t, duplicate.Duplicate)
	require.Equal(t, 1, duplicate.Violation.WindowCount)

	for sequence := 2; sequence <= 4; sequence++ {
		result, recordErr := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, sequence, now+int64(sequence)))
		require.NoError(t, recordErr)
		if sequence < 4 {
			require.Equal(t, ContentSafetyActionWarning, result.Violation.Action)
		} else {
			require.Equal(t, 4, result.Violation.WindowCount)
			require.Equal(t, ContentSafetyActionDisabled, result.Violation.Action)
		}
	}

	var updated User
	require.NoError(t, DB.First(&updated, user.Id).Error)
	require.Equal(t, common.UserStatusDisabled, updated.Status)
	var count int64
	require.NoError(t, DB.Model(&ContentSafetyViolation{}).Where("user_id = ?", user.Id).Count(&count).Error)
	require.EqualValues(t, 4, count)
}

func TestRecordContentSafetyViolationDoesNotAutoDisableAdmins(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-admin", common.RoleAdminUser)
	now := time.Now().Unix()
	var latest *ContentSafetyEnforcementResult
	for sequence := 1; sequence <= 4; sequence++ {
		var err error
		latest, err = RecordContentSafetyViolation(contentSafetyTestParams(user.Id, sequence, now+int64(sequence)))
		require.NoError(t, err)
	}
	require.Equal(t, ContentSafetyActionReviewRequired, latest.Violation.Action)

	var updated User
	require.NoError(t, DB.First(&updated, user.Id).Error)
	require.Equal(t, common.UserStatusEnabled, updated.Status)
}

func TestRecordContentSafetyViolationUsesRollingWindow(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-window", common.RoleCommonUser)
	now := time.Now().Unix()
	require.NoError(t, DB.Create(&ContentSafetyViolation{
		UserId: user.Id, EventKey: "old-event", ErrorCode: "cyber_policy",
		CreatedAt: now - int64((31 * 24 * time.Hour).Seconds()), Action: ContentSafetyActionWarning,
	}).Error)

	for sequence := 1; sequence <= 3; sequence++ {
		result, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, sequence, now+int64(sequence)))
		require.NoError(t, err)
		require.Equal(t, sequence, result.Violation.WindowCount)
		require.Equal(t, ContentSafetyActionWarning, result.Violation.Action)
	}

	var updated User
	require.NoError(t, DB.First(&updated, user.Id).Error)
	require.Equal(t, common.UserStatusEnabled, updated.Status)
}

func TestRecordContentSafetyViolationSerializesConcurrentEvents(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-concurrent", common.RoleCommonUser)
	now := time.Now().Unix()

	const eventCount = 8
	errCh := make(chan error, eventCount)
	var wg sync.WaitGroup
	for sequence := 1; sequence <= eventCount; sequence++ {
		sequence := sequence
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, sequence, now+int64(sequence)))
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	var count int64
	require.NoError(t, DB.Model(&ContentSafetyViolation{}).Where("user_id = ?", user.Id).Count(&count).Error)
	require.EqualValues(t, eventCount, count)
	var updated User
	require.NoError(t, DB.First(&updated, user.Id).Error)
	require.Equal(t, common.UserStatusDisabled, updated.Status)
}

func TestGetContentSafetyViolationsFiltersAndJoinsUsername(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-query", common.RoleCommonUser)
	now := time.Now().Unix()
	_, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 1, now))
	require.NoError(t, err)

	items, total, err := GetContentSafetyViolations(ContentSafetyViolationQuery{
		Username: "policy-query", ErrorCode: "CYBER_POLICY", StartTimestamp: now - 1,
	}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, user.Id, items[0].UserId)
	require.Equal(t, "policy-query", items[0].Username)
	encoded, err := common.Marshal(items[0])
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "event_key")
	require.NotContains(t, string(encoded), "input_hash")
}

func TestTruncateSafetyAuditValueKeepsUTF8Valid(t *testing.T) {
	require.Equal(t, "安全策", truncateSafetyAuditValue("安全策略", 3))
}
