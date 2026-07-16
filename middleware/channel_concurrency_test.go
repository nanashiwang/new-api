package middleware

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forceMemoryMode 把并发计数强制切到内存模式（不依赖 Redis），并在测试结束还原。
func forceMemoryMode(t *testing.T) {
	t.Helper()
	orig := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = orig })
}

// TestTryAcquireChannelSlot_ConcurrentExactlyMax 是核心并发正确性测试：
// N+K 个 goroutine 同时争抢上限为 N 的渠道名额，必须恰好 N 个成功、不超发。
// 用 go test -race 检测数据竞争。
func TestTryAcquireChannelSlot_ConcurrentExactlyMax(t *testing.T) {
	forceMemoryMode(t)

	const maxConcurrency = 10
	const goroutines = 60
	const channelID = 990101

	var granted int64
	var wg sync.WaitGroup
	releases := make(chan func(), goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, ok := TryAcquireChannelSlot(channelID, maxConcurrency)
			if ok {
				atomic.AddInt64(&granted, 1)
				releases <- release
			}
		}()
	}
	wg.Wait()
	close(releases)

	assert.Equal(t, int64(maxConcurrency), granted, "恰好 maxConcurrency 个请求应获得名额，不超发")

	inflight, err := QueryChannelConcurrency(channelID)
	require.NoError(t, err)
	assert.Equal(t, int64(maxConcurrency), inflight, "在途并发应等于已授予名额数")

	// 全部释放后应归零。
	for release := range releases {
		release()
	}
	inflight, err = QueryChannelConcurrency(channelID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), inflight, "全部释放后在途并发应归零")
}

// TestChannelSlotReleaseIdempotent 验证 release 幂等：重复调用不会把计数打成负数。
func TestChannelSlotReleaseIdempotent(t *testing.T) {
	forceMemoryMode(t)
	const channelID = 990102

	release, ok := TryAcquireChannelSlot(channelID, 5)
	require.True(t, ok)

	release()
	release()
	release()

	inflight, err := QueryChannelConcurrency(channelID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), inflight, "重复 release 不应把计数打成负数")
}

// TestChannelSlotReacquireAfterRelease 验证释放名额后可被后续请求重新获取。
func TestChannelSlotReacquireAfterRelease(t *testing.T) {
	forceMemoryMode(t)
	const channelID = 990103
	const maxConcurrency = 2

	r1, ok1 := TryAcquireChannelSlot(channelID, maxConcurrency)
	require.True(t, ok1)
	r2, ok2 := TryAcquireChannelSlot(channelID, maxConcurrency)
	require.True(t, ok2)

	// 已满，第三个应失败。
	_, ok3 := TryAcquireChannelSlot(channelID, maxConcurrency)
	assert.False(t, ok3, "达到上限后应拒绝")

	// 释放一个，名额应可被重新获取。
	r1()
	r3, ok4 := TryAcquireChannelSlot(channelID, maxConcurrency)
	assert.True(t, ok4, "释放后名额应可重新获取")

	r2()
	r3()
	inflight, _ := QueryChannelConcurrency(channelID)
	assert.Equal(t, int64(0), inflight)
}

// TestTryAcquireChannelSlot_ZeroMaxUnlimited 验证 maxConcurrency<=0 视为不限并发。
func TestTryAcquireChannelSlot_ZeroMaxUnlimited(t *testing.T) {
	forceMemoryMode(t)
	const channelID = 990104

	for i := 0; i < 100; i++ {
		release, ok := TryAcquireChannelSlot(channelID, 0)
		require.True(t, ok, "maxConcurrency=0 应始终放行")
		release() // no-op，不应 panic
	}
}

// TestEnterChannelWaitQueue_Bounded 验证有界等待队列：N+K 个并发进入、上限 N，恰好 N 个成功。
func TestEnterChannelWaitQueue_Bounded(t *testing.T) {
	// 重置全局等待计数基线，避免其它测试残留干扰。
	channelConcurrencyMetrics.currentWaiting.Store(0)

	const maxLen = 5
	const goroutines = 40

	var granted int64
	var wg sync.WaitGroup
	entered := make(chan struct{}, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if EnterChannelWaitQueue(maxLen) {
				atomic.AddInt64(&granted, 1)
				entered <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(entered)

	assert.Equal(t, int64(maxLen), granted, "恰好 maxLen 个请求可进入等待队列，其余快速失败")

	// 配对释放，恢复基线。
	for range entered {
		LeaveChannelWaitQueue()
	}
	assert.Equal(t, int64(0), channelConcurrencyMetrics.currentWaiting.Load(), "全部离开后等待计数应归零")
}

// TestEnterChannelWaitQueue_Unlimited 验证 maxLen<=0 表示不限队列。
func TestEnterChannelWaitQueue_Unlimited(t *testing.T) {
	channelConcurrencyMetrics.currentWaiting.Store(0)
	for i := 0; i < 100; i++ {
		assert.True(t, EnterChannelWaitQueue(0), "maxLen=0 应不限队列长度")
	}
	for i := 0; i < 100; i++ {
		LeaveChannelWaitQueue()
	}
	assert.Equal(t, int64(0), channelConcurrencyMetrics.currentWaiting.Load())
}

// TestResolveChannelMaxConcurrency 验证渠道级 max_concurrency 覆盖全局默认。
func TestResolveChannelMaxConcurrency(t *testing.T) {
	setting := operation_setting.GetChannelConcurrencySetting()
	origDefault := setting.DefaultMaxConcurrency
	setting.DefaultMaxConcurrency = 100
	t.Cleanup(func() { setting.DefaultMaxConcurrency = origDefault })

	// 无渠道级覆盖 → 用全局默认。
	assert.Equal(t, 100, ResolveChannelMaxConcurrency(&model.Channel{}))

	// 渠道级 max_concurrency>0 → 覆盖全局。
	override := `{"max_concurrency": 30}`
	assert.Equal(t, 30, ResolveChannelMaxConcurrency(&model.Channel{Setting: &override}))

	// 渠道级为 0 → 回退全局默认。
	zero := `{"max_concurrency": 0}`
	assert.Equal(t, 100, ResolveChannelMaxConcurrency(&model.Channel{Setting: &zero}))

	// nil 渠道 → 全局默认。
	assert.Equal(t, 100, ResolveChannelMaxConcurrency(nil))
}

// TestChannelConcurrencyMetricsSnapshot 验证指标快照可正常计算平均等待时长。
func TestChannelConcurrencyMetricsSnapshot(t *testing.T) {
	recordChannelWaitDuration(150 * time.Millisecond)
	snap := GetChannelConcurrencyMetrics()
	// 记录等待时长后，平均值应可计算且非负（具体值受其它测试累加影响）。
	assert.GreaterOrEqual(t, snap.AvgWaitMs, int64(0))
}
