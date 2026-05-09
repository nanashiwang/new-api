package model

import (
	"errors"
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

func TestDeleteIPBlacklistByIDNotFound(t *testing.T) {
	setupIPBlacklistTestDB(t)

	err := DeleteIPBlacklistByID(12345)
	require.True(t, errors.Is(err, ErrIPBlacklistNotFound) || errors.Is(err, gorm.ErrRecordNotFound))
}
