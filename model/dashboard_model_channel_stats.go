package model

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
)

const dashboardModelChannelUntaggedLabel = "未设置标签"

type DashboardModelSpendStat struct {
	ModelName    string  `json:"model_name" gorm:"column:model_name"`
	Quota        int64   `json:"quota" gorm:"column:quota"`
	RequestCount int64   `json:"request_count" gorm:"column:request_count"`
	Share        float64 `json:"share" gorm:"-"`
}

type DashboardChannelTagSpendStat struct {
	ModelName    string  `json:"model_name" gorm:"column:model_name"`
	Tag          string  `json:"tag" gorm:"column:tag"`
	Quota        int64   `json:"quota" gorm:"column:quota"`
	RequestCount int64   `json:"request_count" gorm:"column:request_count"`
	Share        float64 `json:"share" gorm:"-"`
}

type DashboardModelChannelStats struct {
	StartTimestamp   int64                          `json:"start_timestamp"`
	EndTimestamp     int64                          `json:"end_timestamp"`
	TotalQuota       int64                          `json:"total_quota"`
	TotalRequest     int64                          `json:"total_request"`
	Models           []DashboardModelSpendStat      `json:"models"`
	ChannelTagShares []DashboardChannelTagSpendStat `json:"channel_tags"`
}

type dashboardModelChannelUsageRow struct {
	ModelName    string `gorm:"column:model_name"`
	ChannelID    int    `gorm:"column:channel_id"`
	Quota        int64  `gorm:"column:quota"`
	RequestCount int64  `gorm:"column:request_count"`
}

type dashboardSpendTotalRow struct {
	Quota        int64 `gorm:"column:quota"`
	RequestCount int64 `gorm:"column:request_count"`
}

var (
	dashboardModelChannelStatsCache     *cachex.HybridCache[DashboardModelChannelStats]
	dashboardModelChannelStatsCacheOnce sync.Once
)

func dashboardModelChannelStatsCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("DATA_DASHBOARD_MODEL_CHANNEL_CACHE_TTL", 60)
	if ttlSeconds <= 0 {
		ttlSeconds = 60
	}
	return time.Duration(ttlSeconds) * time.Second
}

func dashboardModelChannelStatsCacheCapacity() int {
	capacity := common.GetEnvOrDefault("DATA_DASHBOARD_MODEL_CHANNEL_CACHE_CAP", 128)
	if capacity <= 0 {
		capacity = 128
	}
	return capacity
}

func getDashboardModelChannelStatsCache() *cachex.HybridCache[DashboardModelChannelStats] {
	dashboardModelChannelStatsCacheOnce.Do(func() {
		ttl := dashboardModelChannelStatsCacheTTL()
		dashboardModelChannelStatsCache = cachex.NewHybridCache[DashboardModelChannelStats](cachex.HybridCacheConfig[DashboardModelChannelStats]{
			Namespace: cachex.Namespace("dashboard_model_channel_stats:v1"),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[DashboardModelChannelStats]{},
			Memory: func() *hot.HotCache[string, DashboardModelChannelStats] {
				return hot.NewHotCache[string, DashboardModelChannelStats](hot.LRU, dashboardModelChannelStatsCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return dashboardModelChannelStatsCache
}

func normalizeDashboardModelChannelRange(startTime int64, endTime int64) (int64, int64) {
	now := time.Now().Unix()
	if endTime <= 0 || endTime > now+3600 {
		endTime = now
	}
	if startTime <= 0 || startTime >= endTime {
		startTime = endTime - 86400
	}
	maxRange := int64(30 * 86400)
	if endTime-startTime > maxRange {
		startTime = endTime - maxRange
	}
	return startTime, endTime
}

func normalizeDashboardModelChannelLimit(limit int) int {
	if limit <= 0 {
		return 12
	}
	if limit > 30 {
		return 30
	}
	return limit
}

func dashboardModelChannelStatsCacheKey(startTime int64, endTime int64, username string, limit int) string {
	return fmt.Sprintf("%d:%d:%s:%d", startTime, endTime, strings.TrimSpace(username), limit)
}

func GetDashboardModelChannelTagStats(startTime int64, endTime int64, username string, limit int) (*DashboardModelChannelStats, error) {
	startTime, endTime = normalizeDashboardModelChannelRange(startTime, endTime)
	limit = normalizeDashboardModelChannelLimit(limit)
	username = strings.TrimSpace(username)

	cache := getDashboardModelChannelStatsCache()
	cacheKey := dashboardModelChannelStatsCacheKey(startTime, endTime, username, limit)
	if cached, found, err := cache.Get(cacheKey); err == nil && found {
		return &cached, nil
	}

	stats, err := loadDashboardModelChannelTagStats(startTime, endTime, username, limit)
	if err != nil {
		return nil, err
	}
	_ = cache.SetWithTTL(cacheKey, *stats, dashboardModelChannelStatsCacheTTL())
	return stats, nil
}

func loadDashboardModelChannelTagStats(startTime int64, endTime int64, username string, limit int) (*DashboardModelChannelStats, error) {
	models, err := queryDashboardModelSpendStats(startTime, endTime, username, limit)
	if err != nil {
		return nil, err
	}
	total, err := queryDashboardSpendTotal(startTime, endTime, username)
	if err != nil {
		return nil, err
	}

	result := &DashboardModelChannelStats{
		StartTimestamp: startTime,
		EndTimestamp:   endTime,
		Models:         models,
		TotalQuota:     total.Quota,
		TotalRequest:   total.RequestCount,
	}

	modelNames := make([]string, 0, len(models))
	modelQuotaMap := make(map[string]int64, len(models))
	topQuota := int64(0)
	topRequest := int64(0)
	for i := range models {
		topQuota += models[i].Quota
		topRequest += models[i].RequestCount
		modelNames = append(modelNames, models[i].ModelName)
		modelQuotaMap[models[i].ModelName] = models[i].Quota
	}
	if otherQuota := result.TotalQuota - topQuota; otherQuota > 0 {
		otherRequest := result.TotalRequest - topRequest
		if otherRequest < 0 {
			otherRequest = 0
		}
		result.Models = append(result.Models, DashboardModelSpendStat{
			ModelName:    "其他",
			Quota:        otherQuota,
			RequestCount: otherRequest,
		})
	}
	if result.TotalQuota > 0 {
		for i := range result.Models {
			result.Models[i].Share = float64(result.Models[i].Quota) / float64(result.TotalQuota)
		}
	}
	if len(modelNames) == 0 {
		return result, nil
	}

	channelRows, err := queryDashboardModelChannelUsageRows(startTime, endTime, username, modelNames)
	if err != nil {
		return nil, err
	}

	channelTagMap, err := getDashboardChannelTagMap(channelRows)
	if err != nil {
		return nil, err
	}

	tagAgg := make(map[string]*DashboardChannelTagSpendStat)
	for _, row := range channelRows {
		tag := channelTagMap[row.ChannelID]
		key := row.ModelName + "\x00" + tag
		item := tagAgg[key]
		if item == nil {
			item = &DashboardChannelTagSpendStat{
				ModelName: row.ModelName,
				Tag:       tag,
			}
			tagAgg[key] = item
		}
		item.Quota += row.Quota
		item.RequestCount += row.RequestCount
	}

	result.ChannelTagShares = make([]DashboardChannelTagSpendStat, 0, len(tagAgg))
	for _, item := range tagAgg {
		if modelQuota := modelQuotaMap[item.ModelName]; modelQuota > 0 {
			item.Share = float64(item.Quota) / float64(modelQuota)
		}
		result.ChannelTagShares = append(result.ChannelTagShares, *item)
	}

	sort.SliceStable(result.ChannelTagShares, func(i, j int) bool {
		left := result.ChannelTagShares[i]
		right := result.ChannelTagShares[j]
		if left.ModelName != right.ModelName {
			return left.ModelName < right.ModelName
		}
		if left.Quota != right.Quota {
			return left.Quota > right.Quota
		}
		return left.Tag < right.Tag
	})

	return result, nil
}

func queryDashboardModelSpendStats(startTime int64, endTime int64, username string, limit int) ([]DashboardModelSpendStat, error) {
	stats := make([]DashboardModelSpendStat, 0, limit)
	tx := LOG_DB.Table("logs").
		Select("model_name, COUNT(*) as request_count, COALESCE(SUM(quota), 0) as quota").
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Where("model_name <> ''")
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	err := tx.Group("model_name").
		Having("COALESCE(SUM(quota), 0) > 0").
		Order("quota DESC").
		Limit(limit).
		Scan(&stats).Error
	return stats, err
}

func queryDashboardSpendTotal(startTime int64, endTime int64, username string) (dashboardSpendTotalRow, error) {
	var total dashboardSpendTotalRow
	tx := LOG_DB.Table("logs").
		Select("COUNT(*) as request_count, COALESCE(SUM(quota), 0) as quota").
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Where("model_name <> ''")
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	err := tx.Scan(&total).Error
	return total, err
}

func queryDashboardModelChannelUsageRows(startTime int64, endTime int64, username string, modelNames []string) ([]dashboardModelChannelUsageRow, error) {
	rows := make([]dashboardModelChannelUsageRow, 0)
	tx := LOG_DB.Table("logs").
		Select("model_name, channel_id, COUNT(*) as request_count, COALESCE(SUM(quota), 0) as quota").
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Where("model_name IN ?", modelNames)
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	err := tx.Group("model_name, channel_id").
		Having("COALESCE(SUM(quota), 0) > 0").
		Scan(&rows).Error
	return rows, err
}

func getDashboardChannelTagMap(rows []dashboardModelChannelUsageRow) (map[int]string, error) {
	channelIDs := make([]int, 0)
	seen := make(map[int]struct{})
	for _, row := range rows {
		if row.ChannelID <= 0 {
			continue
		}
		if _, ok := seen[row.ChannelID]; ok {
			continue
		}
		seen[row.ChannelID] = struct{}{}
		channelIDs = append(channelIDs, row.ChannelID)
	}

	tagMap := make(map[int]string, len(channelIDs))
	if len(channelIDs) > 0 {
		var channels []Channel
		if err := DB.Model(&Channel{}).Select("id, tag").Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
			return nil, err
		}
		for _, channel := range channels {
			tag := strings.TrimSpace(channel.GetTag())
			if tag == "" {
				tag = dashboardModelChannelUntaggedLabel
			}
			tagMap[channel.Id] = tag
		}
	}

	for _, row := range rows {
		if _, ok := tagMap[row.ChannelID]; !ok {
			tagMap[row.ChannelID] = dashboardModelChannelUntaggedLabel
		}
	}
	return tagMap, nil
}
