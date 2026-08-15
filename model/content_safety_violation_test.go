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
	DB, LOG_DB, common.RedisEnabled = db, db, false
	t.Cleanup(func() {
		DB, LOG_DB, common.RedisEnabled = originalDB, originalLogDB, originalRedisEnabled
	})
	require.NoError(t, db.AutoMigrate(&User{}, &Log{}, &UserSubscription{}, &ContentSafetyViolation{}, &ContentSafetyReviewCase{}, &ContentSafetyEvidence{}, &ContentSafetyNotification{}))
}

func createContentSafetyTestUser(t *testing.T, username string, role int) *User {
	t.Helper()
	user := &User{Username: username, Password: "password123", Role: role, Status: common.UserStatusEnabled, AffCode: username + "-aff"}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func contentSafetyTestParams(userID int, sequence int, now int64) RecordContentSafetyViolationParams {
	return RecordContentSafetyViolationParams{
		UserId: userID, TokenId: 100 + sequence, ChannelId: 200 + sequence,
		RequestId: fmt.Sprintf("req-%d", sequence), EventKey: fmt.Sprintf("event-%d-%d", userID, sequence),
		ModelName: "gpt-5.6-sol", ErrorType: "invalid_request", ErrorCode: "cyber_policy",
		OfficialMessage: "request rejected by cyber policy", FineCategory: "credential_theft_phishing",
		ReasonSource: "local_rule", ReasonConfidence: "medium", ReasonSummary: "本地规则识别到钓鱼风险信号；未保存原文。",
		ClassifierVersion: "test-v1", InputHash: fmt.Sprintf("hash-%d", sequence), IsStream: true,
		CreatedAt: now, WindowStart: now - int64((30 * 24 * time.Hour).Seconds()),
		BurstWindowStart: now - int64((10 * time.Minute).Seconds()), BurstThreshold: 3,
		CooldownSeconds: int64((10 * time.Minute).Seconds()), ReviewAfterCooldowns: 3,
	}
}

func TestRecordContentSafetyViolationStartsCooldownWithoutAutoDisable(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-user", common.RoleCommonUser)
	now := time.Now().Unix()

	first, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 1, now))
	require.NoError(t, err)
	require.Equal(t, ContentSafetyActionWarning, first.Violation.Action)
	require.Equal(t, 1, first.Violation.BurstCount)

	duplicate, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 1, now+1))
	require.NoError(t, err)
	require.True(t, duplicate.Duplicate)

	second, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 2, now+2))
	require.NoError(t, err)
	require.Equal(t, 2, second.Violation.BurstCount)
	require.Equal(t, ContentSafetyActionWarning, second.Violation.Action)

	third, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 3, now+3))
	require.NoError(t, err)
	require.Equal(t, ContentSafetyActionCooldownStarted, third.Violation.Action)
	require.Equal(t, now+603, third.Violation.CooldownUntil)

	var updated User
	require.NoError(t, DB.First(&updated, user.Id).Error)
	require.Equal(t, common.UserStatusEnabled, updated.Status)
	require.Zero(t, third.ReviewCase)

	activeUntil, err := GetActiveContentSafetyCooldown(user.Id, now+4)
	require.NoError(t, err)
	require.Equal(t, third.Violation.CooldownUntil, activeUntil)
	expired, err := GetActiveContentSafetyCooldown(user.Id, third.Violation.CooldownUntil)
	require.NoError(t, err)
	require.Zero(t, expired)
	var eventCount int64
	require.NoError(t, DB.Model(&ContentSafetyViolation{}).Where("user_id = ?", user.Id).Count(&eventCount).Error)
	require.EqualValues(t, 3, eventCount, "cooldown lookups and local blocks must not create violations")
}

func TestRecordContentSafetyViolationRecordOnlyDoesNotAffectEnforcement(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "record-only-user", common.RoleCommonUser)
	now := time.Now().Unix()

	for sequence := 1; sequence <= 5; sequence++ {
		params := contentSafetyTestParams(user.Id, sequence, now+int64(sequence))
		params.RecordOnly = true
		params.BurstThreshold = 0
		params.CooldownSeconds = 0
		params.ReviewAfterCooldowns = 0
		result, err := RecordContentSafetyViolation(params)
		require.NoError(t, err)
		require.Equal(t, ContentSafetyActionRecorded, result.Violation.Action)
		require.Zero(t, result.Violation.WindowCount)
		require.Zero(t, result.Violation.BurstCount)
		require.Zero(t, result.Violation.CooldownUntil)
		require.Equal(t, result.Violation.CreatedAt, result.Violation.WarningReadAt)
		require.Nil(t, result.ReviewCase)
	}

	duplicateParams := contentSafetyTestParams(user.Id, 1, now+1)
	duplicateParams.RecordOnly = true
	duplicate, err := RecordContentSafetyViolation(duplicateParams)
	require.NoError(t, err)
	require.True(t, duplicate.Duplicate)
	require.Equal(t, ContentSafetyActionRecorded, duplicate.Violation.Action)

	state, err := GetUserContentSafetyState(user.Id)
	require.NoError(t, err)
	require.Equal(t, ContentSafetyLevelNormal, state.Level)
	require.Zero(t, state.WindowCount)
	require.Zero(t, state.BurstCount)
	require.Zero(t, state.CooldownUntil)
	require.Zero(t, state.ReviewCaseId)
	require.False(t, state.HasUnreadWarning)
	require.Nil(t, state.LatestViolation)

	activeUntil, err := GetActiveContentSafetyCooldown(user.Id, now+10)
	require.NoError(t, err)
	require.Zero(t, activeUntil)
	var reviews int64
	require.NoError(t, DB.Model(&ContentSafetyReviewCase{}).Where("user_id = ?", user.Id).Count(&reviews).Error)
	require.Zero(t, reviews)
	var records int64
	require.NoError(t, DB.Model(&ContentSafetyViolation{}).Where("user_id = ? AND action = ?", user.Id, ContentSafetyActionRecorded).Count(&records).Error)
	require.EqualValues(t, 5, records)

	normalParams := contentSafetyTestParams(user.Id, 99, now+20)
	normalResult, err := RecordContentSafetyViolation(normalParams)
	require.NoError(t, err)
	require.Equal(t, ContentSafetyActionWarning, normalResult.Violation.Action)
	require.Equal(t, 1, normalResult.Violation.WindowCount)
	require.Equal(t, 1, normalResult.Violation.BurstCount)
}

func TestRecordContentSafetyViolationNeverAutoDisablesAnyRole(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	roles := []int{common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser}
	for index, role := range roles {
		user := createContentSafetyTestUser(t, fmt.Sprintf("policy-role-%d", index), role)
		now := time.Now().Unix() + int64(index*10000)
		sequence := 0
		for episode := 0; episode < 3; episode++ {
			for event := 0; event < 3; event++ {
				sequence++
				_, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, sequence, now+int64(episode*700+event)))
				require.NoError(t, err)
			}
		}
		var updated User
		require.NoError(t, DB.First(&updated, user.Id).Error)
		require.Equal(t, common.UserStatusEnabled, updated.Status)
		var pending int64
		require.NoError(t, DB.Model(&ContentSafetyReviewCase{}).Where("user_id = ? AND status = ?", user.Id, ContentSafetyReviewPending).Count(&pending).Error)
		require.EqualValues(t, 1, pending)
	}
}

func TestThreeCooldownEpisodesCreateExactlyOneReviewCase(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-review", common.RoleCommonUser)
	now := time.Now().Unix()
	sequence := 0
	var latest *ContentSafetyEnforcementResult
	for episode := 0; episode < 4; episode++ {
		for event := 0; event < 3; event++ {
			sequence++
			var err error
			latest, err = RecordContentSafetyViolation(contentSafetyTestParams(user.Id, sequence, now+int64(episode*700+event)))
			require.NoError(t, err)
		}
	}
	require.NotNil(t, latest.ReviewCase)
	var cases int64
	require.NoError(t, DB.Model(&ContentSafetyReviewCase{}).Where("user_id = ? AND status = ?", user.Id, ContentSafetyReviewPending).Count(&cases).Error)
	require.EqualValues(t, 1, cases)
}

func TestPermanentDisableRequiresApprovedAdminReview(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-approval", common.RoleCommonUser)
	reviewer := createContentSafetyTestUser(t, "policy-reviewer", common.RoleAdminUser)
	now := time.Now().Unix()
	sequence := 0
	var reviewCase *ContentSafetyReviewCase
	for episode := 0; episode < 3; episode++ {
		for event := 0; event < 3; event++ {
			sequence++
			result, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, sequence, now+int64(episode*700+event)))
			require.NoError(t, err)
			if result.ReviewCase != nil {
				reviewCase = result.ReviewCase
			}
		}
	}
	require.NotNil(t, reviewCase)

	resolved, disabled, err := ResolveContentSafetyReviewCase(reviewCase.Id, reviewer.Id, ContentSafetyReviewObserving, "继续观察")
	require.NoError(t, err)
	require.False(t, disabled)
	require.Equal(t, ContentSafetyReviewObserving, resolved.Status)
	var unchanged User
	require.NoError(t, DB.First(&unchanged, user.Id).Error)
	require.Equal(t, common.UserStatusEnabled, unchanged.Status)

	trigger := &ContentSafetyViolation{UserId: user.Id, EventKey: "new-review-trigger", ErrorCode: "cyber_policy", CreatedAt: now + 4000}
	require.NoError(t, DB.Create(trigger).Error)
	secondCase, err := createPendingContentSafetyReviewCase(DB, trigger, 4)
	require.NoError(t, err)
	_, disabled, err = ResolveContentSafetyReviewCase(secondCase.Id, reviewer.Id, ContentSafetyReviewApprovedDisable, "")
	require.Error(t, err)
	require.False(t, disabled)
	_, disabled, err = ResolveContentSafetyReviewCase(secondCase.Id, reviewer.Id, ContentSafetyReviewApprovedDisable, "确认停用")
	require.NoError(t, err)
	require.True(t, disabled)
	var disabledUser User
	require.NoError(t, DB.First(&disabledUser, user.Id).Error)
	require.Equal(t, common.UserStatusDisabled, disabledUser.Status)
}

func TestConcurrentReviewResolutionsOnlyApplyOnce(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-review-race", common.RoleCommonUser)
	reviewer := createContentSafetyTestUser(t, "policy-review-race-admin", common.RoleAdminUser)
	trigger := &ContentSafetyViolation{UserId: user.Id, EventKey: "review-race-trigger", ErrorCode: "cyber_policy", CreatedAt: time.Now().Unix()}
	require.NoError(t, DB.Create(trigger).Error)
	reviewCase, err := createPendingContentSafetyReviewCase(DB, trigger, 3)
	require.NoError(t, err)

	resolutions := []string{ContentSafetyReviewObserving, ContentSafetyReviewDismissed}
	errCh := make(chan error, len(resolutions))
	var wg sync.WaitGroup
	for _, resolution := range resolutions {
		resolution := resolution
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, resolveErr := ResolveContentSafetyReviewCase(reviewCase.Id, reviewer.Id, resolution, "并发审核测试")
			errCh <- resolveErr
		}()
	}
	wg.Wait()
	close(errCh)
	successes, failures := 0, 0
	for resolveErr := range errCh {
		if resolveErr == nil {
			successes++
		} else {
			failures++
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, failures)
}

func TestRecordContentSafetyViolationUsesRollingWindows(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-window", common.RoleCommonUser)
	now := time.Now().Unix()
	require.NoError(t, DB.Create(&ContentSafetyViolation{UserId: user.Id, EventKey: "old-event", ErrorCode: "cyber_policy", CreatedAt: now - int64((31 * 24 * time.Hour).Seconds()), Action: ContentSafetyActionWarning}).Error)
	first, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 1, now))
	require.NoError(t, err)
	require.Equal(t, 1, first.Violation.WindowCount)
	require.Equal(t, 1, first.Violation.BurstCount)
	second, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 2, now+601))
	require.NoError(t, err)
	require.Equal(t, 2, second.Violation.WindowCount)
	require.Equal(t, 1, second.Violation.BurstCount)
}

func TestRecordContentSafetyViolationSerializesConcurrentCooldownStart(t *testing.T) {
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
	var starts int64
	require.NoError(t, DB.Model(&ContentSafetyViolation{}).Where("user_id = ? AND action = ?", user.Id, ContentSafetyActionCooldownStarted).Count(&starts).Error)
	require.EqualValues(t, 1, starts)
	var updated User
	require.NoError(t, DB.First(&updated, user.Id).Error)
	require.Equal(t, common.UserStatusEnabled, updated.Status)
}

func TestViolationAuditHidesRequestIdentitySecrets(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "policy-query", common.RoleCommonUser)
	now := time.Now().Unix()
	_, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, 1, now))
	require.NoError(t, err)
	items, total, err := GetContentSafetyViolations(ContentSafetyViolationQuery{Username: "policy-query", ErrorCode: "CYBER_POLICY", StartTimestamp: now - 1}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	encoded, err := common.Marshal(items[0])
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "event_key")
	require.NotContains(t, string(encoded), "input_hash")
}

func TestAttachUserContentSafetyMetadataShowsDistinctStates(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	now := time.Now().Unix()
	normal := createContentSafetyTestUser(t, "state-normal", common.RoleCommonUser)
	warning := createContentSafetyTestUser(t, "state-warning", common.RoleCommonUser)
	cooling := createContentSafetyTestUser(t, "state-cooling", common.RoleCommonUser)
	focus := createContentSafetyTestUser(t, "state-focus", common.RoleCommonUser)
	pending := createContentSafetyTestUser(t, "state-pending", common.RoleCommonUser)

	require.NoError(t, DB.Create(&ContentSafetyViolation{UserId: warning.Id, EventKey: "warning", ErrorCode: "cyber_policy", CreatedAt: now - 10, BurstCount: 1, Action: ContentSafetyActionWarning, FineCategory: "malware"}).Error)
	require.NoError(t, DB.Create(&ContentSafetyViolation{UserId: cooling.Id, EventKey: "cooling", ErrorCode: "cyber_policy", CreatedAt: now - 10, BurstCount: 3, CooldownUntil: now + 500, Action: ContentSafetyActionCooldownStarted}).Error)
	for index := 0; index < 2; index++ {
		require.NoError(t, DB.Create(&ContentSafetyViolation{UserId: focus.Id, EventKey: fmt.Sprintf("focus-%d", index), ErrorCode: "cyber_policy", CreatedAt: now - int64(2000-index), Action: ContentSafetyActionCooldownStarted}).Error)
	}
	trigger := &ContentSafetyViolation{UserId: pending.Id, EventKey: "pending", ErrorCode: "cyber_policy", CreatedAt: now - 20, Action: ContentSafetyActionCooldownStarted}
	require.NoError(t, DB.Create(trigger).Error)
	require.NoError(t, DB.Create(&ContentSafetyReviewCase{UserId: pending.Id, Status: ContentSafetyReviewPending, TriggerViolationId: trigger.Id, CreatedAt: now - 20}).Error)

	users := []*User{normal, warning, cooling, focus, pending}
	require.NoError(t, AttachUserContentSafetyMetadata(DB, users))
	require.Equal(t, ContentSafetyLevelNormal, normal.ContentSafetyLevel)
	require.Equal(t, ContentSafetyLevelWarning1, warning.ContentSafetyLevel)
	require.Equal(t, ContentSafetyLevelCoolingOff, cooling.ContentSafetyLevel)
	require.Equal(t, ContentSafetyLevelFocus, focus.ContentSafetyLevel)
	require.Equal(t, ContentSafetyLevelReviewPending, pending.ContentSafetyLevel)
	require.Equal(t, "malware", warning.ContentSafetyLastCategory)
}

func TestUserContentSafetyFiltersApplyBeforePagination(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	now := time.Now().Unix()
	for index := 1; index <= 4; index++ {
		user := createContentSafetyTestUser(t, fmt.Sprintf("filter-%d", index), common.RoleCommonUser)
		require.NoError(t, DB.Create(&ContentSafetyViolation{UserId: user.Id, EventKey: fmt.Sprintf("filter-event-%d", index), ErrorCode: "cyber_policy", CreatedAt: now - int64(index), BurstCount: 1, Action: ContentSafetyActionWarning}).Error)
	}
	otherCode := createContentSafetyTestUser(t, "filter-other", common.RoleCommonUser)
	require.NoError(t, DB.Create(&ContentSafetyViolation{UserId: otherCode.Id, EventKey: "other", ErrorCode: "content_filter", CreatedAt: now - 1, BurstCount: 1, Action: ContentSafetyActionWarning}).Error)

	page, total, err := SearchUsersWithParams(UserSearchParams{ContentSafetyStatus: ContentSafetyLevelTriggered, ContentSafetyCodes: []string{"cyber_policy"}, SortBy: "id", SortOrder: "asc", PageSize: 2})
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, page, 2)

	var warnings int64
	query := applyUserContentSafetyFilters(DB, DB.Unscoped().Model(&User{}), UserSearchParams{ContentSafetyStatus: ContentSafetyLevelWarning1})
	require.NoError(t, query.Count(&warnings).Error)
	require.EqualValues(t, 5, warnings)
}

func TestSearchUsersWithParamsSortsByLatestContentSafetyBeforePagination(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	now := time.Now().Unix()
	older := createContentSafetyTestUser(t, "safety-sort-older", common.RoleCommonUser)
	newer := createContentSafetyTestUser(t, "safety-sort-newer", common.RoleCommonUser)
	withoutRecent := createContentSafetyTestUser(t, "safety-sort-none", common.RoleCommonUser)

	require.NoError(t, DB.Create(&ContentSafetyViolation{
		UserId: older.Id, EventKey: "safety-sort-older-event", ErrorCode: "content_filter",
		CreatedAt: now - 120, Action: ContentSafetyActionWarning,
	}).Error)
	require.NoError(t, DB.Create(&ContentSafetyViolation{
		UserId: newer.Id, EventKey: "safety-sort-newer-event", ErrorCode: "cyber_policy",
		CreatedAt: now - 30, Action: ContentSafetyActionWarning,
	}).Error)
	// 超出 30 天窗口的历史记录不应参与排序，也不应显示为最近触发时间。
	require.NoError(t, DB.Create(&ContentSafetyViolation{
		UserId: withoutRecent.Id, EventKey: "safety-sort-expired-event", ErrorCode: "safety",
		CreatedAt: now - int64(contentSafetyWindow.Seconds()) - 1, Action: ContentSafetyActionWarning,
	}).Error)

	users, total, err := SearchUsersWithParams(UserSearchParams{
		ContentSafetySortOrder: "desc", SortBy: "id", SortOrder: "desc", PageSize: 2,
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, users, 2)
	require.Equal(t, newer.Id, users[0].Id)
	require.Equal(t, older.Id, users[1].Id)
	require.Equal(t, now-30, users[0].ContentSafetyLastAt)
	require.Equal(t, now-120, users[1].ContentSafetyLastAt)

	ascending, _, err := SearchUsersWithParams(UserSearchParams{
		ContentSafetySortOrder: "asc", SortBy: "id", SortOrder: "desc", PageSize: 3,
	})
	require.NoError(t, err)
	require.Len(t, ascending, 3)
	require.Equal(t, older.Id, ascending[0].Id)
	require.Equal(t, newer.Id, ascending[1].Id)
	require.Equal(t, withoutRecent.Id, ascending[2].Id)
	require.Zero(t, ascending[2].ContentSafetyLastAt)
}

func TestAttachUserContentSafetyMetadataWithoutAuditTableIsNormal(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}))
	user := &User{Id: 1, Role: common.RoleCommonUser, Status: common.UserStatusDisabled}
	require.NoError(t, AttachUserContentSafetyMetadata(db, []*User{user}))
	require.Equal(t, ContentSafetyLevelNormal, user.ContentSafetyLevel)
	query := applyUserContentSafetyFilters(db, db.Unscoped().Model(&User{}), UserSearchParams{ContentSafetyStatus: ContentSafetyLevelTriggered})
	var total int64
	require.NoError(t, query.Count(&total).Error)
	require.Zero(t, total)
}

func TestContentSafetyNotificationRateLimitIsEnforcedInTransaction(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	user := createContentSafetyTestUser(t, "notification-limit", common.RoleCommonUser)
	now := time.Now().Unix()
	for index := 1; index <= 4; index++ {
		notification := &ContentSafetyNotification{
			ViolationId: int64(index), UserId: user.Id, DeliveryKey: fmt.Sprintf("delivery-%d", index),
			Kind: ContentSafetyActionWarning, Recipient: "user@example.com", RecipientSource: "email",
			TemplateVersion: "test-v1", Status: ContentSafetyNotificationPending, CreatedAt: now, UpdatedAt: now,
		}
		created, err := CreateContentSafetyNotification(notification, now-3600, 3)
		require.NoError(t, err)
		require.Equal(t, index <= 3, created)
	}
	var count int64
	require.NoError(t, DB.Model(&ContentSafetyNotification{}).Where("user_id = ?", user.Id).Count(&count).Error)
	require.EqualValues(t, 3, count)
}

func TestTruncateSafetyAuditValueKeepsUTF8Valid(t *testing.T) {
	require.Equal(t, "安全策", truncateSafetyAuditValue("安全策略", 3))
}
