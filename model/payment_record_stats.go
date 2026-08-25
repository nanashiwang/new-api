package model

import (
	"sort"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type PaymentRecordStatsItem struct {
	Money      float64            `json:"money"`
	Amounts    map[string]float64 `json:"amounts"`
	OrderCount int64              `json:"order_count"`
}

type PaymentRecordStats struct {
	Totals         PaymentRecordStatsItem            `json:"totals"`
	Statuses       map[string]PaymentRecordStatsItem `json:"statuses"`
	PaymentMethods map[string]PaymentRecordStatsItem `json:"payment_methods"`
}

type PaymentRecordRanking struct {
	UserId           int                `json:"user_id"`
	Username         string             `json:"username"`
	DisplayName      string             `json:"display_name,omitempty"`
	Money            float64            `json:"money"`
	Amounts          map[string]float64 `json:"amounts"`
	OrderCount       int64              `json:"order_count"`
	SuccessMoney     float64            `json:"success_money"`
	SuccessAmounts   map[string]float64 `json:"success_amounts"`
	PendingMoney     float64            `json:"pending_money"`
	PendingAmounts   map[string]float64 `json:"pending_amounts"`
	ExpiredMoney     float64            `json:"expired_money"`
	ExpiredAmounts   map[string]float64 `json:"expired_amounts"`
	CancelledMoney   float64            `json:"cancelled_money"`
	CancelledAmounts map[string]float64 `json:"cancelled_amounts"`
}

type paymentRecordAggregateRow struct {
	MetricKey  string  `gorm:"column:metric_key"`
	Currency   string  `gorm:"column:currency"`
	Money      float64 `gorm:"column:money"`
	OrderCount int64   `gorm:"column:order_count"`
}

type paymentRecordAggregateTotalRow struct {
	Money      float64 `gorm:"column:money"`
	OrderCount int64   `gorm:"column:order_count"`
}

type paymentRecordRankingRow struct {
	UserId         int     `gorm:"column:user_id"`
	Username       string  `gorm:"column:username"`
	DisplayName    string  `gorm:"column:display_name"`
	Currency       string  `gorm:"column:currency"`
	Money          float64 `gorm:"column:money"`
	OrderCount     int64   `gorm:"column:order_count"`
	SuccessMoney   float64 `gorm:"column:success_money"`
	PendingMoney   float64 `gorm:"column:pending_money"`
	ExpiredMoney   float64 `gorm:"column:expired_money"`
	CancelledMoney float64 `gorm:"column:cancelled_money"`
}

func newPaymentRecordStats() PaymentRecordStats {
	return PaymentRecordStats{
		Totals: PaymentRecordStatsItem{Amounts: map[string]float64{}},
		Statuses: map[string]PaymentRecordStatsItem{
			common.TopUpStatusSuccess:    {Amounts: map[string]float64{}},
			common.TopUpStatusPending:    {Amounts: map[string]float64{}},
			common.TopUpStatusExpired:    {Amounts: map[string]float64{}},
			PaymentRecordStatusCancelled: {Amounts: map[string]float64{}},
		},
		PaymentMethods: map[string]PaymentRecordStatsItem{},
	}
}

func mergePaymentRecordStatsItem(current PaymentRecordStatsItem, delta paymentRecordAggregateRow) PaymentRecordStatsItem {
	if current.Amounts == nil {
		current.Amounts = map[string]float64{}
	}
	currency := NormalizePaymentCurrency(delta.Currency)
	if currency != "" {
		current.Amounts[currency] += delta.Money
		if currency == "CNY" {
			current.Money += delta.Money
		}
	}
	current.OrderCount += delta.OrderCount
	return current
}

func GetPaymentRecordStats(params PaymentRecordSearchParams) (PaymentRecordStats, error) {
	stats := newPaymentRecordStats()

	topupStatusRows, err := queryTopUpPaymentRecordAggregateRows(params, "top_ups.status", "top_ups.status")
	if err != nil {
		return stats, err
	}
	walletStatusRows, err := querySellableTokenPaymentStatusRows(params)
	if err != nil {
		return stats, err
	}
	for _, row := range append(topupStatusRows, walletStatusRows...) {
		stats.Statuses[row.MetricKey] = mergePaymentRecordStatsItem(stats.Statuses[row.MetricKey], row)
		stats.Totals = mergePaymentRecordStatsItem(stats.Totals, row)
	}

	topupMethodRows, err := queryTopUpPaymentRecordAggregateRows(params, "top_ups.payment_method", "top_ups.payment_method")
	if err != nil {
		return stats, err
	}
	walletMethodRows, err := querySellableTokenPaymentMethodRows(params)
	if err != nil {
		return stats, err
	}
	for _, row := range append(topupMethodRows, walletMethodRows...) {
		stats.PaymentMethods[row.MetricKey] = mergePaymentRecordStatsItem(stats.PaymentMethods[row.MetricKey], row)
	}

	return stats, nil
}

func GetPaymentRecordRankings(params PaymentRecordSearchParams, limit int) ([]PaymentRecordRanking, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	topupRows, err := queryTopUpPaymentRecordRankingRows(params)
	if err != nil {
		return nil, err
	}
	walletRows, err := querySellableTokenPaymentRankingRows(params)
	if err != nil {
		return nil, err
	}

	merged := make(map[int]*PaymentRecordRanking, len(topupRows)+len(walletRows))
	mergeRow := func(row paymentRecordRankingRow) {
		entry, ok := merged[row.UserId]
		if !ok {
			entry = &PaymentRecordRanking{
				UserId:           row.UserId,
				Amounts:          map[string]float64{},
				SuccessAmounts:   map[string]float64{},
				PendingAmounts:   map[string]float64{},
				ExpiredAmounts:   map[string]float64{},
				CancelledAmounts: map[string]float64{},
			}
			merged[row.UserId] = entry
		}
		if entry.Username == "" {
			entry.Username = row.Username
		}
		if entry.DisplayName == "" {
			entry.DisplayName = row.DisplayName
		}
		entry.OrderCount += row.OrderCount
		currency := NormalizePaymentCurrency(row.Currency)
		if currency != "" {
			entry.Amounts[currency] += row.Money
			entry.SuccessAmounts[currency] += row.SuccessMoney
			entry.PendingAmounts[currency] += row.PendingMoney
			entry.ExpiredAmounts[currency] += row.ExpiredMoney
			entry.CancelledAmounts[currency] += row.CancelledMoney
			if currency == "CNY" {
				entry.Money += row.Money
				entry.SuccessMoney += row.SuccessMoney
				entry.PendingMoney += row.PendingMoney
				entry.ExpiredMoney += row.ExpiredMoney
				entry.CancelledMoney += row.CancelledMoney
			}
		}
	}

	for _, row := range topupRows {
		mergeRow(row)
	}
	for _, row := range walletRows {
		mergeRow(row)
	}

	rankings := make([]PaymentRecordRanking, 0, len(merged))
	for _, row := range merged {
		rankings = append(rankings, *row)
	}

	sort.SliceStable(rankings, func(i, j int) bool {
		if rankings[i].Money != rankings[j].Money {
			return rankings[i].Money > rankings[j].Money
		}
		if rankings[i].OrderCount != rankings[j].OrderCount {
			return rankings[i].OrderCount > rankings[j].OrderCount
		}
		return rankings[i].UserId < rankings[j].UserId
	})

	if len(rankings) > limit {
		rankings = rankings[:limit]
	}
	return rankings, nil
}

func paymentRecordTopUpAggregateQuery() *gorm.DB {
	return applyPaymentDashboardTopUpRiskFilter(
		DB.Table("top_ups").
			Joins("LEFT JOIN users ON users.id = top_ups.user_id"),
	)
}

func applyPaymentDashboardTopUpRiskFilter(query *gorm.DB) *gorm.DB {
	if query == nil {
		return nil
	}
	// 对账看板只保留“可结算”订单：待处理、已回退、已作废的异常单都不再计入；
	// 已确认表示审核通过，仍然保留在看板口径中。
	return query.Where(
		`NOT EXISTS (
			SELECT 1
			FROM payment_risk_cases
			WHERE payment_risk_cases.trade_no = top_ups.trade_no
			  AND payment_risk_cases.status IN (?, ?, ?)
			  AND payment_risk_cases.record_type IN (?, ?)
		)`,
		PaymentRiskStatusOpen,
		PaymentRiskStatusReversed,
		PaymentRiskStatusVoided,
		PaymentRiskRecordTypeTopUp,
		PaymentRiskRecordTypeSubscription,
	)
}

func stripeTopUpSQLCondition() string {
	return "(LOWER(COALESCE(top_ups.payment_provider, '')) = 'stripe' OR " +
		"(COALESCE(top_ups.payment_provider, '') = '' AND LOWER(COALESCE(top_ups.payment_method, '')) = 'stripe'))"
}

func topUpEffectivePaymentMoneyExpr() string {
	return "CASE " +
		"WHEN top_ups.presentment_money > 0 AND LENGTH(TRIM(COALESCE(top_ups.presentment_currency, ''))) = 3 THEN top_ups.presentment_money " +
		"WHEN top_ups.paid_money > 0 THEN top_ups.paid_money " +
		"WHEN NOT " + stripeTopUpSQLCondition() + " AND top_ups.money > 0 THEN top_ups.money " +
		"ELSE 0 END"
}

func topUpEffectivePaymentCurrencyExpr() string {
	return "CASE " +
		"WHEN top_ups.presentment_money > 0 AND LENGTH(TRIM(COALESCE(top_ups.presentment_currency, ''))) = 3 THEN UPPER(TRIM(top_ups.presentment_currency)) " +
		"WHEN top_ups.paid_money > 0 AND LENGTH(TRIM(COALESCE(top_ups.paid_currency, ''))) = 3 THEN UPPER(TRIM(top_ups.paid_currency)) " +
		"WHEN top_ups.paid_money > 0 AND NOT " + stripeTopUpSQLCondition() + " THEN 'CNY' " +
		"WHEN NOT " + stripeTopUpSQLCondition() + " AND top_ups.money > 0 THEN 'CNY' " +
		"ELSE '' END"
}

func topUpCNYPaymentMoneyExpr() string {
	return "CASE " +
		"WHEN top_ups.presentment_money > 0 AND UPPER(TRIM(COALESCE(top_ups.presentment_currency, ''))) IN ('CNY', 'RMB') THEN top_ups.presentment_money " +
		"WHEN top_ups.presentment_money > 0 THEN 0 " +
		"WHEN top_ups.paid_money > 0 AND UPPER(TRIM(COALESCE(top_ups.paid_currency, ''))) IN ('CNY', 'RMB') THEN top_ups.paid_money " +
		"WHEN top_ups.paid_money > 0 AND TRIM(COALESCE(top_ups.paid_currency, '')) <> '' THEN 0 " +
		"WHEN top_ups.paid_money > 0 AND " + stripeTopUpSQLCondition() + " THEN 0 " +
		"WHEN top_ups.paid_money > 0 THEN top_ups.paid_money " +
		"WHEN NOT " + stripeTopUpSQLCondition() + " AND top_ups.money > 0 THEN top_ups.money " +
		"ELSE 0 END"
}

func queryTopUpPaymentRecordAggregateRows(params PaymentRecordSearchParams, keyExpr string, groupExpr string) ([]paymentRecordAggregateRow, error) {
	rows := make([]paymentRecordAggregateRow, 0)
	query := paymentRecordTopUpAggregateQuery()
	query, err := applyTopUpSearch(query, toTopUpSearchParams(params), true)
	if err != nil {
		return nil, err
	}
	moneyExpr := topUpEffectivePaymentMoneyExpr()
	currencyExpr := topUpEffectivePaymentCurrencyExpr()
	if err := query.
		Select(keyExpr + " AS metric_key, " + currencyExpr + " AS currency, COALESCE(SUM(" + moneyExpr + "), 0) AS money, COUNT(*) AS order_count").
		Group(groupExpr + ", " + currencyExpr).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func querySellableTokenPaymentStatusRows(params PaymentRecordSearchParams) ([]paymentRecordAggregateRow, error) {
	rows := make([]paymentRecordAggregateRow, 0)
	statusExpr := sellableTokenPaymentStatusExpr()
	query := sellableTokenPaymentQuery(true)
	query, err := applySellableTokenPaymentSearch(query, params, true)
	if err != nil {
		return nil, err
	}
	if err := query.
		Select(statusExpr + " AS metric_key, 'CNY' AS currency, 0 AS money, COUNT(*) AS order_count").
		Group(statusExpr).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func querySellableTokenPaymentMethodRows(params PaymentRecordSearchParams) ([]paymentRecordAggregateRow, error) {
	rows := make([]paymentRecordAggregateRow, 0)
	query := sellableTokenPaymentQuery(true)
	query, err := applySellableTokenPaymentSearch(query, params, true)
	if err != nil {
		return nil, err
	}
	total := paymentRecordAggregateTotalRow{}
	if err := query.
		Select("0 AS money, COUNT(*) AS order_count").
		Scan(&total).Error; err != nil {
		return nil, err
	}
	if total.OrderCount == 0 {
		return rows, nil
	}
	rows = append(rows, paymentRecordAggregateRow{
		MetricKey:  PaymentMethodWallet,
		Currency:   "CNY",
		Money:      0,
		OrderCount: total.OrderCount,
	})
	return rows, nil
}

func queryTopUpPaymentRecordRankingRows(params PaymentRecordSearchParams) ([]paymentRecordRankingRow, error) {
	rows := make([]paymentRecordRankingRow, 0)
	query := paymentRecordTopUpAggregateQuery()
	query, err := applyTopUpSearch(query, toTopUpSearchParams(params), true)
	if err != nil {
		return nil, err
	}
	moneyExpr := topUpEffectivePaymentMoneyExpr()
	currencyExpr := topUpEffectivePaymentCurrencyExpr()
	if err := query.
		Select(
			"top_ups.user_id AS user_id, users.username AS username, users.display_name AS display_name, " +
				currencyExpr + " AS currency, " +
				"COALESCE(SUM(" + moneyExpr + "), 0) AS money, COUNT(*) AS order_count, " +
				"COALESCE(SUM(CASE WHEN top_ups.status = '" + common.TopUpStatusSuccess + "' THEN " + moneyExpr + " ELSE 0 END), 0) AS success_money, " +
				"COALESCE(SUM(CASE WHEN top_ups.status = '" + common.TopUpStatusPending + "' THEN " + moneyExpr + " ELSE 0 END), 0) AS pending_money, " +
				"COALESCE(SUM(CASE WHEN top_ups.status = '" + common.TopUpStatusExpired + "' THEN " + moneyExpr + " ELSE 0 END), 0) AS expired_money, " +
				"COALESCE(SUM(CASE WHEN top_ups.status = '" + PaymentRecordStatusCancelled + "' THEN " + moneyExpr + " ELSE 0 END), 0) AS cancelled_money",
		).
		Group("top_ups.user_id, users.username, users.display_name, " + currencyExpr).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func querySellableTokenPaymentRankingRows(params PaymentRecordSearchParams) ([]paymentRecordRankingRow, error) {
	rows := make([]paymentRecordRankingRow, 0)
	query := sellableTokenPaymentQuery(true)
	query, err := applySellableTokenPaymentSearch(query, params, true)
	if err != nil {
		return nil, err
	}
	if err := query.
		Select(
			"sellable_token_orders.user_id AS user_id, users.username AS username, users.display_name AS display_name, " +
				"'CNY' AS currency, 0 AS money, COUNT(*) AS order_count, 0 AS success_money, 0 AS pending_money, 0 AS expired_money, 0 AS cancelled_money",
		).
		Group("sellable_token_orders.user_id, users.username, users.display_name").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
