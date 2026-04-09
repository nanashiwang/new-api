package model

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

const (
	BatchUpdateTypeUserQuota = iota
	BatchUpdateTypeTokenQuota
	BatchUpdateTypeUsedQuota
	BatchUpdateTypeChannelUsedQuota
	BatchUpdateTypeRequestCount
	BatchUpdateTypeCount // if you add a new type, you need to add a new map and a new lock
)

const batchUpdateChunkSize = 100

var batchUpdateStores []map[int]int
var batchUpdateLocks []sync.Mutex

func init() {
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateStores = append(batchUpdateStores, make(map[int]int))
		batchUpdateLocks = append(batchUpdateLocks, sync.Mutex{})
	}
}

func InitBatchUpdater() {
	gopool.Go(func() {
		for {
			time.Sleep(time.Duration(common.BatchUpdateInterval) * time.Second)
			batchUpdate()
		}
	})
}

func addNewRecord(type_ int, id int, value int) {
	batchUpdateLocks[type_].Lock()
	defer batchUpdateLocks[type_].Unlock()
	if _, ok := batchUpdateStores[type_][id]; !ok {
		batchUpdateStores[type_][id] = value
	} else {
		batchUpdateStores[type_][id] += value
	}
}

func batchUpdate() {
	stores := make([]map[int]int, BatchUpdateTypeCount)
	hasData := false
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateLocks[i].Lock()
		if len(batchUpdateStores[i]) > 0 {
			hasData = true
			stores[i] = batchUpdateStores[i]
			batchUpdateStores[i] = make(map[int]int)
		} else {
			stores[i] = make(map[int]int)
		}
		batchUpdateLocks[i].Unlock()
	}

	if !hasData {
		return
	}

	common.SysLog("batch update started")
	if err := batchUpdateSingleDeltaColumn(&User{}, "quota", stores[BatchUpdateTypeUserQuota]); err != nil {
		common.SysLog("failed to batch update user quota: " + err.Error())
	}
	if err := batchUpdateTokenQuota(stores[BatchUpdateTypeTokenQuota]); err != nil {
		common.SysLog("failed to batch update token quota: " + err.Error())
	}
	if err := batchUpdateUserUsageAndRequests(stores[BatchUpdateTypeUsedQuota], stores[BatchUpdateTypeRequestCount]); err != nil {
		common.SysLog("failed to batch update user usage stats: " + err.Error())
	}
	if err := batchUpdateSingleDeltaColumn(&Channel{}, "used_quota", stores[BatchUpdateTypeChannelUsedQuota]); err != nil {
		common.SysLog("failed to batch update channel used quota: " + err.Error())
	}
	common.SysLog("batch update finished")
}

func batchUpdateSingleDeltaColumn(model interface{}, column string, store map[int]int) error {
	ids := sortedBatchUpdateIDs(store)
	for _, chunk := range chunkIntSlice(ids, batchUpdateChunkSize) {
		chunkStore := filterBatchUpdateStore(store, chunk)
		expr, args, _ := buildBatchDeltaCaseExpr("id", column, chunkStore)
		if err := DB.Model(model).
			Where("id IN ?", chunk).
			Update(column, gorm.Expr(expr, args...)).Error; err != nil {
			return err
		}
	}
	return nil
}

func batchUpdateTokenQuota(store map[int]int) error {
	ids := sortedBatchUpdateIDs(store)
	now := common.GetTimestamp()
	for _, chunk := range chunkIntSlice(ids, batchUpdateChunkSize) {
		chunkStore := filterBatchUpdateStore(store, chunk)
		remainExpr, remainArgs, _ := buildBatchDeltaCaseExpr("id", "remain_quota", chunkStore)
		usedExpr, usedArgs, _ := buildBatchDeltaCaseExpr("id", "used_quota", negateBatchUpdateStore(chunkStore))
		if err := DB.Model(&Token{}).
			Where("id IN ?", chunk).
			Updates(map[string]interface{}{
				"remain_quota":  gorm.Expr(remainExpr, remainArgs...),
				"used_quota":    gorm.Expr(usedExpr, usedArgs...),
				"accessed_time": now,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func batchUpdateUserUsageAndRequests(usedQuotaStore map[int]int, requestCountStore map[int]int) error {
	ids := unionBatchUpdateIDs(usedQuotaStore, requestCountStore)
	for _, chunk := range chunkIntSlice(ids, batchUpdateChunkSize) {
		updates := make(map[string]interface{}, 2)
		if usedQuotaChunk := filterBatchUpdateStore(usedQuotaStore, chunk); len(usedQuotaChunk) > 0 {
			expr, args, _ := buildBatchDeltaCaseExpr("id", "used_quota", usedQuotaChunk)
			updates["used_quota"] = gorm.Expr(expr, args...)
		}
		if requestCountChunk := filterBatchUpdateStore(requestCountStore, chunk); len(requestCountChunk) > 0 {
			expr, args, _ := buildBatchDeltaCaseExpr("id", "request_count", requestCountChunk)
			updates["request_count"] = gorm.Expr(expr, args...)
		}
		if len(updates) == 0 {
			continue
		}
		if err := DB.Model(&User{}).Where("id IN ?", chunk).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func sortedBatchUpdateIDs(store map[int]int) []int {
	if len(store) == 0 {
		return nil
	}
	ids := make([]int, 0, len(store))
	for id, value := range store {
		if value == 0 {
			continue
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func unionBatchUpdateIDs(stores ...map[int]int) []int {
	seen := make(map[int]struct{})
	for _, store := range stores {
		for id, value := range store {
			if value == 0 {
				continue
			}
			seen[id] = struct{}{}
		}
	}
	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func chunkIntSlice(values []int, size int) [][]int {
	if len(values) == 0 {
		return nil
	}
	chunks := make([][]int, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func filterBatchUpdateStore(store map[int]int, ids []int) map[int]int {
	filtered := make(map[int]int, len(ids))
	for _, id := range ids {
		if value, ok := store[id]; ok && value != 0 {
			filtered[id] = value
		}
	}
	return filtered
}

func negateBatchUpdateStore(store map[int]int) map[int]int {
	negated := make(map[int]int, len(store))
	for id, value := range store {
		if value == 0 {
			continue
		}
		negated[id] = -value
	}
	return negated
}

func buildBatchDeltaCaseExpr(idColumn string, column string, store map[int]int) (string, []interface{}, []int) {
	ids := sortedBatchUpdateIDs(store)
	if len(ids) == 0 {
		return column, nil, nil
	}
	var builder strings.Builder
	builder.WriteString("CASE ")
	builder.WriteString(idColumn)
	args := make([]interface{}, 0, len(ids)*2)
	for _, id := range ids {
		builder.WriteString(" WHEN ? THEN ")
		builder.WriteString(column)
		builder.WriteString(" + ?")
		args = append(args, id, store[id])
	}
	builder.WriteString(" ELSE ")
	builder.WriteString(column)
	builder.WriteString(" END")
	return builder.String(), args, ids
}

func buildBatchStringCaseExpr(idColumn string, column string, values map[int]string) (string, []interface{}, []int) {
	if len(values) == 0 {
		return column, nil, nil
	}
	ids := make([]int, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var builder strings.Builder
	builder.WriteString("CASE ")
	builder.WriteString(idColumn)
	args := make([]interface{}, 0, len(ids)*2)
	for _, id := range ids {
		builder.WriteString(" WHEN ? THEN ?")
		args = append(args, id, values[id])
	}
	builder.WriteString(" ELSE ")
	builder.WriteString(column)
	builder.WriteString(" END")
	return builder.String(), args, ids
}

func RecordExist(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func shouldUpdateRedis(fromDB bool, err error) bool {
	return common.RedisEnabled && fromDB && err == nil
}
