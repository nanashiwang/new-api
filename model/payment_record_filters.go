package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

func parseExplicitUserIDSearch(raw string) (int, bool) {
	normalized := strings.TrimSpace(strings.ReplaceAll(raw, "：", ":"))
	if normalized == "" {
		return 0, false
	}
	lower := strings.ToLower(normalized)
	if !strings.HasPrefix(lower, "id:") {
		return 0, false
	}
	value := strings.TrimSpace(normalized[len("id:"):])
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func applyPaymentRecordUsernameFilter(query *gorm.DB, raw string, idColumn string, usernameColumn string) *gorm.DB {
	username := strings.TrimSpace(raw)
	if username == "" {
		return query
	}

	if explicitID, ok := parseExplicitUserIDSearch(username); ok {
		return query.Where(idColumn+" = ?", explicitID)
	}

	if numericID, err := strconv.Atoi(username); err == nil && numericID > 0 {
		pattern := buildContainsLikePattern(username)
		return query.Where(
			query.Where(idColumn+" = ?", numericID).
				Or(usernameColumn+" LIKE ? ESCAPE '!'", pattern),
		)
	}

	pattern := buildContainsLikePattern(username)
	return query.Where(usernameColumn+" LIKE ? ESCAPE '!'", pattern)
}

func buildPaymentRecordEffectiveTimestampExpr(statusExpr string, createTimeCol string, completeTimeCol string) string {
	return fmt.Sprintf(
		"CASE WHEN (%s) = '%s' AND %s > 0 THEN %s ELSE %s END",
		statusExpr,
		common.TopUpStatusSuccess,
		completeTimeCol,
		completeTimeCol,
		createTimeCol,
	)
}

func applyPaymentRecordTimeRange(query *gorm.DB, startTimestamp int64, endTimestamp int64, effectiveTimestampExpr string) *gorm.DB {
	if startTimestamp != 0 {
		query = query.Where(effectiveTimestampExpr+" >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		query = query.Where(effectiveTimestampExpr+" <= ?", endTimestamp)
	}
	return query
}
