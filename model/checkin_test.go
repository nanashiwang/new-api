package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCheckinTestDB(t *testing.T) {
	t.Helper()

	originDB := DB
	originLogDB := LOG_DB
	originUsingSQLite := common.UsingSQLite
	originSetting := *operation_setting.GetCheckinSetting()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Checkin{}))

	DB = db
	LOG_DB = db
	common.UsingSQLite = false
	setting := operation_setting.GetCheckinSetting()
	setting.Enabled = true
	setting.MinQuota = 100
	setting.MaxQuota = 100
	setting.IPDailyLimit = 2

	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.UsingSQLite = originUsingSQLite
		*operation_setting.GetCheckinSetting() = originSetting
	})
}

func createCheckinTestUser(t *testing.T, username string, registerIP string) int {
	t.Helper()
	user := &User{
		Username:   username,
		Password:   "password123",
		AffCode:    username,
		RegisterIP: registerIP,
	}
	require.NoError(t, DB.Create(user).Error)
	return user.Id
}

func TestUserCheckinBlocksTooManyUsersFromSameIP(t *testing.T) {
	setupCheckinTestDB(t)

	user1 := createCheckinTestUser(t, "checkin_ip_1", "198.51.100.1")
	user2 := createCheckinTestUser(t, "checkin_ip_2", "198.51.100.2")
	user3 := createCheckinTestUser(t, "checkin_ip_3", "198.51.100.3")

	_, err := UserCheckin(user1, "203.0.113.9")
	require.NoError(t, err)
	_, err = UserCheckin(user2, "203.0.113.9")
	require.NoError(t, err)

	_, err = UserCheckin(user3, "203.0.113.9")
	require.ErrorContains(t, err, "当前网络今日签到账号过多")
}

func TestUserCheckinBlocksTooManyUsersFromSameRegisterIP(t *testing.T) {
	setupCheckinTestDB(t)
	operation_setting.GetCheckinSetting().IPDailyLimit = 1

	user1 := createCheckinTestUser(t, "checkin_reg_1", "198.51.100.10")
	user2 := createCheckinTestUser(t, "checkin_reg_2", "198.51.100.10")

	_, err := UserCheckin(user1, "203.0.113.11")
	require.NoError(t, err)

	_, err = UserCheckin(user2, "203.0.113.12")
	require.ErrorContains(t, err, "当前注册来源今日签到账号过多")
}
