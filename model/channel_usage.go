package model

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
)

// ChannelUsage is based on retained consumption/error logs, not upstream cost
// or all retry attempts. Keep log aggregation independent of the main database.
type ChannelUsage struct {
	ChannelID       int   `json:"channel_id"`
	TotalQuota      int64 `json:"total_quota"`
	TotalRequests   int64 `json:"total_requests"`
	SuccessRequests int64 `json:"success_requests"`
	FailedRequests  int64 `json:"failed_requests"`
}

type ChannelUsageSnapshot struct {
	Rows []ChannelUsage `json:"rows"`
	AsOf int64          `json:"as_of"`
}

var usageCacheOnce sync.Once
var usageCache *cachex.HybridCache[ChannelUsageSnapshot]

func GetChannelUsage(start, end int64, group, modelName string) (ChannelUsageSnapshot, bool, error) {
	usageCacheOnce.Do(func() {
		usageCache = cachex.NewHybridCache[ChannelUsageSnapshot](cachex.HybridCacheConfig[ChannelUsageSnapshot]{
			Namespace: cachex.Namespace("channel_usage:v1"), Redis: common.RDB,
			RedisEnabled: func() bool { return common.RedisEnabled && common.RDB != nil },
			RedisCodec:   cachex.JSONCodec[ChannelUsageSnapshot]{},
			Memory: func() *hot.HotCache[string, ChannelUsageSnapshot] {
				return hot.NewHotCache[string, ChannelUsageSnapshot](hot.LRU, channelMonitorStatsCacheCapacity()).WithTTL(channelMonitorStatsCacheTTL()).WithJanitor().Build()
			},
		})
	})
	// Exact boundaries: do not round a selected range to a cache bucket.
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%q:%q", start, end, group, modelName))))
	if v, ok, err := usageCache.Get(key); err == nil && ok {
		return v, true, nil
	}
	rows, err := loadChannelUsage(start, end, group, modelName)
	if err != nil {
		return ChannelUsageSnapshot{}, false, err
	}
	v := ChannelUsageSnapshot{Rows: rows, AsOf: time.Now().Unix()}
	_ = usageCache.SetWithTTL(key, v, channelMonitorStatsCacheTTL())
	return v, false, nil
}

func loadChannelUsage(start, end int64, group, modelName string) ([]ChannelUsage, error) {
	query := LOG_DB.Table("logs").Select("channel_id,"+channelMonitorAggregates()).
		Where("created_at >= ? AND created_at < ?", start, end).
		Where("type IN (?, ?)", LogTypeConsume, LogTypeError).Group("channel_id")
	if group != "" {
		query = query.Where(map[string]interface{}{"group": group})
	}
	if modelName != "" {
		var err error
		query, err = applyExplicitLogTextFilter(query, "model_name", "%"+strings.Trim(modelName, "%")+"%")
		if err != nil {
			return nil, err
		}
	}
	rows := make([]ChannelUsage, 0)
	err := query.Scan(&rows).Error
	return rows, err
}

type ChannelUsageItem struct {
	*Channel
	Usage ChannelUsage `json:"usage"`
}

// PaginateChannelUsage sorts the full filtered set before pagination. A tag's
// children stay together; its quota and success rate must never use just a page.
func PaginateChannelUsage(channels []*Channel, stats []ChannelUsage, tagMode bool, ascending bool, offset, limit int) ([]ChannelUsageItem, int, ChannelUsage) {
	byID := make(map[int]ChannelUsage, len(stats))
	for _, s := range stats {
		byID[s.ChannelID] = s
	}
	type bucket struct {
		key   string
		items []ChannelUsageItem
		quota int64
		id    int
	}
	buckets := make([]*bucket, 0)
	byTag := make(map[string]*bucket)
	summary := ChannelUsage{}
	for _, ch := range channels {
		if tagMode && (ch.Tag == nil || strings.TrimSpace(*ch.Tag) == "") {
			continue
		}
		s := byID[ch.Id]
		s.ChannelID = ch.Id
		summary.TotalQuota += s.TotalQuota
		summary.TotalRequests += s.TotalRequests
		summary.SuccessRequests += s.SuccessRequests
		summary.FailedRequests += s.FailedRequests
		item := ChannelUsageItem{Channel: ch, Usage: s}
		var b *bucket
		if tagMode {
			b = byTag[*ch.Tag]
		}
		if b == nil {
			b = &bucket{id: ch.Id}
			if tagMode {
				b.key = *ch.Tag
				byTag[b.key] = b
			}
			buckets = append(buckets, b)
		}
		b.items = append(b.items, item)
		b.quota += s.TotalQuota
	}
	sort.SliceStable(buckets, func(i, j int) bool {
		if buckets[i].quota == buckets[j].quota {
			return buckets[i].id > buckets[j].id
		}
		if ascending {
			return buckets[i].quota < buckets[j].quota
		}
		return buckets[i].quota > buckets[j].quota
	})
	total := len(buckets)
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	if limit <= 0 {
		return []ChannelUsageItem{}, total, summary
	}
	end := offset + limit
	if end > total || end < offset {
		end = total
	}
	items := make([]ChannelUsageItem, 0)
	for _, b := range buckets[offset:end] {
		items = append(items, b.items...)
	}
	return items, total, summary
}

func ChannelUsageCacheTTL() time.Duration { return channelMonitorStatsCacheTTL() }

// Deleted channels are reported separately and never enter a manageable
// channel's share denominator. Deleted configuration cannot satisfy list filters.
func GetDeletedChannelUsage(stats []ChannelUsage) (ChannelUsage, error) {
	var ids []int
	if err := DB.Model(&Channel{}).Pluck("id", &ids).Error; err != nil {
		return ChannelUsage{}, err
	}
	existing := make(map[int]bool, len(ids))
	for _, id := range ids {
		existing[id] = true
	}
	result := ChannelUsage{}
	for _, s := range stats {
		if s.ChannelID <= 0 || existing[s.ChannelID] {
			continue
		}
		result.TotalQuota += s.TotalQuota
		result.TotalRequests += s.TotalRequests
		result.SuccessRequests += s.SuccessRequests
		result.FailedRequests += s.FailedRequests
	}
	return result, nil
}
