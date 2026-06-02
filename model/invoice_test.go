package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func setupInvoiceTestDB(t *testing.T) {
	t.Helper()
	setupPaymentRecordTestDB(t)
	require.NoError(t, DB.AutoMigrate(&InvoiceRequest{}, &InvoiceRequestItem{}))
}

func createInvoiceTestSubscriptionTopUp(t *testing.T, userID int, tradeNo string, createTime int64) *TopUp {
	t.Helper()
	topup := &TopUp{
		UserId:        userID,
		Amount:        0,
		Money:         88.8,
		TradeNo:       tradeNo,
		PaymentMethod: "stripe",
		CreateTime:    createTime,
		CompleteTime:  createTime + 10,
		Status:        common.TopUpStatusSuccess,
	}
	require.NoError(t, topup.Insert())
	return topup
}

func createInvoiceRequestInput(orderType string, orderID int) CreateInvoiceRequestInput {
	return CreateInvoiceRequestInput{
		TitleType: InvoiceTitleTypeCompany,
		Title:     "测试公司",
		TaxNumber: "TAX123456",
		Email:     "invoice@example.com",
		Phone:     "13800138000",
		Orders: []InvoiceOrderRef{{
			OrderType: orderType,
			Id:        orderID,
		}},
	}
}

func TestGetEligibleInvoiceOrders_FiltersSuccessfulUnoccupiedOrders(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "alice")
	successTopup := createPaymentRecordTopUp(t, user.Id, "T-INV-001", 100, common.TopUpStatusSuccess)
	createPaymentRecordTopUp(t, user.Id, "T-INV-002", 200, common.TopUpStatusPending)
	createPaymentRecordTopUp(t, user.Id, "T-INV-003", 300, common.TopUpStatusExpired)
	subscriptionTopup := createInvoiceTestSubscriptionTopUp(t, user.Id, "sub-inv-001", 400)

	records, total, err := GetEligibleInvoiceOrders(user.Id, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, records, 2)
	require.Equal(t, subscriptionTopup.Id, records[0].Id)
	require.Equal(t, PaymentOrderTypeSubscription, records[0].OrderType)
	require.Equal(t, successTopup.Id, records[1].Id)
	require.Equal(t, PaymentRecordTypeTopUp, records[1].OrderType)
}

func TestCreateInvoiceRequest_OccupiesOrderAndRejectedReleasesIt(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "alice")
	topup := createPaymentRecordTopUp(t, user.Id, "T-INV-004", 100, common.TopUpStatusSuccess)

	request, err := CreateInvoiceRequest(user.Id, createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id))
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusPending, request.Status)
	require.Len(t, request.Items, 1)

	records, total, err := GetEligibleInvoiceOrders(user.Id, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.Empty(t, records)

	rejected, err := RejectInvoiceRequest(request.Id, user.Id, "资料不完整")
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusRejected, rejected.Status)

	records, total, err = GetEligibleInvoiceOrders(user.Id, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, records, 1)
	require.Equal(t, topup.Id, records[0].Id)
}

func TestCreateInvoiceRequest_RejectsOtherUsersOrder(t *testing.T) {
	setupInvoiceTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	bob := createPaymentRecordTestUser(t, "bob")
	bobTopup := createPaymentRecordTopUp(t, bob.Id, "T-INV-005", 100, common.TopUpStatusSuccess)

	_, err := CreateInvoiceRequest(alice.Id, createInvoiceRequestInput(PaymentRecordTypeTopUp, bobTopup.Id))
	require.Error(t, err)
}

func TestCreateInvoiceRequest_SpecialInvoiceRequiresTaxNumberAndStoresInfo(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "alice")
	topup := createPaymentRecordTopUp(t, user.Id, "T-INV-SPECIAL", 100, common.TopUpStatusSuccess)
	input := createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id)
	input.InvoiceType = InvoiceTypeSpecial
	input.TitleType = InvoiceTitleTypePersonal
	input.Title = ""
	input.TaxNumber = "SPECIAL-TAX-001"

	_, err := CreateInvoiceRequest(user.Id, input)
	require.ErrorContains(t, err, "单位名称不能为空")

	input.Title = "测试公司"
	input.TaxNumber = ""

	_, err = CreateInvoiceRequest(user.Id, input)
	require.ErrorContains(t, err, "专票需要填写税号")

	input.TaxNumber = "SPECIAL-TAX-001"
	input.RegisteredAddress = "上海市浦东新区"
	input.RegisteredPhone = "021-12345678"
	input.BankName = "测试银行"
	input.BankAccount = "6222000000000000"

	request, err := CreateInvoiceRequest(user.Id, input)
	require.NoError(t, err)
	require.Equal(t, InvoiceTypeSpecial, request.InvoiceType)
	require.Equal(t, InvoiceTitleTypeCompany, request.TitleType)
	require.Equal(t, "SPECIAL-TAX-001", request.TaxNumber)
	require.Equal(t, "上海市浦东新区", request.RegisteredAddress)
	require.Equal(t, "021-12345678", request.RegisteredPhone)
	require.Equal(t, "测试银行", request.BankName)
	require.Equal(t, "6222000000000000", request.BankAccount)
}

func TestApproveInvoiceRequest_KeepsOrderOccupiedAndPreventsSecondReview(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "alice")
	topup := createPaymentRecordTopUp(t, user.Id, "T-INV-006", 100, common.TopUpStatusSuccess)
	request, err := CreateInvoiceRequest(user.Id, createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id))
	require.NoError(t, err)

	approved, err := ApproveInvoiceRequest(request.Id, user.Id, InvoiceReviewInput{
		InvoiceNo:         "FP-001",
		InvoiceUrl:        "https://example.com/invoice.pdf",
		InvoiceFileName:   "invoice.pdf",
		InvoiceFilePath:   "/tmp/invoice.pdf",
		InvoiceSentTo:     "invoice@example.com",
		InvoiceSendStatus: InvoiceSendStatusPending,
	})
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusInvoiced, approved.Status)
	require.Equal(t, "FP-001", approved.InvoiceNo)
	require.Equal(t, "invoice.pdf", approved.InvoiceFileName)
	require.Equal(t, "invoice@example.com", approved.InvoiceSentTo)
	require.Equal(t, InvoiceSendStatusPending, approved.InvoiceSendStatus)

	records, total, err := GetEligibleInvoiceOrders(user.Id, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.Empty(t, records)

	_, err = RejectInvoiceRequest(request.Id, user.Id, "重复审核")
	require.ErrorIs(t, err, ErrInvoiceRequestAlreadyReviewed)
}

func TestInvoiceSellableTokenPurchaseCanBeRequested(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "alice")
	order := createPaymentRecordSellablePurchase(t, user.Id, "钱包包月", 100, SellableTokenIssuanceStatusIssued)

	records, total, err := GetEligibleInvoiceOrders(user.Id, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, records, 1)
	require.Equal(t, PaymentRecordTypeSellableTokenPurchase, records[0].OrderType)

	request, err := CreateInvoiceRequest(user.Id, createInvoiceRequestInput(PaymentRecordTypeSellableTokenPurchase, order.Id))
	require.NoError(t, err)
	require.Equal(t, int64(200), request.TotalQuota)
	require.Len(t, request.Items, 1)
	require.Equal(t, "钱包包月", request.Items[0].ProductName)
}

func TestGetUserInvoiceRequestDetail_EnforcesOwnerAndLoadsItems(t *testing.T) {
	setupInvoiceTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	bob := createPaymentRecordTestUser(t, "bob")
	topup := createPaymentRecordTopUp(t, alice.Id, "T-INV-007", 100, common.TopUpStatusSuccess)
	request, err := CreateInvoiceRequest(alice.Id, createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id))
	require.NoError(t, err)

	detail, err := GetUserInvoiceRequestDetail(alice.Id, request.Id)
	require.NoError(t, err)
	require.Equal(t, request.Id, detail.Id)
	require.Equal(t, "alice", detail.Username)
	require.Len(t, detail.Items, 1)
	require.Equal(t, "T-INV-007", detail.Items[0].TradeNo)

	_, err = GetUserInvoiceRequestDetail(bob.Id, request.Id)
	require.ErrorIs(t, err, ErrInvoiceRequestNotFound)
}

func TestGetInvoiceRequestDetail_IncludesReviewerInfo(t *testing.T) {
	setupInvoiceTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	reviewer := createPaymentRecordTestUser(t, "admin")
	topup := createPaymentRecordTopUp(t, alice.Id, "T-INV-008", 100, common.TopUpStatusSuccess)
	request, err := CreateInvoiceRequest(alice.Id, createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id))
	require.NoError(t, err)
	_, err = ApproveInvoiceRequest(request.Id, reviewer.Id, InvoiceReviewInput{InvoiceNo: "FP-008"})
	require.NoError(t, err)

	detail, err := GetInvoiceRequestDetail(request.Id)
	require.NoError(t, err)
	require.Equal(t, "alice", detail.Username)
	require.Equal(t, reviewer.Id, detail.ReviewerUserId)
	require.Equal(t, "admin", detail.ReviewerUsername)
	require.Equal(t, "FP-008", detail.InvoiceNo)
	require.Len(t, detail.Items, 1)
}

func TestUpdateInvoiceSendStatus(t *testing.T) {
	setupInvoiceTestDB(t)

	alice := createPaymentRecordTestUser(t, "alice")
	reviewer := createPaymentRecordTestUser(t, "admin")
	topup := createPaymentRecordTopUp(t, alice.Id, "T-INV-009", 100, common.TopUpStatusSuccess)
	request, err := CreateInvoiceRequest(alice.Id, createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id))
	require.NoError(t, err)
	_, err = ApproveInvoiceRequest(request.Id, reviewer.Id, InvoiceReviewInput{
		InvoiceNo:         "FP-009",
		InvoiceFileName:   "invoice.pdf",
		InvoiceFilePath:   "/tmp/invoice.pdf",
		InvoiceSentTo:     "invoice@example.com",
		InvoiceSendStatus: InvoiceSendStatusPending,
	})
	require.NoError(t, err)

	failed, err := UpdateInvoiceSendStatus(request.Id, InvoiceSendStatusFailed, "smtp failed")
	require.NoError(t, err)
	require.Equal(t, InvoiceSendStatusFailed, failed.InvoiceSendStatus)
	require.Equal(t, "smtp failed", failed.InvoiceSendError)
	require.Equal(t, int64(0), failed.InvoiceSentAt)

	sent, err := UpdateInvoiceSendStatus(request.Id, InvoiceSendStatusSent, "")
	require.NoError(t, err)
	require.Equal(t, InvoiceSendStatusSent, sent.InvoiceSendStatus)
	require.Empty(t, sent.InvoiceSendError)
	require.Greater(t, sent.InvoiceSentAt, int64(0))
}
