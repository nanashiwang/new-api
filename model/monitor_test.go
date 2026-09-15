package model

import (
	"os"
	"strconv"
	"testing"
	"time"
)

func setMonitorCacheTTL(t *testing.T, seconds int) {
	t.Helper()
	origin, had := os.LookupEnv("CHANNEL_MONITOR_CACHE_TTL")
	if err := os.Setenv("CHANNEL_MONITOR_CACHE_TTL", strconv.Itoa(seconds)); err != nil {
		t.Fatalf("set ttl: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("CHANNEL_MONITOR_CACHE_TTL", origin)
		} else {
			_ = os.Unsetenv("CHANNEL_MONITOR_CACHE_TTL")
		}
	})
}

// 前端每次刷新都把 end_time 取为当前秒。若窗口不对齐，缓存键每秒都不同，
// 缓存永远不会命中——对齐是这个缓存有没有意义的前提。
func TestNormalizeMonitorRange_AlignsWindowSoKeysRepeat(t *testing.T) {
	setMonitorCacheTTL(t, 60)

	now := time.Now().Unix()
	firstStart, firstEnd := normalizeMonitorRange(now-86400, now)
	secondStart, secondEnd := normalizeMonitorRange(now-1-86400, now-1)

	if firstEnd%60 != 0 || firstStart%60 != 0 {
		t.Fatalf("window not aligned to ttl: start=%d end=%d", firstStart, firstEnd)
	}
	// 相邻两秒的请求落在同一个桶里时必须产生同一个键。
	if firstEnd == secondEnd && firstStart != secondStart {
		t.Fatalf("aligned ends but different starts: %d vs %d", firstStart, secondStart)
	}
	if channelMonitorStatsCacheKey(firstStart, firstEnd, "channel", "") == "" {
		t.Fatal("empty cache key")
	}
}

func TestNormalizeMonitorRange_ClampsRangeAndFuture(t *testing.T) {
	setMonitorCacheTTL(t, 60)
	now := time.Now().Unix()

	// 未来时间被收敛到当前时刻。
	_, end := normalizeMonitorRange(now-3600, now+86400)
	if end > now {
		t.Fatalf("end=%d is in the future (now=%d)", end, now)
	}

	// 超长范围被截断到上限。
	start, end := normalizeMonitorRange(now-400*86400, now)
	if end-start > monitorMaxTimeRange {
		t.Fatalf("range %d exceeds max %d", end-start, monitorMaxTimeRange)
	}

	// 起止颠倒时被交换，窗口始终非空。
	start, end = normalizeMonitorRange(now, now-3600)
	if start >= end {
		t.Fatalf("empty window: start=%d end=%d", start, end)
	}
}

// 归一化后窗口不能塌缩成零长度，否则统计查询会返回空结果。
func TestNormalizeMonitorRange_KeepsWindowNonEmptyAfterAlignment(t *testing.T) {
	setMonitorCacheTTL(t, 3600)
	now := time.Now().Unix()

	start, end := normalizeMonitorRange(now-10, now)
	if start >= end {
		t.Fatalf("window collapsed: start=%d end=%d", start, end)
	}
}

func TestChannelMonitorStatsCacheKey_SeparatesDimensions(t *testing.T) {
	base := channelMonitorStatsCacheKey(100, 200, "channel", "")
	cases := map[string]string{
		"分组维度": channelMonitorStatsCacheKey(100, 200, "group", ""),
		"用户名":  channelMonitorStatsCacheKey(100, 200, "channel", "alice"),
		"起始时间": channelMonitorStatsCacheKey(101, 200, "channel", ""),
		"结束时间": channelMonitorStatsCacheKey(100, 201, "channel", ""),
	}
	for name, key := range cases {
		if key == base {
			t.Fatalf("%s 未体现在缓存键中", name)
		}
	}
}
