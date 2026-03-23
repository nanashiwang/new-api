package model

import (
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

	require.NoError(t, db.AutoMigrate(&User{}, &TopUp{}, &SellableTokenProduct{}, &SellableTokenOrder{}, &SellableTokenIssuance{}))
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
	t.Helper()
	topup := &TopUp{
		UserId:        userID,
		Amount:        120,
		Money:         12.5,
		TradeNo:       tradeNo,
		PaymentMethod: "stripe",
		CreateTime:    createTime,
		CompleteTime:  createTime,
		Status:        status,
	}
	require.NoError(t, topup.Insert())
	return topup
}

func createPaymentRecordSellablePurchase(t *testing.T, userID int, productName string, createTime int64, issuanceStatus string) *SellableTokenOrder {
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
		PriceQuota: 200,
	}
	require.NoError(t, DB.Create(order).Error)
	require.NoError(t, DB.Model(&SellableTokenOrder{}).Where("id = ?", order.Id).Updates(map[string]any{
		"create_time":   createTime,
		"complete_time": createTime,
	}).Error)

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
	require.Equal(t, "Alpha", records[1].ProductName)
	require.Equal(t, common.TopUpStatusPending, records[2].Status)
	require.Equal(t, "Beta", records[2].ProductName)
	require.Equal(t, "T-001", records[3].TradeNo)
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
	require.Equal(t, PaymentRecordStatusCancelled, cancelledRecords[0].Status)
}
