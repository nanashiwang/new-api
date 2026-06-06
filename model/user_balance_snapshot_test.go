package model

import (
	"math"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserBalanceSnapshotTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	originDB := DB
	originLogDB := LOG_DB
	originUsingSQLite := common.UsingSQLite
	originUsingMySQL := common.UsingMySQL
	originUsingPostgreSQL := common.UsingPostgreSQL
	DB = db
	LOG_DB = db
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	initCol()
	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
		initCol()
	})

	require.NoError(t, db.AutoMigrate(&User{}, &UserBalanceSnapshot{}))
}

func createUserBalanceSnapshotTestUser(t *testing.T, username string, quota int) *User {
	t.Helper()
	user := &User{
		Username:             username,
		Password:             "password123",
		Role:                 common.RoleCommonUser,
		Status:               common.UserStatusEnabled,
		Quota:                quota,
		AffCode:              username + "-aff",
		BalanceTrendDisabled: false,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestSaveUserBalanceSnapshotAggregatesAndUpdatesSameDay(t *testing.T) {
	setupUserBalanceSnapshotTestDB(t)
	createUserBalanceSnapshotTestUser(t, "alice", 1000)
	createUserBalanceSnapshotTestUser(t, "bob", 500)
	createUserBalanceSnapshotTestUser(t, "carol", 0)
	createUserBalanceSnapshotTestUser(t, "dave", -50)

	snapshotTime := time.Date(2026, 6, 3, 4, 0, 0, 0, time.Local)
	snapshot, err := SaveUserBalanceSnapshot(snapshotTime)
	require.NoError(t, err)
	require.Equal(t, "2026-06-03", snapshot.SnapshotDate)
	require.EqualValues(t, 1450, snapshot.TotalQuota)
	require.EqualValues(t, 1500, snapshot.TotalPositiveQuota)
	require.Equal(t, 4, snapshot.UserCount)
	require.Equal(t, 2, snapshot.PositiveUserCount)
	require.Equal(t, 1, snapshot.NegativeUserCount)
	require.EqualValues(t, 1500, snapshot.Top10Quota)

	report, err := GetUserBalanceSnapshotReport(snapshotTime.Add(-time.Hour).Unix(), snapshotTime.Add(time.Hour).Unix())
	require.NoError(t, err)
	require.NotNil(t, report.Latest)
	require.EqualValues(t, 1450, report.Latest.TotalQuota)
	require.Len(t, report.TopUsers, 2)
	require.Equal(t, "alice", report.TopUsers[0].Username)
	require.Len(t, report.NegativeUsers, 1)
	require.Equal(t, "dave", report.NegativeUsers[0].Username)
	require.InDelta(t, 1.0, report.Latest.Top10Share, 0.0001)

	require.NoError(t, DB.Model(&User{}).Where("username = ?", "alice").Update("quota", 2000).Error)
	updated, err := SaveUserBalanceSnapshot(snapshotTime.Add(2 * time.Hour))
	require.NoError(t, err)
	require.Equal(t, snapshot.Id, updated.Id)
	require.EqualValues(t, 2450, updated.TotalQuota)
}

func TestGetUserBalanceSnapshotReportDelta(t *testing.T) {
	setupUserBalanceSnapshotTestDB(t)
	createUserBalanceSnapshotTestUser(t, "alice", 1000)

	firstTime := time.Date(2026, 6, 2, 4, 0, 0, 0, time.Local)
	_, err := SaveUserBalanceSnapshot(firstTime)
	require.NoError(t, err)

	require.NoError(t, DB.Model(&User{}).Where("username = ?", "alice").Update("quota", 1200).Error)
	secondTime := firstTime.AddDate(0, 0, 1)
	_, err = SaveUserBalanceSnapshot(secondTime)
	require.NoError(t, err)

	report, err := GetUserBalanceSnapshotReport(firstTime.Add(-time.Hour).Unix(), secondTime.Add(time.Hour).Unix())
	require.NoError(t, err)
	require.Len(t, report.Snapshots, 2)
	require.EqualValues(t, 200, report.DeltaQuota)
	require.True(t, math.Abs(report.DeltaRate-0.2) < 0.0001)
}

func TestSaveUserBalanceSnapshotSkipsDisabledUsers(t *testing.T) {
	setupUserBalanceSnapshotTestDB(t)
	createUserBalanceSnapshotTestUser(t, "alice", 1000)
	disabled := createUserBalanceSnapshotTestUser(t, "bob", 900)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", disabled.Id).Update("balance_trend_disabled", true).Error)

	snapshotTime := time.Date(2026, 6, 4, 4, 0, 0, 0, time.Local)
	snapshot, err := SaveUserBalanceSnapshot(snapshotTime)
	require.NoError(t, err)
	require.EqualValues(t, 1000, snapshot.TotalQuota)
	require.Equal(t, 1, snapshot.UserCount)
	require.EqualValues(t, 1000, snapshot.Top10Quota)

	report, err := GetUserBalanceSnapshotReport(snapshotTime.Add(-time.Hour).Unix(), snapshotTime.Add(time.Hour).Unix())
	require.NoError(t, err)
	require.Len(t, report.TopUsers, 1)
	require.Equal(t, "alice", report.TopUsers[0].Username)
}

func TestSearchAndUpdateUserBalanceTrendUsers(t *testing.T) {
	setupUserBalanceSnapshotTestDB(t)
	alice := createUserBalanceSnapshotTestUser(t, "alice", 1000)
	createUserBalanceSnapshotTestUser(t, "bob", 500)

	updated, err := UpdateUserBalanceTrendDisabled(alice.Id, true)
	require.NoError(t, err)
	require.True(t, updated.BalanceTrendDisabled)

	excluded, total, err := SearchUserBalanceTrendUsers(UserBalanceTrendUserSearchParams{
		IncludeStatus: "excluded",
		StartIdx:      0,
		PageSize:      10,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, excluded, 1)
	require.Equal(t, "alice", excluded[0].Username)

	included, total, err := SearchUserBalanceTrendUsers(UserBalanceTrendUserSearchParams{
		IncludeStatus: "included",
		StartIdx:      0,
		PageSize:      10,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, included, 1)
	require.Equal(t, "bob", included[0].Username)
}
