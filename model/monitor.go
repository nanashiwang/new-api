package model

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// ChannelMonitorStats 渠道监控统计数据
type ChannelMonitorStats struct {
	ChannelID       int     `json:"channel_id" gorm:"column:channel_id"`
	ChannelName     string  `json:"channel_name" gorm:"column:channel_name"`
	ChannelGroup    string  `json:"channel_group" gorm:"column:channel_group"`
	ChannelType     int     `json:"channel_type" gorm:"column:channel_type"`
	TotalRequests   int64   `json:"total_requests" gorm:"column:total_requests"`
	SuccessRequests int64   `json:"success_requests" gorm:"column:success_requests"`
	FailedRequests  int64   `json:"failed_requests" gorm:"column:failed_requests"`
	SuccessRate     float64 `json:"success_rate" gorm:"column:success_rate"`
	TotalTokens     int64   `json:"total_tokens" gorm:"column:total_tokens"`
	TotalQuota      int64   `json:"total_quota" gorm:"column:total_quota"`
	AvgResponseTime float64 `json:"avg_response_time" gorm:"column:avg_response_time"`
	LastUsedAt      int64   `json:"last_used_at" gorm:"column:last_used_at"`
}

// monitorStatsRow 是统计查询的原始行，渠道名称等信息在查询后二次填充。
// 日志库可能与主库分离，因此不能在 SQL 中 JOIN channels 表。
type monitorStatsRow struct {
	ChannelID       int     `gorm:"column:channel_id"`
	LogGroup        string  `gorm:"column:log_group"`
	TotalRequests   int64   `gorm:"column:total_requests"`
	SuccessRequests int64   `gorm:"column:success_requests"`
	FailedRequests  int64   `gorm:"column:failed_requests"`
	TotalTokens     int64   `gorm:"column:total_tokens"`
	TotalQuota      int64   `gorm:"column:total_quota"`
	AvgUseTime      float64 `gorm:"column:avg_use_time"`
	LastUsedAt      int64   `gorm:"column:last_used_at"`
}

const monitorMaxTimeRange = int64(30 * 24 * 3600)

// GetChannelMonitorStats 获取渠道监控统计数据
// groupBy 为 "group" 时按日志分组聚合，否则按渠道聚合。
func GetChannelMonitorStats(startTime, endTime int64, groupBy, username string) ([]ChannelMonitorStats, error) {
	now := time.Now().Unix()
	if endTime <= 0 {
		endTime = now
	}
	if startTime <= 0 {
		startTime = endTime - 86400
	}
	if startTime > endTime {
		startTime, endTime = endTime, startTime
	}
	if endTime-startTime > monitorMaxTimeRange {
		startTime = endTime - monitorMaxTimeRange
	}

	groupCol := logGroupCol
	if groupCol == "" {
		groupCol = commonGroupCol
	}

	// 消费日志计为成功，错误日志计为失败；use_time 记录的是秒。
	aggregates := fmt.Sprintf(`
		COUNT(*) as total_requests,
		SUM(CASE WHEN type = %d THEN 1 ELSE 0 END) as success_requests,
		SUM(CASE WHEN type = %d THEN 1 ELSE 0 END) as failed_requests,
		COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens,
		COALESCE(SUM(quota), 0) as total_quota,
		COALESCE(AVG(use_time), 0) as avg_use_time,
		COALESCE(MAX(created_at), 0) as last_used_at`,
		LogTypeConsume, LogTypeError)

	var selectClause, groupClause string
	if groupBy == "group" {
		selectClause = fmt.Sprintf("%s as log_group,%s", groupCol, aggregates)
		groupClause = groupCol
	} else {
		selectClause = fmt.Sprintf("channel_id,%s", aggregates)
		groupClause = "channel_id"
	}

	query := LOG_DB.Table("logs").
		Select(selectClause).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Where("type IN (?, ?)", LogTypeConsume, LogTypeError).
		Group(groupClause).
		Order("total_requests DESC")

	if username != "" {
		query = query.Where("username = ?", username)
	}

	var rows []monitorStatsRow
	if err := query.Scan(&rows).Error; err != nil {
		common.SysError("获取渠道监控统计失败: " + err.Error())
		return nil, err
	}

	stats := make([]ChannelMonitorStats, 0, len(rows))
	for _, row := range rows {
		stat := ChannelMonitorStats{
			ChannelID:       row.ChannelID,
			ChannelGroup:    row.LogGroup,
			TotalRequests:   row.TotalRequests,
			SuccessRequests: row.SuccessRequests,
			FailedRequests:  row.FailedRequests,
			TotalTokens:     row.TotalTokens,
			TotalQuota:      row.TotalQuota,
			AvgResponseTime: row.AvgUseTime * 1000, // 秒转毫秒
			LastUsedAt:      row.LastUsedAt,
		}
		if row.TotalRequests > 0 {
			stat.SuccessRate = float64(row.SuccessRequests) * 100 / float64(row.TotalRequests)
		}
		stats = append(stats, stat)
	}

	if groupBy != "group" {
		fillChannelInfo(stats)
	}

	return stats, nil
}

// fillChannelInfo 用主库的 channels 表补全渠道名称、类型和分组。
func fillChannelInfo(stats []ChannelMonitorStats) {
	ids := make([]int, 0, len(stats))
	for _, stat := range stats {
		if stat.ChannelID > 0 {
			ids = append(ids, stat.ChannelID)
		}
	}
	if len(ids) == 0 {
		return
	}

	var channels []Channel
	err := DB.Select("id, name, type, " + commonGroupCol).
		Where("id IN ?", ids).
		Find(&channels).Error
	if err != nil {
		common.SysError("补全渠道监控信息失败: " + err.Error())
		return
	}

	channelMap := make(map[int]Channel, len(channels))
	for _, channel := range channels {
		channelMap[channel.Id] = channel
	}

	for i := range stats {
		if channel, ok := channelMap[stats[i].ChannelID]; ok {
			stats[i].ChannelName = channel.Name
			stats[i].ChannelType = channel.Type
			stats[i].ChannelGroup = channel.Group
		}
	}
}
