package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/bytedance/gopkg/util/gopool"
)

const (
	channelPeriodQuotaResetTickInterval = 1 * time.Minute
	channelPeriodQuotaResetBatchSize    = 300
	channelPeriodQuotaCleanupInterval   = 30 * time.Minute
)

var (
	channelPeriodQuotaResetOnce    sync.Once
	channelPeriodQuotaResetRunning atomic.Bool
	channelPeriodQuotaCleanupLast  atomic.Int64
)

func StartChannelPeriodQuotaResetTask() {
	channelPeriodQuotaResetOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("channel period quota reset task started: tick=%s", channelPeriodQuotaResetTickInterval))
			ticker := time.NewTicker(channelPeriodQuotaResetTickInterval)
			defer ticker.Stop()
			runChannelPeriodQuotaResetOnce()
			for range ticker.C {
				runChannelPeriodQuotaResetOnce()
			}
		})
	})
}

func runChannelPeriodQuotaResetOnce() {
	if !channelPeriodQuotaResetRunning.CompareAndSwap(false, true) {
		return
	}
	defer channelPeriodQuotaResetRunning.Store(false)

	ctx := context.Background()
	total := 0
	for {
		usages, err := model.ListExpiredTriggeredUsages(common.GetTimestamp(), channelPeriodQuotaResetBatchSize)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("channel period quota reset task failed: %v", err))
			return
		}
		if len(usages) == 0 {
			break
		}
		for _, usage := range usages {
			if recoverPeriodQuotaUsage(usage) {
				total++
			}
			_ = model.ClearTriggeredFlag(usage.Id)
		}
		if len(usages) < channelPeriodQuotaResetBatchSize {
			break
		}
	}
	lastCleanup := time.Unix(channelPeriodQuotaCleanupLast.Load(), 0)
	if time.Since(lastCleanup) >= channelPeriodQuotaCleanupInterval {
		channelPeriodQuotaCleanupLast.Store(time.Now().Unix())
	}
	if common.DebugEnabled && total > 0 {
		logger.LogDebug(ctx, "channel period quota maintenance: recovered_count=%d", total)
	}
}

func recoverPeriodQuotaUsage(usage model.ChannelQuotaUsage) bool {
	if usage.Scope == periodQuotaScopeChannel {
		id := 0
		_, _ = fmt.Sscanf(usage.ScopeKey, "%d", &id)
		if id <= 0 {
			return false
		}
		ch, err := model.GetChannelById(id, true)
		if err != nil || !shouldRecoverPeriodQuotaChannel(ch, usage.PeriodEnd) {
			return false
		}
		ch.Status = common.ChannelStatusEnabled
		model.ClearPeriodQuotaMeta(ch)
		if err := ch.SaveWithoutKey(); err != nil {
			return false
		}
		return model.UpdateAbilityStatus(ch.Id, true) == nil
	}
	if usage.Scope == periodQuotaScopeTag {
		channels, err := model.GetChannelsByTag(usage.ScopeKey, false, true)
		if err != nil {
			return false
		}
		ok := false
		for _, ch := range channels {
			if !shouldRecoverPeriodQuotaChannel(ch, usage.PeriodEnd) {
				continue
			}
			ch.Status = common.ChannelStatusEnabled
			model.ClearPeriodQuotaMeta(ch)
			if ch.SaveWithoutKey() == nil {
				ok = true
			}
		}
		if ok {
			_ = model.UpdateAbilityStatusByTag(usage.ScopeKey, true)
		}
		return ok
	}
	return false
}

func shouldRecoverPeriodQuotaChannel(ch *model.Channel, periodEnd int64) bool {
	return ch != nil && ch.Status == common.ChannelStatusAutoDisabled && model.HasPeriodQuotaMeta(ch) && common.GetTimestamp() >= model.GetPeriodQuotaUntil(ch) && model.GetPeriodQuotaUntil(ch) == periodEnd
}
