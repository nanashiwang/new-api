package model

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserAffInviteTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&User{}, &Log{}, &TopUp{}, &InviteCommissionLedger{}, &InviteCommissionDailyCapState{}))
	require.NoError(t, DB.Exec("DELETE FROM logs").Error)
	require.NoError(t, DB.Exec("DELETE FROM invite_commission_ledgers").Error)
	require.NoError(t, DB.Exec("DELETE FROM invite_commission_daily_cap_states").Error)
	require.NoError(t, DB.Exec("DELETE FROM top_ups").Error)
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

func withInviteBindingTestSettings(t *testing.T, threshold int, rate int, roll func() (int, error)) {
	t.Helper()
	originalSettings := common.GetInviteBindingSettings()
	originalRoll := inviteBindingRoll
	require.NoError(t, common.SetInviteBindingSettings(common.InviteBindingSettings{
		Threshold:          threshold,
		RateAfterThreshold: rate,
	}))
	if roll != nil {
		inviteBindingRoll = roll
	}
	t.Cleanup(func() {
		require.NoError(t, common.SetInviteBindingSettings(originalSettings))
		inviteBindingRoll = originalRoll
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
	// A zero threshold disables throttling regardless of the stored rate.
	withInviteBindingTestSettings(t, 0, 0, nil)

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
	withInviteBindingTestSettings(t, 0, 100, nil)

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

func TestInviteBindingUsesGuaranteedSlotBeforeProbability(t *testing.T) {
	setupUserAffInviteTestDB(t)
	withInviteQuotaSettings(t, 13, 7, 11)
	rollCalls := 0
	withInviteBindingTestSettings(t, 2, 20, func() (int, error) {
		rollCalls++
		return 9_999, nil
	})

	inviter := createUserAffInviteTestInviter(t, "guaranteed_inviter", "guaranteed_aff", common.UserStatusEnabled)
	require.NoError(t, DB.Model(inviter).Update("aff_count", 1).Error)
	invitee := &User{
		Username: "guaranteed_invitee", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
	}

	require.NoError(t, invitee.Insert(inviter.Id))
	assert.Equal(t, 0, rollCalls, "the final guaranteed slot must not use probability")

	var savedInvitee User
	require.NoError(t, DB.First(&savedInvitee, invitee.Id).Error)
	assert.Equal(t, inviter.Id, savedInvitee.InviterId)
	assert.Equal(t, 20, savedInvitee.Quota)

	var savedInviter User
	require.NoError(t, DB.First(&savedInviter, inviter.Id).Error)
	assert.Equal(t, 2, savedInviter.AffCount)
	assert.Equal(t, 11, savedInviter.AffQuota)
	assert.Equal(t, 11, savedInviter.AffHistoryQuota)
}

func TestInviteBindingAfterThreshold(t *testing.T) {
	tests := []struct {
		name      string
		rate      int
		roll      int
		rollErr   error
		wantBound bool
	}{
		{name: "probability hit", rate: 20, roll: 1_999, wantBound: true},
		{name: "probability boundary misses", rate: 20, roll: 2_000, wantBound: false},
		{name: "zero percent always misses", rate: 0, roll: 0, wantBound: false},
		{name: "one hundred percent always binds", rate: 100, roll: 9_999, wantBound: true},
		{name: "random failure fails closed", rate: 20, rollErr: errors.New("random unavailable"), wantBound: false},
		{name: "negative random value fails closed", rate: 20, roll: -1, wantBound: false},
		{name: "oversized random value fails closed", rate: 20, roll: 10_000, wantBound: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUserAffInviteTestDB(t)
			withInviteQuotaSettings(t, 13, 7, 11)
			withInviteBindingTestSettings(t, 1, tt.rate, func() (int, error) {
				return tt.roll, tt.rollErr
			})

			inviter := createUserAffInviteTestInviter(t, "threshold_inviter", "threshold_aff", common.UserStatusEnabled)
			require.NoError(t, DB.Model(inviter).Updates(map[string]any{
				"aff_count": 1, "aff_quota": 5, "aff_history": 5,
			}).Error)
			invitee := &User{
				Username: "threshold_invitee", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
			}

			require.NoError(t, invitee.Insert(inviter.Id), "registration must succeed regardless of binding outcome")

			var savedInvitee User
			require.NoError(t, DB.First(&savedInvitee, invitee.Id).Error)
			var savedInviter User
			require.NoError(t, DB.First(&savedInviter, inviter.Id).Error)
			if tt.wantBound {
				assert.Equal(t, inviter.Id, savedInvitee.InviterId)
				assert.Equal(t, 20, savedInvitee.Quota)
				assert.Equal(t, 2, savedInviter.AffCount)
				assert.Equal(t, 16, savedInviter.AffQuota)
				assert.Equal(t, 16, savedInviter.AffHistoryQuota)
			} else {
				assert.Zero(t, savedInvitee.InviterId)
				assert.Equal(t, 13, savedInvitee.Quota)
				assert.Equal(t, 1, savedInviter.AffCount)
				assert.Equal(t, 5, savedInviter.AffQuota)
				assert.Equal(t, 5, savedInviter.AffHistoryQuota)

				var userLogs []Log
				require.NoError(t, LOG_DB.Where("user_id = ?", savedInvitee.Id).Find(&userLogs).Error)
				for _, log := range userLogs {
					assert.False(t, strings.Contains(log.Content, "邀请"), "probability miss must be silent to the invitee")
				}
			}
		})
	}
}

func TestInviteBindingIgnoresInvalidInviter(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		delete bool
	}{
		{name: "disabled", status: common.UserStatusDisabled},
		{name: "deleted", status: common.UserStatusEnabled, delete: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setupUserAffInviteTestDB(t)
			withInviteQuotaSettings(t, 13, 7, 11)
			withInviteBindingTestSettings(t, 0, 100, nil)

			inviter := createUserAffInviteTestInviter(t, "invalid_inviter", "invalid_aff", tt.status)
			if tt.delete {
				require.NoError(t, DB.Delete(inviter).Error)
			}
			invitee := &User{
				Username: "invalid_invitee", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
			}

			require.NoError(t, invitee.Insert(inviter.Id))
			var savedInvitee User
			require.NoError(t, DB.First(&savedInvitee, invitee.Id).Error)
			assert.Zero(t, savedInvitee.InviterId)
			assert.Equal(t, 13, savedInvitee.Quota)

			var savedInviter User
			query := DB.Unscoped().First(&savedInviter, inviter.Id)
			require.NoError(t, query.Error)
			assert.Zero(t, savedInviter.AffCount)
			assert.Zero(t, savedInviter.AffQuota)
		})
	}
}

func TestInviteBindingRollsBackWithRegistration(t *testing.T) {
	setupUserAffInviteTestDB(t)
	withInviteQuotaSettings(t, 13, 7, 11)
	withInviteBindingTestSettings(t, 0, 100, nil)

	inviter := createUserAffInviteTestInviter(t, "rollback_inviter", "rollback_aff", common.UserStatusEnabled)
	invitee := &User{
		Username: "rollback_invitee", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
	}
	wantErr := errors.New("rollback registration")
	err := DB.Transaction(func(tx *gorm.DB) error {
		require.NoError(t, invitee.InsertWithTx(tx, inviter.Id))
		return wantErr
	})
	require.ErrorIs(t, err, wantErr)

	var inviteeCount int64
	require.NoError(t, DB.Model(&User{}).Where("username = ?", invitee.Username).Count(&inviteeCount).Error)
	assert.Zero(t, inviteeCount)
	var savedInviter User
	require.NoError(t, DB.First(&savedInviter, inviter.Id).Error)
	assert.Zero(t, savedInviter.AffCount)
	assert.Zero(t, savedInviter.AffQuota)
	assert.Zero(t, savedInviter.AffHistoryQuota)
}

func TestFinalizeOAuthUserCreationIsIdempotent(t *testing.T) {
	setupUserAffInviteTestDB(t)
	withInviteQuotaSettings(t, 13, 7, 11)
	withInviteBindingTestSettings(t, 0, 100, nil)

	inviter := createUserAffInviteTestInviter(t, "idempotent_inviter", "idempotent_aff", common.UserStatusEnabled)
	invitee := &User{
		Username: "idempotent_invitee", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
	}
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return invitee.InsertWithTx(tx, inviter.Id)
	}))

	invitee.FinalizeOAuthUserCreation(inviter.Id)
	invitee.FinalizeOAuthUserCreation(inviter.Id)

	var inviteeLogCount int64
	require.NoError(t, LOG_DB.Model(&Log{}).Where("user_id = ?", invitee.Id).Count(&inviteeLogCount).Error)
	assert.EqualValues(t, 2, inviteeLogCount)
	var inviterLogCount int64
	require.NoError(t, LOG_DB.Model(&Log{}).Where("user_id = ?", inviter.Id).Count(&inviterLogCount).Error)
	assert.EqualValues(t, 1, inviterLogCount)

	var savedInviter User
	require.NoError(t, DB.First(&savedInviter, inviter.Id).Error)
	assert.Equal(t, 1, savedInviter.AffCount)
	assert.Equal(t, 11, savedInviter.AffQuota)
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
	invitee1 := &User{Username: "commission_invitee_1", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "commission_invitee_1_aff", InviterId: inviter.Id, CreatedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local).Unix()}
	invitee2 := &User{Username: "commission_invitee_2", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "commission_invitee_2_aff", InviterId: inviter.Id, CreatedAt: time.Date(2026, 7, 2, 12, 0, 0, 0, time.Local).Unix()}
	require.NoError(t, DB.Create(invitee1).Error)
	require.NoError(t, DB.Create(invitee2).Error)
	require.NoError(t, DB.Create(&[]TopUp{
		{UserId: invitee1.Id, Amount: 100, Money: 100, PaidMoney: 80, TradeNo: "invitee-1-topup-1", Status: common.TopUpStatusSuccess},
		{UserId: invitee1.Id, Amount: 200, Money: 120, TradeNo: "invitee-1-topup-2", Status: common.TopUpStatusSuccess},
		{UserId: invitee2.Id, Amount: 300, Money: 300, PaidMoney: 300, TradeNo: "invitee-2-pending", Status: common.TopUpStatusPending},
	}).Error)
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
	require.Equal(t, "2026-07-01", items[0].RegisteredDate)
	assert.InDelta(t, 200, items[0].RechargeTotalMoney, 0.000001)
	require.Equal(t, 30, items[0].RechargeCommissionQuota)
	require.Equal(t, "用户2", items[1].Alias)
	require.Equal(t, "2026-07-02", items[1].RegisteredDate)
	assert.Zero(t, items[1].RechargeTotalMoney)
	require.Equal(t, 0, items[1].RechargeCommissionQuota)
}
