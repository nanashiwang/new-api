package model

import (
	"errors"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupIPBlacklistTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&User{}, &IPBlacklist{}))
	require.NoError(t, DB.Exec("DELETE FROM ip_blacklists").Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
	InvalidateIPBlacklistCache()
	t.Cleanup(InvalidateIPBlacklistCache)
}

func TestNormalizeIPBlacklistRule(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCIDR    string
		wantVersion int
		wantErr     bool
	}{
		{name: "ipv4 single", input: "203.0.113.8", wantCIDR: "203.0.113.8/32", wantVersion: 4},
		{name: "ipv4 cidr", input: "203.0.113.88/24", wantCIDR: "203.0.113.0/24", wantVersion: 4},
		{name: "ipv6 single", input: "2001:db8::1", wantCIDR: "2001:db8::1/128", wantVersion: 6},
		{name: "ipv6 cidr", input: "2001:db8::abcd/64", wantCIDR: "2001:db8::/64", wantVersion: 6},
		{name: "invalid", input: "not-an-ip", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCIDR, gotVersion, err := NormalizeIPBlacklistRule(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidIPBlacklistRule)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantCIDR, gotCIDR)
			require.Equal(t, tt.wantVersion, gotVersion)
		})
	}
}

func TestCreateFindDeleteIPBlacklist(t *testing.T) {
	setupIPBlacklistTestDB(t)

	item, created, err := CreateIPBlacklist("203.0.113.88/24", "batch signup", 7, 100)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "203.0.113.0/24", item.CIDR)
	require.Equal(t, 4, item.IPVersion)

	duplicate, created, err := CreateIPBlacklist("203.0.113.0/24", "duplicate", 0, 100)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, item.Id, duplicate.Id)

	match, err := FindIPBlacklistMatch("203.0.113.44")
	require.NoError(t, err)
	require.NotNil(t, match)
	require.Equal(t, item.Id, match.Id)

	require.NoError(t, DeleteIPBlacklistByID(item.Id))
	match, err = FindIPBlacklistMatch("203.0.113.44")
	require.NoError(t, err)
	require.Nil(t, match)
}

func TestBatchCreateIPBlacklistFromUsers(t *testing.T) {
	setupIPBlacklistTestDB(t)

	users := []User{
		{Username: "ip_user_1", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "ip_user_1_aff", RegisterIP: "203.0.113.10"},
		{Username: "ip_user_2", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "ip_user_2_aff", RegisterIP: "203.0.113.10"},
		{Username: "ip_user_3", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "ip_user_3_aff", RegisterIP: "2001:db8::1"},
		{Username: "ip_user_4", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "ip_user_4_aff"},
	}
	require.NoError(t, DB.Create(&users).Error)

	result, err := BatchCreateIPBlacklistFromUsers(
		[]int{users[0].Id, users[1].Id, users[2].Id, users[3].Id, 99999},
		"bulk",
		100,
	)
	require.NoError(t, err)
	require.Equal(t, 2, result.CreatedCount)
	require.Equal(t, 2, result.SkippedCount)
	require.Equal(t, 1, result.FailedCount)
	require.Len(t, result.Items, 2)

	match, err := FindUserRegisterIPMatch([]int{users[0].Id, users[2].Id}, "2001:db8::1")
	require.NoError(t, err)
	require.Equal(t, "2001:db8::1", match)
}

func TestSearchUsersByIPBlacklistIDAggregatesSingleIPAndCIDR(t *testing.T) {
	setupIPBlacklistTestDB(t)

	users := []User{
		{Username: "ip_match_single_1", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "ip_match_single_1_aff", RegisterIP: "203.0.113.10"},
		{Username: "ip_match_single_2", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusDisabled, AffCode: "ip_match_single_2_aff", RegisterIP: "203.0.113.10"},
		{Username: "ip_match_cidr_1", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "ip_match_cidr_1_aff", RegisterIP: "203.0.113.99"},
		{Username: "ip_match_outside", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "ip_match_outside_aff", RegisterIP: "198.51.100.20"},
	}
	require.NoError(t, DB.Create(&users).Error)

	singleRule, created, err := CreateIPBlacklist("203.0.113.10", "single ip", 0, 100)
	require.NoError(t, err)
	require.True(t, created)
	singleUsers, singleTotal, err := SearchUsersByIPBlacklistID(singleRule.Id, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 2, singleTotal)
	require.Len(t, singleUsers, 2)
	require.Equal(t, users[1].Id, singleUsers[0].Id)
	require.Equal(t, users[0].Id, singleUsers[1].Id)

	cidrRule, created, err := CreateIPBlacklist("203.0.113.0/24", "cidr", 0, 100)
	require.NoError(t, err)
	require.True(t, created)
	cidrUsers, cidrTotal, err := SearchUsersByIPBlacklistID(cidrRule.Id, &common.PageInfo{Page: 1, PageSize: 2})
	require.NoError(t, err)
	require.EqualValues(t, 3, cidrTotal)
	require.Len(t, cidrUsers, 2)

	require.NoError(t, AttachIPBlacklistUserPreview([]*IPBlacklist{cidrRule}, 5))
	require.Equal(t, 3, cidrRule.MatchedUserCount)
	require.Len(t, cidrRule.MatchedUsers, 3)
}

func TestGetUserIDsByIPBlacklistIDReturnsAllCIDRMatches(t *testing.T) {
	setupIPBlacklistTestDB(t)

	users := make([]User, 0, 105)
	for i := 0; i < 105; i++ {
		users = append(users, User{
			Username:   fmt.Sprintf("ip_all_match_%03d", i),
			Password:   "password123",
			Role:       common.RoleCommonUser,
			Status:     common.UserStatusEnabled,
			AffCode:    fmt.Sprintf("ip_all_match_%03d_aff", i),
			RegisterIP: fmt.Sprintf("198.51.100.%d", (i%200)+1),
		})
	}
	require.NoError(t, DB.Create(&users).Error)

	rule, created, err := CreateIPBlacklist("198.51.100.0/24", "all", 0, 100)
	require.NoError(t, err)
	require.True(t, created)

	ids, err := GetUserIDsByIPBlacklistID(rule.Id)
	require.NoError(t, err)
	require.Len(t, ids, 105)

	pageUsers, total, err := SearchUsersByIPBlacklistID(rule.Id, &common.PageInfo{Page: 1, PageSize: 1000})
	require.NoError(t, err)
	require.EqualValues(t, 105, total)
	require.Len(t, pageUsers, 100)
}

func TestDeleteIPBlacklistByIDNotFound(t *testing.T) {
	setupIPBlacklistTestDB(t)

	err := DeleteIPBlacklistByID(12345)
	require.True(t, errors.Is(err, ErrIPBlacklistNotFound) || errors.Is(err, gorm.ErrRecordNotFound))
}
