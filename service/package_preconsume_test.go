package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayhelper "github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func seedBillingPackageToken(t *testing.T, id int, userId int, key string, remainQuota int, packageLimit int, packageUsed int) {
	t.Helper()
	token := &model.Token{
		Id:                id,
		UserId:            userId,
		Key:               key,
		Name:              "package_preconsume_token",
		Status:            common.TokenStatusEnabled,
		RemainQuota:       remainQuota,
		UsedQuota:         0,
		PackageEnabled:    true,
		PackageLimitQuota: packageLimit,
		PackagePeriod:     model.TokenPackagePeriodHourly,
		PackageUsedQuota:  packageUsed,
		PackagePeriodMode: model.TokenPackagePeriodModeRelative,
	}
	require.NoError(t, model.DB.Create(token).Error)
}

func newPackageBillingContext(packageEnabled bool) *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyTokenPackageEnabled, packageEnabled)
	common.SetContextKey(ctx, constant.ContextKeyTokenBillingMode, model.TokenBillingModeTokenOnly)
	return ctx
}

func TestPreConsumeBilling_PackageTokenUsesConservativeTextQuota(t *testing.T) {
	truncate(t)

	originPreConsumedQuota := common.PreConsumedQuota
	t.Cleanup(func() {
		common.PreConsumedQuota = originPreConsumedQuota
	})
	common.PreConsumedQuota = 0

	const userID = 3001
	const tokenID = 4001
	const tokenKey = "package_text_preconsume_key"

	seedUser(t, userID, 0)
	seedBillingPackageToken(t, tokenID, userID, tokenKey, 200, 80, 0)

	ctx := newPackageBillingContext(true)
	request := &dto.GeneralOpenAIRequest{Model: "test-text-model", MaxTokens: 20}
	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        tokenKey,
		OriginModelName: "test-text-model",
		Request:         request,
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 4,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	relayInfo.PriceData.ConservativeQuotaToPreConsume = relayhelper.EstimateConservativePreConsumeQuota(relayInfo, 10, &types.TokenCountMeta{MaxTokens: 20}, relayInfo.PriceData)

	apiErr := PreConsumeBilling(ctx, 30, relayInfo)
	require.Error(t, apiErr)
	require.Contains(t, apiErr.Error(), "令牌套餐周期额度不足")

	token, err := model.GetTokenById(tokenID)
	require.NoError(t, err)
	require.Equal(t, 200, token.RemainQuota)
	require.Equal(t, 0, token.PackageUsedQuota)
}

func TestPreConsumeBilling_PackageTokenUsesConservativeImageQuota(t *testing.T) {
	truncate(t)

	originPreConsumedQuota := common.PreConsumedQuota
	t.Cleanup(func() {
		common.PreConsumedQuota = originPreConsumedQuota
	})
	common.PreConsumedQuota = 0

	const userID = 3002
	const tokenID = 4002
	const tokenKey = "package_image_preconsume_key"

	seedUser(t, userID, 0)
	seedBillingPackageToken(t, tokenID, userID, tokenKey, 200, 15, 0)

	ctx := newPackageBillingContext(true)
	request := &dto.ImageRequest{Model: "test-image-model", N: 1}
	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        tokenKey,
		OriginModelName: "test-image-model",
		Request:         request,
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 1,
			ImageRatio:      2,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	relayInfo.PriceData.ConservativeQuotaToPreConsume = relayhelper.EstimateConservativePreConsumeQuota(relayInfo, 10, &types.TokenCountMeta{}, relayInfo.PriceData)

	apiErr := PreConsumeBilling(ctx, 10, relayInfo)
	require.Error(t, apiErr)
	require.Contains(t, apiErr.Error(), "令牌套餐周期额度不足")

	token, err := model.GetTokenById(tokenID)
	require.NoError(t, err)
	require.Equal(t, 200, token.RemainQuota)
	require.Equal(t, 0, token.PackageUsedQuota)
}

func TestPreConsumeBilling_NormalTokenKeepsLegacyQuota(t *testing.T) {
	truncate(t)

	const userID = 3003
	const tokenID = 4003
	const tokenKey = "normal_preconsume_key"

	seedUser(t, userID, 0)
	seedToken(t, tokenID, userID, tokenKey, 100)

	ctx := newPackageBillingContext(false)
	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        tokenKey,
		OriginModelName: "test-normal-model",
		PriceData: types.PriceData{
			ConservativeQuotaToPreConsume: 20,
		},
	}

	apiErr := PreConsumeBilling(ctx, 10, relayInfo)
	require.Nil(t, apiErr)
	require.NotNil(t, relayInfo.Billing)
	require.Equal(t, 10, relayInfo.FinalPreConsumedQuota)

	token, err := model.GetTokenById(tokenID)
	require.NoError(t, err)
	require.Equal(t, 90, token.RemainQuota)
	require.Equal(t, 10, token.UsedQuota)
}

func TestPreConsumeBilling_PackageTokenWithoutMaxTokensUsesDefaultCompletionReserve(t *testing.T) {
	truncate(t)

	originPreConsumedQuota := common.PreConsumedQuota
	t.Cleanup(func() {
		common.PreConsumedQuota = originPreConsumedQuota
	})
	common.PreConsumedQuota = 0

	const userID = 3004
	const tokenID = 4004
	const tokenKey = "package_text_default_completion_key"

	seedUser(t, userID, 0)
	seedBillingPackageToken(t, tokenID, userID, tokenKey, 20000, 5000, 0)

	ctx := newPackageBillingContext(true)
	request := &dto.GeneralOpenAIRequest{Model: "test-text-model"}
	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        tokenKey,
		OriginModelName: "test-text-model",
		Request:         request,
		PriceData: types.PriceData{
			ModelRatio:      1,
			CompletionRatio: 2,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	relayInfo.PriceData.ConservativeQuotaToPreConsume = relayhelper.EstimateConservativePreConsumeQuota(relayInfo, 10, &types.TokenCountMeta{}, relayInfo.PriceData)

	apiErr := PreConsumeBilling(ctx, 10, relayInfo)
	require.Error(t, apiErr)
	require.Contains(t, apiErr.Error(), "令牌套餐周期额度不足")

	token, err := model.GetTokenById(tokenID)
	require.NoError(t, err)
	require.Equal(t, 20000, token.RemainQuota)
	require.Equal(t, 0, token.PackageUsedQuota)
}
