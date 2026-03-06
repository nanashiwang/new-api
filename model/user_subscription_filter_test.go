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
	require.NoError(t, DB.Exec("DELETE FROM user_subscriptions").Error)
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
		UserId:    userID,
		PlanId:    1,
		Status:    status,
		Source:    "admin",
		StartTime: now - 60,
		EndTime:   endTime,
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
	assert.False(t, userMap[userWithoutSubs.Id].HasActiveSubscription)
	assert.Equal(t, 0, userMap[userWithoutSubs.Id].ActiveSubscriptionCount)
}
