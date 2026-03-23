package model

import (
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	PaymentRecordTypeTopUp                 = "topup"
	PaymentRecordTypeSellableTokenPurchase = "sellable_token_purchase"
	PaymentMethodWallet                    = "wallet"
	PaymentRecordStatusCancelled           = "cancelled"
)

type PaymentRecord struct {
	Id            int     `json:"id"`
	RecordType    string  `json:"record_type"`
	UserId        int     `json:"user_id"`
	Username      string  `json:"username,omitempty"`
	DisplayName   string  `json:"display_name,omitempty"`
	TradeNo       string  `json:"trade_no,omitempty"`
	PaymentMethod string  `json:"payment_method,omitempty"`
	Amount        int64   `json:"amount"`
	Money         float64 `json:"money"`
	Status        string  `json:"status"`
	CreateTime    int64   `json:"create_time"`
	CompleteTime  int64   `json:"complete_time"`
	ProductId     int     `json:"product_id,omitempty"`
	ProductName   string  `json:"product_name,omitempty"`
}

type PaymentRecordSearchParams struct {
	Keyword       string
	Username      string
	Status        string
	PaymentMethod string
}

type sellableTokenPaymentRecordRow struct {
	Id             int    `gorm:"column:id"`
	UserId         int    `gorm:"column:user_id"`
	ProductId      int    `gorm:"column:product_id"`
	ProductName    string `gorm:"column:product_name"`
	PriceQuota     int    `gorm:"column:price_quota"`
	CreateTime     int64  `gorm:"column:create_time"`
	CompleteTime   int64  `gorm:"column:complete_time"`
	IssuanceStatus string `gorm:"column:issuance_status"`
	Username       string `gorm:"column:username"`
	DisplayName    string `gorm:"column:display_name"`
}

func GetUserPaymentRecordsByParams(userId int, params PaymentRecordSearchParams, pageInfo *common.PageInfo) ([]*PaymentRecord, int64, error) {
	if userId <= 0 {
		return []*PaymentRecord{}, 0, nil
	}

	topupTotal, err := countTopUpPaymentRecords(&userId, params, false)
	if err != nil {
		return nil, 0, err
	}
	walletTotal, err := countSellableTokenPaymentRecords(&userId, params, false)
	if err != nil {
		return nil, 0, err
	}

	fetchLimit := paymentRecordFetchLimit(pageInfo)
	topups, err := listTopUpPaymentRecords(&userId, params, fetchLimit, false)
	if err != nil {
		return nil, 0, err
	}
	walletPurchases, err := listSellableTokenPaymentRecords(&userId, params, fetchLimit, false)
	if err != nil {
		return nil, 0, err
	}

	total := topupTotal + walletTotal
	return mergePaymentRecordPage(pageInfo, topups, walletPurchases), total, nil
}

func GetAllPaymentRecordsByParams(params PaymentRecordSearchParams, pageInfo *common.PageInfo) ([]*PaymentRecord, int64, error) {
	topupTotal, err := countTopUpPaymentRecords(nil, params, true)
	if err != nil {
		return nil, 0, err
	}
	walletTotal, err := countSellableTokenPaymentRecords(nil, params, true)
	if err != nil {
		return nil, 0, err
	}

	fetchLimit := paymentRecordFetchLimit(pageInfo)
	topups, err := listTopUpPaymentRecords(nil, params, fetchLimit, true)
	if err != nil {
		return nil, 0, err
	}
	walletPurchases, err := listSellableTokenPaymentRecords(nil, params, fetchLimit, true)
	if err != nil {
		return nil, 0, err
	}

	total := topupTotal + walletTotal
	return mergePaymentRecordPage(pageInfo, topups, walletPurchases), total, nil
}

func paymentRecordFetchLimit(pageInfo *common.PageInfo) int {
	if pageInfo == nil {
		return common.ItemsPerPage
	}
	fetchLimit := pageInfo.GetEndIdx()
	if fetchLimit < pageInfo.GetPageSize() {
		fetchLimit = pageInfo.GetPageSize()
	}
	if fetchLimit <= 0 {
		fetchLimit = common.ItemsPerPage
	}
	return fetchLimit
}

func mergePaymentRecordPage(pageInfo *common.PageInfo, recordGroups ...[]*PaymentRecord) []*PaymentRecord {
	merged := make([]*PaymentRecord, 0)
	for _, group := range recordGroups {
		merged = append(merged, group...)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return paymentRecordLess(merged[i], merged[j])
	})

	if pageInfo == nil {
		return merged
	}
	start := pageInfo.GetStartIdx()
	if start >= len(merged) {
		return []*PaymentRecord{}
	}
	end := pageInfo.GetEndIdx()
	if end > len(merged) {
		end = len(merged)
	}
	if end < start {
		end = start
	}
	return merged[start:end]
}

func paymentRecordLess(left *PaymentRecord, right *PaymentRecord) bool {
	if left == nil || right == nil {
		return left != nil
	}
	if left.CreateTime != right.CreateTime {
		return left.CreateTime > right.CreateTime
	}
	if left.CompleteTime != right.CompleteTime {
		return left.CompleteTime > right.CompleteTime
	}
	if left.RecordType != right.RecordType {
		return left.RecordType < right.RecordType
	}
	return left.Id > right.Id
}

func countTopUpPaymentRecords(userId *int, params PaymentRecordSearchParams, includeUser bool) (int64, error) {
	query := DB.Model(&TopUp{})
	if userId != nil {
		query = query.Where("user_id = ?", *userId)
	}
	query = applyTopUpSearch(query, toTopUpSearchParams(params), includeUser)
	if includeUser {
		query = query.Joins("LEFT JOIN users ON users.id = top_ups.user_id")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func listTopUpPaymentRecords(userId *int, params PaymentRecordSearchParams, limit int, includeUser bool) ([]*PaymentRecord, error) {
	query := topUpBaseQuery(DB, includeUser)
	if userId != nil {
		query = query.Where("top_ups.user_id = ?", *userId)
	}
	query = applyTopUpSearch(query, toTopUpSearchParams(params), includeUser)
	query = query.Order("top_ups.create_time desc, top_ups.id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}

	var topups []*TopUp
	if err := query.Find(&topups).Error; err != nil {
		return nil, err
	}

	records := make([]*PaymentRecord, 0, len(topups))
	for _, topup := range topups {
		records = append(records, &PaymentRecord{
			Id:            topup.Id,
			RecordType:    PaymentRecordTypeTopUp,
			UserId:        topup.UserId,
			Username:      topup.Username,
			DisplayName:   topup.DisplayName,
			TradeNo:       topup.TradeNo,
			PaymentMethod: topup.PaymentMethod,
			Amount:        topup.Amount,
			Money:         topup.Money,
			Status:        topup.Status,
			CreateTime:    topup.CreateTime,
			CompleteTime:  topup.CompleteTime,
		})
	}
	return records, nil
}

func countSellableTokenPaymentRecords(userId *int, params PaymentRecordSearchParams, includeUser bool) (int64, error) {
	query := sellableTokenPaymentQuery(includeUser)
	if userId != nil {
		query = query.Where("sellable_token_orders.user_id = ?", *userId)
	}
	query = applySellableTokenPaymentSearch(query, params, includeUser)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func listSellableTokenPaymentRecords(userId *int, params PaymentRecordSearchParams, limit int, includeUser bool) ([]*PaymentRecord, error) {
	query := sellableTokenPaymentSelectQuery(includeUser)
	if userId != nil {
		query = query.Where("sellable_token_orders.user_id = ?", *userId)
	}
	query = applySellableTokenPaymentSearch(query, params, includeUser)
	query = query.Order("sellable_token_orders.create_time desc, sellable_token_orders.id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}

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
			Username:      row.Username,
			DisplayName:   row.DisplayName,
			PaymentMethod: PaymentMethodWallet,
			Amount:        int64(row.PriceQuota),
			Money:         0,
			Status:        toSellableTokenPaymentStatus(row.IssuanceStatus),
			CreateTime:    row.CreateTime,
			CompleteTime:  row.CompleteTime,
			ProductId:     row.ProductId,
			ProductName:   row.ProductName,
		})
	}
	return records, nil
}

func toTopUpSearchParams(params PaymentRecordSearchParams) TopUpSearchParams {
	return TopUpSearchParams{
		Keyword:       strings.TrimSpace(params.Keyword),
		Username:      strings.TrimSpace(params.Username),
		Status:        strings.TrimSpace(params.Status),
		PaymentMethod: strings.TrimSpace(params.PaymentMethod),
	}
}

func sellableTokenPaymentQuery(includeUser bool) *gorm.DB {
	query := DB.Table("sellable_token_orders").
		Joins("LEFT JOIN sellable_token_products ON sellable_token_products.id = sellable_token_orders.product_id").
		Joins("LEFT JOIN sellable_token_issuances ON sellable_token_issuances.source_id = sellable_token_orders.id AND sellable_token_issuances.source_type = ?", SellableTokenSourceTypeWallet)
	if includeUser {
		query = query.Joins("LEFT JOIN users ON users.id = sellable_token_orders.user_id")
	}
	return query
}

func sellableTokenPaymentSelectQuery(includeUser bool) *gorm.DB {
	selectClause := []string{
		"sellable_token_orders.id AS id",
		"sellable_token_orders.user_id AS user_id",
		"sellable_token_orders.product_id AS product_id",
		"sellable_token_products.name AS product_name",
		"sellable_token_orders.price_quota AS price_quota",
		"sellable_token_orders.create_time AS create_time",
		"sellable_token_orders.complete_time AS complete_time",
		"sellable_token_issuances.status AS issuance_status",
	}
	if includeUser {
		selectClause = append(selectClause,
			"users.username AS username",
			"users.display_name AS display_name",
		)
	}
	return sellableTokenPaymentQuery(includeUser).Select(strings.Join(selectClause, ", "))
}

func applySellableTokenPaymentSearch(query *gorm.DB, params PaymentRecordSearchParams, includeUsername bool) *gorm.DB {
	keyword := strings.TrimSpace(params.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		keywordQuery := DB.Where("sellable_token_products.name LIKE ?", like)
		if orderID, ok := parseSellableTokenOrderKeyword(keyword); ok {
			keywordQuery = keywordQuery.Or("sellable_token_orders.id = ?", orderID)
		}
		query = query.Where(keywordQuery)
	}

	status := strings.TrimSpace(params.Status)
	switch status {
	case common.TopUpStatusPending:
		query = query.Where("sellable_token_issuances.status = ?", SellableTokenIssuanceStatusPending)
	case common.TopUpStatusSuccess:
		query = query.Where("(sellable_token_issuances.status = ? OR sellable_token_issuances.id IS NULL)", SellableTokenIssuanceStatusIssued)
	case PaymentRecordStatusCancelled:
		query = query.Where("sellable_token_issuances.status = ?", SellableTokenIssuanceStatusCancelled)
	case common.TopUpStatusExpired:
		query = query.Where("1 = 0")
	}

	paymentMethod := strings.TrimSpace(params.PaymentMethod)
	if paymentMethod != "" {
		if paymentMethod != PaymentMethodWallet {
			return query.Where("1 = 0")
		}
	}

	if includeUsername {
		username := strings.TrimSpace(params.Username)
		if username != "" {
			if uid, err := strconv.Atoi(username); err == nil {
				query = query.Where("users.id = ?", uid)
			} else {
				like := "%" + username + "%"
				query = query.Where("users.username LIKE ?", like)
			}
		}
	}

	return query
}

func parseSellableTokenOrderKeyword(keyword string) (int, bool) {
	normalized := strings.TrimSpace(keyword)
	upper := strings.ToUpper(normalized)
	if strings.HasPrefix(upper, "STO-") && len(normalized) > 4 {
		normalized = strings.TrimSpace(normalized[4:])
	}
	orderID, err := strconv.Atoi(normalized)
	if err != nil || orderID <= 0 {
		return 0, false
	}
	return orderID, true
}

func toSellableTokenPaymentStatus(issuanceStatus string) string {
	switch issuanceStatus {
	case SellableTokenIssuanceStatusPending:
		return common.TopUpStatusPending
	case SellableTokenIssuanceStatusCancelled:
		return PaymentRecordStatusCancelled
	default:
		return common.TopUpStatusSuccess
	}
}
