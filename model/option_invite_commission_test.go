package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionMapValidatesCombinedInviteCommissionRates(t *testing.T) {
	originalOptionMap := common.OptionMap
	originalFirst := common.InviterRechargeCommissionRate
	originalSecond := common.InviterRechargeSecondLevelCommissionRate
	common.OptionMap = map[string]string{}
	t.Cleanup(func() {
		common.OptionMap = originalOptionMap
		common.InviterRechargeCommissionRate = originalFirst
		common.InviterRechargeSecondLevelCommissionRate = originalSecond
	})

	common.InviterRechargeCommissionRate = 0.1
	common.InviterRechargeSecondLevelCommissionRate = 0

	require.NoError(t, updateOptionMap("InviterRechargeSecondLevelCommissionRate", "0.2"))
	require.Equal(t, 0.2, common.InviterRechargeSecondLevelCommissionRate)

	err := updateOptionMap("InviterRechargeCommissionRate", "0.9")
	require.Error(t, err)
	require.Equal(t, 0.1, common.InviterRechargeCommissionRate)
	require.Equal(t, 0.2, common.InviterRechargeSecondLevelCommissionRate)

	err = updateOptionMap("InviterRechargeSecondLevelCommissionRate", "NaN")
	require.Error(t, err)
	require.Equal(t, 0.2, common.InviterRechargeSecondLevelCommissionRate)
}

func TestUpdateOptionMapRejectsNegativeInviteCommissionDailyCap(t *testing.T) {
	originalOptionMap := common.OptionMap
	originalDailyCap := common.InviterCommissionDailyCap
	common.OptionMap = map[string]string{}
	t.Cleanup(func() {
		common.OptionMap = originalOptionMap
		common.InviterCommissionDailyCap = originalDailyCap
	})

	common.InviterCommissionDailyCap = 0
	require.Error(t, updateOptionMap("InviterCommissionDailyCap", "-1"))
	require.Equal(t, 0, common.InviterCommissionDailyCap)
	require.NoError(t, updateOptionMap("InviterCommissionDailyCap", "1000"))
	require.Equal(t, 1000, common.InviterCommissionDailyCap)
}
