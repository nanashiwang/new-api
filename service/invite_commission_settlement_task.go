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
	inviteCommissionSettlementBatchSize = 300
)

var (
	inviteCommissionSettlementOnce    sync.Once
	inviteCommissionSettlementRunning atomic.Bool
)

func StartInviteCommissionSettlementTask() {
	inviteCommissionSettlementOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		gopool.Go(func() {
			// 统一按“自然日 00:00”触发 T+1 结算：
			// - 不是“充值后 +24 小时”
			// - 而是进入次日后，在本地时区午夜执行上一自然日及以前的待结算台账。
			logger.LogInfo(context.Background(), "invite commission settlement task started: schedule=00:00 local time")
			for {
				wait := durationUntilNextLocalMidnight(time.Now())
				logger.LogDebug(context.Background(), "invite commission settlement next run in %s", wait)
				timer := time.NewTimer(wait)
				<-timer.C
				timer.Stop()
				runInviteCommissionSettlementOnce()
			}
		})
	})
}

func durationUntilNextLocalMidnight(now time.Time) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	return next.Sub(now)
}

func runInviteCommissionSettlementOnce() {
	// 防重入保护：避免慢查询或大积压导致多轮并发结算。
	if !inviteCommissionSettlementRunning.CompareAndSwap(false, true) {
		return
	}
	defer inviteCommissionSettlementRunning.Store(false)

	if !common.InviterCommissionEnabled || common.InviterRechargeCommissionRate <= 0 {
		return
	}

	// T+1 结算：每次只处理“昨天及以前”的 pending 台账，任务可重复执行且不会重复入账。
	targetDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	totalSettled := 0
	totalSkipped := 0
	for {
		// 分批处理，避免单次任务过大导致长事务和高延迟。
		settled, skipped, processed, err := model.SettleInviteCommissionByBizDate(targetDate, inviteCommissionSettlementBatchSize)
		if err != nil {
			logger.LogWarn(context.Background(), fmt.Sprintf("invite commission settlement failed: %v", err))
			return
		}
		totalSettled += settled
		totalSkipped += skipped
		// 当前批次不足上限，说明本轮没有更多待处理记录。
		if processed < inviteCommissionSettlementBatchSize {
			break
		}
	}

	if common.DebugEnabled && (totalSettled > 0 || totalSkipped > 0) {
		logger.LogDebug(context.Background(), "invite commission settlement: settled=%d skipped=%d", totalSettled, totalSkipped)
	}
}
