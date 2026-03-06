package model

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	inviteCommissionRedemptionMigrateOnce sync.Once
	inviteCommissionRedemptionMigrateErr  error
)

func setupInviteCommissionRedemptionTest(t *testing.T) {
	t.Helper()
	inviteCommissionRedemptionMigrateOnce.Do(func() {
		inviteCommissionRedemptionMigrateErr = DB.AutoMigrate(&User{}, &Redemption{}, &InviteCommissionLedger{}, &InviteCommissionDailyCapState{})
	})
	require.NoError(t, inviteCommissionRedemptionMigrateErr)
	require.NoError(t, DB.Exec("DELETE FROM invite_commission_ledgers").Error)
	require.NoError(t, DB.Exec("DELETE FROM invite_commission_daily_cap_states").Error)
	require.NoError(t, DB.Exec("DELETE FROM redemptions").Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
}

func createInviteCommissionTestUser(t *testing.T, username string, inviterID int) *User {
	t.Helper()
	user := &User{
		Username:    username,
		Password:    "test-password",
		DisplayName: username,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		InviterId:   inviterID,
		AffCode:     fmt.Sprintf("aff_%s", username),
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestEnqueueInviteCommissionFromRedemption_CreatesLedger(t *testing.T) {
	setupInviteCommissionRedemptionTest(t)

	originEnabled := common.InviterCommissionEnabled
	originRate := common.InviterRechargeCommissionRate
	t.Cleanup(func() {
		common.InviterCommissionEnabled = originEnabled
		common.InviterRechargeCommissionRate = originRate
	})
	common.InviterCommissionEnabled = true
	common.InviterRechargeCommissionRate = 0.1

	inviter := createInviteCommissionTestUser(t, "inviter_redemption", 0)
	invitee := createInviteCommissionTestUser(t, "invitee_redemption", inviter.Id)

	redeemedAt := int64(1700000000)
	dbRedemption := &Redemption{
		Id:           101,
		Key:          "0123456789abcdef0123456789abcdef",
		Status:       common.RedemptionCodeStatusUsed,
		UsedUserId:   invitee.Id,
		Quota:        300,
		RedeemedTime: redeemedAt,
	}
	require.NoError(t, DB.Create(dbRedemption).Error)

	// 传入对象中的关键字段被篡改，也应以 DB 中记录为准。
	redemption := &Redemption{
		Id:           101,
		UsedUserId:   99999,
		Quota:        1,
		RedeemedTime: 1,
	}

	require.NoError(t, EnqueueInviteCommissionFromRedemption(redemption))
	// 同一个兑换码重复入池应被唯一索引忽略（幂等）。
	require.NoError(t, EnqueueInviteCommissionFromRedemption(redemption))

	var ledger InviteCommissionLedger
	require.NoError(t, DB.Where("topup_trade_no = ? AND inviter_user_id = ?", "redeem:101", inviter.Id).First(&ledger).Error)

	assert.Equal(t, invitee.Id, ledger.InviteeUserId)
	assert.Equal(t, inviter.Id, ledger.InviterUserId)
	assert.Equal(t, 300, ledger.BaseQuota)
	assert.Equal(t, 30, ledger.CommissionQuota)
	assert.Equal(t, InviteCommissionStatusPending, ledger.Status)
	assert.Equal(t, time.Unix(redeemedAt, 0).Format("2006-01-02"), ledger.BizDate)

	var count int64
	require.NoError(t, DB.Model(&InviteCommissionLedger{}).Where("topup_trade_no = ? AND inviter_user_id = ?", "redeem:101", inviter.Id).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestAdminDirectQuotaUpdate_DoesNotCreateInviteCommissionLedger(t *testing.T) {
	setupInviteCommissionRedemptionTest(t)

	originEnabled := common.InviterCommissionEnabled
	originRate := common.InviterRechargeCommissionRate
	t.Cleanup(func() {
		common.InviterCommissionEnabled = originEnabled
		common.InviterRechargeCommissionRate = originRate
	})
	common.InviterCommissionEnabled = true
	common.InviterRechargeCommissionRate = 0.1

	inviter := createInviteCommissionTestUser(t, "inviter_direct_add", 0)
	invitee := createInviteCommissionTestUser(t, "invitee_direct_add", inviter.Id)

	// 模拟管理员直接修改余额（非充值、非兑换码），应不触发返佣入池。
	require.NoError(t, DB.Model(&User{}).Where("id = ?", invitee.Id).Update("quota", gorm.Expr("quota + ?", 500)).Error)

	var count int64
	require.NoError(t, DB.Model(&InviteCommissionLedger{}).Count(&count).Error)
	assert.EqualValues(t, 0, count)
}
