package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func setupInvoiceTestDB(t *testing.T) {
	t.Helper()
	setupPaymentRecordTestDB(t)
	originInvoiceServiceFeeRate := common.InvoiceServiceFeeRate
	common.InvoiceServiceFeeRate = 0
	t.Cleanup(func() {
		common.InvoiceServiceFeeRate = originInvoiceServiceFeeRate
	})
	require.NoError(t, DB.AutoMigrate(&InvoiceRequest{}, &InvoiceRequestItem{}, &InvoiceRequestProductItem{}))
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
		TitleType:      InvoiceTitleTypeCompany,
		Title:          "测试公司",
		TaxNumber:      "TAX123456",
		Email:          "invoice@example.com",
		Phone:          "13800138000",
		NeedDetailBill: true,
		Orders: []InvoiceOrderRef{{
			OrderType: orderType,
			Id:        orderID,
		}},
	}
}

func createManualInvoiceRequestInput(tradeNo string) CreateInvoiceRequestInput {
	return CreateInvoiceRequestInput{
		TitleType:      InvoiceTitleTypeCompany,
		Title:          "测试公司",
		TaxNumber:      "TAX123456",
		Email:          "invoice@example.com",
		NeedDetailBill: true,
		ManualTransactions: []InvoiceManualTransactionInput{{
			TradeNo:          tradeNo,
			PayerName:        "付款测试公司",
			PayeeName:        InvoiceManualTransferPayeeName,
			TransferBankName: "测试银行",
			Money:            1000,
			PaidAt:           common.GetTimestamp() - 60,
			Remark:           "公对公转账",
		}},
		ProductItems: []InvoiceProductItemInput{{
			ProductName:   "API 调用服务",
			Specification: "标准版",
			Unit:          "项",
			Quantity:      1,
			UnitPrice:     1000,
			Money:         1000,
			Quota:         500000,
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

func TestCreateInvoiceRequest_StoresAttachmentOptions(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "alice")
	topup := createPaymentRecordTopUp(t, user.Id, "T-INV-SERVICE", 100, common.TopUpStatusSuccess)
	input := createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id)
	input.NeedDetailBill = false
	input.NeedServiceConfirmation = true

	request, err := CreateInvoiceRequest(user.Id, input)
	require.NoError(t, err)
	require.False(t, request.NeedDetailBill)
	require.True(t, request.NeedServiceConfirmation)

	detail, err := GetUserInvoiceRequestDetail(user.Id, request.Id)
	require.NoError(t, err)
	require.False(t, detail.NeedDetailBill)
	require.True(t, detail.NeedServiceConfirmation)
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

func TestApproveInvoiceRequest_AllowsInvoiceFileWithoutInvoiceNo(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "alice")
	topup := createPaymentRecordTopUp(t, user.Id, "T-INV-NO-NUMBER", 100, common.TopUpStatusSuccess)
	request, err := CreateInvoiceRequest(user.Id, createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id))
	require.NoError(t, err)

	approved, err := ApproveInvoiceRequest(request.Id, user.Id, InvoiceReviewInput{
		InvoiceFileName: "invoice.pdf",
		InvoiceFilePath: "/tmp/invoice.pdf",
		InvoiceSentTo:   "invoice@example.com",
	})
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusInvoiced, approved.Status)
	require.Empty(t, approved.InvoiceNo)
	require.Equal(t, "invoice.pdf", approved.InvoiceFileName)
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

func TestCreateInvoiceRequest_ManualTransferStoresAuditableSnapshotsWithoutTopUp(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "manual-transfer-user")
	var beforeTopUps int64
	require.NoError(t, DB.Model(&TopUp{}).Where("user_id = ?", user.Id).Count(&beforeTopUps).Error)
	beforeQuota := user.Quota

	request, err := CreateInvoiceRequest(user.Id, createManualInvoiceRequestInput("BANK-TRANSFER-001"))
	require.NoError(t, err)
	require.Equal(t, InvoiceSourceTypeManualTransfer, request.SourceType)
	require.Equal(t, 1000.0, request.TotalMoney)
	require.Equal(t, int64(500000), request.TotalQuota)
	require.Len(t, request.Items, 1)
	require.Len(t, request.ProductItems, 1)
	require.Equal(t, InvoiceSourceTypeManualTransfer, request.Items[0].SourceType)
	require.Equal(t, InvoiceOrderTypeManualTransfer, request.Items[0].OrderType)
	require.Equal(t, 0, request.Items[0].OrderId)
	require.Equal(t, InvoicePaymentMethodBankTransfer, request.Items[0].PaymentMethod)
	require.Equal(t, "付款测试公司", request.Items[0].PayerName)
	require.Equal(t, "API 调用服务", request.ProductItems[0].ProductName)

	var afterTopUps int64
	require.NoError(t, DB.Model(&TopUp{}).Where("user_id = ?", user.Id).Count(&afterTopUps).Error)
	require.Equal(t, beforeTopUps, afterTopUps)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, "id = ?", user.Id).Error)
	require.Equal(t, beforeQuota, refreshed.Quota)

	detail, err := GetUserInvoiceRequestDetail(user.Id, request.Id)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	require.Len(t, detail.ProductItems, 1)
}

func TestCreateInvoiceRequest_ManualTransferRejectsMixedSources(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "mixed-source-user")
	topup := createPaymentRecordTopUp(t, user.Id, "T-INV-MIXED", 100, common.TopUpStatusSuccess)
	input := createManualInvoiceRequestInput("BANK-TRANSFER-MIXED")
	input.Orders = []InvoiceOrderRef{{OrderType: PaymentRecordTypeTopUp, Id: topup.Id}}

	_, err := CreateInvoiceRequest(user.Id, input)
	require.ErrorContains(t, err, "不能在同一申请中混用")
}

func TestCreateInvoiceRequest_ManualTransferRequiresMatchingProductTotal(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "manual-total-user")
	input := createManualInvoiceRequestInput("BANK-TRANSFER-TOTAL")
	input.ProductItems[0].Money = 999
	input.ProductItems[0].UnitPrice = 999

	_, err := CreateInvoiceRequest(user.Id, input)
	require.ErrorContains(t, err, "产品明细金额合计")
}

func TestCreateInvoiceRequest_ManualTransferRejectsOneCentProductMismatch(t *testing.T) {
	t.Run("数量单价与金额相差一分", func(t *testing.T) {
		setupInvoiceTestDB(t)

		user := createPaymentRecordTestUser(t, "manual-line-cent-mismatch-user")
		input := createManualInvoiceRequestInput("BANK-TRANSFER-LINE-CENT-MISMATCH")
		input.ProductItems[0].UnitPrice = 999.99
		input.ProductItems[0].Money = 1000

		_, err := CreateInvoiceRequest(user.Id, input)
		require.ErrorContains(t, err, "数量、单价与金额不一致")
	})

	t.Run("产品合计与转账金额相差一分", func(t *testing.T) {
		setupInvoiceTestDB(t)

		user := createPaymentRecordTestUser(t, "manual-total-cent-mismatch-user")
		input := createManualInvoiceRequestInput("BANK-TRANSFER-TOTAL-CENT-MISMATCH")
		input.ProductItems[0].UnitPrice = 999.99
		input.ProductItems[0].Money = 999.99

		_, err := CreateInvoiceRequest(user.Id, input)
		require.ErrorContains(t, err, "产品明细金额合计")
	})
}

func TestCreateInvoiceRequest_ManualTransferDuplicateReleasedAfterRejection(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "manual-duplicate-user")
	input := createManualInvoiceRequestInput("BANK-TRANSFER-DUPLICATE")
	first, err := CreateInvoiceRequest(user.Id, input)
	require.NoError(t, err)

	_, err = CreateInvoiceRequest(user.Id, input)
	require.ErrorIs(t, err, ErrInvoiceManualTransferOccupied)

	_, err = RejectInvoiceRequest(first.Id, user.Id, "转账资料需要修正")
	require.NoError(t, err)

	second, err := CreateInvoiceRequest(user.Id, input)
	require.NoError(t, err)
	require.NotEqual(t, first.Id, second.Id)
}

func TestCreateInvoiceRequest_ManualTransferRejectsFuturePaymentAndInvalidMoney(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "manual-invalid-user")
	future := createManualInvoiceRequestInput("BANK-TRANSFER-FUTURE")
	future.ManualTransactions[0].PaidAt = common.GetTimestamp() + 48*60*60
	_, err := CreateInvoiceRequest(user.Id, future)
	require.ErrorContains(t, err, "转账时间不能晚于当前时间")

	fraction := createManualInvoiceRequestInput("BANK-TRANSFER-FRACTION")
	fraction.ManualTransactions[0].Money = 1000.001
	_, err = CreateInvoiceRequest(user.Id, fraction)
	require.ErrorContains(t, err, "金额最多保留 2 位小数")
}

func TestCreateInvoiceRequest_SystemOrderCreatesDefaultProductSnapshot(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "system-product-user")
	topup := createPaymentRecordTopUp(t, user.Id, "T-INV-PRODUCT", 100, common.TopUpStatusSuccess)
	request, err := CreateInvoiceRequest(user.Id, createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id))
	require.NoError(t, err)
	require.Equal(t, InvoiceSourceTypeSystemOrder, request.SourceType)
	require.Len(t, request.ProductItems, 1)
	require.Equal(t, request.TotalMoney, request.ProductItems[0].Money)
}

func TestCreateInvoiceRequest_ManualTransferForcesDocumentsAndFixedPayee(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "manual-fixed-payee-user")
	input := createManualInvoiceRequestInput("BANK-TRANSFER-FIXED-PAYEE")
	input.NeedDetailBill = false
	input.NeedServiceConfirmation = false
	input.ManualTransactions[0].PayeeName = " 上海曜算智能科技有限公司 "

	request, err := CreateInvoiceRequest(user.Id, input)
	require.NoError(t, err)
	require.True(t, request.NeedDetailBill)
	require.True(t, request.NeedServiceConfirmation)
	require.Equal(t, InvoiceManualTransferPayeeName, request.Items[0].PayeeName)

	invalid := createManualInvoiceRequestInput("BANK-TRANSFER-WRONG-PAYEE")
	invalid.ManualTransactions[0].PayeeName = "其他公司"
	_, err = CreateInvoiceRequest(user.Id, invalid)
	require.ErrorContains(t, err, "收款方必须为")
}

func TestCreateInvoiceRequest_SystemOrderRejectsCustomProductItems(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "system-custom-product-user")
	topup := createPaymentRecordTopUp(t, user.Id, "T-INV-CUSTOM-PRODUCT", 100, common.TopUpStatusSuccess)
	input := createInvoiceRequestInput(PaymentRecordTypeTopUp, topup.Id)
	input.ProductItems = []InvoiceProductItemInput{{
		ProductName: "伪造产品",
		Unit:        "项",
		Quantity:    1,
		UnitPrice:   topup.Money,
		Money:       topup.Money,
	}}

	_, err := CreateInvoiceRequest(user.Id, input)
	require.ErrorContains(t, err, "不能自定义")
}

func TestCreateInvoiceRequest_ManualTransferRejectsZeroUnitPrice(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "manual-zero-price-user")
	input := createManualInvoiceRequestInput("BANK-TRANSFER-ZERO-PRICE")
	input.ProductItems[0].UnitPrice = 0

	_, err := CreateInvoiceRequest(user.Id, input)
	require.ErrorContains(t, err, "单价必须大于 0")
}

func TestCreateInvoiceRequest_ManualTransferFingerprintNormalizesWidthAndSeparators(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "manual-normalized-fingerprint-user")
	first := createManualInvoiceRequestInput("BANK-001")
	_, err := CreateInvoiceRequest(user.Id, first)
	require.NoError(t, err)

	duplicate := createManualInvoiceRequestInput("ＢＡＮＫ ００１")
	duplicate.ManualTransactions[0].TransferBankName = "测 试-银行"
	_, err = CreateInvoiceRequest(user.Id, duplicate)
	require.ErrorIs(t, err, ErrInvoiceManualTransferOccupied)
}

func TestCreateInvoiceRequest_ManualTransferFeeFailureRollsBackAllSnapshots(t *testing.T) {
	setupInvoiceTestDB(t)

	originPrice := operation_setting.Price
	common.InvoiceServiceFeeRate = 1
	operation_setting.Price = 1
	t.Cleanup(func() {
		operation_setting.Price = originPrice
	})

	user := createPaymentRecordTestUser(t, "manual-fee-rollback-user")
	_, err := CreateInvoiceRequest(user.Id, createManualInvoiceRequestInput("BANK-TRANSFER-FEE-ROLLBACK"))
	require.ErrorIs(t, err, ErrInsufficientQuotaForInvoiceFee)

	var requestCount int64
	require.NoError(t, DB.Model(&InvoiceRequest{}).Where("user_id = ?", user.Id).Count(&requestCount).Error)
	require.Zero(t, requestCount)
	var itemCount int64
	require.NoError(t, DB.Model(&InvoiceRequestItem{}).Where("user_id = ?", user.Id).Count(&itemCount).Error)
	require.Zero(t, itemCount)
	var productCount int64
	require.NoError(t, DB.Model(&InvoiceRequestProductItem{}).Count(&productCount).Error)
	require.Zero(t, productCount)
}

func TestCreateInvoiceRequest_ManualTransferRejectsNonCanonicalIdentifiersAndPrecisionOverflow(t *testing.T) {
	setupInvoiceTestDB(t)

	user := createPaymentRecordTestUser(t, "manual-canonical-validation-user")
	invalidTradeNo := createManualInvoiceRequestInput("---")
	_, err := CreateInvoiceRequest(user.Id, invalidTradeNo)
	require.ErrorContains(t, err, "银行流水号必须包含字母或数字")

	invalidPrice := createManualInvoiceRequestInput("BANK-TRANSFER-UNIT-PRICE-PRECISION")
	invalidPrice.ProductItems[0].UnitPrice = 1000.001
	_, err = CreateInvoiceRequest(user.Id, invalidPrice)
	require.ErrorContains(t, err, "单价最多保留 2 位小数")

	invalidQuantity := createManualInvoiceRequestInput("BANK-TRANSFER-QUANTITY-PRECISION")
	invalidQuantity.ProductItems[0].Quantity = 1.0000001
	_, err = CreateInvoiceRequest(user.Id, invalidQuantity)
	require.ErrorContains(t, err, "数量最多保留 6 位小数")
}

func TestRejectInvoiceRequest_ManualTransferRefundsFeeAndReleasesFingerprintAtomically(t *testing.T) {
	setupInvoiceTestDB(t)

	originPrice := operation_setting.Price
	common.InvoiceServiceFeeRate = 0.1
	operation_setting.Price = 1
	t.Cleanup(func() {
		operation_setting.Price = originPrice
	})

	user := createPaymentRecordTestUser(t, "manual-fee-refund-user")
	const initialQuota = 100_000_000
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", initialQuota).Error)

	input := createManualInvoiceRequestInput("BANK-TRANSFER-FEE-REFUND")
	request, err := CreateInvoiceRequest(user.Id, input)
	require.NoError(t, err)
	require.Greater(t, request.ServiceFeeQuota, int64(0))

	var deducted User
	require.NoError(t, DB.First(&deducted, "id = ?", user.Id).Error)
	require.Equal(t, initialQuota-int(request.ServiceFeeQuota), deducted.Quota)

	_, err = RejectInvoiceRequest(request.Id, user.Id, "到账资料需要修正")
	require.NoError(t, err)
	var refunded User
	require.NoError(t, DB.First(&refunded, "id = ?", user.Id).Error)
	require.Equal(t, initialQuota, refunded.Quota)

	second, err := CreateInvoiceRequest(user.Id, input)
	require.NoError(t, err)
	require.NotEqual(t, request.Id, second.Id)
}
