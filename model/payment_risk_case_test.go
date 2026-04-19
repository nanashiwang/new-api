package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPaymentRiskCaseTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	originDB := DB
	originLogDB := LOG_DB
	originRedisEnabled := common.RedisEnabled
	originUsingSQLite := common.UsingSQLite
	originUsingMySQL := common.UsingMySQL
	originUsingPostgreSQL := common.UsingPostgreSQL
	originInviterCommissionEnabled := common.InviterCommissionEnabled

	DB = db
	LOG_DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.InviterCommissionEnabled = false

	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.RedisEnabled = originRedisEnabled
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
		common.InviterCommissionEnabled = originInviterCommissionEnabled
	})

	require.NoError(t, db.AutoMigrate(&User{}, &TopUp{}, &PaymentRiskCase{}, &Log{}))
}

func createPaymentRiskCaseTestUser(t *testing.T, username string) *User {
	t.Helper()

	user := &User{
		Username: username,
		Password: "password123",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		AffCode:  username + "-aff",
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func createPaymentRiskCaseTestTopUp(t *testing.T, userID int, tradeNo string, status string, amount int64, money float64, paymentMethod string) *TopUp {
	t.Helper()

	topup := &TopUp{
		UserId:        userID,
		Amount:        amount,
		Money:         money,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethod,
		CreateTime:    1_760_000_000,
		Status:        status,
	}
	if status == common.TopUpStatusSuccess {
		topup.CompleteTime = topup.CreateTime + 60
	}
	require.NoError(t, topup.Insert())
	return topup
}

func createPaymentRiskCaseRecord(t *testing.T, topUp *TopUp, reason string) *PaymentRiskCase {
	t.Helper()

	riskCase, err := UpsertPaymentRiskCase(PaymentRiskCaseUpsertInput{
		RecordType:            PaymentRiskRecordTypeTopUp,
		TradeNo:               topUp.TradeNo,
		UserId:                topUp.UserId,
		PaymentMethod:         topUp.PaymentMethod,
		ProviderPaymentMethod: topUp.PaymentMethod,
		ExpectedAmount:        topUp.Amount,
		ExpectedMoney:         topUp.Money,
		ReceivedMoney:         topUp.Money,
		Source:                "test",
		Reason:                reason,
		OrderStatus:           topUp.Status,
		ProviderPayload:       `{"source":"test"}`,
	})
	require.NoError(t, err)
	return riskCase
}

func TestResolvePaymentRiskCase_ConfirmCompletesPendingTopUp(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	user := createPaymentRiskCaseTestUser(t, "alice")
	topup := createPaymentRiskCaseTestTopUp(t, user.Id, "RISK-CONFIRM-001", common.TopUpStatusPending, 2, 16, "alipay")
	riskCase := createPaymentRiskCaseRecord(t, topup, PaymentRiskReasonAmountMismatch)

	grantedQuota, err := CalculateGrantedQuotaForTopUp(topup)
	require.NoError(t, err)

	require.NoError(t, ResolvePaymentRiskCase(riskCase.Id, 99, PaymentRiskActionConfirm, "manual confirm"))

	updatedCase, err := GetPaymentRiskCaseByID(riskCase.Id)
	require.NoError(t, err)
	require.Equal(t, PaymentRiskStatusConfirmed, updatedCase.Status)
	require.Equal(t, 99, updatedCase.HandlerAdminId)
	require.Equal(t, "manual confirm", updatedCase.HandlerNote)
	require.Equal(t, 0, updatedCase.AppliedQuotaDelta)
	require.NotZero(t, updatedCase.ResolvedAt)

	updatedTopUp := GetTopUpByTradeNo(topup.TradeNo)
	require.NotNil(t, updatedTopUp)
	require.Equal(t, common.TopUpStatusSuccess, updatedTopUp.Status)
	require.NotZero(t, updatedTopUp.CompleteTime)

	updatedUser, err := GetUserById(user.Id, false)
	require.NoError(t, err)
	require.Equal(t, grantedQuota, updatedUser.Quota)
}

func TestResolvePaymentRiskCase_ReverseDeductsGrantedQuota(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	user := createPaymentRiskCaseTestUser(t, "bob")
	topup := createPaymentRiskCaseTestTopUp(t, user.Id, "RISK-REVERSE-001", common.TopUpStatusSuccess, 3, 24, "alipay")
	riskCase := createPaymentRiskCaseRecord(t, topup, PaymentRiskReasonManualReview)

	grantedQuota, err := CalculateGrantedQuotaForTopUp(topup)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", grantedQuota).Error)

	require.NoError(t, ResolvePaymentRiskCase(riskCase.Id, 7, PaymentRiskActionReverse, "reverse suspicious credit"))

	updatedCase, err := GetPaymentRiskCaseByID(riskCase.Id)
	require.NoError(t, err)
	require.Equal(t, PaymentRiskStatusReversed, updatedCase.Status)
	require.Equal(t, -grantedQuota, updatedCase.AppliedQuotaDelta)
	require.Equal(t, 7, updatedCase.HandlerAdminId)

	updatedUser, err := GetUserById(user.Id, false)
	require.NoError(t, err)
	require.Equal(t, 0, updatedUser.Quota)
}

func TestResolvePaymentRiskCase_VoidExpiresPendingTopUp(t *testing.T) {
	setupPaymentRiskCaseTestDB(t)

	user := createPaymentRiskCaseTestUser(t, "charlie")
	topup := createPaymentRiskCaseTestTopUp(t, user.Id, "RISK-VOID-001", common.TopUpStatusPending, 1, 8, "alipay")
	riskCase := createPaymentRiskCaseRecord(t, topup, PaymentRiskReasonManualReview)

	require.NoError(t, ResolvePaymentRiskCase(riskCase.Id, 12, PaymentRiskActionVoid, "void invalid callback"))

	updatedCase, err := GetPaymentRiskCaseByID(riskCase.Id)
	require.NoError(t, err)
	require.Equal(t, PaymentRiskStatusVoided, updatedCase.Status)
	require.Equal(t, "void invalid callback", updatedCase.HandlerNote)

	updatedTopUp := GetTopUpByTradeNo(topup.TradeNo)
	require.NotNil(t, updatedTopUp)
	require.Equal(t, common.TopUpStatusExpired, updatedTopUp.Status)
	require.NotZero(t, updatedTopUp.CompleteTime)
}
