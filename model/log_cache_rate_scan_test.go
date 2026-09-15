package model

import (
	"os"
	"strconv"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCacheRateScanDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	originLogDB := LOG_DB
	LOG_DB = db
	t.Cleanup(func() { LOG_DB = originLogDB })
	if err := db.AutoMigrate(&Log{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
}

func setCacheRateScanLimit(t *testing.T, limit int) {
	t.Helper()
	origin, had := os.LookupEnv("LOG_CACHE_RATE_SCAN_LIMIT")
	if err := os.Setenv("LOG_CACHE_RATE_SCAN_LIMIT", strconv.Itoa(limit)); err != nil {
		t.Fatalf("set scan limit: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("LOG_CACHE_RATE_SCAN_LIMIT", origin)
		} else {
			_ = os.Unsetenv("LOG_CACHE_RATE_SCAN_LIMIT")
		}
	})
}

// 流式扫描必须与旧的“全部读入内存后计算”得到相同的比值。
func TestSumUsedQuota_StreamedCacheRatesMatchBatchCalculation(t *testing.T) {
	setupCacheRateScanDB(t)
	setCacheRateScanLimit(t, 0)
	seedCacheRateLogFixtures(t)

	stat, err := SumUsedQuota(LogTypeConsume, 0, 0, "", "alice", "", 0, "", "", false)
	if err != nil {
		t.Fatalf("sum used quota: %v", err)
	}

	var rows []logCacheRateRow
	if err := LOG_DB.Table("logs").
		Select("prompt_tokens, other").
		Where("type = ?", LogTypeConsume).
		Scan(&rows).Error; err != nil {
		t.Fatalf("load rows: %v", err)
	}
	wantHit, wantGlobal := calculateCacheRates(rows)

	assertFloatEquals(t, stat.CacheHitRate, wantHit)
	assertFloatEquals(t, stat.CacheGlobalRate, wantGlobal)
}

// 上限存在的意义是让一次统计的代价与日志总量脱钩，因此必须真正生效，
// 且取样要落在最新的行上。
func TestSumUsedQuota_CacheRateScanLimitSamplesNewestRows(t *testing.T) {
	setupCacheRateScanDB(t)
	setCacheRateScanLimit(t, 1)
	seedCacheRateLogFixtures(t)

	stat, err := SumUsedQuota(LogTypeConsume, 0, 0, "", "alice", "", 0, "", "", false)
	if err != nil {
		t.Fatalf("sum used quota: %v", err)
	}

	// 最新一行 created_at=103：claude 语义，prompt=20、读=30、写=10，
	// 展示输入 = 20+30+10 = 60，全局缓存率 = 30/60。
	assertFloatEquals(t, stat.CacheGlobalRate, 0.5)
	assertFloatEquals(t, stat.CacheHitRate, 0.5)

	// 上限不应影响额度合计，它是独立的聚合查询。
	if stat.Quota != 290 {
		t.Fatalf("quota = %d, want 290", stat.Quota)
	}
}

// other 为 NULL 的历史行不能让整个统计失败。
func TestSumUsedQuota_ToleratesNullOtherColumn(t *testing.T) {
	setupCacheRateScanDB(t)
	setCacheRateScanLimit(t, 0)

	if err := LOG_DB.Exec(
		"INSERT INTO logs (user_id, username, type, model_name, quota, prompt_tokens, completion_tokens, created_at, other) VALUES (1, 'alice', ?, 'gpt-5.4', 10, 100, 5, 100, NULL)",
		LogTypeConsume,
	).Error; err != nil {
		t.Fatalf("seed null row: %v", err)
	}

	stat, err := SumUsedQuota(LogTypeConsume, 0, 0, "", "alice", "", 0, "", "", false)
	if err != nil {
		t.Fatalf("sum used quota: %v", err)
	}
	assertFloatEquals(t, stat.CacheGlobalRate, 0)
	assertFloatEquals(t, stat.CacheHitRate, 0)
}
