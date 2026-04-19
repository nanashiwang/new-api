package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPaymentRecordTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	DB = db
	LOG_DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	require.NoError(t, db.AutoMigrate(
		&User{},
		&TopUp{},
		&PaymentRiskCase{},
		&SellableTokenProduct{},
		&SellableTokenOrder{},
		&SellableTokenIssuance{},
	))
}

func createPaymentRecordTestUser(t *testing.T, username string) *User {
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

func createPaymentRecordTopUp(t *testing.T, userID int, tradeNo string, createTime int64, status string) *TopUp {
	return createPaymentRecordTopUpWithDetail(t, userID, tradeNo, createTime, createTime, status, "stripe", 12.5)
}

func createPaymentRecordTopUpWithDetail(t *testing.T, userID int, tradeNo string, createTime int64, completeTime int64, status string, paymentMethod string, money float64) *TopUp {
	t.Helper()
	topup := &TopUp{
		UserId:        userID,
		Amount:        120,
		Money:         money,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethod,
		CreateTime:    createTime,
		CompleteTime:  completeTime,
		Status:        status,
	}
	require.NoError(t, topup.Insert())
	return topup
}

func createPaymentRecordRiskCase(t *testing.T, recordType string, tradeNo string, userID int, paymentMethod string, status string) *PaymentRiskCase {
	t.Helper()
	riskCase, err := UpsertPaymentRiskCase(PaymentRiskCaseUpsertInput{
		RecordType:            recordType,
		TradeNo:               tradeNo,
		UserId:                userID,
		PaymentMethod:         paymentMethod,
		ProviderPaymentMethod: paymentMethod,
		Source:                "test",
		Reason:                PaymentRiskReasonManualReview,
	})
	require.NoError(t, err)
	if status != "" && status != PaymentRiskStatusOpen {
		require.NoError(t, DB.Model(&PaymentRiskCase{}).Where("id = ?", riskCase.Id).Updates(map[string]any{
			"status":      status,
			"resolved_at": 123,
		}).Error)
		require.NoError(t, DB.First(riskCase, "id = ?", riskCase.Id).Error)
	}
	return riskCase
}

func createPaymentRecordSellablePurchase(t *testing.T, userID int, productName string, createTime int64, issuanceStatus string) *SellableTokenOrder {
	return createPaymentRecordSellablePurchaseWithTradeNo(t, userID, productName, createTime, issuanceStatus, "")
}

func createPaymentRecordSellablePurchaseWithTradeNo(t *testing.T, userID int, productName string, createTime int64, issuanceStatus string, tradeNo string) *SellableTokenOrder {
	t.Helper()
	product := &SellableTokenProduct{
		Name:       productName,
		Status:     SellableTokenProductStatusEnabled,
		PriceQuota: 200,
		TotalQuota: 1000,
	}
	require.NoError(t, CreateSellableTokenProduct(product))

	order := &SellableTokenOrder{
		UserId:     userID,
		ProductId:  product.Id,
		TradeNo:    tradeNo,
		PriceQuota: 200,
	}
	require.NoError(t, DB.Create(order).Error)
	updates := map[string]any{
		"create_time":   createTime,
		"complete_time": createTime,
	}
	if tradeNo == "" {
		updates["trade_no"] = ""
		order.TradeNo = ""
	}
	require.NoError(t, DB.Model(&SellableTokenOrder{}).Where("id = ?", order.Id).Updates(updates).Error)

	issuance := &SellableTokenIssuance{
		UserId:     userID,
		ProductId:  product.Id,
		SourceType: SellableTokenSourceTypeWallet,
		SourceId:   order.Id,
	}
	require.NoError(t, CreateSellableTokenIssuanceTx(DB, issuance))
	if issuanceStatus != "" && issuanceStatus != SellableTokenIssuanceStatusPending {
		require.NoError(t, DB.Model(&SellableTokenIssuance{}).Where("id = ?", issuance.Id).Update("status", issuanceStatus).Error)
	}
	return order
}

func TestGetUserPaymentRecordsByParams_MergesAndSortsSources(t *testing.T) {
	setupPaymentRecordTestDB(t)

	user := createPaymentRecordTestUser(t, "alice")
	createPaymentRecordTopUp(t, user.Id, "T-001", 100, common.TopUpStatusSuccess)
	createPaymentRecordSellablePurchase(t, user.Id, "Alpha", 250, SellableTokenIssuanceStatusIssued)
	createPaymentRecordSellablePurchase(t, user.Id, "Beta", 200, SellableTokenIssuanceStatusPending)
	createPaymentRecordTopUp(t, user.Id, "T-002", 300, common.TopUpStatusPending)

	records, total, err := GetUserPaymentRecordsByParams(user.Id, PaymentRecordSearchParams{}, &common.PageInfo{Page: 1, PageSize: 4})
	require.NoError(t, err)
	require.Equal(t, int64(4), total)
	require.Len(t, records, 4)
	require.Equal(t, PaymentRecordTypeTopUp, records[0].RecordType)
	require.Equal(t, "T-002", records[0].TradeNo)
	require.Equal(t, PaymentRecordTypeSellableTokenPurchase, records[1].RecordType)
	require.Equal(t, "USR1STO1", records[1].TradeNo)
	require.Equal(t, "Alpha", records[1].ProductName)
	require.Equal(t, common.TopUpStatusPending, records[2].Status)
	require.Equal(t, "USR1STO2", records[2].TradeNo)
	require.Equal(t, "Beta", records[2].ProductName)
	require.Equal(t, "T-001", records[3].TradeNo)
}

func TestGetUserPaymentRecordsByParams_UsesPersistedSellableTokenTradeNoForNewOrders(t *testing.T) {
	setupPaymentRecordTestDB(t)

	user := createPaymentRecordTestUser(t, "alice")
	product := &SellableTokenProduct{
		Name:       "Alpha",
		Status:     SellableTokenProductStatusEnabled,
		PriceQuota: 200,
		TotalQuota: 1000,
	}
	require.NoError(t, CreateSellableTokenProduct(product))
	order := &SellableTokenOrder{
		UserId:     user.Id,
		ProductId:  product.Id,
		PriceQuota: 200,
	}
	require.NoError(t, DB.Create(order).Error)
	require.NoError(t, DB.Model(&SellableTokenOrder{}).Where("id = ?", order.Id).Updates(map[string]any{
		"create_time":   250,
		"complete_time": 250,
	}).Error)
	issuance := &SellableTokenIssuance{
		UserId:     user.Id,
		ProductId:  product.Id,
		SourceType: SellableTokenSourceTypeWallet,
		SourceId:   order.Id,
	}
	require.NoError(t, CreateSellableTokenIssuanceTx(DB, issuance))
	require.NoError(t, DB.Model(&SellableTokenIssuance{}).Where("id = ?", issuance.Id).Update("status", SellableTokenIssuanceStatusIssued).Error)
	require.NotEmpty(t, order.TradeNo)
	require.True(t, strings.HasPrefix(order.TradeNo, "USR1NO"))

	records, total, err := GetUserPaymentRecordsByParams(user.Id, PaymentRecordSearchParams{}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, records, 1)
	require.Equal(t, order.TradeNo, records[0].TradeNo)
	require.Equal(t, PaymentRecordTypeSellableTokenPurchase, records[0].RecordType)
}

func TestGetAllPaymentRecordsByParams_FiltersWalletPurchaseStatusAndUser(t *testing.T) {
	setupPaymentRecordTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	bob := createPaymentRecordTestUser(t, "bob")
	createPaymentRecordTopUp(t, alice.Id, "T-003", 100, common.TopUpStatusSuccess)
	createPaymentRecordSellablePurchase(t, alice.Id, "Gamma", 220, SellableTokenIssuanceStatusPending)
	createPaymentRecordSellablePurchase(t, alice.Id, "Cancelled Product", 180, SellableTokenIssuanceStatusCancelled)
	createPaymentRecordSellablePurchase(t, bob.Id, "Other User Product", 260, SellableTokenIssuanceStatusIssued)

	records, total, err := GetAllPaymentRecordsByParams(PaymentRecordSearchParams{
		Username:      "alice",
		PaymentMethod: PaymentMethodWallet,
		Status:        common.TopUpStatusPending,
	}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, records, 1)
	require.Equal(t, PaymentRecordTypeSellableTokenPurchase, records[0].RecordType)
	require.Equal(t, "Gamma", records[0].ProductName)
	require.Equal(t, common.TopUpStatusPending, records[0].Status)
	require.Equal(t, alice.Id, records[0].UserId)

	cancelledRecords, cancelledTotal, err := GetAllPaymentRecordsByParams(PaymentRecordSearchParams{
		PaymentMethod: PaymentMethodWallet,
		Status:        PaymentRecordStatusCancelled,
	}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), cancelledTotal)
	require.Len(t, cancelledRecords, 1)
	require.Equal(t, "Cancelled Product", cancelledRecords[0].ProductName)
	require.Equal(t, "USR1STO2", cancelledRecords[0].TradeNo)
	require.Equal(t, PaymentRecordStatusCancelled, cancelledRecords[0].Status)
}

func TestGetAllPaymentRecordsByParams_FiltersWalletPurchaseByUnifiedTradeNo(t *testing.T) {
	setupPaymentRecordTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	createPaymentRecordSellablePurchase(t, alice.Id, "Gamma", 220, SellableTokenIssuanceStatusPending)
	order := createPaymentRecordSellablePurchaseWithTradeNo(t, alice.Id, "Delta", 180, SellableTokenIssuanceStatusIssued, "USR1NOABC123456789")

	records, total, err := GetAllPaymentRecordsByParams(PaymentRecordSearchParams{
		Keyword:       "USR1NOABC123456789",
		PaymentMethod: PaymentMethodWallet,
	}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, records, 1)
	require.Equal(t, "Delta", records[0].ProductName)
	require.Equal(t, order.TradeNo, records[0].TradeNo)

	legacyRecords, legacyTotal, err := GetAllPaymentRecordsByParams(PaymentRecordSearchParams{
		Keyword:       "STO-1",
		PaymentMethod: PaymentMethodWallet,
	}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), legacyTotal)
	require.Len(t, legacyRecords, 1)
	require.Equal(t, "Gamma", legacyRecords[0].ProductName)
}

func TestGetAllPaymentRecordsByParams_FiltersNumericUsernameAndExplicitID(t *testing.T) {
	setupPaymentRecordTestDB(t)

	numericUser := createPaymentRecordTestUser(t, "7399605")
	otherUser := createPaymentRecordTestUser(t, "alice")
	createPaymentRecordTopUpWithDetail(t, numericUser.Id, "NUM-001", 100, 120, common.TopUpStatusSuccess, "stripe", 19.9)
	createPaymentRecordTopUpWithDetail(t, otherUser.Id, "ALICE-001", 100, 120, common.TopUpStatusSuccess, "alipay", 29.9)

	records, total, err := GetAllPaymentRecordsByParams(PaymentRecordSearchParams{
		Username: "7399605",
	}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, records, 1)
	require.Equal(t, numericUser.Id, records[0].UserId)
	require.Equal(t, "7399605", records[0].Username)

	idRecords, idTotal, err := GetAllPaymentRecordsByParams(PaymentRecordSearchParams{
		Username: "ID:7399605",
	}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(0), idTotal)
	require.Len(t, idRecords, 0)

	explicitIDRecords, explicitIDTotal, err := GetAllPaymentRecordsByParams(PaymentRecordSearchParams{
		Username: "ID:1",
	}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), explicitIDTotal)
	require.Len(t, explicitIDRecords, 1)
	require.Equal(t, numericUser.Id, explicitIDRecords[0].UserId)
	require.Equal(t, "7399605", explicitIDRecords[0].Username)
}

func TestGetPaymentRecordStats_AggregatesStatusesMethodsAndEffectiveTime(t *testing.T) {
	setupPaymentRecordTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	bob := createPaymentRecordTestUser(t, "bob")

	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-SUCC-A", 100, 150, common.TopUpStatusSuccess, "stripe", 10)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-PEND-A", 200, 0, common.TopUpStatusPending, "wxpay", 20)
	createPaymentRecordTopUpWithDetail(t, bob.Id, "T-SUCC-B", 300, 400, common.TopUpStatusSuccess, "alipay", 30)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-EXP-A", 500, 0, common.TopUpStatusExpired, "creem", 40)
	createPaymentRecordSellablePurchase(t, alice.Id, "Wallet Success", 250, SellableTokenIssuanceStatusIssued)
	createPaymentRecordSellablePurchase(t, bob.Id, "Wallet Pending", 350, SellableTokenIssuanceStatusPending)
	createPaymentRecordSellablePurchase(t, bob.Id, "Wallet Cancelled", 550, SellableTokenIssuanceStatusCancelled)

	stats, err := GetPaymentRecordStats(PaymentRecordSearchParams{})
	require.NoError(t, err)
	require.Equal(t, int64(7), stats.Totals.OrderCount)
	require.InDelta(t, 100.0, stats.Totals.Money, 0.0001)
	require.InDelta(t, 40.0, stats.Statuses[common.TopUpStatusSuccess].Money, 0.0001)
	require.Equal(t, int64(3), stats.Statuses[common.TopUpStatusSuccess].OrderCount)
	require.InDelta(t, 20.0, stats.Statuses[common.TopUpStatusPending].Money, 0.0001)
	require.Equal(t, int64(2), stats.Statuses[common.TopUpStatusPending].OrderCount)
	require.InDelta(t, 40.0, stats.Statuses[common.TopUpStatusExpired].Money, 0.0001)
	require.Equal(t, int64(1), stats.Statuses[common.TopUpStatusExpired].OrderCount)
	require.InDelta(t, 0.0, stats.Statuses[PaymentRecordStatusCancelled].Money, 0.0001)
	require.Equal(t, int64(1), stats.Statuses[PaymentRecordStatusCancelled].OrderCount)
	require.InDelta(t, 10.0, stats.PaymentMethods["stripe"].Money, 0.0001)
	require.Equal(t, int64(1), stats.PaymentMethods["stripe"].OrderCount)
	require.InDelta(t, 0.0, stats.PaymentMethods[PaymentMethodWallet].Money, 0.0001)
	require.Equal(t, int64(3), stats.PaymentMethods[PaymentMethodWallet].OrderCount)

	windowedStats, err := GetPaymentRecordStats(PaymentRecordSearchParams{
		StartTimestamp: 240,
		EndTimestamp:   420,
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), windowedStats.Totals.OrderCount)
	require.InDelta(t, 30.0, windowedStats.Totals.Money, 0.0001)
	require.Equal(t, int64(2), windowedStats.Statuses[common.TopUpStatusSuccess].OrderCount)
	require.Equal(t, int64(1), windowedStats.Statuses[common.TopUpStatusPending].OrderCount)
	require.Equal(t, int64(0), windowedStats.Statuses[common.TopUpStatusExpired].OrderCount)
	require.Equal(t, int64(0), windowedStats.Statuses[PaymentRecordStatusCancelled].OrderCount)
}

func TestGetPaymentRecordRankings_SortsByMoneyAndUsesEffectiveTime(t *testing.T) {
	setupPaymentRecordTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	bob := createPaymentRecordTestUser(t, "bob")

	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-SUCC-A", 100, 150, common.TopUpStatusSuccess, "stripe", 10)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-PEND-A", 200, 0, common.TopUpStatusPending, "wxpay", 20)
	createPaymentRecordTopUpWithDetail(t, bob.Id, "T-SUCC-B", 300, 400, common.TopUpStatusSuccess, "alipay", 30)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-EXP-A", 500, 0, common.TopUpStatusExpired, "creem", 40)
	createPaymentRecordSellablePurchase(t, alice.Id, "Wallet Success", 250, SellableTokenIssuanceStatusIssued)
	createPaymentRecordSellablePurchase(t, bob.Id, "Wallet Pending", 350, SellableTokenIssuanceStatusPending)
	createPaymentRecordSellablePurchase(t, bob.Id, "Wallet Cancelled", 550, SellableTokenIssuanceStatusCancelled)

	rankings, err := GetPaymentRecordRankings(PaymentRecordSearchParams{}, 10)
	require.NoError(t, err)
	require.Len(t, rankings, 2)
	require.Equal(t, alice.Id, rankings[0].UserId)
	require.Equal(t, "alice", rankings[0].Username)
	require.InDelta(t, 70.0, rankings[0].Money, 0.0001)
	require.Equal(t, int64(4), rankings[0].OrderCount)
	require.InDelta(t, 10.0, rankings[0].SuccessMoney, 0.0001)
	require.InDelta(t, 20.0, rankings[0].PendingMoney, 0.0001)
	require.InDelta(t, 40.0, rankings[0].ExpiredMoney, 0.0001)
	require.InDelta(t, 0.0, rankings[0].CancelledMoney, 0.0001)
	require.Equal(t, bob.Id, rankings[1].UserId)
	require.InDelta(t, 30.0, rankings[1].Money, 0.0001)
	require.Equal(t, int64(3), rankings[1].OrderCount)

	windowedRankings, err := GetPaymentRecordRankings(PaymentRecordSearchParams{
		StartTimestamp: 240,
		EndTimestamp:   420,
	}, 10)
	require.NoError(t, err)
	require.Len(t, windowedRankings, 2)
	require.Equal(t, bob.Id, windowedRankings[0].UserId)
	require.InDelta(t, 30.0, windowedRankings[0].Money, 0.0001)
	require.Equal(t, int64(2), windowedRankings[0].OrderCount)
	require.Equal(t, alice.Id, windowedRankings[1].UserId)
	require.InDelta(t, 0.0, windowedRankings[1].Money, 0.0001)
	require.Equal(t, int64(1), windowedRankings[1].OrderCount)
}

func TestGetPaymentRecordStats_ExcludesOpenRiskCasesFromDashboard(t *testing.T) {
	setupPaymentRecordTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")

	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-CLEAN-001", 100, 120, common.TopUpStatusSuccess, "stripe", 20)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "sub-open-001", 200, 0, common.TopUpStatusPending, "alipay", 85)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "sub-confirmed-001", 300, 0, common.TopUpStatusPending, "wxpay", 40)

	createPaymentRecordRiskCase(t, PaymentRiskRecordTypeSubscription, "sub-open-001", alice.Id, "alipay", PaymentRiskStatusOpen)
	createPaymentRecordRiskCase(t, PaymentRiskRecordTypeSubscription, "sub-confirmed-001", alice.Id, "wxpay", PaymentRiskStatusConfirmed)

	stats, err := GetPaymentRecordStats(PaymentRecordSearchParams{})
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.Totals.OrderCount)
	require.InDelta(t, 60.0, stats.Totals.Money, 0.0001)
	require.Equal(t, int64(1), stats.Statuses[common.TopUpStatusSuccess].OrderCount)
	require.InDelta(t, 20.0, stats.Statuses[common.TopUpStatusSuccess].Money, 0.0001)
	require.Equal(t, int64(1), stats.Statuses[common.TopUpStatusPending].OrderCount)
	require.InDelta(t, 40.0, stats.Statuses[common.TopUpStatusPending].Money, 0.0001)
	require.Equal(t, int64(0), stats.Statuses[common.TopUpStatusExpired].OrderCount)
	require.Equal(t, int64(1), stats.PaymentMethods["stripe"].OrderCount)
	require.Equal(t, int64(1), stats.PaymentMethods["wxpay"].OrderCount)
	require.Equal(t, int64(0), stats.PaymentMethods["alipay"].OrderCount)
}

func TestGetPaymentRecordRankings_ExcludesOpenRiskCasesFromDashboard(t *testing.T) {
	setupPaymentRecordTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	bob := createPaymentRecordTestUser(t, "bob")

	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-CLEAN-001", 100, 120, common.TopUpStatusSuccess, "stripe", 20)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "sub-open-001", 200, 0, common.TopUpStatusPending, "alipay", 85)
	createPaymentRecordTopUpWithDetail(t, bob.Id, "T-BOB-001", 150, 180, common.TopUpStatusSuccess, "creem", 50)

	createPaymentRecordRiskCase(t, PaymentRiskRecordTypeSubscription, "sub-open-001", alice.Id, "alipay", PaymentRiskStatusOpen)

	rankings, err := GetPaymentRecordRankings(PaymentRecordSearchParams{}, 10)
	require.NoError(t, err)
	require.Len(t, rankings, 2)
	require.Equal(t, bob.Id, rankings[0].UserId)
	require.InDelta(t, 50.0, rankings[0].Money, 0.0001)
	require.Equal(t, int64(1), rankings[0].OrderCount)
	require.Equal(t, alice.Id, rankings[1].UserId)
	require.InDelta(t, 20.0, rankings[1].Money, 0.0001)
	require.Equal(t, int64(1), rankings[1].OrderCount)
}

func TestGetPaymentRecordStats_ExcludesReversedAndVoidedRiskCasesFromDashboard(t *testing.T) {
	setupPaymentRecordTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")

	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-CLEAN-001", 100, 120, common.TopUpStatusSuccess, "stripe", 20)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "sub-confirmed-001", 200, 0, common.TopUpStatusPending, "wxpay", 40)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "sub-reversed-001", 300, 0, common.TopUpStatusPending, "creem", 70)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "sub-voided-001", 400, 0, common.TopUpStatusPending, "alipay", 30)

	createPaymentRecordRiskCase(t, PaymentRiskRecordTypeSubscription, "sub-confirmed-001", alice.Id, "wxpay", PaymentRiskStatusConfirmed)
	createPaymentRecordRiskCase(t, PaymentRiskRecordTypeSubscription, "sub-reversed-001", alice.Id, "creem", PaymentRiskStatusReversed)
	createPaymentRecordRiskCase(t, PaymentRiskRecordTypeSubscription, "sub-voided-001", alice.Id, "alipay", PaymentRiskStatusVoided)

	stats, err := GetPaymentRecordStats(PaymentRecordSearchParams{})
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.Totals.OrderCount)
	require.InDelta(t, 60.0, stats.Totals.Money, 0.0001)
	require.Equal(t, int64(1), stats.Statuses[common.TopUpStatusSuccess].OrderCount)
	require.InDelta(t, 20.0, stats.Statuses[common.TopUpStatusSuccess].Money, 0.0001)
	require.Equal(t, int64(1), stats.Statuses[common.TopUpStatusPending].OrderCount)
	require.InDelta(t, 40.0, stats.Statuses[common.TopUpStatusPending].Money, 0.0001)
	require.Equal(t, int64(1), stats.PaymentMethods["stripe"].OrderCount)
	require.Equal(t, int64(1), stats.PaymentMethods["wxpay"].OrderCount)
	require.Equal(t, int64(0), stats.PaymentMethods["creem"].OrderCount)
	require.Equal(t, int64(0), stats.PaymentMethods["alipay"].OrderCount)
}

func TestGetPaymentRecordRankings_KeepConfirmedOrdersButExcludeOtherResolvedRiskCases(t *testing.T) {
	setupPaymentRecordTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	bob := createPaymentRecordTestUser(t, "bob")

	createPaymentRecordTopUpWithDetail(t, alice.Id, "T-CLEAN-001", 100, 120, common.TopUpStatusSuccess, "stripe", 20)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "sub-confirmed-001", 200, 0, common.TopUpStatusPending, "wxpay", 40)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "sub-reversed-001", 300, 0, common.TopUpStatusPending, "creem", 70)
	createPaymentRecordTopUpWithDetail(t, alice.Id, "sub-voided-001", 400, 0, common.TopUpStatusPending, "alipay", 30)
	createPaymentRecordTopUpWithDetail(t, bob.Id, "T-BOB-001", 150, 180, common.TopUpStatusSuccess, "stripe", 50)

	createPaymentRecordRiskCase(t, PaymentRiskRecordTypeSubscription, "sub-confirmed-001", alice.Id, "wxpay", PaymentRiskStatusConfirmed)
	createPaymentRecordRiskCase(t, PaymentRiskRecordTypeSubscription, "sub-reversed-001", alice.Id, "creem", PaymentRiskStatusReversed)
	createPaymentRecordRiskCase(t, PaymentRiskRecordTypeSubscription, "sub-voided-001", alice.Id, "alipay", PaymentRiskStatusVoided)

	rankings, err := GetPaymentRecordRankings(PaymentRecordSearchParams{}, 10)
	require.NoError(t, err)
	require.Len(t, rankings, 2)
	require.Equal(t, alice.Id, rankings[0].UserId)
	require.InDelta(t, 60.0, rankings[0].Money, 0.0001)
	require.Equal(t, int64(2), rankings[0].OrderCount)
	require.InDelta(t, 20.0, rankings[0].SuccessMoney, 0.0001)
	require.InDelta(t, 40.0, rankings[0].PendingMoney, 0.0001)
	require.InDelta(t, 0.0, rankings[0].ExpiredMoney, 0.0001)
	require.InDelta(t, 0.0, rankings[0].CancelledMoney, 0.0001)
	require.Equal(t, bob.Id, rankings[1].UserId)
	require.InDelta(t, 50.0, rankings[1].Money, 0.0001)
	require.Equal(t, int64(1), rankings[1].OrderCount)
}
