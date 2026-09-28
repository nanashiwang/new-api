package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChannelUsageSeparateLogDBAndExactWindow(t *testing.T) {
	setupChannelQueryTestDB(t)
	// Only the log database contains logs, and it contains no channels table.
	logs, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	LOG_DB = logs
	require.NoError(t, logs.AutoMigrate(&Log{}))
	require.NoError(t, logs.Create(&[]Log{
		{ChannelId: 1, CreatedAt: 100, Type: LogTypeConsume, Quota: 10, ModelName: "gpt-4o", Group: "vip"},
		{ChannelId: 1, CreatedAt: 199, Type: LogTypeError, Quota: 2, ModelName: "gpt-4o", Group: "vip"},
		{ChannelId: 1, CreatedAt: 200, Type: LogTypeConsume, Quota: 1000, ModelName: "gpt-4o", Group: "vip"},
		{ChannelId: 1, CreatedAt: 99, Type: LogTypeConsume, Quota: 1000, ModelName: "gpt-4o", Group: "vip"},
		{ChannelId: 1, CreatedAt: 150, Type: LogTypeRefund, Quota: 1000, ModelName: "gpt-4o", Group: "vip"},
		{ChannelId: 2, CreatedAt: 150, Type: LogTypeConsume, Quota: 400, ModelName: "gemini", Group: "vip"},
		{ChannelId: 3, CreatedAt: 150, Type: LogTypeConsume, Quota: 500, ModelName: "gpt-4o", Group: "default"},
	}).Error)
	rows, err := loadChannelUsage(100, 200, "vip", "gpt")
	require.NoError(t, err)
	require.Equal(t, []ChannelUsage{{ChannelID: 1, TotalQuota: 12, TotalRequests: 2, SuccessRequests: 1, FailedRequests: 1}}, rows)
	rows, err = loadChannelUsage(100, 200, "missing", "")
	require.NoError(t, err)
	require.Empty(t, rows)
	monitor, err := loadChannelMonitorStats(100, 200, "group", "", true)
	require.NoError(t, err)
	var monitorQuota int64
	for _, s := range monitor {
		monitorQuota += s.TotalQuota
	}
	all, err := loadChannelUsage(100, 200, "", "")
	require.NoError(t, err)
	var usageQuota int64
	for _, s := range all {
		usageQuota += s.TotalQuota
	}
	require.Equal(t, monitorQuota, usageQuota, "dashboard and channel usage must share accounting and exact boundaries")
	deleted, err := GetDeletedChannelUsage(all)
	require.NoError(t, err)
	require.Equal(t, usageQuota, deleted.TotalQuota)
	require.NoError(t, DB.Create(&Channel{Id: 1, Key: "not-returned", Name: "present"}).Error)
	deleted, err = GetDeletedChannelUsage(all)
	require.NoError(t, err)
	require.EqualValues(t, 900, deleted.TotalQuota)
	require.NoError(t, logs.Migrator().DropTable(&Log{}))
	_, err = loadChannelUsage(100, 200, "", "")
	require.Error(t, err, "query failure must not become a zero-usage result")
}

func TestChannelUsageSortBeforePaginationAndTagTotals(t *testing.T) {
	a, b := "a", "b"
	channels := []*Channel{{Id: 1, Tag: &a}, {Id: 2, Tag: &b}, {Id: 3, Tag: &a}, {Id: 4}}
	stats := []ChannelUsage{{ChannelID: 1, TotalQuota: 60, TotalRequests: 2, SuccessRequests: 1, FailedRequests: 1}, {ChannelID: 2, TotalQuota: 100, TotalRequests: 1, SuccessRequests: 1}, {ChannelID: 3, TotalQuota: 70, TotalRequests: 3, SuccessRequests: 3}, {ChannelID: 999, TotalQuota: 9999}}
	items, total, summary := PaginateChannelUsage(channels, stats, false, false, 1, 1)
	require.Equal(t, 4, total)
	require.Len(t, items, 1)
	require.Equal(t, 3, items[0].Id)
	require.EqualValues(t, 230, summary.TotalQuota)
	require.EqualValues(t, 6, summary.TotalRequests)
	items, total, summary = PaginateChannelUsage(channels, stats, true, false, 0, 1)
	require.Equal(t, 2, total)
	require.Len(t, items, 2)
	require.Equal(t, 1, items[0].Id)
	require.Equal(t, 3, items[1].Id)
	require.EqualValues(t, 230, summary.TotalQuota)
	items, _, _ = PaginateChannelUsage(channels, stats, false, true, 0, 1)
	require.Equal(t, 4, items[0].Id)
	require.Zero(t, items[0].Usage.TotalQuota)
	items, _, _ = PaginateChannelUsage(channels, stats, false, true, 100, 10)
	require.Empty(t, items)
}
