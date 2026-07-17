package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// channel_concurrency_relay.go 是渠道并发控制（含 Layer C 有界过载处理）的编排层。
//
// 它包装 getChannel：选中渠道后尝试占用该渠道一个在途并发名额；若该渠道已满则自动换到
// 其它未满的渠道（不消耗外层 RetryTimes，因为这不是转发失败）；当所有候选渠道都满时进入
// Layer C——有界等待容量释放：
//   - 客户端断开：立即取消等待（不返回错误响应，客户端已走）；
//   - 等待超时：返回 503 + Retry-After，指导客户端退避；
//   - 等待队列已满：快速失败 503，防止过载时等待请求无界堆积；
//   - 等到容量：清除因满而排除的渠道，重新选择并占用。
//
// 关键正确性保证：
//   - 全程在请求自身 goroutine 内 select 等待，不创建任何额外 goroutine（无 goroutine 泄漏）；
//   - 等待时长由 WaitTimeoutMs 上界，等待并发数由 MaxQueueLength 上界（双重有界）；
//   - 返回的 releaseSlot 由调用方在本轮上游调用结束后 defer 释放，覆盖所有退出路径。
//
// 未启用（Enabled=false）时完全短路，直接透传 getChannel，行为与未引入本功能一致。

// selectChannelWithConcurrency 选择一个可用且未超并发上限的渠道。
// 返回 (channel, releaseSlot, apiErr, isOverloadControl)：
//   - 成功：channel 非 nil，releaseSlot 非 nil（须在本轮上游调用后释放一次）；
//   - 客户端在等待中断开：apiErr 为可跳过重试的 client-canceled 错误，isOverloadControl=true；
//   - 等待超时/队列满：apiErr 为 503（已设置 Retry-After 响应头），isOverloadControl=true；
//   - 真无可用渠道（非并发原因）：透传 getChannel 的原始错误，isOverloadControl=false。
//
// isOverloadControl=true 表示这是 Layer C 过载控制的终态错误，调用方应直接采用它，
// 不要用历史重试错误（lastRelayError）覆盖——因为“系统整体过载”比“之前某次上游报错”
// 对用户更有指导性。未启用并发控制时 releaseSlot 为 nil、isOverloadControl=false。
func selectChannelWithConcurrency(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	retryParam *service.RetryParam,
) (*model.Channel, func(), *types.NewAPIError, bool) {
	if !middleware.ChannelConcurrencyEnabled() {
		channel, apiErr := getChannel(c, info, retryParam)
		return channel, nil, apiErr, false
	}

	setting := operation_setting.GetChannelConcurrencySetting()
	waitTimeout := time.Duration(setting.NormalizedWaitTimeoutMs()) * time.Millisecond
	pollInterval := time.Duration(setting.NormalizedPollIntervalMs()) * time.Millisecond
	maxQueueLen := setting.NormalizedMaxQueueLength()
	deadline := time.Now().Add(waitTimeout)

	// ---- 阶段1：亲和粘性优先（保上游 prompt cache 命中）----
	// 仅在首轮（未发生真实上游失败）尝试。重试轮意味着亲和渠道可能真的故障了，
	// 不该再死粘它，直接进入阶段2 的正常负载均衡以保障请求成功。
	if setting.AffinityStickyEnabled && retryParam.GetRetry() == 0 {
		ch, release, outcome := tryStickToAffinityChannel(c, info, retryParam, setting, deadline)
		switch outcome {
		case affinityStickAcquired:
			return ch, release, nil, false
		case affinityStickClientGone:
			return nil, nil, buildClientCanceledError(), true
		case affinityStickSkip, affinityStickDegrade:
			// 无亲和绑定 / 亲和渠道不可用 / 等待其超时 → 落到阶段2 正常选择（保体验）。
		}
	}

	// ---- 阶段2：正常负载均衡 + 并发控制（有界过载兜底）----
	// 本次请求内因“并发已满”而被临时排除的渠道集合。等到容量后从 ExcludeChannels 中移除它们重试。
	fullChannels := make(map[int]bool)
	inWaitQueue := false
	var waitStart time.Time
	defer func() {
		if inWaitQueue {
			middleware.LeaveChannelWaitQueue()
			middleware.RecordChannelWaitDuration(time.Since(waitStart))
		}
	}()

	for {
		channel, chErr := getChannel(c, info, retryParam)

		needWait := false
		switch {
		case chErr != nil:
			if len(fullChannels) == 0 {
				// 与并发无关的“无可用渠道”（如模型确实无渠道）：透传原始错误。
				return nil, nil, chErr, false
			}
			// 有渠道，但都被并发占满排除了 → 进入等待。
			needWait = true
		case fullChannels[channel.Id]:
			// getChannel 又返回了已知满的渠道（无其它候选，如指定渠道或单渠道场景）→ 等待其释放。
			needWait = true
		default:
			maxConcurrency := middleware.ResolveChannelMaxConcurrency(channel)
			release, acquired := middleware.AcquireChannelSlotWithMetric(channel.Id, maxConcurrency)
			if acquired {
				// 成功换到未满渠道：把本请求因“临时并发满”而排除的渠道恢复为可选，
				// 避免临时排除退化成永久排除（否则它们在本请求后续（含外层失败重试）中一直被跳过，
				// 即便早已空闲，会持续缩小可用渠道集）。removeChannelsFromExclusion 按集合精确移除，
				// 不影响外层因真实失败而排除的渠道。
				if len(fullChannels) > 0 {
					retryParam.ExcludeChannels = removeChannelsFromExclusion(retryParam.ExcludeChannels, fullChannels)
				}
				if inWaitQueue {
					middleware.RecordChannelWaitAcquired()
				}
				return channel, release, nil, false
			}
			// 该渠道已满：标记并排除，换下一个候选（不消耗外层 RetryTimes）。
			fullChannels[channel.Id] = true
			retryParam.ExcludeChannels = appendUniqueInt(retryParam.ExcludeChannels, channel.Id)
			continue
		}

		if !needWait {
			continue
		}

		// ---- Layer C 有界等待 ----
		if !inWaitQueue {
			if !middleware.EnterChannelWaitQueue(maxQueueLen) {
				// 等待队列已满：快速失败，防止过载时等待请求无界堆积。
				middleware.RecordChannelQueueReject()
				return nil, nil, buildChannelOverload503(c, setting, "concurrency wait queue is full"), true
			}
			inWaitQueue = true
			waitStart = time.Now()
			middleware.RecordChannelWaitEnter()
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			middleware.RecordChannelWaitTimeout()
			return nil, nil, buildChannelOverload503(c, setting, "all channels reached concurrency limit"), true
		}
		wait := pollInterval
		if wait > remaining {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-c.Request.Context().Done():
			timer.Stop()
			middleware.RecordChannelWaitCancel()
			return nil, nil, buildClientCanceledError(), true
		case <-timer.C:
			// 一个轮询周期过去：清除因满而排除的渠道，重新参与选择（可能已有渠道释放容量）。
			retryParam.ExcludeChannels = removeChannelsFromExclusion(retryParam.ExcludeChannels, fullChannels)
			fullChannels = make(map[int]bool)
			continue
		}
	}
}

// buildChannelOverload503 构造过载 503 错误，并设置 Retry-After 响应头指导客户端退避重试。
func buildChannelOverload503(c *gin.Context, setting *operation_setting.ChannelConcurrencySetting, reason string) *types.NewAPIError {
	if c != nil {
		c.Header("Retry-After", strconv.Itoa(setting.NormalizedRetryAfterSeconds()))
	}
	return types.NewErrorWithStatusCode(
		errors.New(reason+", please retry later"),
		types.ErrorCodeGetChannelFailed,
		http.StatusServiceUnavailable,
		types.ErrOptionWithSkipRetry(),
	)
}

// buildClientCanceledError 构造客户端断开取消的错误：跳过重试、隐藏内部细节，不污染错误率指标。
func buildClientCanceledError() *types.NewAPIError {
	return types.NewError(
		context.Canceled,
		types.ErrorCodeDoRequestFailed,
		types.ErrOptionWithSkipRetry(),
		types.ErrOptionWithHideErrMsg("client canceled while waiting for channel capacity"),
	)
}

// affinityStickOutcome 表示"尽力粘住亲和渠道"的结果。
type affinityStickOutcome int

const (
	affinityStickSkip       affinityStickOutcome = iota // 无亲和绑定，跳过粘性（新会话走正常选择，自然建立绑定）
	affinityStickAcquired                               // 成功粘住亲和渠道并占用名额（缓存最优）
	affinityStickClientGone                             // 等待亲和渠道时客户端断开
	affinityStickDegrade                                // 亲和渠道不可用/等待超时/队列满 → 降级到正常负载均衡
)

// tryStickToAffinityChannel 尽力把请求粘在其亲和绑定渠道上，以命中上游 prompt cache。
//
// 缓存与体验的平衡点：
//   - 亲和渠道可用且未满 → 立即占用并返回（缓存最优，无额外延迟）；
//   - 亲和渠道并发满 → 在 AffinityWaitMs 内有界等待它释放，等到即用（保住缓存局部性）；
//   - 亲和渠道熔断/禁用/等待超时/等待队列已满 → 返回 degrade，交由阶段2 正常负载均衡兜底（保体验）；
//   - 客户端断开 → 立即取消（不空等）。
//
// 全程在请求自身 goroutine 内 select，等待时长由 AffinityWaitMs 上界、等待并发由队列上界，无额外 goroutine。
// 返回 acquired 时已通过 SetupContextForSelectedChannel 把渠道写入 context，调用方可直接转发。
func tryStickToAffinityChannel(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	retryParam *service.RetryParam,
	setting *operation_setting.ChannelConcurrencySetting,
	overallDeadline time.Time,
) (*model.Channel, func(), affinityStickOutcome) {
	// 复用 distributor 的完整亲和校验（可用性/熔断、token 与客户端权限、codex 兼容、
	// 分组内模型启用、auto 分组展开），避免自行实现校验遗漏把请求粘到无权/未启用的渠道。
	affinityCh, selectGroup, ok := middleware.ResolveAffinityChannelForRelay(
		c, info.OriginModelName, info.UsingGroup, retryParam.ExcludeChannels,
	)
	if !ok {
		// 无亲和绑定，或绑定渠道不可用/无权限 → 交由阶段2 正常负载均衡（新会话会自然建立绑定）。
		return nil, nil, affinityStickSkip
	}

	maxConcurrency := middleware.ResolveChannelMaxConcurrency(affinityCh)
	// 亲和等待预算：不超过 AffinityWaitMs，也不超过总 deadline（把剩余时间留给降级阶段的等待）。
	affinityDeadline := time.Now().Add(time.Duration(setting.NormalizedAffinityWaitMs()) * time.Millisecond)
	if affinityDeadline.After(overallDeadline) {
		affinityDeadline = overallDeadline
	}
	pollInterval := time.Duration(setting.NormalizedPollIntervalMs()) * time.Millisecond
	maxQueueLen := setting.NormalizedMaxQueueLength()

	inWaitQueue := false
	var waitStart time.Time
	defer func() {
		if inWaitQueue {
			middleware.LeaveChannelWaitQueue()
			middleware.RecordChannelWaitDuration(time.Since(waitStart))
		}
	}()

	for {
		release, acquired := middleware.AcquireChannelSlotWithMetric(affinityCh.Id, maxConcurrency)
		if acquired {
			// 阶段1 绕过了 getChannel，需自己把亲和渠道写入 context 供后续转发链路使用。
			if setupErr := middleware.SetupContextForSelectedChannel(c, affinityCh, info.OriginModelName); setupErr != nil {
				release()
				return nil, nil, affinityStickDegrade
			}
			// 与 distributor 路径对齐：标记亲和已用（供 SkipRetryOnFailure 规则与 admin 日志/统计生效），
			// 用 tryAffinityChannel 返回的生效分组，避免 auto 分组下的 group 维度错乱。
			service.MarkChannelAffinityUsed(c, selectGroup, affinityCh.Id)
			if inWaitQueue {
				middleware.RecordChannelWaitAcquired()
			}
			return affinityCh, release, affinityStickAcquired
		}

		// 亲和渠道并发满：在预算内有界等待它释放（而非立即换渠道，以保住缓存）。
		if !inWaitQueue {
			if !middleware.EnterChannelWaitQueue(maxQueueLen) {
				// 等待队列已满 → 降级换渠道，保障体验（不无界堆积）。
				middleware.RecordChannelQueueReject()
				return nil, nil, affinityStickDegrade
			}
			inWaitQueue = true
			waitStart = time.Now()
			middleware.RecordChannelWaitEnter()
		}

		remaining := time.Until(affinityDeadline)
		if remaining <= 0 {
			// 等待亲和渠道超时 → 降级换渠道（用有界延迟换缓存，超时则让位给可用性）。
			middleware.RecordChannelWaitTimeout()
			return nil, nil, affinityStickDegrade
		}
		wait := pollInterval
		if wait > remaining {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-c.Request.Context().Done():
			timer.Stop()
			middleware.RecordChannelWaitCancel()
			return nil, nil, affinityStickClientGone
		case <-timer.C:
			// 继续尝试占用亲和渠道（可能已释放容量）。
		}
	}
}

// removeChannelsFromExclusion 返回移除了 remove 集合中所有渠道 ID 的新排除列表。
func removeChannelsFromExclusion(exclude []int, remove map[int]bool) []int {
	if len(remove) == 0 || len(exclude) == 0 {
		return exclude
	}
	result := make([]int, 0, len(exclude))
	for _, id := range exclude {
		if !remove[id] {
			result = append(result, id)
		}
	}
	return result
}
