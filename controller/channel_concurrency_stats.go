package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// channelConcurrencyChannelStat 单个渠道的在途并发快照。
type channelConcurrencyChannelStat struct {
	ChannelId      int    `json:"channel_id"`
	ChannelName    string `json:"channel_name"`
	Inflight       int64  `json:"inflight"`
	MaxConcurrency int    `json:"max_concurrency"`
}

// GetChannelConcurrencyStats 只读返回渠道并发控制的运行时统计：
// 全局配置、过载指标（等待/超时/取消/队列拒绝/平均等待时长/当前等待数）、
// 以及当前有在途并发的各渠道明细。供管理端监控过载情况、判断是否需要扩容。
func GetChannelConcurrencyStats(c *gin.Context) {
	setting := operation_setting.GetChannelConcurrencySetting()
	metrics := middleware.GetChannelConcurrencyMetrics()

	channelStats := make([]channelConcurrencyChannelStat, 0)
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err == nil {
		for _, ch := range channels {
			if ch == nil {
				continue
			}
			inflight, qErr := middleware.QueryChannelConcurrency(ch.Id)
			if qErr != nil || inflight <= 0 {
				// 只展示当前有在途并发的渠道，避免渠道很多时刷屏。
				continue
			}
			channelStats = append(channelStats, channelConcurrencyChannelStat{
				ChannelId:      ch.Id,
				ChannelName:    ch.Name,
				Inflight:       inflight,
				MaxConcurrency: middleware.ResolveChannelMaxConcurrency(ch),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":  setting.Enabled,
			"config":   setting,
			"metrics":  metrics,
			"channels": channelStats,
		},
	})
}
