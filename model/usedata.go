package model

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
	"gorm.io/gorm"
)

// QuotaData 柱状图数据
type QuotaData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"index;index:idx_qdt_user_created,priority:1"`
	Username  string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;index:idx_qdt_username_created,priority:1;index:idx_qdt_created_username,priority:2;size:64;default:''"`
	ModelName string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;index:idx_qdt_created_model,priority:2;size:64;default:''"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2;index:idx_qdt_user_created,priority:2;index:idx_qdt_username_created,priority:2;index:idx_qdt_created_model,priority:1;index:idx_qdt_created_username,priority:1"`
	TokenUsed int    `json:"token_used" gorm:"default:0"`
	Count     int    `json:"count" gorm:"default:0"`
	Quota     int    `json:"quota" gorm:"default:0"`
}

func UpdateQuotaData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheQuotaData = make(map[string]*QuotaData)
var CacheQuotaDataLock = sync.Mutex{}

var (
	quotaDataQueryCache     *cachex.HybridCache[[]*QuotaData]
	quotaDataQueryCacheOnce sync.Once
)

func quotaDataQueryCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("DATA_DASHBOARD_QUERY_CACHE_TTL", 30)
	if ttlSeconds <= 0 {
		ttlSeconds = 30
	}
	return time.Duration(ttlSeconds) * time.Second
}

func quotaDataQueryCacheCapacity() int {
	capacity := common.GetEnvOrDefault("DATA_DASHBOARD_QUERY_CACHE_CAP", 256)
	if capacity <= 0 {
		capacity = 256
	}
	return capacity
}

func getQuotaDataQueryCache() *cachex.HybridCache[[]*QuotaData] {
	quotaDataQueryCacheOnce.Do(func() {
		ttl := quotaDataQueryCacheTTL()
		quotaDataQueryCache = cachex.NewHybridCache[[]*QuotaData](cachex.HybridCacheConfig[[]*QuotaData]{
			Namespace: cachex.Namespace("quota_data_query:v1"),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[[]*QuotaData]{},
			Memory: func() *hot.HotCache[string, []*QuotaData] {
				return hot.NewHotCache[string, []*QuotaData](hot.LRU, quotaDataQueryCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return quotaDataQueryCache
}

func quotaDataEffectiveRange(startTime int64, endTime int64) (int64, int64) {
	if startTime%3600 != 0 {
		startTime = startTime + (3600 - startTime%3600)
	}
	endTime = endTime - (endTime % 3600)
	return startTime, endTime
}

func quotaDataCacheKey(scope string, startTime int64, endTime int64, parts ...string) string {
	effectiveStart, effectiveEnd := quotaDataEffectiveRange(startTime, endTime)
	return fmt.Sprintf("%s:%d:%d:%s", scope, effectiveStart, effectiveEnd, strings.Join(parts, ":"))
}

func getCachedQuotaData(key string, loader func() ([]*QuotaData, error)) ([]*QuotaData, error) {
	cache := getQuotaDataQueryCache()
	if cached, found, err := cache.Get(key); err == nil && found {
		return cached, nil
	}
	quotaData, err := loader()
	if err != nil {
		return quotaData, err
	}
	_ = cache.SetWithTTL(key, quotaData, quotaDataQueryCacheTTL())
	return quotaData, nil
}

func logQuotaDataCache(userId int, username string, modelName string, quota int, createdAt int64, tokenUsed int) {
	key := fmt.Sprintf("%d-%s-%s-%d", userId, username, modelName, createdAt)
	quotaData, ok := CacheQuotaData[key]
	if ok {
		quotaData.Count += 1
		quotaData.Quota += quota
		quotaData.TokenUsed += tokenUsed
	} else {
		quotaData = &QuotaData{
			UserID:    userId,
			Username:  username,
			ModelName: modelName,
			CreatedAt: createdAt,
			Count:     1,
			Quota:     quota,
			TokenUsed: tokenUsed,
		}
	}
	CacheQuotaData[key] = quotaData
}

func LogQuotaData(userId int, username string, modelName string, quota int, createdAt int64, tokenUsed int) {
	// 只精确到小时
	createdAt = createdAt - (createdAt % 3600)

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	logQuotaDataCache(userId, username, modelName, quota, createdAt, tokenUsed)
}

func SaveQuotaDataCache() {
	CacheQuotaDataLock.Lock()
	pendingQuotaData := CacheQuotaData
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	size := len(pendingQuotaData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, quotaData := range pendingQuotaData {
		quotaDataDB := &QuotaData{}
		DB.Table("quota_data").Where("user_id = ? and username = ? and model_name = ? and created_at = ?",
			quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt).First(quotaDataDB)
		if quotaDataDB.Id > 0 {
			//quotaDataDB.Count += quotaData.Count
			//quotaDataDB.Quota += quotaData.Quota
			//DB.Table("quota_data").Save(quotaDataDB)
			increaseQuotaData(quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.Count, quotaData.Quota, quotaData.CreatedAt, quotaData.TokenUsed)
		} else {
			DB.Table("quota_data").Create(quotaData)
		}
	}
	_ = getQuotaDataQueryCache().Purge()
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseQuotaData(userId int, username string, modelName string, count int, quota int, createdAt int64, tokenUsed int) {
	err := DB.Table("quota_data").Where("user_id = ? and username = ? and model_name = ? and created_at = ?",
		userId, username, modelName, createdAt).Updates(map[string]interface{}{
		"count":      gorm.Expr("count + ?", count),
		"quota":      gorm.Expr("quota + ?", quota),
		"token_used": gorm.Expr("token_used + ?", tokenUsed),
	}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseQuotaData error: %s", err))
	}
}

func GetQuotaDataByUsername(username string, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	return getCachedQuotaData(quotaDataCacheKey("username", startTime, endTime, username), func() ([]*QuotaData, error) {
		var quotaDatas []*QuotaData
		// 从quota_data表中查询数据
		err := DB.Table("quota_data").
			Select("model_name, count, quota, token_used, created_at").
			Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).
			Find(&quotaDatas).Error
		return quotaDatas, err
	})
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	return getCachedQuotaData(quotaDataCacheKey("user_id", startTime, endTime, fmt.Sprintf("%d", userId)), func() ([]*QuotaData, error) {
		var quotaDatas []*QuotaData
		// 从quota_data表中查询数据
		err := DB.Table("quota_data").
			Select("model_name, count, quota, token_used, created_at").
			Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).
			Find(&quotaDatas).Error
		return quotaDatas, err
	})
}

func GetQuotaDataGroupByUser(startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	return getCachedQuotaData(quotaDataCacheKey("group_user", startTime, endTime), func() ([]*QuotaData, error) {
		var quotaDatas []*QuotaData
		err := DB.Table("quota_data").
			Select("username, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
			Where("created_at >= ? and created_at <= ?", startTime, endTime).
			Group("username, created_at").
			Find(&quotaDatas).Error
		return quotaDatas, err
	})
}

func GetAllQuotaDates(startTime int64, endTime int64, username string) (quotaData []*QuotaData, err error) {
	if username != "" {
		return GetQuotaDataByUsername(username, startTime, endTime)
	}
	return getCachedQuotaData(quotaDataCacheKey("all_models", startTime, endTime), func() ([]*QuotaData, error) {
		var quotaDatas []*QuotaData
		// 从quota_data表中查询数据
		// only select model_name, sum(count) as count, sum(quota) as quota, model_name, created_at from quota_data group by model_name, created_at;
		//err = DB.Table("quota_data").Where("created_at >= ? and created_at <= ?", startTime, endTime).Find(&quotaDatas).Error
		err := DB.Table("quota_data").Select("model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, created_at").Where("created_at >= ? and created_at <= ?", startTime, endTime).Group("model_name, created_at").Find(&quotaDatas).Error
		return quotaDatas, err
	})
}
