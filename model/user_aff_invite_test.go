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
	require.NoError(t, DB.AutoMigrate(&User{}, &Log{}, &InviteCommissionLedger{}, &InviteCommissionDailyCapState{}))
	require.NoError(t, DB.Exec("DELETE FROM logs").Error)
	require.NoError(t, DB.Exec("DELETE FROM invite_commission_ledgers").Error)
	require.NoError(t, DB.Exec("DELETE FROM invite_commission_daily_cap_states").Error)
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

func TestGetUserInviteRechargeCommissionsAnonymizesAndAggregates(t *testing.T) {
	setupUserAffInviteTestDB(t)

	inviter := createUserAffInviteTestInviter(t, "commission_inviter", "commission_inviter_aff", common.UserStatusEnabled)
	invitee1 := &User{Username: "commission_invitee_1", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "commission_invitee_1_aff", InviterId: inviter.Id}
	invitee2 := &User{Username: "commission_invitee_2", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "commission_invitee_2_aff", InviterId: inviter.Id}
	require.NoError(t, DB.Create(invitee1).Error)
	require.NoError(t, DB.Create(invitee2).Error)
	require.NoError(t, DB.Create(&[]InviteCommissionLedger{
		{
			InviteeUserId:   invitee1.Id,
			InviterUserId:   inviter.Id,
			TopupTradeNo:    "invitee-1-topup-1",
			BizDate:         "2026-07-05",
			BaseQuota:       100,
			CommissionRate:  0.1,
			CommissionQuota: 10,
			SettledQuota:    10,
			Status:          InviteCommissionStatusSettled,
		},
		{
			InviteeUserId:   invitee1.Id,
			InviterUserId:   inviter.Id,
			TopupTradeNo:    "invitee-1-topup-2",
			BizDate:         "2026-07-05",
			BaseQuota:       200,
			CommissionRate:  0.1,
			CommissionQuota: 20,
			SettledQuota:    20,
			Status:          InviteCommissionStatusSettled,
		},
		{
			InviteeUserId:   invitee2.Id,
			InviterUserId:   inviter.Id,
			TopupTradeNo:    "invitee-2-pending",
			BizDate:         "2026-07-05",
			BaseQuota:       300,
			CommissionRate:  0.1,
			CommissionQuota: 30,
			Status:          InviteCommissionStatusPending,
		},
	}).Error)

	items, total, summary, err := GetUserInviteRechargeCommissions(inviter.Id, 0, 10)

	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Equal(t, 30, summary.RechargeTotalQuota)
	require.Len(t, items, 2)
	require.Equal(t, "用户1", items[0].Alias)
	require.Equal(t, 30, items[0].RechargeCommissionQuota)
	require.Equal(t, "用户2", items[1].Alias)
	require.Equal(t, 0, items[1].RechargeCommissionQuota)
}
