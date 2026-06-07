package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	InvoiceStatusPending  = "pending"
	InvoiceStatusInvoiced = "invoiced"
	InvoiceStatusRejected = "rejected"

	InvoiceSendStatusPending = "pending"
	InvoiceSendStatusSent    = "sent"
	InvoiceSendStatusFailed  = "failed"

	InvoiceTypeNormal  = "normal"
	InvoiceTypeSpecial = "special"

	InvoiceTitleTypePersonal = "personal"
	InvoiceTitleTypeCompany  = "company"
)

var (
	ErrInvoiceRequestNotFound        = errors.New("发票申请不存在")
	ErrInvoiceRequestAlreadyReviewed = errors.New("发票申请已审核")
	ErrInvoiceOrderUnavailable       = errors.New("订单不可申请发票")
)

type InvoiceRequest struct {
	Id                      int     `json:"id"`
	UserId                  int     `json:"user_id" gorm:"index;not null"`
	InvoiceType             string  `json:"invoice_type" gorm:"type:varchar(16);not null;default:'normal'"`
	TitleType               string  `json:"title_type" gorm:"type:varchar(16);not null;default:'personal'"`
	Title                   string  `json:"title" gorm:"type:varchar(128);not null;default:''"`
	TaxNumber               string  `json:"tax_number" gorm:"type:varchar(64);not null;default:''"`
	RegisteredAddress       string  `json:"registered_address" gorm:"type:varchar(255);not null;default:''"`
	RegisteredPhone         string  `json:"registered_phone" gorm:"type:varchar(64);not null;default:''"`
	BankName                string  `json:"bank_name" gorm:"type:varchar(128);not null;default:''"`
	BankAccount             string  `json:"bank_account" gorm:"type:varchar(128);not null;default:''"`
	Email                   string  `json:"email" gorm:"type:varchar(128);not null;default:''"`
	Phone                   string  `json:"phone" gorm:"type:varchar(32);not null;default:''"`
	Remark                  string  `json:"remark" gorm:"type:text"`
	NeedServiceConfirmation bool    `json:"need_service_confirmation" gorm:"not null;default:false"`
	Status                  string  `json:"status" gorm:"type:varchar(16);index;not null;default:'pending'"`
	TotalMoney              float64 `json:"total_money" gorm:"type:decimal(20,6);not null;default:0"`
	TotalQuota              int64   `json:"total_quota" gorm:"type:bigint;not null;default:0"`
	InvoiceNo               string  `json:"invoice_no" gorm:"type:varchar(128);not null;default:''"`
	InvoiceUrl              string  `json:"invoice_url" gorm:"type:text"`
	InvoiceFileName         string  `json:"invoice_file_name" gorm:"type:varchar(255);not null;default:''"`
	InvoiceFilePath         string  `json:"-" gorm:"type:text"`
	InvoiceSentTo           string  `json:"invoice_sent_to" gorm:"type:varchar(128);not null;default:''"`
	InvoiceSentAt           int64   `json:"invoice_sent_at" gorm:"index;not null;default:0"`
	InvoiceSendStatus       string  `json:"invoice_send_status" gorm:"type:varchar(16);index;not null;default:''"`
	InvoiceSendError        string  `json:"invoice_send_error" gorm:"type:text"`
	AdminRemark             string  `json:"admin_remark" gorm:"type:text"`
	ReviewerUserId          int     `json:"reviewer_user_id" gorm:"index;not null;default:0"`
	CreatedAt               int64   `json:"created_at" gorm:"index"`
	ReviewedAt              int64   `json:"reviewed_at" gorm:"index;not null;default:0"`

	Username            string               `json:"username,omitempty" gorm:"column:username;->"`
	DisplayName         string               `json:"display_name,omitempty" gorm:"column:display_name;->"`
	ReviewerUsername    string               `json:"reviewer_username,omitempty" gorm:"column:reviewer_username;->"`
	ReviewerDisplayName string               `json:"reviewer_display_name,omitempty" gorm:"column:reviewer_display_name;->"`
	Items               []InvoiceRequestItem `json:"items" gorm:"foreignKey:InvoiceRequestId"`
}

type InvoiceRequestItem struct {
	Id               int     `json:"id"`
	InvoiceRequestId int     `json:"invoice_request_id" gorm:"index;not null"`
	UserId           int     `json:"user_id" gorm:"index;not null"`
	OrderType        string  `json:"order_type" gorm:"type:varchar(32);index;not null"`
	OrderId          int     `json:"order_id" gorm:"index;not null"`
	TradeNo          string  `json:"trade_no" gorm:"type:varchar(255);index;not null;default:''"`
	PaymentMethod    string  `json:"payment_method" gorm:"type:varchar(50);not null;default:''"`
	Amount           int64   `json:"amount" gorm:"type:bigint;not null;default:0"`
	Money            float64 `json:"money" gorm:"type:decimal(20,6);not null;default:0"`
	ProductName      string  `json:"product_name" gorm:"type:varchar(255);not null;default:''"`
	CreateTime       int64   `json:"create_time" gorm:"index"`
	CompleteTime     int64   `json:"complete_time" gorm:"index"`
}

type InvoiceOrderRef struct {
	OrderType string `json:"order_type"`
	Id        int    `json:"id"`
}

type CreateInvoiceRequestInput struct {
	InvoiceType             string
	TitleType               string
	Title                   string
	TaxNumber               string
	RegisteredAddress       string
	RegisteredPhone         string
	BankName                string
	BankAccount             string
	Email                   string
	Phone                   string
	Remark                  string
	NeedServiceConfirmation bool
	Orders                  []InvoiceOrderRef
}

type InvoiceRequestSearchParams struct {
	Status   string
	Username string
}

type InvoiceReviewInput struct {
	InvoiceNo         string
	InvoiceUrl        string
	InvoiceFileName   string
	InvoiceFilePath   string
	InvoiceSentTo     string
	InvoiceSendStatus string
	AdminRemark       string
}

func GetEligibleInvoiceOrders(userID int, pageInfo *common.PageInfo) ([]*PaymentRecord, int64, error) {
	if userID <= 0 {
		return nil, 0, errors.New("用户不存在")
	}
	if pageInfo == nil {
		pageInfo = &common.PageInfo{Page: 1, PageSize: common.ItemsPerPage}
	}

	topups, err := listEligibleInvoiceTopUpRecords(DB, userID)
	if err != nil {
		return nil, 0, err
	}
	walletPurchases, err := listEligibleInvoiceSellableTokenRecords(DB, userID)
	if err != nil {
		return nil, 0, err
	}
	records := mergePaymentRecordPage(nil, topups, walletPurchases)
	total := int64(len(records))
	start := pageInfo.GetStartIdx()
	if start >= len(records) {
		return []*PaymentRecord{}, total, nil
	}
	end := pageInfo.GetEndIdx()
	if end > len(records) {
		end = len(records)
	}
	return records[start:end], total, nil
}

func CreateInvoiceRequest(userID int, input CreateInvoiceRequestInput) (*InvoiceRequest, error) {
	if userID <= 0 {
		return nil, errors.New("用户不存在")
	}
	input = normalizeCreateInvoiceInput(input)
	if err := validateCreateInvoiceInput(input); err != nil {
		return nil, err
	}

	var request InvoiceRequest
	err := DB.Transaction(func(tx *gorm.DB) error {
		items := make([]InvoiceRequestItem, 0, len(input.Orders))
		seen := make(map[string]struct{}, len(input.Orders))
		var totalMoney float64
		var totalQuota int64

		for _, ref := range input.Orders {
			orderType := normalizeInvoiceOrderType(ref.OrderType)
			if orderType == "" || ref.Id <= 0 {
				return errors.New("订单参数错误")
			}
			key := fmt.Sprintf("%s:%d", orderType, ref.Id)
			if _, ok := seen[key]; ok {
				return errors.New("订单重复选择")
			}
			seen[key] = struct{}{}

			item, err := buildInvoiceRequestItemTx(tx, userID, orderType, ref.Id)
			if err != nil {
				return err
			}
			items = append(items, *item)
			totalMoney += item.Money
			totalQuota += item.Amount
		}

		request = InvoiceRequest{
			UserId:                  userID,
			InvoiceType:             input.InvoiceType,
			TitleType:               input.TitleType,
			Title:                   input.Title,
			TaxNumber:               input.TaxNumber,
			RegisteredAddress:       input.RegisteredAddress,
			RegisteredPhone:         input.RegisteredPhone,
			BankName:                input.BankName,
			BankAccount:             input.BankAccount,
			Email:                   input.Email,
			Phone:                   input.Phone,
			Remark:                  input.Remark,
			NeedServiceConfirmation: input.NeedServiceConfirmation,
			Status:                  InvoiceStatusPending,
			TotalMoney:              totalMoney,
			TotalQuota:              totalQuota,
			CreatedAt:               common.GetTimestamp(),
		}
		if err := tx.Create(&request).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].InvoiceRequestId = request.Id
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		request.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func GetUserInvoiceRequestsByParams(userID int, params InvoiceRequestSearchParams, pageInfo *common.PageInfo) ([]*InvoiceRequest, int64, error) {
	if userID <= 0 {
		return nil, 0, errors.New("用户不存在")
	}
	if pageInfo == nil {
		pageInfo = &common.PageInfo{Page: 1, PageSize: common.ItemsPerPage}
	}

	base := applyInvoiceRequestSearch(DB.Model(&InvoiceRequest{}).Where("user_id = ?", userID), params, false)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var requests []*InvoiceRequest
	query := applyInvoiceRequestSearch(DB.Model(&InvoiceRequest{}).Where("user_id = ?", userID), params, false)
	if err := query.Preload("Items").Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&requests).Error; err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

func GetAllInvoiceRequestsByParams(params InvoiceRequestSearchParams, pageInfo *common.PageInfo) ([]*InvoiceRequest, int64, error) {
	if pageInfo == nil {
		pageInfo = &common.PageInfo{Page: 1, PageSize: common.ItemsPerPage}
	}

	countQuery := applyInvoiceRequestSearch(invoiceRequestListWithUser(DB), params, true)
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var requests []*InvoiceRequest
	dataQuery := applyInvoiceRequestSearch(invoiceRequestListWithUser(DB), params, true)
	if err := dataQuery.Preload("Items").Order("invoice_requests.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&requests).Error; err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

func GetUserInvoiceRequestDetail(userID int, id int) (*InvoiceRequest, error) {
	if userID <= 0 || id <= 0 {
		return nil, ErrInvoiceRequestNotFound
	}

	var request InvoiceRequest
	err := invoiceRequestDetailWithUser(DB).
		Preload("Items").
		Where("invoice_requests.id = ? AND invoice_requests.user_id = ?", id, userID).
		First(&request).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceRequestNotFound
		}
		return nil, err
	}
	return &request, nil
}

func GetInvoiceRequestDetail(id int) (*InvoiceRequest, error) {
	if id <= 0 {
		return nil, ErrInvoiceRequestNotFound
	}

	var request InvoiceRequest
	err := invoiceRequestDetailWithUser(DB).
		Preload("Items").
		Where("invoice_requests.id = ?", id).
		First(&request).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceRequestNotFound
		}
		return nil, err
	}
	return &request, nil
}

func ApproveInvoiceRequest(id int, reviewerUserID int, input InvoiceReviewInput) (*InvoiceRequest, error) {
	input.InvoiceNo = strings.TrimSpace(input.InvoiceNo)
	input.InvoiceUrl = strings.TrimSpace(input.InvoiceUrl)
	input.InvoiceFileName = strings.TrimSpace(input.InvoiceFileName)
	input.InvoiceFilePath = strings.TrimSpace(input.InvoiceFilePath)
	input.InvoiceSentTo = strings.TrimSpace(input.InvoiceSentTo)
	input.InvoiceSendStatus = strings.TrimSpace(input.InvoiceSendStatus)
	input.AdminRemark = strings.TrimSpace(input.AdminRemark)
	if input.InvoiceNo == "" && input.InvoiceUrl == "" && input.InvoiceFilePath == "" {
		return nil, errors.New("发票号、发票链接或发票 PDF 不能为空")
	}
	return reviewInvoiceRequest(id, reviewerUserID, InvoiceStatusInvoiced, input)
}

func RejectInvoiceRequest(id int, reviewerUserID int, adminRemark string) (*InvoiceRequest, error) {
	remark := strings.TrimSpace(adminRemark)
	if remark == "" {
		return nil, errors.New("驳回原因不能为空")
	}
	return reviewInvoiceRequest(id, reviewerUserID, InvoiceStatusRejected, InvoiceReviewInput{AdminRemark: remark})
}

func normalizeCreateInvoiceInput(input CreateInvoiceRequestInput) CreateInvoiceRequestInput {
	input.InvoiceType = normalizeInvoiceType(input.InvoiceType)
	input.TitleType = normalizeInvoiceTitleType(input.TitleType)
	if input.InvoiceType == InvoiceTypeSpecial {
		input.TitleType = InvoiceTitleTypeCompany
	}
	input.Title = strings.TrimSpace(input.Title)
	input.TaxNumber = strings.TrimSpace(input.TaxNumber)
	input.RegisteredAddress = strings.TrimSpace(input.RegisteredAddress)
	input.RegisteredPhone = strings.TrimSpace(input.RegisteredPhone)
	input.BankName = strings.TrimSpace(input.BankName)
	input.BankAccount = strings.TrimSpace(input.BankAccount)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Remark = strings.TrimSpace(input.Remark)
	return input
}

func validateCreateInvoiceInput(input CreateInvoiceRequestInput) error {
	if len(input.Orders) == 0 {
		return errors.New("请选择需要开票的订单")
	}
	if len(input.Orders) > 100 {
		return errors.New("单次最多选择 100 笔订单")
	}
	if input.Title == "" {
		if input.InvoiceType == InvoiceTypeSpecial {
			return errors.New("单位名称不能为空")
		}
		return errors.New("发票抬头不能为空")
	}
	if len([]rune(input.Title)) > 128 {
		return errors.New("发票抬头不能超过 128 个字符")
	}
	if input.InvoiceType == InvoiceTypeSpecial && input.TaxNumber == "" {
		return errors.New("专票需要填写税号")
	}
	if input.TitleType == InvoiceTitleTypeCompany && input.TaxNumber == "" {
		return errors.New("企业抬头需要填写税号")
	}
	if len([]rune(input.TaxNumber)) > 64 {
		return errors.New("税号不能超过 64 个字符")
	}
	if len([]rune(input.RegisteredAddress)) > 255 {
		return errors.New("注册地址不能超过 255 个字符")
	}
	if len([]rune(input.RegisteredPhone)) > 64 {
		return errors.New("注册电话不能超过 64 个字符")
	}
	if len([]rune(input.BankName)) > 128 {
		return errors.New("开户银行不能超过 128 个字符")
	}
	if len([]rune(input.BankAccount)) > 128 {
		return errors.New("银行账号不能超过 128 个字符")
	}
	if input.Email == "" {
		return errors.New("接收邮箱不能为空")
	}
	if len([]rune(input.Email)) > 128 {
		return errors.New("接收邮箱不能超过 128 个字符")
	}
	if len([]rune(input.Phone)) > 32 {
		return errors.New("手机号不能超过 32 个字符")
	}
	if len([]rune(input.Remark)) > 1000 {
		return errors.New("备注不能超过 1000 个字符")
	}
	return nil
}

func normalizeInvoiceType(invoiceType string) string {
	switch strings.ToLower(strings.TrimSpace(invoiceType)) {
	case InvoiceTypeSpecial:
		return InvoiceTypeSpecial
	default:
		return InvoiceTypeNormal
	}
}

func normalizeInvoiceTitleType(titleType string) string {
	switch strings.ToLower(strings.TrimSpace(titleType)) {
	case InvoiceTitleTypeCompany:
		return InvoiceTitleTypeCompany
	default:
		return InvoiceTitleTypePersonal
	}
}

func normalizeInvoiceOrderType(orderType string) string {
	switch strings.ToLower(strings.TrimSpace(orderType)) {
	case PaymentRecordTypeTopUp:
		return PaymentRecordTypeTopUp
	case PaymentOrderTypeSubscription:
		return PaymentOrderTypeSubscription
	case PaymentRecordTypeSellableTokenPurchase:
		return PaymentRecordTypeSellableTokenPurchase
	default:
		return ""
	}
}

func buildInvoiceRequestItemTx(tx *gorm.DB, userID int, orderType string, orderID int) (*InvoiceRequestItem, error) {
	occupied, err := isInvoiceOrderOccupiedTx(tx, orderType, orderID)
	if err != nil {
		return nil, err
	}
	if occupied {
		return nil, errors.New("订单已存在待审核或已开票申请")
	}

	switch orderType {
	case PaymentRecordTypeTopUp, PaymentOrderTypeSubscription:
		return buildTopUpInvoiceItemTx(tx, userID, orderType, orderID)
	case PaymentRecordTypeSellableTokenPurchase:
		return buildSellableTokenInvoiceItemTx(tx, userID, orderID)
	default:
		return nil, errors.New("订单类型不支持")
	}
}

func buildTopUpInvoiceItemTx(tx *gorm.DB, userID int, orderType string, orderID int) (*InvoiceRequestItem, error) {
	var topup TopUp
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&topup, "id = ? AND user_id = ?", orderID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceOrderUnavailable
		}
		return nil, err
	}
	if topup.Status != common.TopUpStatusSuccess {
		return nil, ErrInvoiceOrderUnavailable
	}
	actualOrderType := resolveTopUpPaymentOrderType(&topup)
	if actualOrderType != orderType {
		return nil, errors.New("订单类型不匹配")
	}
	return &InvoiceRequestItem{
		UserId:        userID,
		OrderType:     orderType,
		OrderId:       topup.Id,
		TradeNo:       topup.TradeNo,
		PaymentMethod: topup.PaymentMethod,
		Amount:        topup.Amount,
		Money:         topup.Money,
		CreateTime:    topup.CreateTime,
		CompleteTime:  topup.CompleteTime,
	}, nil
}

func buildSellableTokenInvoiceItemTx(tx *gorm.DB, userID int, orderID int) (*InvoiceRequestItem, error) {
	var row sellableTokenPaymentRecordRow
	query := sellableTokenPaymentSelectQueryWithTx(tx, false).
		Where("sellable_token_orders.id = ? AND sellable_token_orders.user_id = ?", orderID, userID).
		Where("sellable_token_orders.status = ?", SellableTokenOrderStatusCompleted).
		Where("(sellable_token_issuances.status = ? OR sellable_token_issuances.id IS NULL)", SellableTokenIssuanceStatusIssued)
	if err := query.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceOrderUnavailable
		}
		return nil, err
	}
	return &InvoiceRequestItem{
		UserId:        userID,
		OrderType:     PaymentRecordTypeSellableTokenPurchase,
		OrderId:       row.Id,
		TradeNo:       formatSellableTokenPaymentTradeNo(row.TradeNo, row.UserId, row.Id),
		PaymentMethod: PaymentMethodWallet,
		Amount:        int64(row.PriceQuota),
		Money:         0,
		ProductName:   row.ProductName,
		CreateTime:    row.CreateTime,
		CompleteTime:  row.CompleteTime,
	}, nil
}

func isInvoiceOrderOccupiedTx(tx *gorm.DB, orderType string, orderID int) (bool, error) {
	if tx == nil {
		tx = DB
	}
	var count int64
	err := tx.Table("invoice_request_items").
		Joins("JOIN invoice_requests ON invoice_requests.id = invoice_request_items.invoice_request_id").
		Where("invoice_request_items.order_type = ? AND invoice_request_items.order_id = ?", orderType, orderID).
		Where("invoice_requests.status IN ?", invoiceOccupiedStatuses()).
		Count(&count).Error
	return count > 0, err
}

func listEligibleInvoiceTopUpRecords(tx *gorm.DB, userID int) ([]*PaymentRecord, error) {
	var topups []*TopUp
	excludeOrderTypes := []string{PaymentRecordTypeTopUp, PaymentOrderTypeSubscription}
	err := tx.Model(&TopUp{}).
		Where("user_id = ? AND status = ?", userID, common.TopUpStatusSuccess).
		Where(`NOT EXISTS (
			SELECT 1 FROM invoice_request_items
			JOIN invoice_requests ON invoice_requests.id = invoice_request_items.invoice_request_id
			WHERE invoice_request_items.order_type IN ?
			AND invoice_request_items.order_id = top_ups.id
			AND invoice_requests.status IN ?
		)`, excludeOrderTypes, invoiceOccupiedStatuses()).
		Order("create_time desc, id desc").
		Find(&topups).Error
	if err != nil {
		return nil, err
	}

	records := make([]*PaymentRecord, 0, len(topups))
	for _, topup := range topups {
		records = append(records, &PaymentRecord{
			Id:            topup.Id,
			RecordType:    PaymentRecordTypeTopUp,
			UserId:        topup.UserId,
			TradeNo:       topup.TradeNo,
			PaymentMethod: topup.PaymentMethod,
			Amount:        topup.Amount,
			Money:         topup.Money,
			Status:        topup.Status,
			CreateTime:    topup.CreateTime,
			CompleteTime:  topup.CompleteTime,
			OrderType:     resolveTopUpPaymentOrderType(topup),
		})
	}
	return records, nil
}

func listEligibleInvoiceSellableTokenRecords(tx *gorm.DB, userID int) ([]*PaymentRecord, error) {
	query := sellableTokenPaymentSelectQueryWithTx(tx, false).
		Where("sellable_token_orders.user_id = ?", userID).
		Where("sellable_token_orders.status = ?", SellableTokenOrderStatusCompleted).
		Where("(sellable_token_issuances.status = ? OR sellable_token_issuances.id IS NULL)", SellableTokenIssuanceStatusIssued).
		Where(`NOT EXISTS (
			SELECT 1 FROM invoice_request_items
			JOIN invoice_requests ON invoice_requests.id = invoice_request_items.invoice_request_id
			WHERE invoice_request_items.order_type = ?
			AND invoice_request_items.order_id = sellable_token_orders.id
			AND invoice_requests.status IN ?
		)`, PaymentRecordTypeSellableTokenPurchase, invoiceOccupiedStatuses()).
		Order("sellable_token_orders.create_time desc, sellable_token_orders.id desc")

	var rows []*sellableTokenPaymentRecordRow
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	records := make([]*PaymentRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, &PaymentRecord{
			Id:            row.Id,
			RecordType:    PaymentRecordTypeSellableTokenPurchase,
			UserId:        row.UserId,
			TradeNo:       formatSellableTokenPaymentTradeNo(row.TradeNo, row.UserId, row.Id),
			PaymentMethod: PaymentMethodWallet,
			Amount:        int64(row.PriceQuota),
			Money:         0,
			Status:        common.TopUpStatusSuccess,
			CreateTime:    row.CreateTime,
			CompleteTime:  row.CompleteTime,
			ProductId:     row.ProductId,
			ProductName:   row.ProductName,
			OrderType:     PaymentRecordTypeSellableTokenPurchase,
		})
	}
	return records, nil
}

func sellableTokenPaymentSelectQueryWithTx(tx *gorm.DB, includeUser bool) *gorm.DB {
	if tx == nil {
		tx = DB
	}
	selectClause := []string{
		"sellable_token_orders.id AS id",
		"sellable_token_orders.user_id AS user_id",
		"sellable_token_orders.product_id AS product_id",
		"sellable_token_products.name AS product_name",
		"sellable_token_orders.trade_no AS trade_no",
		"sellable_token_orders.price_quota AS price_quota",
		"sellable_token_orders.create_time AS create_time",
		"sellable_token_orders.complete_time AS complete_time",
		"sellable_token_issuances.status AS issuance_status",
	}
	query := tx.Table("sellable_token_orders").
		Joins("LEFT JOIN sellable_token_products ON sellable_token_products.id = sellable_token_orders.product_id").
		Joins("LEFT JOIN sellable_token_issuances ON sellable_token_issuances.source_id = sellable_token_orders.id AND sellable_token_issuances.source_type = ?", SellableTokenSourceTypeWallet)
	if includeUser {
		selectClause = append(selectClause, "users.username AS username", "users.display_name AS display_name")
		query = query.Joins("LEFT JOIN users ON users.id = sellable_token_orders.user_id")
	}
	return query.Select(strings.Join(selectClause, ", "))
}

func invoiceOccupiedStatuses() []string {
	return []string{InvoiceStatusPending, InvoiceStatusInvoiced}
}

func invoiceRequestListWithUser(tx *gorm.DB) *gorm.DB {
	return tx.Model(&InvoiceRequest{}).
		Select("invoice_requests.*, users.username AS username, users.display_name AS display_name").
		Joins("LEFT JOIN users ON users.id = invoice_requests.user_id")
}

func invoiceRequestDetailWithUser(tx *gorm.DB) *gorm.DB {
	return tx.Model(&InvoiceRequest{}).
		Select("invoice_requests.*, users.username AS username, users.display_name AS display_name, reviewers.username AS reviewer_username, reviewers.display_name AS reviewer_display_name").
		Joins("LEFT JOIN users ON users.id = invoice_requests.user_id").
		Joins("LEFT JOIN users AS reviewers ON reviewers.id = invoice_requests.reviewer_user_id")
}

func applyInvoiceRequestSearch(query *gorm.DB, params InvoiceRequestSearchParams, includeUsername bool) *gorm.DB {
	if params.Status != "" {
		query = query.Where("invoice_requests.status = ?", strings.TrimSpace(params.Status))
	}
	if includeUsername && strings.TrimSpace(params.Username) != "" {
		query = applyPaymentRecordUsernameFilter(query, params.Username, "users.id", "users.username")
	}
	return query
}

func reviewInvoiceRequest(id int, reviewerUserID int, targetStatus string, input InvoiceReviewInput) (*InvoiceRequest, error) {
	if id <= 0 {
		return nil, ErrInvoiceRequestNotFound
	}
	if reviewerUserID <= 0 {
		return nil, errors.New("审核人不存在")
	}
	if len([]rune(input.InvoiceNo)) > 128 {
		return nil, errors.New("发票号不能超过 128 个字符")
	}
	if len([]rune(input.InvoiceFileName)) > 255 {
		return nil, errors.New("发票文件名不能超过 255 个字符")
	}
	if len([]rune(input.InvoiceSentTo)) > 128 {
		return nil, errors.New("发票接收邮箱不能超过 128 个字符")
	}
	if input.InvoiceSendStatus != "" && !isValidInvoiceSendStatus(input.InvoiceSendStatus) {
		return nil, errors.New("邮件发送状态不合法")
	}
	if len([]rune(input.AdminRemark)) > 1000 {
		return nil, errors.New("审核备注不能超过 1000 个字符")
	}
	if targetStatus != InvoiceStatusInvoiced && targetStatus != InvoiceStatusRejected {
		return nil, errors.New("审核动作不合法")
	}

	var request InvoiceRequest
	now := common.GetTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Preload("Items").First(&request, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceRequestNotFound
			}
			return err
		}
		if request.Status != InvoiceStatusPending {
			return ErrInvoiceRequestAlreadyReviewed
		}
		updates := map[string]interface{}{
			"status":           targetStatus,
			"reviewer_user_id": reviewerUserID,
			"admin_remark":     input.AdminRemark,
			"reviewed_at":      now,
		}
		if targetStatus == InvoiceStatusInvoiced {
			updates["invoice_no"] = input.InvoiceNo
			updates["invoice_url"] = input.InvoiceUrl
			updates["invoice_file_name"] = input.InvoiceFileName
			updates["invoice_file_path"] = input.InvoiceFilePath
			updates["invoice_sent_to"] = input.InvoiceSentTo
			updates["invoice_send_status"] = input.InvoiceSendStatus
			updates["invoice_send_error"] = ""
		}
		result := tx.Model(&InvoiceRequest{}).Where("id = ? AND status = ?", id, InvoiceStatusPending).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrInvoiceRequestAlreadyReviewed
		}
		request.Status = targetStatus
		request.ReviewerUserId = reviewerUserID
		request.AdminRemark = input.AdminRemark
		request.ReviewedAt = now
		if targetStatus == InvoiceStatusInvoiced {
			request.InvoiceNo = input.InvoiceNo
			request.InvoiceUrl = input.InvoiceUrl
			request.InvoiceFileName = input.InvoiceFileName
			request.InvoiceFilePath = input.InvoiceFilePath
			request.InvoiceSentTo = input.InvoiceSentTo
			request.InvoiceSendStatus = input.InvoiceSendStatus
			request.InvoiceSendError = ""
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func UpdateInvoiceSendStatus(id int, status string, errorMessage string) (*InvoiceRequest, error) {
	status = strings.TrimSpace(status)
	errorMessage = strings.TrimSpace(errorMessage)
	if id <= 0 {
		return nil, ErrInvoiceRequestNotFound
	}
	if !isValidInvoiceSendStatus(status) {
		return nil, errors.New("邮件发送状态不合法")
	}
	if len([]rune(errorMessage)) > 1000 {
		errorMessage = string([]rune(errorMessage)[:1000])
	}
	updates := map[string]interface{}{
		"invoice_send_status": status,
		"invoice_send_error":  errorMessage,
	}
	if status == InvoiceSendStatusSent {
		updates["invoice_sent_at"] = common.GetTimestamp()
	} else {
		updates["invoice_sent_at"] = int64(0)
	}
	result := DB.Model(&InvoiceRequest{}).
		Where("id = ? AND status = ?", id, InvoiceStatusInvoiced).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrInvoiceRequestNotFound
	}
	return GetInvoiceRequestDetail(id)
}

func isValidInvoiceSendStatus(status string) bool {
	switch status {
	case InvoiceSendStatusPending, InvoiceSendStatusSent, InvoiceSendStatusFailed:
		return true
	default:
		return false
	}
}
