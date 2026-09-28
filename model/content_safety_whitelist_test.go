package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestContentSafetyWhitelistExemptsExistingCooldownAndPreservesHistory(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	admin := createContentSafetyTestUser(t, "whitelist-admin", common.RoleAdminUser)
	user := createContentSafetyTestUser(t, "whitelist-user", common.RoleCommonUser)
	now := time.Now().Unix()
	for i := 1; i <= 3; i++ {
		_, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, i, now))
		require.NoError(t, err)
	}
	until, err := GetActiveContentSafetyCooldown(user.Id, now)
	require.NoError(t, err)
	require.Equal(t, now+600, until)
	require.NoError(t, SetContentSafetyWhitelist(user.Id, admin.Id, true))
	until, err = GetActiveContentSafetyCooldown(user.Id, now)
	require.NoError(t, err)
	require.Zero(t, until)
	state, err := GetUserContentSafetyState(user.Id)
	require.NoError(t, err)
	require.True(t, state.Whitelisted)
	require.False(t, state.HasUnreadWarning)
	require.Zero(t, state.CooldownUntil)
	require.Equal(t, 3, state.WindowCount)
	require.Equal(t, ContentSafetyLevelWhitelisted, state.Level)
	for i := 4; i <= 7; i++ {
		result, err := RecordContentSafetyViolation(contentSafetyTestParams(user.Id, i, now+1))
		require.NoError(t, err)
		require.Equal(t, ContentSafetyActionRecorded, result.Violation.Action)
		require.Zero(t, result.Violation.CooldownUntil)
		require.Nil(t, result.ReviewCase)
	}
	var count int64
	require.NoError(t, DB.Model(&ContentSafetyViolation{}).Where("user_id = ?", user.Id).Count(&count).Error)
	require.EqualValues(t, 7, count)
	var matches []User
	require.NoError(t, applyUserContentSafetyFilters(DB, DB.Model(&User{}), UserSearchParams{ContentSafetyStatus: ContentSafetyLevelWhitelisted}).Find(&matches).Error)
	require.Len(t, matches, 1)
	require.Equal(t, user.Id, matches[0].Id)
	var cooling []User
	require.NoError(t, applyUserContentSafetyFilters(DB, DB.Model(&User{}), UserSearchParams{ContentSafetyStatus: ContentSafetyLevelCoolingOff}).Find(&cooling).Error)
	require.Empty(t, cooling)
	// Removing the exemption restores an unexpired historical cooldown.
	require.NoError(t, SetContentSafetyWhitelist(user.Id, admin.Id, false))
	until, err = GetActiveContentSafetyCooldown(user.Id, now+2)
	require.NoError(t, err)
	require.Equal(t, now+600, until)
}

func TestContentSafetyWhitelistPermissionsAndDisabledAccount(t *testing.T) {
	setupContentSafetyViolationTestDB(t)
	admin := createContentSafetyTestUser(t, "admin-whitelist", common.RoleAdminUser)
	root := createContentSafetyTestUser(t, "root-whitelist", common.RoleRootUser)
	user := createContentSafetyTestUser(t, "user-whitelist", common.RoleCommonUser)
	require.Error(t, SetContentSafetyWhitelist(user.Id, user.Id, true))
	require.Error(t, SetContentSafetyWhitelist(root.Id, admin.Id, true))
	require.Error(t, SetContentSafetyWhitelist(admin.Id, admin.Id, true))
	require.NoError(t, DB.Model(user).Update("status", common.UserStatusDisabled).Error)
	require.NoError(t, SetContentSafetyWhitelist(user.Id, admin.Id, true))
	require.NoError(t, DB.First(user, user.Id).Error)
	require.True(t, user.ContentSafetyWhitelisted)
	require.Equal(t, common.UserStatusDisabled, user.Status)
	// Generic profile updates must never overwrite a separately managed exemption.
	user.ContentSafetyWhitelisted = false
	require.NoError(t, user.Update(false))
	require.NoError(t, DB.First(user, user.Id).Error)
	require.True(t, user.ContentSafetyWhitelisted)
	user.ContentSafetyWhitelisted = true
	require.NoError(t, SetContentSafetyWhitelist(user.Id, root.Id, false))
	require.NoError(t, user.Update(false))
	require.NoError(t, DB.First(user, user.Id).Error)
	require.False(t, user.ContentSafetyWhitelisted)
}
