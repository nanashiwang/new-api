package model

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGroupHealthPersistenceAcrossProcessesAndExactFilters(t *testing.T) {
	previous := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	t.Cleanup(func() { DB = previous })
	require.NoError(t, db.AutoMigrate(&GroupHealthMetric{}, &Ability{}))
	ctx := context.Background()
	now := time.Now()
	ts := now.Unix() / 3600 * 3600
	rows := []GroupHealthMetric{
		{ID: "process-a", GroupName: "public", ModelName: "model-a", BucketTs: ts, RequestCount: 10, SuccessCount: 8, TTFTSumMs: 800, TTFTCount: 8, CacheReadTokens: 80, CacheInputTokens: 100, CacheSampleCount: 1, UpdatedAt: now.Unix()},
		{ID: "process-b", GroupName: "public", ModelName: "model-a", BucketTs: ts, RequestCount: 5, SuccessCount: 5, CacheReadTokens: 30, CacheInputTokens: 170, CacheSampleCount: 2},
		{ID: "other-model", GroupName: "public", ModelName: "model-b", BucketTs: ts, RequestCount: 100, SuccessCount: 100},
		{ID: "private", GroupName: "private", ModelName: "model-a", BucketTs: ts, RequestCount: 999, SuccessCount: 999},
		{ID: "expired", GroupName: "public", ModelName: "model-a", BucketTs: now.Add(-8 * 24 * time.Hour).Unix(), RequestCount: 1},
	}
	require.NoError(t, SaveGroupHealthSnapshots(ctx, rows))
	require.NoError(t, SaveGroupHealthSnapshots(ctx, rows), "retrying snapshots must not add counts")
	rows[0].RequestCount = 11
	rows[0].CacheInputTokens = 200
	rows[0].CacheSampleCount = 2
	require.NoError(t, SaveGroupHealthSnapshots(ctx, rows[:1]))
	got, err := QueryGroupHealthMetrics(ctx, []string{"public"}, "model-a", ts, now.Unix())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.EqualValues(t, 16, got[0].RequestCount)
	require.EqualValues(t, 13, got[0].SuccessCount)
	require.EqualValues(t, 800, got[0].TTFTSumMs)
	require.EqualValues(t, 110, got[0].CacheReadTokens)
	require.EqualValues(t, 370, got[0].CacheInputTokens)
	require.EqualValues(t, 4, got[0].CacheSampleCount)
	got, err = QueryGroupHealthMetrics(ctx, nil, "", ts, now.Unix())
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, DeleteExpiredGroupHealthMetrics(ctx, now))
	var count int64
	require.NoError(t, db.Model(&GroupHealthMetric{}).Where("id = ?", "expired").Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Create(&Ability{Group: "public", Model: "model-a", ChannelId: 1, Enabled: false}).Error)
	require.True(t, HasConfiguredGroupModel("public", "model-a", nil), "disabled channel is a real outage")
	require.False(t, HasConfiguredGroupModel("public", "typo", nil))
	require.False(t, HasConfiguredGroupModel("public", "model-a", []int{}), "empty channel allowlist denies all")
	require.False(t, HasConfiguredGroupModel("public", "model-a", []int{2}))
	require.True(t, HasConfiguredGroupModel("public", "model-a", []int{1}))
}

func TestGroupHealthCacheMigrationKeepsLegacyRowsUnknown(t *testing.T) {
	previous := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	t.Cleanup(func() { DB = previous })
	require.NoError(t, db.AutoMigrate(&GroupHealthMetric{}))
	// Recreate the pre-cache schema, then exercise the same additive migration
	// used at startup. Existing health outcomes must not turn into cache samples.
	for _, field := range []string{"CacheReadTokens", "CacheInputTokens", "CacheSampleCount"} {
		require.NoError(t, db.Migrator().DropColumn(&GroupHealthMetric{}, field))
	}
	require.NoError(t, db.Table("group_health_metrics").Create(map[string]interface{}{
		"id": "legacy", "group_name": "g", "model_name": "m", "bucket_ts": 3600,
		"request_count": 10, "success_count": 9,
	}).Error)
	require.NoError(t, db.AutoMigrate(&GroupHealthMetric{}))
	rows, err := QueryGroupHealthMetrics(context.Background(), []string{"g"}, "m", 3600, 7200)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.EqualValues(t, 10, rows[0].RequestCount)
	require.Zero(t, rows[0].CacheSampleCount)
	require.Zero(t, rows[0].CacheInputTokens)
}
