package middleware

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/go-redis/redis/v8"
)

// channel_concurrency.go 实现渠道级在途并发计数与过载指标，是渠道并发控制（含 Layer C
// 有界等待）的底层原语层。设计与 token_runtime_limit.go 的令牌并发计数完全同构：
//   - Redis 优先（跨多实例准确，INCR/DECR + Expire 兜底防泄漏）
//   - Redis 未启用时回退进程内内存（sync.Map + atomic，单实例场景）
//
// 编排逻辑（选渠道 → acquire → 满则换渠道 → 全满进入有界等待）在 controller 层，
// 这里只暴露被其调用的原语，避免 middleware → controller 的反向依赖。

const (
	// Redis key 前缀，独立命名空间，避免与 token:concurrency:* 撞车。
	channelConcurrencyKeyPrefix = "channel:concurrency:"
	// Redis 计数 key 的 TTL 兜底：远大于最长可能的单次上游请求（含超长流式生成），
	// 防止 release 因进程崩溃丢失导致计数永久占用；正常路径 release 会主动 DECR。
	channelConcurrencyKeyTTL = 2 * time.Hour
)

// channelConcurrencyIncrScript 原子执行 INCR + PEXPIRE，保证“有计数必有 TTL”，
// 消除 INCR 成功而 Expire 单独失败导致 key 无 TTL、进而永久占用名额的泄漏路径。
var channelConcurrencyIncrScript = redis.NewScript(`
local v = redis.call('INCR', KEYS[1])
redis.call('PEXPIRE', KEYS[1], ARGV[1])
return v
`)

// ---- 内存态计数（Redis 未启用时的回退） ----

var channelConcurrencyCounters sync.Map // key(string) -> *channelConcurrencyCounter

type channelConcurrencyCounter struct {
	value int64
	// idleRounds 仅由 cleanup goroutine 单线程访问：连续为 0 的清理轮数。
	idleRounds int
}

func getChannelConcurrencyCounter(key string) *channelConcurrencyCounter {
	if counter, ok := channelConcurrencyCounters.Load(key); ok {
		return counter.(*channelConcurrencyCounter)
	}
	counter := &channelConcurrencyCounter{}
	actual, _ := channelConcurrencyCounters.LoadOrStore(key, counter)
	return actual.(*channelConcurrencyCounter)
}

func init() {
	go cleanupIdleChannelConcurrencyCounters()
}

// 定期清理归零的内存计数器 key，防止 sync.Map 无界增长。
func cleanupIdleChannelConcurrencyCounters() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		channelConcurrencyCounters.Range(func(key, value any) bool {
			counter := value.(*channelConcurrencyCounter)
			if atomic.LoadInt64(&counter.value) == 0 {
				// 连续两轮（≥5 分钟）都为 0 才删除，规避与“已读到 counter 指针、尚未自增”的
				// acquire 之间的 TOCTOU 竞态：一个进行中的 acquire 不可能让计数跨越两个 tick
				// 仍保持 0，因此“连续两轮为 0”可安全判定该 key 确实空闲。
				counter.idleRounds++
				if counter.idleRounds >= 2 {
					channelConcurrencyCounters.Delete(key)
				}
			} else {
				counter.idleRounds = 0
			}
			return true
		})
	}
}

func channelConcurrencyKey(channelID int) string {
	return fmt.Sprintf("%s%d", channelConcurrencyKeyPrefix, channelID)
}

// ---- 过载指标（进程内 atomic，重启清零；仅用于观测，不作硬依据） ----

var channelConcurrencyMetrics struct {
	acquireTotal      atomic.Int64 // 成功占用名额的总次数
	waitEnterTotal    atomic.Int64 // 进入 Layer C 有界等待的次数（所有渠道满）
	waitAcquiredTotal atomic.Int64 // 等待后成功拿到容量的次数
	waitTimeoutTotal  atomic.Int64 // 等待超时返回 503 的次数
	waitCancelTotal   atomic.Int64 // 等待中客户端断开取消的次数
	queueRejectTotal  atomic.Int64 // 等待队列已满、快速失败 503 的次数
	waitTotalMs       atomic.Int64 // 等待累计毫秒（用于算平均等待时长）
	waitDurationCount atomic.Int64 // 参与等待时长统计的样本数
	currentWaiting    atomic.Int64 // 当前处于 Layer C 等待中的请求数（gauge，即队列占用）
}

// ChannelConcurrencyMetricsSnapshot 是指标的只读快照，供 admin stats 端点返回。
type ChannelConcurrencyMetricsSnapshot struct {
	AcquireTotal      int64 `json:"acquire_total"`
	WaitEnterTotal    int64 `json:"wait_enter_total"`
	WaitAcquiredTotal int64 `json:"wait_acquired_total"`
	WaitTimeoutTotal  int64 `json:"wait_timeout_total"`
	WaitCancelTotal   int64 `json:"wait_cancel_total"`
	QueueRejectTotal  int64 `json:"queue_reject_total"`
	CurrentWaiting    int64 `json:"current_waiting"`
	AvgWaitMs         int64 `json:"avg_wait_ms"`
}

// GetChannelConcurrencyMetrics 返回当前过载指标快照。
func GetChannelConcurrencyMetrics() ChannelConcurrencyMetricsSnapshot {
	count := channelConcurrencyMetrics.waitDurationCount.Load()
	avg := int64(0)
	if count > 0 {
		avg = channelConcurrencyMetrics.waitTotalMs.Load() / count
	}
	return ChannelConcurrencyMetricsSnapshot{
		AcquireTotal:      channelConcurrencyMetrics.acquireTotal.Load(),
		WaitEnterTotal:    channelConcurrencyMetrics.waitEnterTotal.Load(),
		WaitAcquiredTotal: channelConcurrencyMetrics.waitAcquiredTotal.Load(),
		WaitTimeoutTotal:  channelConcurrencyMetrics.waitTimeoutTotal.Load(),
		WaitCancelTotal:   channelConcurrencyMetrics.waitCancelTotal.Load(),
		QueueRejectTotal:  channelConcurrencyMetrics.queueRejectTotal.Load(),
		CurrentWaiting:    channelConcurrencyMetrics.currentWaiting.Load(),
		AvgWaitMs:         avg,
	}
}

// 指标记录器（供 controller 编排层调用）。
func recordChannelAcquire()      { channelConcurrencyMetrics.acquireTotal.Add(1) }
func recordChannelWaitEnter()    { channelConcurrencyMetrics.waitEnterTotal.Add(1) }
func recordChannelWaitAcquired() { channelConcurrencyMetrics.waitAcquiredTotal.Add(1) }
func recordChannelWaitTimeout()  { channelConcurrencyMetrics.waitTimeoutTotal.Add(1) }
func recordChannelWaitCancel()   { channelConcurrencyMetrics.waitCancelTotal.Add(1) }
func recordChannelQueueReject()  { channelConcurrencyMetrics.queueRejectTotal.Add(1) }
func recordChannelWaitDuration(d time.Duration) {
	channelConcurrencyMetrics.waitTotalMs.Add(d.Milliseconds())
	channelConcurrencyMetrics.waitDurationCount.Add(1)
}

// Exported wrappers（controller 包调用）。
func RecordChannelWaitEnter()                   { recordChannelWaitEnter() }
func RecordChannelWaitAcquired()                { recordChannelWaitAcquired() }
func RecordChannelWaitTimeout()                 { recordChannelWaitTimeout() }
func RecordChannelWaitCancel()                  { recordChannelWaitCancel() }
func RecordChannelQueueReject()                 { recordChannelQueueReject() }
func RecordChannelWaitDuration(d time.Duration) { recordChannelWaitDuration(d) }

// ---- 有界等待队列准入（防止过载时等待请求无界堆积） ----

// EnterChannelWaitQueue 尝试进入 Layer C 等待队列。返回 false 表示队列已满，调用方应快速失败（503）。
// 用原子自增 + 越界回退实现无锁准入闸门。maxLen<=0 视为不限。
func EnterChannelWaitQueue(maxLen int) bool {
	n := channelConcurrencyMetrics.currentWaiting.Add(1)
	if maxLen > 0 && n > int64(maxLen) {
		channelConcurrencyMetrics.currentWaiting.Add(-1)
		return false
	}
	return true
}

// LeaveChannelWaitQueue 离开等待队列，必须与 EnterChannelWaitQueue 成功调用配对。
func LeaveChannelWaitQueue() {
	channelConcurrencyMetrics.currentWaiting.Add(-1)
}

// ---- 渠道并发信号量 ----

// ChannelConcurrencyEnabled 全局开关，关闭时所有并发控制逻辑短路。
func ChannelConcurrencyEnabled() bool {
	return operation_setting.GetChannelConcurrencySetting().Enabled
}

// ResolveChannelMaxConcurrency 计算某渠道的在途并发上限：渠道级 max_concurrency>0 时覆盖全局默认。
func ResolveChannelMaxConcurrency(channel *model.Channel) int {
	setting := operation_setting.GetChannelConcurrencySetting()
	if channel != nil {
		if override := channel.GetSetting().MaxConcurrency; override > 0 {
			return override
		}
	}
	return setting.NormalizedDefaultMaxConcurrency()
}

// TryAcquireChannelSlot 尝试占用渠道一个在途并发名额（不等待）。
//   - 成功：返回 (release, true)，release 必须在本次上游调用结束后调用一次且仅一次。
//   - 已满：返回 (nil, false)。
//   - maxConcurrency<=0：视为不限并发，返回一个 no-op release。
//
// release 用 sync.Once 保护，防止重复 DECR 把计数打成负数。
func TryAcquireChannelSlot(channelID int, maxConcurrency int) (func(), bool) {
	if maxConcurrency <= 0 {
		return func() {}, true
	}
	key := channelConcurrencyKey(channelID)

	if common.RedisEnabled {
		ctx := context.Background()
		// 原子 INCR+PEXPIRE：保证有计数必有 TTL（消除 Expire 单独失败的无 TTL 泄漏）。
		value, err := channelConcurrencyIncrScript.Run(ctx, common.RDB, []string{key}, channelConcurrencyKeyTTL.Milliseconds()).Int64()
		if err != nil {
			// Redis 异常时不阻断请求：放行并给一个 no-op release。
			// 权衡：Redis 故障期间该渠道暂不受并发限制（宁可短暂不限流也不误杀正常流量）。
			// 若脚本实际已在 Redis 端 +1 但响应超时，该名额由 channelConcurrencyKeyTTL 兜底最终回收。
			return func() {}, true
		}
		if value > int64(maxConcurrency) {
			_, _ = common.RDB.Decr(ctx, key).Result()
			return nil, false
		}
		return redisReleaseOnce(key), true
	}

	counter := getChannelConcurrencyCounter(key)
	next := atomic.AddInt64(&counter.value, 1)
	if next > int64(maxConcurrency) {
		atomic.AddInt64(&counter.value, -1)
		return nil, false
	}
	return memoryReleaseOnce(counter), true
}

func redisReleaseOnce(key string) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			// 用 Background ctx：release 可能在客户端断开（请求 ctx 已 cancel）后执行，
			// 不能因请求 ctx 失效而漏放名额。DecrBy 的下限由 max(0) 语义近似保证——
			// 这里 DECR 后若为负说明有异常，但 TTL 兜底会最终清理。
			_, _ = common.RDB.Decr(context.Background(), key).Result()
		})
	}
}

func memoryReleaseOnce(counter *channelConcurrencyCounter) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			atomic.AddInt64(&counter.value, -1)
		})
	}
}

// AcquireChannelSlotWithMetric 在 TryAcquireChannelSlot 基础上记录成功指标，供 controller 编排层用。
func AcquireChannelSlotWithMetric(channelID int, maxConcurrency int) (func(), bool) {
	release, ok := TryAcquireChannelSlot(channelID, maxConcurrency)
	if ok {
		recordChannelAcquire()
	}
	return release, ok
}

// QueryChannelConcurrency 只读返回某渠道当前在途并发数（供监控/仪表盘展示）。
func QueryChannelConcurrency(channelID int) (int64, error) {
	key := channelConcurrencyKey(channelID)
	if common.RedisEnabled {
		ctx := context.Background()
		val, err := common.RDB.Get(ctx, key).Int64()
		if err != nil {
			// redis: nil 表示 key 不存在，即并发为 0（与 QueryTokenConcurrency 判定一致）。
			if err.Error() == "redis: nil" {
				return 0, nil
			}
			return 0, err
		}
		return val, nil
	}
	if counter, ok := channelConcurrencyCounters.Load(key); ok {
		return atomic.LoadInt64(&counter.(*channelConcurrencyCounter).value), nil
	}
	return 0, nil
}

// QueryChannelConcurrencies 批量返回渠道当前在途并发数，供容量感知选渠计算压力。
// Redis 模式使用一次 MGET，避免候选渠道逐个 GET；内存模式直接读取各原子计数器。
func QueryChannelConcurrencies(channelIDs []int) (map[int]int64, error) {
	result := make(map[int]int64, len(channelIDs))
	if len(channelIDs) == 0 {
		return result, nil
	}

	uniqueIDs := make([]int, 0, len(channelIDs))
	seen := make(map[int]struct{}, len(channelIDs))
	for _, channelID := range channelIDs {
		if _, exists := seen[channelID]; exists {
			continue
		}
		seen[channelID] = struct{}{}
		uniqueIDs = append(uniqueIDs, channelID)
		result[channelID] = 0
	}

	if common.RedisEnabled {
		keys := make([]string, len(uniqueIDs))
		for i, channelID := range uniqueIDs {
			keys[i] = channelConcurrencyKey(channelID)
		}
		values, err := common.RDB.MGet(context.Background(), keys...).Result()
		if err != nil {
			return nil, err
		}
		for i, value := range values {
			if value == nil {
				continue
			}
			parsed, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
			if err != nil {
				return nil, err
			}
			result[uniqueIDs[i]] = parsed
		}
		return result, nil
	}

	for _, channelID := range uniqueIDs {
		key := channelConcurrencyKey(channelID)
		if counter, ok := channelConcurrencyCounters.Load(key); ok {
			result[channelID] = atomic.LoadInt64(&counter.(*channelConcurrencyCounter).value)
		}
	}
	return result, nil
}
