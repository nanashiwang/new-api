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

var (
	logRetentionOnce    sync.Once
	logRetentionRunning atomic.Bool
)

// logRetentionDays 返回日志保留天数，非正值表示关闭自动清理。
//
// 默认关闭：删除日志不可逆，且不同部署对审计留存期的要求不同，必须由运维显式
// 选择保留窗口，不能由升级动作替他们决定。
func logRetentionDays() int {
	return common.GetEnvOrDefault("LOG_RETENTION_DAYS", 0)
}

// logRetentionInterval 返回两次清理之间的间隔。
func logRetentionInterval() time.Duration {
	hours := common.GetEnvOrDefault("LOG_RETENTION_INTERVAL_HOURS", 24)
	if hours <= 0 {
		hours = 24
	}
	return time.Duration(hours) * time.Hour
}

// LogRetentionBatchSize 返回单次删除的行数。
//
// 批量删除持有的行锁与 binlog 体积都随批大小增长，过大会影响正在写入日志的
// 请求；过小则会把一次清理拆成大量往返。数千行是两者之间的折中。
func LogRetentionBatchSize() int {
	size := common.GetEnvOrDefault("LOG_RETENTION_BATCH_SIZE", 2000)
	if size <= 0 {
		size = 2000
	}
	return size
}

// StartLogRetentionTask 启动日志保留期清理任务。
//
// 日志表只增不减是统计查询逐渐变慢的根因之一：表和索引最终会远大于缓冲池，
// 任何按时间范围聚合的查询都要走磁盘。设定保留期可以让表大小稳定下来。
func StartLogRetentionTask() {
	logRetentionOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		days := logRetentionDays()
		if days <= 0 {
			return
		}

		interval := logRetentionInterval()
		common.SysLog(fmt.Sprintf("log retention task started: keep %d days, interval %s", days, interval))

		gopool.Go(func() {
			for {
				// 启动后先等一个间隔，避开进程启动时的迁移与预热。
				timer := time.NewTimer(interval)
				<-timer.C
				timer.Stop()
				runLogRetentionOnce()
			}
		})
	})
}

func runLogRetentionOnce() {
	days := logRetentionDays()
	if days <= 0 {
		return
	}
	// 上一轮仍在进行时跳过，避免两轮删除互相争锁。
	if !logRetentionRunning.CompareAndSwap(false, true) {
		return
	}
	defer logRetentionRunning.Store(false)

	ctx := context.Background()
	targetTimestamp := time.Now().AddDate(0, 0, -days).Unix()
	deleted, err := model.DeleteOldLog(ctx, targetTimestamp, LogRetentionBatchSize())
	if err != nil {
		logger.LogError(ctx, "log retention failed: "+err.Error())
		return
	}
	if deleted > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("log retention removed %d rows older than %d days", deleted, days))
	}
}
