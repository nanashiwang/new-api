package model

import (
	"sort"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type PaymentRecordStatsItem struct {
	Money      float64 `json:"money"`
	OrderCount int64   `json:"order_count"`
}

type PaymentRecordStats struct {
	Totals         PaymentRecordStatsItem            `json:"totals"`
	Statuses       map[string]PaymentRecordStatsItem `json:"statuses"`
	PaymentMethods map[string]PaymentRecordStatsItem `json:"payment_methods"`
}

type PaymentRecordRanking struct {
	UserId         int     `json:"user_id"`
	Username       string  `json:"username"`
	DisplayName    string  `json:"display_name,omitempty"`
	Money          float64 `json:"money"`
	OrderCount     int64   `json:"order_count"`
	SuccessMoney   float64 `json:"success_money"`
	PendingMoney   float64 `json:"pending_money"`
	ExpiredMoney   float64 `json:"expired_money"`
	CancelledMoney float64 `json:"cancelled_money"`
}

type paymentRecordAggregateRow struct {
	MetricKey  string  `gorm:"column:metric_key"`
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
	Money          float64 `gorm:"column:money"`
	OrderCount     int64   `gorm:"column:order_count"`
	SuccessMoney   float64 `gorm:"column:success_money"`
	PendingMoney   float64 `gorm:"column:pending_money"`
	ExpiredMoney   float64 `gorm:"column:expired_money"`
	CancelledMoney float64 `gorm:"column:cancelled_money"`
}

func newPaymentRecordStats() PaymentRecordStats {
	return PaymentRecordStats{
		Statuses: map[string]PaymentRecordStatsItem{
			common.TopUpStatusSuccess:    {},
			common.TopUpStatusPending:    {},
			common.TopUpStatusExpired:    {},
			PaymentRecordStatusCancelled: {},
		},
		PaymentMethods: map[string]PaymentRecordStatsItem{},
	}
}

func mergePaymentRecordStatsItem(current PaymentRecordStatsItem, delta paymentRecordAggregateRow) PaymentRecordStatsItem {
	current.Money += delta.Money
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
		stats.Totals.Money += row.Money
		stats.Totals.OrderCount += row.OrderCount
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
				UserId: row.UserId,
			}
			merged[row.UserId] = entry
		}
		if entry.Username == "" {
			entry.Username = row.Username
		}
		if entry.DisplayName == "" {
			entry.DisplayName = row.DisplayName
		}
		entry.Money += row.Money
		entry.OrderCount += row.OrderCount
		entry.SuccessMoney += row.SuccessMoney
		entry.PendingMoney += row.PendingMoney
		entry.ExpiredMoney += row.ExpiredMoney
		entry.CancelledMoney += row.CancelledMoney
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
	return DB.Table("top_ups").
		Joins("LEFT JOIN users ON users.id = top_ups.user_id")
}

func queryTopUpPaymentRecordAggregateRows(params PaymentRecordSearchParams, keyExpr string, groupExpr string) ([]paymentRecordAggregateRow, error) {
	rows := make([]paymentRecordAggregateRow, 0)
	query := paymentRecordTopUpAggregateQuery()
	query = applyTopUpSearch(query, toTopUpSearchParams(params), true)
	if err := query.
		Select(keyExpr + " AS metric_key, COALESCE(SUM(top_ups.money), 0) AS money, COUNT(*) AS order_count").
		Group(groupExpr).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func querySellableTokenPaymentStatusRows(params PaymentRecordSearchParams) ([]paymentRecordAggregateRow, error) {
	rows := make([]paymentRecordAggregateRow, 0)
	statusExpr := sellableTokenPaymentStatusExpr()
	query := sellableTokenPaymentQuery(true)
	query = applySellableTokenPaymentSearch(query, params, true)
	if err := query.
		Select(statusExpr + " AS metric_key, 0 AS money, COUNT(*) AS order_count").
		Group(statusExpr).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func querySellableTokenPaymentMethodRows(params PaymentRecordSearchParams) ([]paymentRecordAggregateRow, error) {
	rows := make([]paymentRecordAggregateRow, 0)
	query := sellableTokenPaymentQuery(true)
	query = applySellableTokenPaymentSearch(query, params, true)
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
		Money:      0,
		OrderCount: total.OrderCount,
	})
	return rows, nil
}

func queryTopUpPaymentRecordRankingRows(params PaymentRecordSearchParams) ([]paymentRecordRankingRow, error) {
	rows := make([]paymentRecordRankingRow, 0)
	query := paymentRecordTopUpAggregateQuery()
	query = applyTopUpSearch(query, toTopUpSearchParams(params), true)
	if err := query.
		Select(
			"top_ups.user_id AS user_id, users.username AS username, users.display_name AS display_name, " +
				"COALESCE(SUM(top_ups.money), 0) AS money, COUNT(*) AS order_count, " +
				"COALESCE(SUM(CASE WHEN top_ups.status = '" + common.TopUpStatusSuccess + "' THEN top_ups.money ELSE 0 END), 0) AS success_money, " +
				"COALESCE(SUM(CASE WHEN top_ups.status = '" + common.TopUpStatusPending + "' THEN top_ups.money ELSE 0 END), 0) AS pending_money, " +
				"COALESCE(SUM(CASE WHEN top_ups.status = '" + common.TopUpStatusExpired + "' THEN top_ups.money ELSE 0 END), 0) AS expired_money, " +
				"COALESCE(SUM(CASE WHEN top_ups.status = '" + PaymentRecordStatusCancelled + "' THEN top_ups.money ELSE 0 END), 0) AS cancelled_money",
		).
		Group("top_ups.user_id, users.username, users.display_name").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func querySellableTokenPaymentRankingRows(params PaymentRecordSearchParams) ([]paymentRecordRankingRow, error) {
	rows := make([]paymentRecordRankingRow, 0)
	query := sellableTokenPaymentQuery(true)
	query = applySellableTokenPaymentSearch(query, params, true)
	if err := query.
		Select(
			"sellable_token_orders.user_id AS user_id, users.username AS username, users.display_name AS display_name, " +
				"0 AS money, COUNT(*) AS order_count, 0 AS success_money, 0 AS pending_money, 0 AS expired_money, 0 AS cancelled_money",
		).
		Group("sellable_token_orders.user_id, users.username, users.display_name").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
