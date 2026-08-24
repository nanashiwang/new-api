package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionMapValidatesRedemptionPolicyLimits(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	originalCreate := common.WalletRedemptionDailyCreateLimit
	originalQuota := common.WalletRedemptionDailyQuotaLimit
	originalCreators := common.WalletRedemptionReviewDistinctCreatorThreshold
	originalSmall := common.WalletRedemptionReviewSmallQuotaLimit
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
		common.WalletRedemptionDailyCreateLimit = originalCreate
		common.WalletRedemptionDailyQuotaLimit = originalQuota
		common.WalletRedemptionReviewDistinctCreatorThreshold = originalCreators
		common.WalletRedemptionReviewSmallQuotaLimit = originalSmall
	})

	require.Error(t, updateOptionMap("WalletRedemptionDailyCreateLimit", "-1"))
	require.NoError(t, updateOptionMap("WalletRedemptionDailyCreateLimit", "100"))
	require.NoError(t, updateOptionMap("WalletRedemptionDailyQuotaLimit", "5000"))
	require.NoError(t, updateOptionMap("WalletRedemptionReviewDistinctCreatorThreshold", "3"))
	require.NoError(t, updateOptionMap("WalletRedemptionReviewSmallQuotaLimit", "100"))
	require.Equal(t, 100, common.WalletRedemptionDailyCreateLimit)
	require.Equal(t, 5000, common.WalletRedemptionDailyQuotaLimit)
	require.Equal(t, 3, common.WalletRedemptionReviewDistinctCreatorThreshold)
	require.Equal(t, 100, common.WalletRedemptionReviewSmallQuotaLimit)
}
