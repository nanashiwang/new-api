package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserAffInviteTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&User{}, &Log{}))
	require.NoError(t, DB.Exec("DELETE FROM logs").Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
}

func withInviteQuotaSettings(t *testing.T, newUserQuota int, inviteeQuota int, inviterQuota int) {
	t.Helper()
	originNewUserQuota := common.QuotaForNewUser
	originInviteeQuota := common.QuotaForInvitee
	originInviterQuota := common.QuotaForInviter
	common.QuotaForNewUser = newUserQuota
	common.QuotaForInvitee = inviteeQuota
	common.QuotaForInviter = inviterQuota
	t.Cleanup(func() {
		common.QuotaForNewUser = originNewUserQuota
		common.QuotaForInvitee = originInviteeQuota
		common.QuotaForInviter = originInviterQuota
	})
}

func createUserAffInviteTestInviter(t *testing.T, username string, affCode string, status int) *User {
	t.Helper()
	user := &User{
		Username:    username,
		Password:    "password123",
		DisplayName: username,
		Role:        common.RoleCommonUser,
		Status:      status,
		AffCode:     affCode,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestUserInsertPersistsInviterAndRewards(t *testing.T) {
	setupUserAffInviteTestDB(t)
	withInviteQuotaSettings(t, 0, 7, 11)

	inviter := createUserAffInviteTestInviter(t, "insert_inviter", "insert_inviter_aff", common.UserStatusEnabled)
	invitee := &User{
		Username:    "insert_invitee",
		Password:    "password123",
		DisplayName: "insert_invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}

	require.NoError(t, invitee.Insert(inviter.Id))

	var savedInvitee User
	require.NoError(t, DB.First(&savedInvitee, invitee.Id).Error)
	assert.Equal(t, inviter.Id, savedInvitee.InviterId)
	assert.Len(t, savedInvitee.AffCode, UserAffCodeLength)
	assert.Equal(t, 7, savedInvitee.Quota)

	var savedInviter User
	require.NoError(t, DB.First(&savedInviter, inviter.Id).Error)
	assert.Equal(t, 1, savedInviter.AffCount)
	assert.Equal(t, 11, savedInviter.AffQuota)
	assert.Equal(t, 11, savedInviter.AffHistoryQuota)
}

func TestUserInsertWithTxPersistsInviterForOAuthFlow(t *testing.T) {
	setupUserAffInviteTestDB(t)
	withInviteQuotaSettings(t, 0, 0, 9)

	inviter := createUserAffInviteTestInviter(t, "oauth_inviter", "oauth_inviter_aff", common.UserStatusEnabled)
	invitee := &User{
		Username:    "oauth_invitee",
		DisplayName: "oauth_invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		return invitee.InsertWithTx(tx, inviter.Id)
	})
	require.NoError(t, err)

	var savedInvitee User
	require.NoError(t, DB.First(&savedInvitee, invitee.Id).Error)
	assert.Equal(t, inviter.Id, savedInvitee.InviterId)

	invitee.FinalizeOAuthUserCreation(inviter.Id)

	var savedInviter User
	require.NoError(t, DB.First(&savedInviter, inviter.Id).Error)
	assert.Equal(t, 1, savedInviter.AffCount)
	assert.Equal(t, 9, savedInviter.AffQuota)
	assert.Equal(t, 9, savedInviter.AffHistoryQuota)
}

func TestGetUserIdByAffCodeRequiresEnabledActiveInviter(t *testing.T) {
	setupUserAffInviteTestDB(t)

	enabled := createUserAffInviteTestInviter(t, "aff_enabled", "aff_enabled_code", common.UserStatusEnabled)
	disabled := createUserAffInviteTestInviter(t, "aff_disabled", "aff_disabled_code", common.UserStatusDisabled)
	deleted := createUserAffInviteTestInviter(t, "aff_deleted", "aff_deleted_code", common.UserStatusEnabled)
	require.NoError(t, DB.Delete(deleted).Error)

	id, err := GetUserIdByAffCode(" aff_enabled_code ")
	require.NoError(t, err)
	assert.Equal(t, enabled.Id, id)

	id, err = GetUserIdByAffCode(disabled.AffCode)
	require.Error(t, err)
	assert.Zero(t, id)

	id, err = GetUserIdByAffCode(deleted.AffCode)
	require.Error(t, err)
	assert.Zero(t, id)
}

func TestGenerateUniqueAffCodeRetriesOnCollision(t *testing.T) {
	setupUserAffInviteTestDB(t)

	createUserAffInviteTestInviter(t, "aff_collision", "duplicate001", common.UserStatusEnabled)

	originGenerator := generateAffCodeCandidate
	calls := 0
	generateAffCodeCandidate = func() (string, error) {
		calls++
		if calls == 1 {
			return "duplicate001", nil
		}
		return "unique000001", nil
	}
	t.Cleanup(func() {
		generateAffCodeCandidate = originGenerator
	})

	code, err := GenerateUniqueAffCode()
	require.NoError(t, err)
	assert.Equal(t, "unique000001", code)
	assert.Equal(t, 2, calls)
}
