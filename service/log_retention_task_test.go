package service

import (
	"os"
	"strconv"
	"testing"
	"time"
)

func setEnvForTest(t *testing.T, key string, value string) {
	t.Helper()
	origin, had := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("set %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, origin)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

// 默认必须关闭：删除日志不可逆，保留期只能由运维显式选择，
// 不能因为升级到新版本就开始删数据。
func TestLogRetentionDefaultsToDisabled(t *testing.T) {
	_ = os.Unsetenv("LOG_RETENTION_DAYS")
	if days := logRetentionDays(); days > 0 {
		t.Fatalf("retention enabled by default: %d days", days)
	}
}

func TestLogRetentionReadsConfiguredDays(t *testing.T) {
	setEnvForTest(t, "LOG_RETENTION_DAYS", "90")
	if days := logRetentionDays(); days != 90 {
		t.Fatalf("days = %d, want 90", days)
	}
}

// 非法或零值必须回落到安全默认值，不能变成 0 行批次导致删除循环空转。
func TestLogRetentionBatchSizeFallsBackOnInvalidValues(t *testing.T) {
	for _, value := range []string{"0", "-1", "not-a-number"} {
		setEnvForTest(t, "LOG_RETENTION_BATCH_SIZE", value)
		if size := LogRetentionBatchSize(); size <= 0 {
			t.Fatalf("batch size %d for env %q", size, value)
		}
	}
}

// 手动清理接口的批大小必须显著大于此前硬编码的 100，
// 否则删除大量历史日志会退化成海量小事务往返。
func TestLogRetentionBatchSizeDefaultIsNotTiny(t *testing.T) {
	_ = os.Unsetenv("LOG_RETENTION_BATCH_SIZE")
	if size := LogRetentionBatchSize(); size < 1000 {
		t.Fatalf("default batch size %d is too small", size)
	}
}

func TestLogRetentionIntervalFallsBackOnInvalidValues(t *testing.T) {
	for _, value := range []string{"0", "-3"} {
		setEnvForTest(t, "LOG_RETENTION_INTERVAL_HOURS", value)
		if interval := logRetentionInterval(); interval <= 0 {
			t.Fatalf("interval %s for env %q", interval, value)
		}
	}
	setEnvForTest(t, "LOG_RETENTION_INTERVAL_HOURS", "6")
	if interval := logRetentionInterval(); interval != 6*time.Hour {
		t.Fatalf("interval = %s, want 6h", interval)
	}
}

// 保留期关闭时即使被直接调用也不能删除任何数据。
func TestRunLogRetentionOnceIsNoopWhenDisabled(t *testing.T) {
	setEnvForTest(t, "LOG_RETENTION_DAYS", "0")
	// model.DeleteOldLog 会在 LOG_DB 为空时 panic，因此这里能正常返回
	// 本身就证明了它没有去访问数据库。
	runLogRetentionOnce()
}

func TestLogRetentionDaysRejectsNegative(t *testing.T) {
	setEnvForTest(t, "LOG_RETENTION_DAYS", strconv.Itoa(-30))
	if days := logRetentionDays(); days > 0 {
		t.Fatalf("negative retention accepted: %d", days)
	}
}
