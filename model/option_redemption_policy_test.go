package model

import (
	"strconv"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpdateOptionMapValidatesRedemptionPolicyLimits(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	originalCreate := common.WalletRedemptionDailyCreateLimit
	originalMinimum := common.WalletRedemptionMinimumQuota
	originalActive := common.WalletRedemptionActiveLimit
	originalQuota := common.WalletRedemptionDailyQuotaLimit
	originalCreators := common.WalletRedemptionReviewDistinctCreatorThreshold
	originalSmall := common.WalletRedemptionReviewSmallQuotaLimit
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
		common.WalletRedemptionDailyCreateLimit = originalCreate
		common.WalletRedemptionMinimumQuota = originalMinimum
		common.WalletRedemptionActiveLimit = originalActive
		common.WalletRedemptionDailyQuotaLimit = originalQuota
		common.WalletRedemptionReviewDistinctCreatorThreshold = originalCreators
		common.WalletRedemptionReviewSmallQuotaLimit = originalSmall
	})

	require.Error(t, updateOptionMap("WalletRedemptionDailyCreateLimit", "-1"))
	require.NoError(t, updateOptionMap("WalletRedemptionDailyCreateLimit", "100"))
	require.NoError(t, updateOptionMap("WalletRedemptionMinimumQuota", "10"))
	require.NoError(t, updateOptionMap("WalletRedemptionActiveLimit", "100"))
	require.NoError(t, updateOptionMap("WalletRedemptionDailyQuotaLimit", "5000"))
	require.NoError(t, updateOptionMap("WalletRedemptionReviewDistinctCreatorThreshold", "3"))
	require.NoError(t, updateOptionMap("WalletRedemptionReviewSmallQuotaLimit", "100"))
	require.Equal(t, 100, common.WalletRedemptionDailyCreateLimit)
	require.Equal(t, 10, common.WalletRedemptionMinimumQuota)
	require.Equal(t, 100, common.WalletRedemptionActiveLimit)
	require.Equal(t, 5000, common.WalletRedemptionDailyQuotaLimit)
	require.Equal(t, 3, common.WalletRedemptionReviewDistinctCreatorThreshold)
	require.Equal(t, 100, common.WalletRedemptionReviewSmallQuotaLimit)
}

func TestConcurrentWalletRedemptionPolicyUpdatesKeepDatabaseAndRuntimeAligned(t *testing.T) {
	originalDB := DB
	originalPolicy := currentWalletRedemptionPolicy()
	originalOptionMap := common.OptionMap
	t.Cleanup(func() {
		DB = originalDB
		publishWalletRedemptionPolicy(originalPolicy)
		common.OptionMap = originalOptionMap
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&Option{}))
	DB = db
	common.OptionMap = make(map[string]string)

	raws := []string{
		`{"daily_create_limit":80,"minimum_quota":12,"active_limit":60,"daily_quota_limit":4000,"review_distinct_creator_threshold":4,"review_small_quota_limit":120}`,
		`{"daily_create_limit":90,"minimum_quota":15,"active_limit":70,"daily_quota_limit":4500,"review_distinct_creator_threshold":5,"review_small_quota_limit":150}`,
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(raws))
	for _, raw := range raws {
		wg.Add(1)
		go func(raw string) {
			defer wg.Done()
			errs <- UpdateWalletRedemptionPolicyOptions(raw)
		}(raw)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	policy := currentWalletRedemptionPolicy()
	values := map[string]string{}
	var options []Option
	require.NoError(t, DB.Find(&options).Error)
	for _, option := range options {
		values[option.Key] = option.Value
	}
	require.Equal(t, strconv.Itoa(policy.DailyCreateLimit), values["WalletRedemptionDailyCreateLimit"])
	require.Equal(t, strconv.Itoa(policy.MinimumQuota), values["WalletRedemptionMinimumQuota"])
	require.Equal(t, strconv.Itoa(policy.ActiveLimit), values["WalletRedemptionActiveLimit"])
	require.Equal(t, strconv.Itoa(policy.DailyQuotaLimit), values["WalletRedemptionDailyQuotaLimit"])
}

func TestUpdateWalletRedemptionPolicyOptionsPublishesAtomically(t *testing.T) {
	originalDB := DB
	originalPolicy := currentWalletRedemptionPolicy()
	originalOptionMap := common.OptionMap
	t.Cleanup(func() {
		DB = originalDB
		publishWalletRedemptionPolicy(originalPolicy)
		common.OptionMap = originalOptionMap
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))
	DB = db
	common.OptionMap = make(map[string]string)

	raw := `{"daily_create_limit":80,"minimum_quota":12,"active_limit":60,"daily_quota_limit":4000,"review_distinct_creator_threshold":4,"review_small_quota_limit":120}`
	require.NoError(t, UpdateWalletRedemptionPolicyOptions(raw))
	require.Equal(t, 80, common.WalletRedemptionDailyCreateLimit)
	require.Equal(t, 12, common.WalletRedemptionMinimumQuota)
	require.Equal(t, 60, common.WalletRedemptionActiveLimit)
	require.Equal(t, 4000, common.WalletRedemptionDailyQuotaLimit)
	require.Equal(t, 4, common.WalletRedemptionReviewDistinctCreatorThreshold)
	require.Equal(t, 120, common.WalletRedemptionReviewSmallQuotaLimit)
	var count int64
	require.NoError(t, DB.Model(&Option{}).Where(commonKeyCol+" IN ?", []string{
		"WalletRedemptionDailyCreateLimit", "WalletRedemptionMinimumQuota", "WalletRedemptionActiveLimit",
		"WalletRedemptionDailyQuotaLimit", "WalletRedemptionReviewDistinctCreatorThreshold", "WalletRedemptionReviewSmallQuotaLimit",
	}).Count(&count).Error)
	require.EqualValues(t, 6, count)

	err = UpdateWalletRedemptionPolicyOptions(`{"daily_create_limit":80,"minimum_quota":120,"active_limit":60,"daily_quota_limit":4000,"review_distinct_creator_threshold":4,"review_small_quota_limit":100}`)
	require.Error(t, err)
	require.Equal(t, 12, common.WalletRedemptionMinimumQuota)
}
