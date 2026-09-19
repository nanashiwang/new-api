package service

import (
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type imageReserveRecorder struct{ quota int }

func (r *imageReserveRecorder) Reserve(q int) error      { r.quota = max(r.quota, q); return nil }
func (r *imageReserveRecorder) GetPreConsumedQuota() int { return r.quota }
func (r *imageReserveRecorder) NeedsRefund() bool        { return false }
func (r *imageReserveRecorder) Refund(*gin.Context)      {}
func (r *imageReserveRecorder) Settle(int) error         { return nil }

func TestImageBillingQuantityRetryAndLocalRatios(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	reserve := &imageReserveRecorder{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeAli, UpstreamModelName: "z-image"},
		Billing:     reserve,
		PriceData: types.PriceData{UsePrice: true, ModelPrice: 100 / common.QuotaPerUnit,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 0.5}, TimeRatioInfo: types.TimeRatioInfo{Ratio: 2}},
	}
	require.Nil(t, PrepareImageBilling(c, info, 3, true))
	require.Equal(t, 600, reserve.quota)
	require.Nil(t, PrepareImageBilling(c, info, 3, true))
	require.Equal(t, 600, info.PriceData.QuotaToPreConsume)
	info.ChannelType, info.UpstreamModelName = constant.ChannelTypeOpenAI, "image"
	require.Nil(t, PrepareImageBilling(c, info, 2, false))
	require.Equal(t, 200, info.PriceData.QuotaToPreConsume)
	require.Equal(t, float64(1), info.PriceData.OtherRatios["prompt_extend"])
	require.Equal(t, float64(2), info.PriceData.OtherRatios["n"])
	require.Equal(t, 600, reserve.quota) // retain reservation; settlement refunds the excess
	actual := 1
	info.ImageResponseCount = &actual
	count, err := ApplyImageResponseQuantity(info)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, float64(1), info.PriceData.OtherRatios["n"])
	require.Equal(t, 600, reserve.quota)
	require.Nil(t, PrepareImageBilling(c, info, 2, false))
	require.Nil(t, info.ImageResponseCount, "response quantities must not leak into retries")
}

func TestImageBillingTokenExpressionsAndBounds(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	r := &imageReserveRecorder{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, Billing: r,
		PriceData: types.PriceData{QuotaToPreConsume: 100}, TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "tiered_expr", ExprString: `tier("base", p)`, ExprHash: billingexpr.ExprHashString(`tier("base", p)`),
			EstimatedPromptTokens: 100, QuotaPerUnit: 1_000_000, GroupRatio: 1}}
	require.Nil(t, PrepareImageBilling(c, info, 4, false))
	require.Equal(t, 100, r.quota) // token totals already cover all images
	for _, n := range []int{0, -1, 129} {
		require.NotNil(t, PrepareImageBilling(c, info, n, false))
	}
	info = &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, Billing: r,
		PriceData: types.PriceData{UsePrice: true, ModelPrice: math.MaxFloat64, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}}}
	require.NotNil(t, PrepareImageBilling(c, info, 128, false))
}

func TestImageExpressionUsesOutboundCountWithoutChangingOtherInputs(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	expr := `tier("images", param("n") * 100)`
	input := &billingexpr.RequestInput{Body: []byte(`{"n":1,"service_tier":"keep"}`)}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, Billing: &imageReserveRecorder{}, BillingRequestInput: input,
		PriceData: types.PriceData{TimeRatioInfo: types.TimeRatioInfo{Ratio: 2}},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{BillingMode: "tiered_expr", ExprString: expr, ExprHash: billingexpr.ExprHashString(expr),
			QuotaPerUnit: 1_000_000, GroupRatio: 0.5, TimeRatio: 2}}
	require.Nil(t, PrepareImageBilling(c, info, 4, false))
	require.Equal(t, 400, info.PriceData.QuotaToPreConsume)
	used, quota, _ := TryTieredSettle(info, billingexpr.TokenParams{})
	require.True(t, used)
	require.Equal(t, 400, quota)
	actual := 1
	info.ImageResponseCount = &actual
	_, err := ApplyImageResponseQuantity(info)
	require.NoError(t, err)
	used, quota, _ = TryTieredSettle(info, billingexpr.TokenParams{})
	require.True(t, used)
	require.Equal(t, 100, quota)
	require.Equal(t, `{"n":1,"service_tier":"keep"}`, string(input.Body))
	require.Contains(t, string(info.BillingRequestInput.Body), `"service_tier":"keep"`)
	require.Nil(t, PrepareImageBilling(c, info, 2, false))
	require.Equal(t, 200, info.PriceData.QuotaToPreConsume)
}

func TestImageBillingTokenOnlyReserveSettleAndReject(t *testing.T) {
	truncate(t)
	seedUser(t, 3201, 0)
	seedBillingPackageToken(t, 4201, 3201, "image-token", 500, 200, 0)
	c := newPackageBillingContext(true)
	info := &relaycommon.RelayInfo{UserId: 3201, TokenId: 4201, TokenKey: "image-token", Request: &dto.ImageRequest{},
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
		PriceData:   types.PriceData{UsePrice: true, ModelPrice: 50 / common.QuotaPerUnit, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}}}
	require.Nil(t, PrepareImageBilling(c, info, 2, false))
	require.Nil(t, PrepareImageBilling(c, info, 3, false))
	require.Equal(t, 150, info.Billing.GetPreConsumedQuota())
	require.NotNil(t, PrepareImageBilling(c, info, 5, false))
	token, err := model.GetTokenById(4201)
	require.NoError(t, err)
	require.Equal(t, 150, token.PackageUsedQuota)
	require.Equal(t, 350, token.RemainQuota)
	require.NoError(t, info.Billing.Settle(100))
	token, err = model.GetTokenById(4201)
	require.NoError(t, err)
	require.Equal(t, 100, token.PackageUsedQuota)
	require.Equal(t, 400, token.RemainQuota)
	quota, err := model.GetUserQuota(3201, true)
	require.NoError(t, err)
	require.Zero(t, quota)
}

func TestImageWalletReservationRejectsDebtAndRollsBack(t *testing.T) {
	truncate(t)
	seedUser(t, 3202, 100)
	seedToken(t, 4202, 3202, "image-wallet", 70)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{UserId: 3202, TokenId: 4202, TokenKey: "image-wallet", Request: &dto.ImageRequest{},
		ChannelMeta: &relaycommon.ChannelMeta{}, ImageRequestCount: 1, ForcePreConsume: true}
	require.Nil(t, PreConsumeBilling(c, 40, info))
	require.Error(t, info.Billing.Reserve(120)) // wallet has only 60 remaining
	require.Error(t, info.Billing.Reserve(90))  // wallet reserve succeeds, token fails and rolls it back
	quota, err := model.GetUserQuota(3202, true)
	require.NoError(t, err)
	require.Equal(t, 60, quota)
	require.Equal(t, 40, info.Billing.GetPreConsumedQuota())
	token, err := model.GetTokenById(4202)
	require.NoError(t, err)
	require.Equal(t, 30, token.RemainQuota)
}

func TestImageSubscriptionReserveAndRefund(t *testing.T) {
	truncate(t)
	seedUser(t, 3203, 0)
	seedToken(t, 4203, 3203, "image-subscription", 500)
	seedSubscription(t, 5203, 3203, 200, 0)
	require.NoError(t, model.PostConsumeUserSubscriptionDelta(5203, 50))
	info := &relaycommon.RelayInfo{UserId: 3203, TokenId: 4203, TokenKey: "image-subscription", SubscriptionId: 5203}
	require.NoError(t, PreConsumeTokenQuota(info, 50))
	s := &BillingSession{relayInfo: info, funding: &SubscriptionFunding{subscriptionId: 5203}, preConsumedQuota: 50, tokenConsumed: 50}
	require.NoError(t, s.Reserve(150))
	require.Error(t, s.Reserve(250))
	require.Equal(t, 150, s.preConsumedQuota)
	require.NoError(t, s.Settle(100))
	require.Equal(t, 400, getTokenRemainQuota(t, 4203))
	var sub model.UserSubscription
	require.NoError(t, model.DB.First(&sub, 5203).Error)
	require.EqualValues(t, 100, sub.AmountUsed)
}

func TestImageTokenOnlyFailureRefundIsIdempotent(t *testing.T) {
	truncate(t)
	seedUser(t, 3204, 0)
	seedBillingPackageToken(t, 4204, 3204, "image-refund", 500, 500, 0)
	c := newPackageBillingContext(true)
	info := &relaycommon.RelayInfo{UserId: 3204, TokenId: 4204, TokenKey: "image-refund", ChannelMeta: &relaycommon.ChannelMeta{},
		PriceData: types.PriceData{UsePrice: true, ModelPrice: 50 / common.QuotaPerUnit, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}}}
	require.Nil(t, PrepareImageBilling(c, info, 2, false))
	require.Nil(t, PrepareImageBilling(c, info, 4, false))
	info.Billing.Refund(c)
	info.Billing.Refund(c)
	require.Eventually(t, func() bool {
		token, err := model.GetTokenById(4204)
		return err == nil && token.RemainQuota == 500 && token.PackageUsedQuota == 0
	}, time.Second, 10*time.Millisecond)
}
