package model

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"gorm.io/gorm"
)

const (
	profitBoardAggregateStateKey     = "logs"
	profitBoardAggregateBucketSecond = int64(3600)
)

type ProfitBoardHourlyStat struct {
	BucketStart                    int64   `json:"bucket_start" gorm:"bigint;uniqueIndex:idx_profit_board_hourly_stats_bucket_channel_model_user,priority:1;index:idx_profit_board_hourly_stats_channel_bucket,priority:2"`
	ChannelId                      int     `json:"channel_id" gorm:"uniqueIndex:idx_profit_board_hourly_stats_bucket_channel_model_user,priority:2;index:idx_profit_board_hourly_stats_channel_bucket,priority:1"`
	ModelName                      string  `json:"model_name" gorm:"type:varchar(255);uniqueIndex:idx_profit_board_hourly_stats_bucket_channel_model_user,priority:3"`
	UserId                         int     `json:"user_id" gorm:"uniqueIndex:idx_profit_board_hourly_stats_bucket_channel_model_user,priority:4"`
	RequestCount                   int     `json:"request_count" gorm:"default:0"`
	QuotaSum                       int64   `json:"quota_sum" gorm:"bigint;default:0"`
	InputTokensSum                 int64   `json:"input_tokens_sum" gorm:"bigint;default:0"`
	CompletionTokensSum            int64   `json:"completion_tokens_sum" gorm:"bigint;default:0"`
	CacheReadTokensSum             int64   `json:"cache_read_tokens_sum" gorm:"bigint;default:0"`
	CacheCreationTokensSum         int64   `json:"cache_creation_tokens_sum" gorm:"bigint;default:0"`
	ReturnedRequestCount           int     `json:"returned_request_count" gorm:"default:0"`
	ReturnedQuotaSum               int64   `json:"returned_quota_sum" gorm:"bigint;default:0"`
	ReturnedInputTokensSum         int64   `json:"returned_input_tokens_sum" gorm:"bigint;default:0"`
	ReturnedCompletionTokensSum    int64   `json:"returned_completion_tokens_sum" gorm:"bigint;default:0"`
	ReturnedCacheReadTokensSum     int64   `json:"returned_cache_read_tokens_sum" gorm:"bigint;default:0"`
	ReturnedCacheCreationTokensSum int64   `json:"returned_cache_creation_tokens_sum" gorm:"bigint;default:0"`
	ReturnedCostSumUSD             float64 `json:"returned_cost_sum_usd" gorm:"default:0"`
}

type ProfitBoardAggregateState struct {
	Key                 string `json:"key" gorm:"primaryKey;type:varchar(32)"`
	CutoverLogID        int    `json:"cutover_log_id" gorm:"default:0"`
	LiveCursorLogID     int    `json:"live_cursor_log_id" gorm:"default:0"`
	BackfillCursorLogID int    `json:"backfill_cursor_log_id" gorm:"default:0"`
	BackfillDone        bool   `json:"backfill_done" gorm:"default:false"`
	UpdatedAt           int64  `json:"updated_at" gorm:"bigint;default:0"`
}

type profitBoardAggregateLogRow struct {
	Id               int    `gorm:"column:id"`
	UserId           int    `gorm:"column:user_id"`
	CreatedAt        int64  `gorm:"column:created_at"`
	ChannelId        int    `gorm:"column:channel_id"`
	ModelName        string `gorm:"column:model_name"`
	Quota            int    `gorm:"column:quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens"`
	Other            string `gorm:"column:other"`
}

type profitBoardUsageSegment struct {
	CreatedAt                      int64
	ChannelId                      int
	UserId                         int
	ModelName                      string
	RequestCount                   int
	QuotaSum                       int64
	InputTokensSum                 int64
	CompletionTokensSum            int64
	CacheReadTokensSum             int64
	CacheCreationTokensSum         int64
	ReturnedRequestCount           int
	ReturnedQuotaSum               int64
	ReturnedInputTokensSum         int64
	ReturnedCompletionTokensSum    int64
	ReturnedCacheReadTokensSum     int64
	ReturnedCacheCreationTokensSum int64
	ReturnedCostSumUSD             float64
	LatestLogId                    int
	LatestLogCreatedAt             int64
}

type profitBoardUsageSlice struct {
	RequestCount           int
	QuotaSum               int64
	InputTokensSum         int64
	CompletionTokensSum    int64
	CacheReadTokensSum     int64
	CacheCreationTokensSum int64
}

type profitBoardAggregateRange struct {
	StartTimestamp int64
	EndTimestamp   int64
	MinLogID       int
	MaxLogID       int
}

var profitBoardAggregateStateLock sync.Mutex
var profitBoardAggregateSyncOnce sync.Once

func profitBoardAggregateStateByKeyQuery(tx *gorm.DB, key string) *gorm.DB {
	return tx.Where(commonKeyCol+" = ?", key)
}

func profitBoardAggregateSummaryEnabled() bool {
	return true
}

func profitBoardAggregateSyncInterval() time.Duration {
	seconds := common.GetEnvOrDefault("PROFIT_BOARD_AGGREGATE_SYNC_INTERVAL_SECONDS", 60)
	if seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func profitBoardAggregateBackfillBatchSize() int {
	size := common.GetEnvOrDefault("PROFIT_BOARD_AGGREGATE_BACKFILL_BATCH_SIZE", 5000)
	if size <= 0 {
		size = 5000
	}
	return size
}

func profitBoardAggregateLiveBatchSize() int {
	size := common.GetEnvOrDefault("PROFIT_BOARD_AGGREGATE_LIVE_BATCH_SIZE", 2000)
	if size <= 0 {
		size = 2000
	}
	return size
}

func profitBoardHourBucket(timestamp int64) int64 {
	if timestamp <= 0 {
		return 0
	}
	return timestamp - (timestamp % profitBoardAggregateBucketSecond)
}

func profitBoardAggregateFullHourWindow(startTimestamp int64, endTimestamp int64) (int64, int64) {
	if endTimestamp <= 0 {
		endTimestamp = common.GetTimestamp()
	}
	if startTimestamp < 0 {
		startTimestamp = 0
	}
	aggregateStart := startTimestamp
	if mod := aggregateStart % profitBoardAggregateBucketSecond; mod != 0 {
		aggregateStart += profitBoardAggregateBucketSecond - mod
	}
	aggregateEnd := endTimestamp - (endTimestamp % profitBoardAggregateBucketSecond)
	if aggregateEnd < aggregateStart {
		return 0, 0
	}
	return aggregateStart, aggregateEnd
}

func profitBoardLatestConsumeLogID() int {
	var maxID int
	LOG_DB.Table("logs").
		Select("COALESCE(MAX(id), 0)").
		Where("type = ?", LogTypeConsume).
		Scan(&maxID)
	return maxID
}

func ensureProfitBoardAggregateState() (*ProfitBoardAggregateState, error) {
	profitBoardAggregateStateLock.Lock()
	defer profitBoardAggregateStateLock.Unlock()

	state := &ProfitBoardAggregateState{}
	err := profitBoardAggregateStateByKeyQuery(DB, profitBoardAggregateStateKey).First(state).Error
	if err == nil {
		return state, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	latestLogID := profitBoardLatestConsumeLogID()
	state = &ProfitBoardAggregateState{
		Key:                 profitBoardAggregateStateKey,
		CutoverLogID:        latestLogID,
		LiveCursorLogID:     latestLogID,
		BackfillCursorLogID: 0,
		BackfillDone:        latestLogID == 0,
		UpdatedAt:           common.GetTimestamp(),
	}
	if err = DB.Create(state).Error; err != nil {
		return nil, err
	}
	return state, nil
}

func loadProfitBoardAggregateState() (*ProfitBoardAggregateState, error) {
	state := &ProfitBoardAggregateState{}
	if err := profitBoardAggregateStateByKeyQuery(DB, profitBoardAggregateStateKey).First(state).Error; err != nil {
		return nil, err
	}
	return state, nil
}

func persistProfitBoardAggregateState(tx *gorm.DB, state *ProfitBoardAggregateState) error {
	state.UpdatedAt = common.GetTimestamp()
	return profitBoardAggregateStateByKeyQuery(tx.Model(&ProfitBoardAggregateState{}), state.Key).
		Updates(map[string]any{
			"cutover_log_id":         state.CutoverLogID,
			"live_cursor_log_id":     state.LiveCursorLogID,
			"backfill_cursor_log_id": state.BackfillCursorLogID,
			"backfill_done":          state.BackfillDone,
			"updated_at":             state.UpdatedAt,
		}).Error
}

func queryProfitBoardLogsForAggregation(minLogID int, maxLogID int, limit int) ([]profitBoardAggregateLogRow, error) {
	rows := make([]profitBoardAggregateLogRow, 0)
	tx := LOG_DB.Table("logs").
		Select("id, user_id, created_at, channel_id, model_name, quota, prompt_tokens, completion_tokens, other").
		Where("type = ?", LogTypeConsume)
	if minLogID > 0 {
		tx = tx.Where("id > ?", minLogID)
	}
	if maxLogID > 0 {
		tx = tx.Where("id <= ?", maxLogID)
	}
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	if err := tx.Order("id asc").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func aggregateProfitBoardLogRows(rows []profitBoardAggregateLogRow) map[string]*ProfitBoardHourlyStat {
	stats := make(map[string]*ProfitBoardHourlyStat)
	for _, row := range rows {
		bucketStart := profitBoardHourBucket(row.CreatedAt)
		key := fmt.Sprintf("%d|%d|%s|%d", bucketStart, row.ChannelId, row.ModelName, row.UserId)
		stat := stats[key]
		if stat == nil {
			stat = &ProfitBoardHourlyStat{
				BucketStart: bucketStart,
				ChannelId:   row.ChannelId,
				ModelName:   row.ModelName,
				UserId:      row.UserId,
			}
			stats[key] = stat
		}

		other := profitBoardOtherInfo{}
		if row.Other != "" {
			_ = common.UnmarshalJsonStr(row.Other, &other)
		}
		cacheReadTokens := other.CacheTokens
		if cacheReadTokens < 0 {
			cacheReadTokens = 0
		}
		cacheCreationTokens := sumCacheCreationTokens(other.tokenUsageOtherInfo)
		if cacheCreationTokens < 0 {
			cacheCreationTokens = 0
		}
		inputTokens := normalizeInputTokens(row.PromptTokens, cacheReadTokens, cacheCreationTokens, other.tokenUsageOtherInfo)
		returnedKnown := (other.UpstreamCostReported || other.UpstreamCost > 0) && other.UpstreamCost >= 0

		stat.RequestCount++
		stat.QuotaSum += int64(row.Quota)
		stat.InputTokensSum += int64(inputTokens)
		stat.CompletionTokensSum += int64(row.CompletionTokens)
		stat.CacheReadTokensSum += int64(cacheReadTokens)
		stat.CacheCreationTokensSum += int64(cacheCreationTokens)
		if returnedKnown {
			stat.ReturnedRequestCount++
			stat.ReturnedQuotaSum += int64(row.Quota)
			stat.ReturnedInputTokensSum += int64(inputTokens)
			stat.ReturnedCompletionTokensSum += int64(row.CompletionTokens)
			stat.ReturnedCacheReadTokensSum += int64(cacheReadTokens)
			stat.ReturnedCacheCreationTokensSum += int64(cacheCreationTokens)
			stat.ReturnedCostSumUSD += other.UpstreamCost
		}
	}
	return stats
}

func mergeProfitBoardHourlyStats(target *ProfitBoardHourlyStat, delta *ProfitBoardHourlyStat) {
	target.RequestCount += delta.RequestCount
	target.QuotaSum += delta.QuotaSum
	target.InputTokensSum += delta.InputTokensSum
	target.CompletionTokensSum += delta.CompletionTokensSum
	target.CacheReadTokensSum += delta.CacheReadTokensSum
	target.CacheCreationTokensSum += delta.CacheCreationTokensSum
	target.ReturnedRequestCount += delta.ReturnedRequestCount
	target.ReturnedQuotaSum += delta.ReturnedQuotaSum
	target.ReturnedInputTokensSum += delta.ReturnedInputTokensSum
	target.ReturnedCompletionTokensSum += delta.ReturnedCompletionTokensSum
	target.ReturnedCacheReadTokensSum += delta.ReturnedCacheReadTokensSum
	target.ReturnedCacheCreationTokensSum += delta.ReturnedCacheCreationTokensSum
	target.ReturnedCostSumUSD += delta.ReturnedCostSumUSD
}

func saveProfitBoardHourlyStats(tx *gorm.DB, stats map[string]*ProfitBoardHourlyStat) error {
	keys := make([]string, 0, len(stats))
	for key := range stats {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		delta := stats[key]
		current := &ProfitBoardHourlyStat{}
		err := tx.Where("bucket_start = ? AND channel_id = ? AND model_name = ? AND user_id = ?",
			delta.BucketStart,
			delta.ChannelId,
			delta.ModelName,
			delta.UserId,
		).First(current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = tx.Create(delta).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		mergeProfitBoardHourlyStats(current, delta)
		if err = tx.Model(&ProfitBoardHourlyStat{}).
			Where("bucket_start = ? AND channel_id = ? AND model_name = ? AND user_id = ?",
				current.BucketStart,
				current.ChannelId,
				current.ModelName,
				current.UserId,
			).
			Updates(map[string]any{
				"request_count":                      current.RequestCount,
				"quota_sum":                          current.QuotaSum,
				"input_tokens_sum":                   current.InputTokensSum,
				"completion_tokens_sum":              current.CompletionTokensSum,
				"cache_read_tokens_sum":              current.CacheReadTokensSum,
				"cache_creation_tokens_sum":          current.CacheCreationTokensSum,
				"returned_request_count":             current.ReturnedRequestCount,
				"returned_quota_sum":                 current.ReturnedQuotaSum,
				"returned_input_tokens_sum":          current.ReturnedInputTokensSum,
				"returned_completion_tokens_sum":     current.ReturnedCompletionTokensSum,
				"returned_cache_read_tokens_sum":     current.ReturnedCacheReadTokensSum,
				"returned_cache_creation_tokens_sum": current.ReturnedCacheCreationTokensSum,
				"returned_cost_sum_usd":              current.ReturnedCostSumUSD,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func syncProfitBoardAggregateRows(rows []profitBoardAggregateLogRow) error {
	if len(rows) == 0 {
		return nil
	}
	stats := aggregateProfitBoardLogRows(rows)
	return DB.Transaction(func(tx *gorm.DB) error {
		return saveProfitBoardHourlyStats(tx, stats)
	})
}

func syncProfitBoardAggregateBackfillStep(limit int) (int, error) {
	if limit <= 0 {
		limit = profitBoardAggregateBackfillBatchSize()
	}
	state, err := ensureProfitBoardAggregateState()
	if err != nil {
		return 0, err
	}
	if state.BackfillDone {
		return 0, nil
	}
	rows, err := queryProfitBoardLogsForAggregation(state.BackfillCursorLogID, state.CutoverLogID, limit)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		state.BackfillDone = true
		return 0, DB.Transaction(func(tx *gorm.DB) error {
			return persistProfitBoardAggregateState(tx, state)
		})
	}
	if err = DB.Transaction(func(tx *gorm.DB) error {
		if errTx := saveProfitBoardHourlyStats(tx, aggregateProfitBoardLogRows(rows)); errTx != nil {
			return errTx
		}
		state.BackfillCursorLogID = rows[len(rows)-1].Id
		if state.BackfillCursorLogID >= state.CutoverLogID {
			state.BackfillDone = true
		}
		return persistProfitBoardAggregateState(tx, state)
	}); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func syncProfitBoardAggregateLiveToLatest() (int, error) {
	state, err := ensureProfitBoardAggregateState()
	if err != nil {
		return 0, err
	}
	latestLogID := profitBoardLatestConsumeLogID()
	if latestLogID <= state.LiveCursorLogID {
		return 0, nil
	}
	rows, err := queryProfitBoardLogsForAggregation(state.LiveCursorLogID, latestLogID, profitBoardAggregateLiveBatchSize())
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err = DB.Transaction(func(tx *gorm.DB) error {
		if errTx := saveProfitBoardHourlyStats(tx, aggregateProfitBoardLogRows(rows)); errTx != nil {
			return errTx
		}
		state.LiveCursorLogID = rows[len(rows)-1].Id
		return persistProfitBoardAggregateState(tx, state)
	}); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func SyncProfitBoardAggregate(forceLive bool) error {
	if _, err := ensureProfitBoardAggregateState(); err != nil {
		return err
	}
	if forceLive {
		for {
			processed, err := syncProfitBoardAggregateLiveToLatest()
			if err != nil {
				return err
			}
			if processed == 0 {
				break
			}
		}
	}
	for i := 0; i < 2; i++ {
		processed, err := syncProfitBoardAggregateBackfillStep(profitBoardAggregateBackfillBatchSize())
		if err != nil {
			return err
		}
		if processed == 0 {
			break
		}
	}
	return nil
}

func StartProfitBoardAggregateSyncTask() {
	profitBoardAggregateSyncOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			common.SysLog("profit board aggregate sync task started")
			_ = SyncProfitBoardAggregate(true)
			ticker := time.NewTicker(profitBoardAggregateSyncInterval())
			defer ticker.Stop()
			for range ticker.C {
				if err := SyncProfitBoardAggregate(true); err != nil {
					common.SysError("profit board aggregate sync failed: " + err.Error())
				}
			}
		}()
	})
}

func profitBoardQueryAggregateRanges(query ProfitBoardQuery) ([]profitBoardAggregateRange, int64, int64, error) {
	state, err := ensureProfitBoardAggregateState()
	if err != nil {
		return nil, 0, 0, err
	}
	aggregateStart, aggregateEnd := profitBoardAggregateFullHourWindow(query.StartTimestamp, query.EndTimestamp)
	ranges := make([]profitBoardAggregateRange, 0, 4)
	if aggregateStart == 0 && aggregateEnd == 0 {
		ranges = append(ranges, profitBoardAggregateRange{
			StartTimestamp: query.StartTimestamp,
			EndTimestamp:   query.EndTimestamp,
		})
		return ranges, 0, 0, nil
	}
	if query.StartTimestamp < aggregateStart {
		ranges = append(ranges, profitBoardAggregateRange{
			StartTimestamp: query.StartTimestamp,
			EndTimestamp:   minInt64(query.EndTimestamp, aggregateStart),
		})
	}
	if aggregateEnd > 0 && aggregateEnd < query.EndTimestamp {
		ranges = append(ranges, profitBoardAggregateRange{
			StartTimestamp: maxInt64(query.StartTimestamp, aggregateEnd),
			EndTimestamp:   query.EndTimestamp,
		})
	}
	if aggregateStart < aggregateEnd {
		if !state.BackfillDone && state.BackfillCursorLogID < state.CutoverLogID {
			ranges = append(ranges, profitBoardAggregateRange{
				StartTimestamp: aggregateStart,
				EndTimestamp:   aggregateEnd,
				MinLogID:       state.BackfillCursorLogID,
				MaxLogID:       state.CutoverLogID,
			})
		}
		if state.LiveCursorLogID < profitBoardLatestConsumeLogID() {
			ranges = append(ranges, profitBoardAggregateRange{
				StartTimestamp: aggregateStart,
				EndTimestamp:   aggregateEnd,
				MinLogID:       state.LiveCursorLogID,
			})
		}
	}
	return ranges, aggregateStart, aggregateEnd, nil
}

func minInt64(a int64, b int64) int64 {
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func queryProfitBoardHourlyStats(query ProfitBoardQuery, channelIDs []int, aggregateStart int64, aggregateEnd int64) ([]ProfitBoardHourlyStat, error) {
	if len(channelIDs) == 0 || aggregateStart >= aggregateEnd {
		return nil, nil
	}
	stats := make([]ProfitBoardHourlyStat, 0)
	err := DB.Model(&ProfitBoardHourlyStat{}).
		Where("channel_id IN ?", channelIDs).
		Where("bucket_start >= ? AND bucket_start < ?", aggregateStart, aggregateEnd).
		Order("bucket_start asc, channel_id asc, model_name asc, user_id asc").
		Find(&stats).Error
	return stats, err
}

func iterateProfitBoardRawRanges(query ProfitBoardQuery, batches []ProfitBoardBatchInfo, ranges []profitBoardAggregateRange, callback func(segment profitBoardUsageSegment) error) error {
	channelIDs := collectProfitBoardChannelIDs(batches)
	if len(channelIDs) == 0 {
		return nil
	}
	batchByChannelID := make(map[int]ProfitBoardBatchInfo, len(channelIDs))
	for _, batch := range batches {
		for _, channelID := range batch.ChannelIDs {
			batchByChannelID[channelID] = batch
		}
	}
	for _, r := range ranges {
		if r.EndTimestamp > 0 && r.StartTimestamp >= r.EndTimestamp {
			continue
		}
		tx := LOG_DB.Table("logs").
			Select("id, user_id, created_at, channel_id, model_name, quota, prompt_tokens, completion_tokens, other").
			Where("type = ?", LogTypeConsume).
			Where("channel_id IN ?", channelIDs)
		if r.StartTimestamp > 0 {
			tx = tx.Where("created_at >= ?", r.StartTimestamp)
		}
		if r.EndTimestamp > 0 {
			tx = tx.Where("created_at < ?", r.EndTimestamp)
		}
		if r.MinLogID > 0 {
			tx = tx.Where("id > ?", r.MinLogID)
		}
		if r.MaxLogID > 0 {
			tx = tx.Where("id <= ?", r.MaxLogID)
		}
		rows, err := tx.Order("id desc").Rows()
		if err != nil {
			return err
		}
		var rowIterErr error
		for rows.Next() {
			var row profitBoardAggregateLogRow
			if err = LOG_DB.ScanRows(rows, &row); err != nil {
				rowIterErr = err
				break
			}
			batch, ok := batchByChannelID[row.ChannelId]
			if !ok {
				continue
			}
			if row.CreatedAt < profitBoardEffectiveStartTimestamp(batch, query.StartTimestamp) {
				continue
			}
			other := profitBoardOtherInfo{}
			if row.Other != "" {
				_ = common.UnmarshalJsonStr(row.Other, &other)
			}
			cacheReadTokens := other.CacheTokens
			if cacheReadTokens < 0 {
				cacheReadTokens = 0
			}
			cacheCreationTokens := sumCacheCreationTokens(other.tokenUsageOtherInfo)
			if cacheCreationTokens < 0 {
				cacheCreationTokens = 0
			}
			inputTokens := normalizeInputTokens(row.PromptTokens, cacheReadTokens, cacheCreationTokens, other.tokenUsageOtherInfo)
			segment := profitBoardUsageSegment{
				CreatedAt:              row.CreatedAt,
				ChannelId:              row.ChannelId,
				UserId:                 row.UserId,
				ModelName:              row.ModelName,
				RequestCount:           1,
				QuotaSum:               int64(row.Quota),
				InputTokensSum:         int64(inputTokens),
				CompletionTokensSum:    int64(row.CompletionTokens),
				CacheReadTokensSum:     int64(cacheReadTokens),
				CacheCreationTokensSum: int64(cacheCreationTokens),
				LatestLogId:            row.Id,
				LatestLogCreatedAt:     row.CreatedAt,
			}
			if (other.UpstreamCostReported || other.UpstreamCost > 0) && other.UpstreamCost >= 0 {
				segment.ReturnedRequestCount = 1
				segment.ReturnedQuotaSum = int64(row.Quota)
				segment.ReturnedInputTokensSum = int64(inputTokens)
				segment.ReturnedCompletionTokensSum = int64(row.CompletionTokens)
				segment.ReturnedCacheReadTokensSum = int64(cacheReadTokens)
				segment.ReturnedCacheCreationTokensSum = int64(cacheCreationTokens)
				segment.ReturnedCostSumUSD = other.UpstreamCost
			}
			if err = callback(segment); err != nil {
				rowIterErr = err
				break
			}
		}
		if closeErr := rows.Close(); rowIterErr == nil && closeErr != nil {
			rowIterErr = closeErr
		}
		if rowIterErr != nil {
			return rowIterErr
		}
	}
	return nil
}

func iterateProfitBoardAggregateSegments(query ProfitBoardQuery, batches []ProfitBoardBatchInfo, callback func(segment profitBoardUsageSegment) error) error {
	channelIDs := collectProfitBoardChannelIDs(batches)
	ranges, aggregateStart, aggregateEnd, err := profitBoardQueryAggregateRanges(query)
	if err != nil {
		return err
	}
	if aggregateStart < aggregateEnd && len(channelIDs) > 0 {
		stats, statErr := queryProfitBoardHourlyStats(query, channelIDs, aggregateStart, aggregateEnd)
		if statErr != nil {
			return statErr
		}
		for _, stat := range stats {
			segment := profitBoardUsageSegment{
				CreatedAt:                      stat.BucketStart,
				ChannelId:                      stat.ChannelId,
				UserId:                         stat.UserId,
				ModelName:                      stat.ModelName,
				RequestCount:                   stat.RequestCount,
				QuotaSum:                       stat.QuotaSum,
				InputTokensSum:                 stat.InputTokensSum,
				CompletionTokensSum:            stat.CompletionTokensSum,
				CacheReadTokensSum:             stat.CacheReadTokensSum,
				CacheCreationTokensSum:         stat.CacheCreationTokensSum,
				ReturnedRequestCount:           stat.ReturnedRequestCount,
				ReturnedQuotaSum:               stat.ReturnedQuotaSum,
				ReturnedInputTokensSum:         stat.ReturnedInputTokensSum,
				ReturnedCompletionTokensSum:    stat.ReturnedCompletionTokensSum,
				ReturnedCacheReadTokensSum:     stat.ReturnedCacheReadTokensSum,
				ReturnedCacheCreationTokensSum: stat.ReturnedCacheCreationTokensSum,
				ReturnedCostSumUSD:             stat.ReturnedCostSumUSD,
			}
			if err = callback(segment); err != nil {
				return err
			}
		}
	}
	return iterateProfitBoardRawRanges(query, batches, ranges, callback)
}

func profitBoardManualRevenueUSDFromTotals(inputTokens int64, completionTokens int64, cacheReadTokens int64, cacheCreationTokens int64, rule ProfitBoardModelPricingRule) float64 {
	return float64(inputTokens)*rule.InputPrice/1_000_000 +
		float64(completionTokens)*rule.OutputPrice/1_000_000 +
		float64(cacheReadTokens)*rule.CacheReadPrice/1_000_000 +
		float64(cacheCreationTokens)*rule.CacheCreationPrice/1_000_000
}

func profitBoardManualSiteRevenueUSDFromUsage(modelName string, inputTokens int64, completionTokens int64, cacheReadTokens int64, cacheCreationTokens int64, rules []ProfitBoardModelPricingRule) (float64, string, bool) {
	rule, usedDefault, ok := profitBoardFindManualRule(modelName, rules)
	if !ok {
		return 0, "manual_missing", false
	}
	if usedDefault {
		return profitBoardManualRevenueUSDFromTotals(inputTokens, completionTokens, cacheReadTokens, cacheCreationTokens, rule), "manual_default", true
	}
	return profitBoardManualRevenueUSDFromTotals(inputTokens, completionTokens, cacheReadTokens, cacheCreationTokens, rule), "manual_rule", true
}

func profitBoardSiteModelRevenueUSDFromUsage(modelName string, requestCount int, inputTokens int64, completionTokens int64, cacheReadTokens int64, cacheCreationTokens int64, config ProfitBoardTokenPricingConfig, pricingMap map[string]Pricing, groupRatios map[string]float64) (float64, string, bool) {
	pricing, ok := pricingMap[modelName]
	if !ok {
		return 0, "site_model_not_found", false
	}
	if len(config.ModelNames) > 0 {
		matched := false
		for _, selected := range config.ModelNames {
			if selected == modelName {
				matched = true
				break
			}
		}
		if !matched {
			return 0, "site_model_not_selected", false
		}
	}
	groupRatio, ok := profitBoardResolveGroupRatio(pricing.EnableGroup, config.Group, groupRatios)
	if !ok {
		return 0, "site_model_group_unmatched", false
	}
	priceFactor := 1.0
	source := "site_model_standard"
	if config.UseRechargePrice {
		priceFactor = profitBoardPriceFactor(true)
		source = "site_model_recharge"
	} else if config.PlanID > 0 {
		priceFactor = profitBoardPlanPriceFactor(config.PlanID)
		source = "site_model_package"
	}
	if pricing.QuotaType == 1 {
		return float64(requestCount) * (pricing.ModelPrice*groupRatio*priceFactor + config.FixedAmount), source, true
	}
	baseInputPrice := pricing.ModelRatio * 2 * groupRatio * priceFactor
	cacheReadPrice := 0.0
	if pricing.SupportsCacheRead {
		cacheReadPrice = baseInputPrice * pricing.CacheRatio
	}
	cacheCreationPrice := 0.0
	if pricing.SupportsCacheCreation {
		cacheCreationPrice = baseInputPrice * pricing.CacheCreationRatio
	}
	outputPrice := pricing.ModelRatio * pricing.CompletionRatio * 2 * groupRatio * priceFactor
	return float64(inputTokens)*baseInputPrice/1_000_000 +
		float64(completionTokens)*outputPrice/1_000_000 +
		float64(cacheReadTokens)*cacheReadPrice/1_000_000 +
		float64(cacheCreationTokens)*cacheCreationPrice/1_000_000 +
		float64(requestCount)*config.FixedAmount, source, true
}

func profitBoardSiteRevenueUSDFromUsage(modelName string, requestCount int, quotaSum int64, inputTokens int64, completionTokens int64, cacheReadTokens int64, cacheCreationTokens int64, config ProfitBoardTokenPricingConfig, pricingMap map[string]Pricing, groupRatios map[string]float64) (float64, string, bool) {
	if config.PricingMode == ProfitBoardSitePricingSiteModel {
		if amount, source, ok := profitBoardSiteModelRevenueUSDFromUsage(modelName, requestCount, inputTokens, completionTokens, cacheReadTokens, cacheCreationTokens, config, pricingMap, groupRatios); ok {
			return amount, source, true
		}
		amount := profitBoardTokenMoneyUSD(int(inputTokens), int(completionTokens), int(cacheReadTokens), int(cacheCreationTokens), config) + float64(max(0, requestCount-1))*config.FixedAmount
		if amount == 0 {
			return 0, "site_model_missing", false
		}
		return amount, "manual_fallback", true
	}
	if strings.TrimSpace(config.PricingMode) == ProfitBoardComboSiteModeLogQuota {
		if quotaSum <= 0 {
			return 0, "log_quota_zero", false
		}
		return float64(quotaSum) / common.QuotaPerUnit, "log_quota", true
	}
	amount := profitBoardTokenMoneyUSD(int(inputTokens), int(completionTokens), int(cacheReadTokens), int(cacheCreationTokens), config) + float64(max(0, requestCount-1))*config.FixedAmount
	return amount, "manual", true
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildProfitBoardAggregateActivityWatermark() string {
	state, err := ensureProfitBoardAggregateState()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d:%d:%d:%d", state.CutoverLogID, state.LiveCursorLogID, state.BackfillCursorLogID, state.UpdatedAt)
}

func profitBoardReturnedUsageSlice(segment profitBoardUsageSegment) profitBoardUsageSlice {
	return profitBoardUsageSlice{
		RequestCount:           segment.ReturnedRequestCount,
		QuotaSum:               segment.ReturnedQuotaSum,
		InputTokensSum:         segment.ReturnedInputTokensSum,
		CompletionTokensSum:    segment.ReturnedCompletionTokensSum,
		CacheReadTokensSum:     segment.ReturnedCacheReadTokensSum,
		CacheCreationTokensSum: segment.ReturnedCacheCreationTokensSum,
	}
}

func profitBoardMissingUsageSlice(segment profitBoardUsageSegment) profitBoardUsageSlice {
	return profitBoardUsageSlice{
		RequestCount:           segment.RequestCount - segment.ReturnedRequestCount,
		QuotaSum:               segment.QuotaSum - segment.ReturnedQuotaSum,
		InputTokensSum:         segment.InputTokensSum - segment.ReturnedInputTokensSum,
		CompletionTokensSum:    segment.CompletionTokensSum - segment.ReturnedCompletionTokensSum,
		CacheReadTokensSum:     segment.CacheReadTokensSum - segment.ReturnedCacheReadTokensSum,
		CacheCreationTokensSum: segment.CacheCreationTokensSum - segment.ReturnedCacheCreationTokensSum,
	}
}

func profitBoardTotalUsageSlice(segment profitBoardUsageSegment) profitBoardUsageSlice {
	return profitBoardUsageSlice{
		RequestCount:           segment.RequestCount,
		QuotaSum:               segment.QuotaSum,
		InputTokensSum:         segment.InputTokensSum,
		CompletionTokensSum:    segment.CompletionTokensSum,
		CacheReadTokensSum:     segment.CacheReadTokensSum,
		CacheCreationTokensSum: segment.CacheCreationTokensSum,
	}
}

func profitBoardSiteRevenueByUsageSlice(modelName string, usage profitBoardUsageSlice, comboPricing profitBoardResolvedComboPricing, pricingMap map[string]Pricing, groupRatios map[string]float64) (float64, string, bool) {
	if usage.RequestCount <= 0 {
		return 0, "empty", true
	}
	switch comboPricing.SiteMode {
	case ProfitBoardComboSiteModeLogQuota:
		if len(comboPricing.SharedSite.ModelNames) > 0 {
			matched := false
			for _, selected := range comboPricing.SharedSite.ModelNames {
				if selected == modelName {
					matched = true
					break
				}
			}
			if !matched {
				return 0, "site_model_not_selected", false
			}
		}
		if usage.QuotaSum <= 0 {
			return 0, "log_quota_zero", false
		}
		return float64(usage.QuotaSum) / common.QuotaPerUnit, "log_quota", true
	case ProfitBoardComboSiteModeSharedSite:
		config := ProfitBoardTokenPricingConfig{
			PricingMode:      ProfitBoardSitePricingSiteModel,
			ModelNames:       comboPricing.SharedSite.ModelNames,
			Group:            comboPricing.SharedSite.Group,
			UseRechargePrice: comboPricing.SharedSite.UseRechargePrice,
			PlanID:           comboPricing.SharedSite.PlanID,
		}
		amount, source, ok := profitBoardSiteModelRevenueUSDFromUsage(
			modelName,
			usage.RequestCount,
			usage.InputTokensSum,
			usage.CompletionTokensSum,
			usage.CacheReadTokensSum,
			usage.CacheCreationTokensSum,
			config,
			pricingMap,
			groupRatios,
		)
		if ok {
			return amount, source, true
		}
		fallbackAmount, fallbackSource, fallbackOK := profitBoardManualSiteRevenueUSDFromUsage(
			modelName,
			usage.InputTokensSum,
			usage.CompletionTokensSum,
			usage.CacheReadTokensSum,
			usage.CacheCreationTokensSum,
			comboPricing.SiteRules,
		)
		if fallbackOK {
			return fallbackAmount, fallbackSource, true
		}
		return 0, profitBoardComposeSiteMissingReason(source, fallbackSource), false
	default:
		return profitBoardManualSiteRevenueUSDFromUsage(
			modelName,
			usage.InputTokensSum,
			usage.CompletionTokensSum,
			usage.CacheReadTokensSum,
			usage.CacheCreationTokensSum,
			comboPricing.SiteRules,
		)
	}
}

func profitBoardManualCostByUsageSlice(modelName string, usage profitBoardUsageSlice, comboPricing profitBoardResolvedComboPricing) (float64, string, bool) {
	if usage.RequestCount <= 0 {
		return 0, "empty", true
	}
	if comboPricing.CostSource == ProfitBoardCostSourceManualOnly &&
		len(comboPricing.UpstreamRules) == 0 &&
		comboPricing.UpstreamFixedTotalAmount > 0 {
		return 0, "manual_fixed_total_only", true
	}
	rule, usedDefault, ok := profitBoardFindManualRule(modelName, comboPricing.UpstreamRules)
	if !ok {
		return 0, "manual_missing", false
	}
	if usedDefault {
		return profitBoardManualRevenueUSDFromTotals(usage.InputTokensSum, usage.CompletionTokensSum, usage.CacheReadTokensSum, usage.CacheCreationTokensSum, rule), "manual_default", true
	}
	return profitBoardManualRevenueUSDFromTotals(usage.InputTokensSum, usage.CompletionTokensSum, usage.CacheReadTokensSum, usage.CacheCreationTokensSum, rule), "manual_rule", true
}

func accumulateProfitBoardSegment(
	report *ProfitBoardReport,
	segment profitBoardUsageSegment,
	batch ProfitBoardBatchInfo,
	comboPricing profitBoardResolvedComboPricing,
	excludedUserSet map[int]struct{},
	pricingMap map[string]Pricing,
	groupRatios map[string]float64,
	batchSummaryMap map[string]*ProfitBoardBatchSummary,
	timeBuckets map[string]*ProfitBoardTimeseriesPoint,
	channelBreakdown map[string]*ProfitBoardBreakdownItem,
	modelBreakdown map[string]*ProfitBoardBreakdownItem,
	channelNameMap map[int]string,
	channelTagMap map[int]string,
	warningAccumulator *profitBoardWarningAccumulator,
	siteRevenueAllocation profitBoardSiteRevenueAllocation,
	wantTimeseries bool,
	wantChannelBreakdown bool,
	wantModelBreakdown bool,
	wantWarningItems bool,
	granularity string,
	customIntervalMinutes int,
) error {
	batchSummary := batchSummaryMap[batch.Id]
	totalUsage := profitBoardTotalUsageSlice(segment)
	returnedUsage := profitBoardReturnedUsageSlice(segment)
	missingUsage := profitBoardMissingUsageSlice(segment)
	actualSiteRevenueUSDTotal := float64(totalUsage.QuotaSum) / common.QuotaPerUnit
	actualSiteRevenueUSDReturned := float64(returnedUsage.QuotaSum) / common.QuotaPerUnit
	actualSiteRevenueUSDMissing := float64(missingUsage.QuotaSum) / common.QuotaPerUnit

	report.Summary.RequestCount += totalUsage.RequestCount
	batchSummary.RequestCount += totalUsage.RequestCount

	_, excludedFromRevenue := excludedUserSet[segment.UserId]
	sitePricingSource := ""
	sitePricingKnown := false
	configuredSiteRevenueUSDTotal := 0.0
	configuredSiteRevenueUSDReturned := 0.0
	configuredSiteRevenueUSDMissing := 0.0
	if excludedFromRevenue {
		sitePricingSource = "excluded_user"
		sitePricingKnown = true
	} else {
		configuredSiteRevenueUSDTotal, sitePricingSource, sitePricingKnown = profitBoardSiteRevenueByUsageSlice(segment.ModelName, totalUsage, comboPricing, pricingMap, groupRatios)
		if sitePricingKnown {
			configuredSiteRevenueUSDReturned, _, _ = profitBoardSiteRevenueByUsageSlice(segment.ModelName, returnedUsage, comboPricing, pricingMap, groupRatios)
			configuredSiteRevenueUSDMissing = configuredSiteRevenueUSDTotal - configuredSiteRevenueUSDReturned
		}
	}

	if !excludedFromRevenue {
		siteRevenueAllocation.EligibleBatchRequestCount[batch.Id] += totalUsage.RequestCount
		siteRevenueAllocation.EligibleChannelRequestCount[batch.Id+"|"+strconv.Itoa(segment.ChannelId)] += totalUsage.RequestCount
		siteRevenueAllocation.EligibleModelRequestCount[batch.Id+"|"+segment.ModelName] += totalUsage.RequestCount
	}
	if sitePricingKnown && strings.HasPrefix(sitePricingSource, "site_model_") {
		report.Summary.SiteModelMatchCount += totalUsage.RequestCount
		batchSummary.SiteModelMatchCount += totalUsage.RequestCount
	}
	if !sitePricingKnown {
		report.Summary.MissingSitePricingCount += totalUsage.RequestCount
		batchSummary.MissingSitePricingCount += totalUsage.RequestCount
		if wantWarningItems && warningAccumulator != nil {
			for i := 0; i < totalUsage.RequestCount; i++ {
				warningAccumulator.add("missing_site_pricing", sitePricingSource, segment.ChannelId, segment.ModelName, channelNameMap, channelTagMap)
			}
		}
	}

	configuredSiteRevenueCNYTotal := 0.0
	configuredSiteRevenueCNYReturned := 0.0
	configuredSiteRevenueCNYMissing := 0.0
	if sitePricingKnown {
		configuredSiteRevenueCNYTotal = profitBoardConfiguredSiteRevenueCNY(configuredSiteRevenueUSDTotal, comboPricing)
		configuredSiteRevenueCNYReturned = profitBoardConfiguredSiteRevenueCNY(configuredSiteRevenueUSDReturned, comboPricing)
		configuredSiteRevenueCNYMissing = configuredSiteRevenueCNYTotal - configuredSiteRevenueCNYReturned
		report.Summary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSDTotal
		report.Summary.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNYTotal
		batchSummary.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSDTotal
		batchSummary.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNYTotal
	}

	report.Summary.ActualSiteRevenueUSD += actualSiteRevenueUSDTotal
	batchSummary.ActualSiteRevenueUSD += actualSiteRevenueUSDTotal

	isWalletCombo := profitBoardComboUsesWalletObserver(comboPricing)
	upstreamCostUSDKnown := 0.0
	upstreamCostCNYKnown := 0.0
	upstreamKnownCount := 0
	manualKnownCount := 0
	if isWalletCombo {
		upstreamKnownCount = totalUsage.RequestCount
	} else {
		switch comboPricing.CostSource {
		case ProfitBoardCostSourceReturnedOnly:
			upstreamKnownCount = returnedUsage.RequestCount
			upstreamCostUSDKnown = segment.ReturnedCostSumUSD
			if returnedUsage.RequestCount > 0 {
				report.Summary.ReturnedCostCount += returnedUsage.RequestCount
				batchSummary.ReturnedCostCount += returnedUsage.RequestCount
			}
			if missingUsage.RequestCount > 0 && wantWarningItems && warningAccumulator != nil {
				for i := 0; i < missingUsage.RequestCount; i++ {
					warningAccumulator.add("missing_upstream_cost", "returned_cost_missing", segment.ChannelId, segment.ModelName, channelNameMap, channelTagMap)
				}
			}
		case ProfitBoardCostSourceManualOnly:
			if cost, source, ok := profitBoardManualCostByUsageSlice(segment.ModelName, totalUsage, comboPricing); ok {
				upstreamKnownCount = totalUsage.RequestCount
				upstreamCostUSDKnown = cost
				if source != "empty" {
					manualKnownCount = totalUsage.RequestCount
					report.Summary.ManualCostCount += totalUsage.RequestCount
					batchSummary.ManualCostCount += totalUsage.RequestCount
				}
			} else if wantWarningItems && warningAccumulator != nil {
				for i := 0; i < totalUsage.RequestCount; i++ {
					warningAccumulator.add("missing_upstream_cost", "manual_missing", segment.ChannelId, segment.ModelName, channelNameMap, channelTagMap)
				}
			}
		default:
			upstreamKnownCount = returnedUsage.RequestCount
			upstreamCostUSDKnown = segment.ReturnedCostSumUSD
			if returnedUsage.RequestCount > 0 {
				report.Summary.ReturnedCostCount += returnedUsage.RequestCount
				batchSummary.ReturnedCostCount += returnedUsage.RequestCount
			}
			if missingUsage.RequestCount > 0 {
				if cost, source, ok := profitBoardManualCostByUsageSlice(segment.ModelName, missingUsage, comboPricing); ok {
					upstreamKnownCount += missingUsage.RequestCount
					upstreamCostUSDKnown += cost
					if source != "empty" {
						manualKnownCount += missingUsage.RequestCount
						report.Summary.ManualCostCount += missingUsage.RequestCount
						batchSummary.ManualCostCount += missingUsage.RequestCount
					}
				} else if wantWarningItems && warningAccumulator != nil {
					for i := 0; i < missingUsage.RequestCount; i++ {
						warningAccumulator.add("missing_upstream_cost", "manual_missing", segment.ChannelId, segment.ModelName, channelNameMap, channelTagMap)
					}
				}
			}
		}
		upstreamCostCNYKnown = profitBoardConfiguredUpstreamCostCNY(upstreamCostUSDKnown, comboPricing)
		if upstreamKnownCount > 0 {
			report.Summary.KnownUpstreamCostCount += upstreamKnownCount
			batchSummary.KnownUpstreamCostCount += upstreamKnownCount
			report.Summary.UpstreamCostUSD += upstreamCostUSDKnown
			report.Summary.UpstreamCostCNY += upstreamCostCNYKnown
			batchSummary.UpstreamCostUSD += upstreamCostUSDKnown
			batchSummary.UpstreamCostCNY += upstreamCostCNYKnown
		}
		missingUpstreamCount := totalUsage.RequestCount - upstreamKnownCount
		if missingUpstreamCount > 0 {
			report.Summary.MissingUpstreamCostCount += missingUpstreamCount
			batchSummary.MissingUpstreamCostCount += missingUpstreamCount
		}
	}

	configuredProfitUSD := 0.0
	configuredProfitCNY := 0.0
	actualProfitUSD := 0.0
	if isWalletCombo {
		actualProfitUSD = actualSiteRevenueUSDTotal
		if sitePricingKnown {
			configuredProfitUSD = configuredSiteRevenueUSDTotal
			configuredProfitCNY = configuredSiteRevenueCNYTotal
		}
	} else {
		knownActualSiteRevenueUSD := actualSiteRevenueUSDReturned
		knownConfiguredSiteRevenueUSD := configuredSiteRevenueUSDReturned
		knownConfiguredSiteRevenueCNY := configuredSiteRevenueCNYReturned
		if manualKnownCount > 0 {
			knownActualSiteRevenueUSD += actualSiteRevenueUSDMissing
			knownConfiguredSiteRevenueUSD += configuredSiteRevenueUSDMissing
			knownConfiguredSiteRevenueCNY += configuredSiteRevenueCNYMissing
		}
		actualProfitUSD = knownActualSiteRevenueUSD - upstreamCostUSDKnown
		if sitePricingKnown {
			configuredProfitUSD = knownConfiguredSiteRevenueUSD - upstreamCostUSDKnown
			configuredProfitCNY = knownConfiguredSiteRevenueCNY - upstreamCostCNYKnown
		}
	}

	report.Summary.ActualProfitUSD += actualProfitUSD
	batchSummary.ActualProfitUSD += actualProfitUSD
	if sitePricingKnown {
		report.Summary.ConfiguredProfitUSD += configuredProfitUSD
		report.Summary.ConfiguredProfitCNY += configuredProfitCNY
		batchSummary.ConfiguredProfitUSD += configuredProfitUSD
		batchSummary.ConfiguredProfitCNY += configuredProfitCNY
	}

	bucketTimestamp, bucketLabel := buildProfitBoardBucket(segment.CreatedAt, granularity, customIntervalMinutes)
	channelLabel := channelNameMap[segment.ChannelId]
	if channelLabel == "" {
		channelLabel = fmt.Sprintf("渠道 #%d", segment.ChannelId)
	}
	if wantTimeseries {
		timeKey := fmt.Sprintf("%s:%d", batch.Id, bucketTimestamp)
		point := timeBuckets[timeKey]
		if point == nil {
			point = &ProfitBoardTimeseriesPoint{
				BatchId:         batch.Id,
				BatchName:       batch.Name,
				Bucket:          bucketLabel,
				BucketTimestamp: bucketTimestamp,
			}
			timeBuckets[timeKey] = point
		}
		point.RequestCount += totalUsage.RequestCount
		point.ActualSiteRevenueUSD += actualSiteRevenueUSDTotal
		point.UpstreamCostUSD += upstreamCostUSDKnown
		point.UpstreamCostCNY += upstreamCostCNYKnown
		point.KnownUpstreamCostCount += upstreamKnownCount
		point.MissingUpstreamCostCount += totalUsage.RequestCount - upstreamKnownCount
		point.ActualProfitUSD += actualProfitUSD
		if sitePricingKnown {
			point.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSDTotal
			point.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNYTotal
			point.ConfiguredProfitUSD += configuredProfitUSD
			point.ConfiguredProfitCNY += configuredProfitCNY
		}
		if sitePricingKnown && strings.HasPrefix(sitePricingSource, "site_model_") {
			point.SiteModelMatchCount += totalUsage.RequestCount
		}
		if !sitePricingKnown {
			point.MissingSitePricingCount += totalUsage.RequestCount
		}
	}
	if wantChannelBreakdown {
		channelKey := batch.Id + "|" + strconv.Itoa(segment.ChannelId)
		item := channelBreakdown[channelKey]
		if item == nil {
			item = &ProfitBoardBreakdownItem{BatchId: batch.Id, BatchName: batch.Name, Key: strconv.Itoa(segment.ChannelId), Label: channelLabel}
			channelBreakdown[channelKey] = item
		}
		item.RequestCount += totalUsage.RequestCount
		item.ActualSiteRevenueUSD += actualSiteRevenueUSDTotal
		item.UpstreamCostUSD += upstreamCostUSDKnown
		item.UpstreamCostCNY += upstreamCostCNYKnown
		item.KnownUpstreamCostCount += upstreamKnownCount
		item.MissingUpstreamCostCount += totalUsage.RequestCount - upstreamKnownCount
		item.ActualProfitUSD += actualProfitUSD
		if sitePricingKnown {
			item.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSDTotal
			item.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNYTotal
			item.ConfiguredProfitUSD += configuredProfitUSD
			item.ConfiguredProfitCNY += configuredProfitCNY
		}
	}
	if wantModelBreakdown {
		modelKey := batch.Id + "|" + segment.ModelName
		item := modelBreakdown[modelKey]
		if item == nil {
			item = &ProfitBoardBreakdownItem{BatchId: batch.Id, BatchName: batch.Name, Key: segment.ModelName, Label: segment.ModelName}
			modelBreakdown[modelKey] = item
		}
		item.RequestCount += totalUsage.RequestCount
		item.ActualSiteRevenueUSD += actualSiteRevenueUSDTotal
		item.UpstreamCostUSD += upstreamCostUSDKnown
		item.UpstreamCostCNY += upstreamCostCNYKnown
		item.KnownUpstreamCostCount += upstreamKnownCount
		item.MissingUpstreamCostCount += totalUsage.RequestCount - upstreamKnownCount
		item.ActualProfitUSD += actualProfitUSD
		if sitePricingKnown {
			item.ConfiguredSiteRevenueUSD += configuredSiteRevenueUSDTotal
			item.ConfiguredSiteRevenueCNY += configuredSiteRevenueCNYTotal
			item.ConfiguredProfitUSD += configuredProfitUSD
			item.ConfiguredProfitCNY += configuredProfitCNY
		}
	}
	return nil
}

func generateProfitBoardSummaryReport(query ProfitBoardQuery) (*ProfitBoardReport, error) {
	return generateProfitBoardSummaryReportInternal(query, false)
}

func generateProfitBoardSummaryReportInternal(query ProfitBoardQuery, cumulativeOverview bool) (*ProfitBoardReport, error) {
	normalizedQuery, signature, err := normalizeProfitBoardQuery(query)
	if err != nil {
		return nil, err
	}
	if cumulativeOverview && query.StartTimestamp <= 0 {
		// Cumulative overview is explicitly all-time; do not let query normalization
		// silently rewrite the range to the default recent window.
		normalizedQuery.StartTimestamp = 0
	}
	resolvedBatches, resolvedBatchWarnings, err := resolveProfitBoardBatches(normalizedQuery.Batches)
	if err != nil {
		return nil, err
	}
	resolvedBatchFingerprint := buildProfitBoardResolvedBatchFingerprint(resolvedBatches, resolvedBatchWarnings)
	if cacheKey := buildProfitBoardReportCacheKey(normalizedQuery, resolvedBatchFingerprint); cacheKey != "" {
		if cached, found, cacheErr := getProfitBoardReportCache().Get(cacheKey); cacheErr == nil && found {
			rebucketProfitBoardTimeseries(&cached, normalizedQuery.Granularity, normalizedQuery.CustomIntervalMinutes)
			return &cached, nil
		}
	}

	pricingMap := make(map[string]Pricing)
	for _, pricing := range GetPricing() {
		pricingMap[pricing.ModelName] = pricing
	}
	groupRatios := ratio_setting.GetGroupRatioCopy()
	comboPricingMap := resolveProfitBoardComboPricingMap(normalizedQuery, resolvedBatches)
	excludedUserSet := profitBoardExcludedUserSet(normalizedQuery.ExcludedUserIDs)
	siteUseRechargePrice, sitePriceFactor, sitePriceFactorNote := profitBoardSharedSiteMeta(comboPricingMap)
	sectionSet := profitBoardSectionSet(normalizedQuery.Sections)
	_, wantTimeseries := sectionSet[profitBoardSectionTimeseries]
	_, wantChannelBreakdown := sectionSet[profitBoardSectionChannelBreakdown]
	_, wantModelBreakdown := sectionSet[profitBoardSectionModelBreakdown]
	_, wantWarningItems := sectionSet[profitBoardSectionWarningItems]

	report := &ProfitBoardReport{
		Signature:      signature,
		Batches:        resolvedBatches,
		BatchSummaries: make([]ProfitBoardBatchSummary, 0, len(resolvedBatches)),
		Warnings:       append([]string(nil), resolvedBatchWarnings...),
		Meta: ProfitBoardMeta{
			SiteUseRechargePrice:      siteUseRechargePrice,
			SitePriceFactor:           roundProfitBoardAmount(sitePriceFactor),
			SitePriceFactorNote:       sitePriceFactorNote,
			GeneratedAt:               common.GetTimestamp(),
			LoadedSections:            append([]string(nil), normalizedQuery.Sections...),
			FixedTotalAmountScope:     "created_at_once",
			FixedAmountAllocationMode: "request_count",
			LegacyUpstreamFixedAmount: profitBoardLegacyFixedAmountEnabled(normalizedQuery.Upstream),
			LegacySiteFixedAmount:     profitBoardLegacyFixedAmountEnabled(normalizedQuery.Site),
		},
	}
	timeBuckets := make(map[string]*ProfitBoardTimeseriesPoint)
	channelBreakdown := make(map[string]*ProfitBoardBreakdownItem)
	modelBreakdown := make(map[string]*ProfitBoardBreakdownItem)
	batchSummaryMap := make(map[string]*ProfitBoardBatchSummary, len(resolvedBatches))
	channelNameMap, channelTagMap := profitBoardResolvedChannelMaps(resolvedBatches)
	warningAccumulator := newProfitBoardWarningAccumulator()
	siteRevenueAllocation := newProfitBoardSiteRevenueAllocation()
	accountWalletCombos := profitBoardWalletObserverCombosByAccount(comboPricingMap)
	accountWalletAggregates := make(map[int]*profitBoardUpstreamAccountObservedAggregate, len(accountWalletCombos))
	for _, batch := range resolvedBatches {
		batchSummaryMap[batch.Id] = &ProfitBoardBatchSummary{BatchId: batch.Id, BatchName: batch.Name}
	}
	for accountID := range accountWalletCombos {
		aggregate, aggregateErr := collectProfitBoardUpstreamAccountObservedAggregate(
			accountID,
			normalizedQuery.StartTimestamp,
			normalizedQuery.EndTimestamp,
			normalizedQuery.Granularity,
			normalizedQuery.CustomIntervalMinutes,
			false,
		)
		if aggregateErr != nil {
			return nil, aggregateErr
		}
		accountWalletAggregates[accountID] = aggregate
	}
	remoteAggregate, remoteErr := collectProfitBoardRemoteObserverAggregate(
		signature,
		resolvedBatches,
		normalizedQuery.ComboConfigs,
		normalizedQuery.StartTimestamp,
		normalizedQuery.EndTimestamp,
		normalizedQuery.Granularity,
		normalizedQuery.CustomIntervalMinutes,
		false,
		wantTimeseries,
	)
	if remoteErr != nil {
		return nil, remoteErr
	}
	report.RemoteObserverStates = remoteAggregate.States
	report.Summary.RemoteObservedCostUSD += remoteAggregate.TotalCostUSD
	report.Warnings = append(report.Warnings, remoteAggregate.Warnings...)
	for batchID, observedCostUSD := range remoteAggregate.BatchCostUSD {
		if summary := batchSummaryMap[batchID]; summary != nil {
			summary.RemoteObservedCostUSD += observedCostUSD
		}
	}
	if wantTimeseries {
		for _, remotePoint := range remoteAggregate.Timeseries {
			timeKey := fmt.Sprintf("%s:%d", remotePoint.BatchId, remotePoint.BucketTimestamp)
			point := timeBuckets[timeKey]
			if point == nil {
				current := remotePoint
				timeBuckets[timeKey] = &current
				continue
			}
			point.RemoteObservedCostUSD += remotePoint.RemoteObservedCostUSD
		}
	}

	batchByChannelID := make(map[int]ProfitBoardBatchInfo)
	for _, batch := range resolvedBatches {
		for _, channelID := range batch.ChannelIDs {
			batchByChannelID[channelID] = batch
		}
	}
	err = iterateProfitBoardAggregateSegments(normalizedQuery, resolvedBatches, func(segment profitBoardUsageSegment) error {
		batch, ok := batchByChannelID[segment.ChannelId]
		if !ok {
			return nil
		}
		return accumulateProfitBoardSegment(
			report,
			segment,
			batch,
			comboPricingMap[batch.Id],
			excludedUserSet,
			pricingMap,
			groupRatios,
			batchSummaryMap,
			timeBuckets,
			channelBreakdown,
			modelBreakdown,
			channelNameMap,
			channelTagMap,
			warningAccumulator,
			siteRevenueAllocation,
			wantTimeseries,
			wantChannelBreakdown,
			wantModelBreakdown,
			wantWarningItems,
			normalizedQuery.Granularity,
			normalizedQuery.CustomIntervalMinutes,
		)
	})
	if err != nil {
		return nil, err
	}
	if report.Summary.RequestCount > 0 {
		knownOrWalletCount := report.Summary.KnownUpstreamCostCount
		for _, batch := range resolvedBatches {
			if profitBoardComboUsesWalletObserver(comboPricingMap[batch.Id]) {
				knownOrWalletCount += batchSummaryMap[batch.Id].RequestCount
			}
		}
		report.Summary.ConfiguredProfitCoverageRate = float64(knownOrWalletCount) / float64(report.Summary.RequestCount)
	}
	if wantWarningItems && report.Summary.MissingUpstreamCostCount > 0 {
		report.Warnings = append(report.Warnings, "部分日志未命中上游成本配置，已按可用规则回退，仍无法确定的记为未知")
	}
	if wantWarningItems && report.Summary.MissingSitePricingCount > 0 {
		report.Warnings = append(report.Warnings, "部分日志没有命中本站模型定价，已按手动价格或零值处理")
	}
	if wantWarningItems && report.Meta.LegacyUpstreamFixedAmount {
		report.Warnings = append(report.Warnings, "当前上游价格配置仍包含旧版按次固定金额，请确认后改成固定总金额")
	}
	if wantWarningItems && report.Meta.LegacySiteFixedAmount {
		report.Warnings = append(report.Warnings, "当前本站价格配置仍包含旧版按次固定金额，请确认后改成固定总金额")
	}

	report.BatchSummaries = make([]ProfitBoardBatchSummary, 0, len(batchSummaryMap))
	for _, batch := range resolvedBatches {
		current := *batchSummaryMap[batch.Id]
		if current.RequestCount > 0 {
			if profitBoardComboUsesWalletObserver(comboPricingMap[batch.Id]) {
				current.ConfiguredProfitCoverageRate = 1
			} else {
				current.ConfiguredProfitCoverageRate = float64(current.KnownUpstreamCostCount) / float64(current.RequestCount)
			}
		}
		current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
		current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
		current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
		current.RemoteObservedCostUSD = roundProfitBoardAmount(current.RemoteObservedCostUSD)
		current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
		current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
		current.ConfiguredProfitCoverageRate = roundProfitBoardAmount(current.ConfiguredProfitCoverageRate)
		roundProfitBoardConfiguredMetrics(&current.ProfitBoardSummary)
		report.BatchSummaries = append(report.BatchSummaries, current)
	}
	if wantTimeseries {
		report.Timeseries = make([]ProfitBoardTimeseriesPoint, 0, len(timeBuckets))
		for _, point := range timeBuckets {
			current := *point
			current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
			current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
			current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
			current.RemoteObservedCostUSD = roundProfitBoardAmount(current.RemoteObservedCostUSD)
			current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
			current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
			roundProfitBoardConfiguredTimeseriesMetrics(&current)
			report.Timeseries = append(report.Timeseries, current)
		}
		sort.Slice(report.Timeseries, func(i, j int) bool {
			if report.Timeseries[i].BucketTimestamp == report.Timeseries[j].BucketTimestamp {
				return report.Timeseries[i].BatchName < report.Timeseries[j].BatchName
			}
			return report.Timeseries[i].BucketTimestamp < report.Timeseries[j].BucketTimestamp
		})
	}
	if wantChannelBreakdown {
		report.ChannelBreakdown = make([]ProfitBoardBreakdownItem, 0, len(channelBreakdown))
		for _, item := range channelBreakdown {
			current := *item
			current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
			current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
			current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
			current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
			current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
			roundProfitBoardConfiguredBreakdownMetrics(&current)
			report.ChannelBreakdown = append(report.ChannelBreakdown, current)
		}
		sort.Slice(report.ChannelBreakdown, func(i, j int) bool {
			if report.ChannelBreakdown[i].BatchName != report.ChannelBreakdown[j].BatchName {
				return report.ChannelBreakdown[i].BatchName < report.ChannelBreakdown[j].BatchName
			}
			if report.ChannelBreakdown[i].ConfiguredProfitUSD == report.ChannelBreakdown[j].ConfiguredProfitUSD {
				return report.ChannelBreakdown[i].ActualSiteRevenueUSD > report.ChannelBreakdown[j].ActualSiteRevenueUSD
			}
			return report.ChannelBreakdown[i].ConfiguredProfitUSD > report.ChannelBreakdown[j].ConfiguredProfitUSD
		})
	}
	if wantModelBreakdown {
		report.ModelBreakdown = make([]ProfitBoardBreakdownItem, 0, len(modelBreakdown))
		for _, item := range modelBreakdown {
			current := *item
			current.ActualSiteRevenueUSD = roundProfitBoardAmount(current.ActualSiteRevenueUSD)
			current.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(current.ConfiguredSiteRevenueUSD)
			current.UpstreamCostUSD = roundProfitBoardAmount(current.UpstreamCostUSD)
			current.ConfiguredProfitUSD = roundProfitBoardAmount(current.ConfiguredProfitUSD)
			current.ActualProfitUSD = roundProfitBoardAmount(current.ActualProfitUSD)
			roundProfitBoardConfiguredBreakdownMetrics(&current)
			report.ModelBreakdown = append(report.ModelBreakdown, current)
		}
		sort.Slice(report.ModelBreakdown, func(i, j int) bool {
			if report.ModelBreakdown[i].BatchName != report.ModelBreakdown[j].BatchName {
				return report.ModelBreakdown[i].BatchName < report.ModelBreakdown[j].BatchName
			}
			if report.ModelBreakdown[i].ConfiguredProfitUSD == report.ModelBreakdown[j].ConfiguredProfitUSD {
				return report.ModelBreakdown[i].RequestCount > report.ModelBreakdown[j].RequestCount
			}
			return report.ModelBreakdown[i].ConfiguredProfitUSD > report.ModelBreakdown[j].ConfiguredProfitUSD
		})
	}
	applyProfitBoardComboFixedTotals(
		report,
		comboPricingMap,
		siteRevenueAllocation,
		resolvedBatches,
		normalizedQuery.StartTimestamp,
		normalizedQuery.EndTimestamp,
		normalizedQuery.Granularity,
		normalizedQuery.CustomIntervalMinutes,
	)
	for accountID, comboIDs := range accountWalletCombos {
		applyProfitBoardObservedWalletCost(
			report,
			accountWalletAggregates[accountID],
			comboPricingMap,
			resolvedBatches,
			comboIDs,
			normalizedQuery.Granularity,
			normalizedQuery.CustomIntervalMinutes,
			cumulativeOverview,
		)
	}
	for index := range report.BatchSummaries {
		report.BatchSummaries[index].ActualSiteRevenueUSD = roundProfitBoardAmount(report.BatchSummaries[index].ActualSiteRevenueUSD)
		report.BatchSummaries[index].ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.BatchSummaries[index].ConfiguredSiteRevenueUSD)
		report.BatchSummaries[index].UpstreamCostUSD = roundProfitBoardAmount(report.BatchSummaries[index].UpstreamCostUSD)
		report.BatchSummaries[index].RemoteObservedCostUSD = roundProfitBoardAmount(report.BatchSummaries[index].RemoteObservedCostUSD)
		report.BatchSummaries[index].ConfiguredProfitUSD = roundProfitBoardAmount(report.BatchSummaries[index].ConfiguredProfitUSD)
		report.BatchSummaries[index].ActualProfitUSD = roundProfitBoardAmount(report.BatchSummaries[index].ActualProfitUSD)
		roundProfitBoardConfiguredMetrics(&report.BatchSummaries[index].ProfitBoardSummary)
	}
	report.Summary.ActualSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ActualSiteRevenueUSD)
	report.Summary.ConfiguredSiteRevenueUSD = roundProfitBoardAmount(report.Summary.ConfiguredSiteRevenueUSD)
	report.Summary.UpstreamCostUSD = roundProfitBoardAmount(report.Summary.UpstreamCostUSD)
	report.Summary.RemoteObservedCostUSD = roundProfitBoardAmount(report.Summary.RemoteObservedCostUSD)
	report.Summary.ConfiguredProfitUSD = roundProfitBoardAmount(report.Summary.ConfiguredProfitUSD)
	report.Summary.ActualProfitUSD = roundProfitBoardAmount(report.Summary.ActualProfitUSD)
	report.Summary.ConfiguredProfitCoverageRate = roundProfitBoardAmount(report.Summary.ConfiguredProfitCoverageRate)
	roundProfitBoardConfiguredMetrics(&report.Summary)
	walletSnapshotWatermark, watermarkErr := buildProfitBoardWalletSnapshotWatermark(comboPricingMap)
	if watermarkErr != nil {
		return nil, watermarkErr
	}
	report.Meta.ActivityWatermark = buildProfitBoardCombinedActivityWatermark(
		report.Summary.RequestCount,
		report.Meta.LatestLogId,
		report.Meta.LatestLogCreatedAt,
		walletSnapshotWatermark+":"+buildProfitBoardAggregateActivityWatermark(),
	)
	if wantWarningItems {
		report.WarningItems = warningAccumulator.items(map[string]string{
			"missing_upstream_cost": "部分日志未命中上游成本配置，已按可用规则回退，仍无法确定的记为未知",
			"missing_site_pricing":  "部分日志没有命中本站模型定价，已按手动价格或零值处理",
		}, profitBoardWarningReasonLabels())
		report.Warnings = uniqueProfitBoardWarnings(report.Warnings)
	}
	if cacheKey := buildProfitBoardReportCacheKey(normalizedQuery, resolvedBatchFingerprint); cacheKey != "" {
		_ = getProfitBoardReportCache().SetWithTTL(cacheKey, *report, profitBoardReportCacheTTL())
	}
	return report, nil
}

func generateProfitBoardOverviewSummary(payload ProfitBoardConfigPayload) (*ProfitBoardReport, error) {
	normalizedBatches, signature, _, err := normalizeProfitBoardBatches(payload.Batches, payload.Selection)
	if err != nil {
		return nil, err
	}
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
	payload.SharedSite = normalizeProfitBoardSharedSiteConfig(payload.SharedSite, payload.Site)
	payload.ExcludedUserIDs = normalizeProfitBoardExcludedUserIDs(payload.ExcludedUserIDs)
	payload.ComboConfigs = normalizeProfitBoardComboConfigs(normalizedBatches, payload.ComboConfigs, payload.SharedSite, payload.Site, payload.Upstream)
	if err = validateProfitBoardPricingConfig(payload.Upstream, false); err != nil {
		return nil, err
	}
	if err = validateProfitBoardPricingConfig(payload.Site, true); err != nil {
		return nil, err
	}
	if err = validateProfitBoardComboConfigs(payload.ComboConfigs); err != nil {
		return nil, err
	}
	resolvedBatches, resolvedBatchWarnings, err := resolveProfitBoardBatches(normalizedBatches)
	if err != nil {
		return nil, err
	}
	resolvedBatchFingerprint := buildProfitBoardResolvedBatchFingerprint(resolvedBatches, resolvedBatchWarnings)
	overviewCacheKey := buildProfitBoardOverviewCacheKey(payload, resolvedBatchFingerprint)
	if overviewCacheKey != "" {
		if cached, found, cacheErr := getProfitBoardOverviewCache().Get(overviewCacheKey); cacheErr == nil && found {
			return &cached, nil
		}
	}
	report, err := generateProfitBoardSummaryReportInternal(ProfitBoardQuery{
		Batches:         normalizedBatches,
		SharedSite:      payload.SharedSite,
		ComboConfigs:    payload.ComboConfigs,
		ExcludedUserIDs: payload.ExcludedUserIDs,
		Upstream:        payload.Upstream,
		Site:            payload.Site,
		StartTimestamp:  0,
		EndTimestamp:    common.GetTimestamp(),
		Granularity:     "day",
		Sections: []string{
			profitBoardSectionTimeseries,
			profitBoardSectionChannelBreakdown,
			profitBoardSectionModelBreakdown,
			profitBoardSectionWarningItems,
		},
	}, true)
	if err != nil {
		return nil, err
	}
	report.Signature = signature
	report.Meta.CumulativeScope = "all_time"
	report.Timeseries = nil
	for index := range report.WarningItems {
		switch report.WarningItems[index].Code {
		case "missing_upstream_cost":
			report.WarningItems[index].Message = "累计总览中部分日志未命中上游成本配置，已按可用规则回退，仍无法确定的记为未知"
		case "missing_site_pricing":
			report.WarningItems[index].Message = "累计总览中部分日志没有命中本站模型定价，已按手动价格或零值处理"
		}
	}
	for index := range report.Warnings {
		report.Warnings[index] = strings.Replace(report.Warnings[index], "部分日志", "累计总览中部分日志", 1)
	}
	if overviewCacheKey != "" {
		_ = getProfitBoardOverviewCache().SetWithTTL(overviewCacheKey, *report, profitBoardReportCacheTTL())
		_ = getProfitBoardOverviewStaleCache().SetWithTTL(overviewCacheKey, *report, 10*time.Minute)
	}
	return report, nil
}
