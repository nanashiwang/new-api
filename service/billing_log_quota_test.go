package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func seedPackageToken(t *testing.T, id int, userId int, key string, remainQuota int, packageLimit int) {
	t.Helper()
	token := &model.Token{
		Id:                id,
		UserId:            userId,
		Key:               key,
		Name:              "package_token",
		Status:            common.TokenStatusEnabled,
		RemainQuota:       remainQuota,
		UsedQuota:         0,
		PackageEnabled:    true,
		PackageLimitQuota: packageLimit,
		PackagePeriod:     model.TokenPackagePeriodHourly,
	}
	require.NoError(t, model.DB.Create(token).Error)
}

func TestResolveLoggedQuotaAfterSettle_TokenOnlyPackageOverrun(t *testing.T) {
	truncate(t)
	seedUser(t, 1001, 0)
	seedPackageToken(t, 2001, 1001, "settle_package_token_overrun_test_key_0000001", 100, 10)

	relayInfo := &relaycommon.RelayInfo{
		UserId:                1001,
		TokenId:               2001,
		TokenKey:              "sk-settle_package_token_overrun_test_key_0000001",
		FinalPreConsumedQuota: 8,
		BillingSource:         BillingSourceToken,
	}

	session := &BillingSession{
		relayInfo:        relayInfo,
		funding:          &TokenFunding{},
		preConsumedQuota: 8,
		tokenConsumed:    8,
	}
	relayInfo.Billing = session

	require.NoError(t, model.DecreaseTokenQuota(relayInfo.TokenId, relayInfo.TokenKey, 8))

	err := session.Settle(12)
	require.Error(t, err)

	loggedQuota, _, _ := FinalizeConsumeLogAfterSettle("", nil, 12, relayInfo, err)
	require.Equal(t, 8, loggedQuota)

	token, getErr := model.GetTokenById(2001)
	require.NoError(t, getErr)
	require.Equal(t, 8, token.PackageUsedQuota)
	require.Equal(t, 92, token.RemainQuota)
}
