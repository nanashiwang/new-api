package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserSubscriptionFilterTest(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&User{}, &UserSubscription{}))
	require.NoError(t, ensureSubscriptionPlanTableSQLite())
	require.NoError(t, DB.Exec("DELETE FROM user_subscriptions").Error)
	require.NoError(t, DB.Exec("DELETE FROM subscription_plans").Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
}

func createUserForSubscriptionFilterTest(t *testing.T, name string) *User {
	t.Helper()
	user := &User{
		Username:    name,
		Password:    "test-password",
		DisplayName: name,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "aff_" + name,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func createUserSubscriptionForFilterTest(t *testing.T, userID int, status string, endTime int64) {
	t.Helper()
	now := common.GetTimestamp()
	sub := &UserSubscription{
		UserId:      userID,
		PlanId:      1,
		AmountTotal: 100,
		AmountUsed:  0,
		Status:      status,
		Source:      "admin",
		StartTime:   now - 60,
		EndTime:     endTime,
	}
	require.NoError(t, DB.Create(sub).Error)
}

func createSubscriptionPlanForFilterTest(t *testing.T, title string, resetPeriod string, totalAmount int64) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Title:               title,
		PriceAmount:         1,
		Currency:            "USD",
		DurationUnit:        SubscriptionDurationMonth,
		DurationValue:       1,
		Enabled:             true,
		TotalAmount:         totalAmount,
		QuotaResetPeriod:    resetPeriod,
		UpgradeGroup:        "",
		PurchaseQuantityMin: 1,
		PurchaseQuantityMax: 1,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func createDetailedUserSubscriptionForFilterTest(
	t *testing.T,
	userID int,
	planID int,
	status string,
	startTime int64,
	endTime int64,
	amountTotal int64,
	amountUsed int64,
	lastReset int64,
	nextReset int64,
) {
	t.Helper()
	sub := &UserSubscription{
		UserId:        userID,
		PlanId:        planID,
		Status:        status,
		Source:        "admin",
		StartTime:     startTime,
		EndTime:       endTime,
		AmountTotal:   amountTotal,
		AmountUsed:    amountUsed,
		LastResetTime: lastReset,
		NextResetTime: nextReset,
	}
	require.NoError(t, DB.Create(sub).Error)
}

func TestSearchUsersWithParams_HasActiveSubscriptionFilter(t *testing.T) {
	setupUserSubscriptionFilterTest(t)
	now := common.GetTimestamp()

	activeUser := createUserForSubscriptionFilterTest(t, "active_user")
	expiredUser := createUserForSubscriptionFilterTest(t, "expired_user")
	noSubUser := createUserForSubscriptionFilterTest(t, "no_sub_user")

	createUserSubscriptionForFilterTest(t, activeUser.Id, "active", now+3600)
	createUserSubscriptionForFilterTest(t, expiredUser.Id, "active", now-30)

	hasActive := true
	activeUsers, totalActive, err := SearchUsersWithParams(UserSearchParams{
		HasActiveSubscription: &hasActive,
		SortBy:                "id",
		SortOrder:             "asc",
		StartIdx:              0,
		PageSize:              20,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, totalActive)
	require.Len(t, activeUsers, 1)
	assert.Equal(t, activeUser.Id, activeUsers[0].Id)
	assert.True(t, activeUsers[0].HasActiveSubscription)
	assert.Equal(t, 1, activeUsers[0].ActiveSubscriptionCount)

	hasNoActive := false
	inactiveUsers, totalInactive, err := SearchUsersWithParams(UserSearchParams{
		HasActiveSubscription: &hasNoActive,
		SortBy:                "id",
		SortOrder:             "asc",
		StartIdx:              0,
		PageSize:              20,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, totalInactive)

	expectedIDs := map[int]struct{}{
		expiredUser.Id: {},
		noSubUser.Id:   {},
	}
	for _, user := range inactiveUsers {
		_, ok := expectedIDs[user.Id]
		assert.True(t, ok, fmt.Sprintf("unexpected user id: %d", user.Id))
		assert.False(t, user.HasActiveSubscription)
		assert.Equal(t, 0, user.ActiveSubscriptionCount)
	}
}

func TestGetAllUsers_AttachesActiveSubscriptionMetadata(t *testing.T) {
	setupUserSubscriptionFilterTest(t)
	now := common.GetTimestamp()

	userWithSubs := createUserForSubscriptionFilterTest(t, "subs_user")
	userWithoutSubs := createUserForSubscriptionFilterTest(t, "plain_user")

	createUserSubscriptionForFilterTest(t, userWithSubs.Id, "active", now+3600)
	createUserSubscriptionForFilterTest(t, userWithSubs.Id, "active", now+7200)

	pageInfo := &common.PageInfo{
		Page:     1,
		PageSize: 20,
	}
	users, total, err := GetAllUsers(pageInfo, "id", "asc", "", "")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)

	userMap := make(map[int]*User, len(users))
	for _, user := range users {
		userMap[user.Id] = user
	}
	require.Contains(t, userMap, userWithSubs.Id)
	require.Contains(t, userMap, userWithoutSubs.Id)

	assert.True(t, userMap[userWithSubs.Id].HasActiveSubscription)
	assert.Equal(t, 2, userMap[userWithSubs.Id].ActiveSubscriptionCount)
	assert.EqualValues(t, 200, userMap[userWithSubs.Id].SubscriptionQuotaRemaining)
	assert.EqualValues(t, 200, userMap[userWithSubs.Id].SubscriptionQuotaTotal)
	assert.False(t, userMap[userWithSubs.Id].SubscriptionQuotaHasUnlimited)
	assert.False(t, userMap[userWithoutSubs.Id].HasActiveSubscription)
	assert.Equal(t, 0, userMap[userWithoutSubs.Id].ActiveSubscriptionCount)
	assert.Zero(t, userMap[userWithoutSubs.Id].SubscriptionQuotaRemaining)
	assert.Zero(t, userMap[userWithoutSubs.Id].SubscriptionQuotaTotal)
	assert.False(t, userMap[userWithoutSubs.Id].SubscriptionQuotaHasUnlimited)
}

func TestSearchUsersWithParams_WalletBalanceFilterUsesCurrentQuotaOnly(t *testing.T) {
	setupUserSubscriptionFilterTest(t)

	walletRich := createUserForSubscriptionFilterTest(t, "wallet_rich")
	usedHeavy := createUserForSubscriptionFilterTest(t, "used_heavy")
	walletLow := createUserForSubscriptionFilterTest(t, "wallet_low")

	require.NoError(t, DB.Model(&User{}).Where("id = ?", walletRich.Id).Updates(map[string]any{
		"quota":      200,
		"used_quota": 5,
	}).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", usedHeavy.Id).Updates(map[string]any{
		"quota":      20,
		"used_quota": 500,
	}).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", walletLow.Id).Updates(map[string]any{
		"quota":      8,
		"used_quota": 0,
	}).Error)

	walletMin := 100
	users, total, err := SearchUsersWithParams(UserSearchParams{
		WalletMin: &walletMin,
		SortBy:    "id",
		SortOrder: "asc",
		StartIdx:  0,
		PageSize:  20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, users, 1)
	assert.Equal(t, walletRich.Id, users[0].Id)
}

func TestGetAllUsers_AggregatesSubscriptionQuotaWithResetAwareView(t *testing.T) {
	setupUserSubscriptionFilterTest(t)
	now := common.GetTimestamp()

	user := createUserForSubscriptionFilterTest(t, "quota_user")
	dailyPlan := createSubscriptionPlanForFilterTest(t, "daily_plan", SubscriptionResetDaily, 100)
	weeklyPlan := createSubscriptionPlanForFilterTest(t, "weekly_plan", SubscriptionResetWeekly, 100)
	monthlyPlan := createSubscriptionPlanForFilterTest(t, "monthly_plan", SubscriptionResetMonthly, 100)
	unlimitedPlan := createSubscriptionPlanForFilterTest(t, "unlimited_plan", SubscriptionResetNever, 0)

	createDetailedUserSubscriptionForFilterTest(
		t,
		user.Id,
		dailyPlan.Id,
		"active",
		now-3*86400,
		now+3*86400,
		100,
		70,
		now-2*86400,
		now-3600,
	)
	createDetailedUserSubscriptionForFilterTest(
		t,
		user.Id,
		weeklyPlan.Id,
		"active",
		now-3*86400,
		now+5*86400,
		100,
		20,
		now-2*86400,
		now+5*86400,
	)
	createDetailedUserSubscriptionForFilterTest(
		t,
		user.Id,
		monthlyPlan.Id,
		"active",
		now-7*86400,
		now+25*86400,
		100,
		0,
		now-7*86400,
		now+20*86400,
	)

	pageInfo := &common.PageInfo{
		Page:     1,
		PageSize: 20,
	}
	users, total, err := GetAllUsers(pageInfo, "id", "asc", "", "")
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, users, 1)
	assert.EqualValues(t, 280, users[0].SubscriptionQuotaRemaining)
	assert.EqualValues(t, 300, users[0].SubscriptionQuotaTotal)
	assert.False(t, users[0].SubscriptionQuotaHasUnlimited)

	createDetailedUserSubscriptionForFilterTest(
		t,
		user.Id,
		unlimitedPlan.Id,
		"active",
		now-3600,
		now+86400,
		0,
		10,
		0,
		0,
	)

	users, total, err = GetAllUsers(pageInfo, "id", "asc", "", "")
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, users, 1)
	assert.True(t, users[0].SubscriptionQuotaHasUnlimited)
	assert.Zero(t, users[0].SubscriptionQuotaRemaining)
	assert.Zero(t, users[0].SubscriptionQuotaTotal)
}
