package model

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

type AdminLogQueryFilters struct {
	LogType        int
	StartTimestamp int64
	EndTimestamp   int64
	ModelName      string
	Username       string
	TokenName      string
	Channel        int
	Group          string
	RequestID      string
}

type TopUserStat struct {
	UserID           int    `json:"user_id" gorm:"column:user_id"`
	Username         string `json:"username" gorm:"column:username"`
	Group            string `json:"group" gorm:"column:user_group"`
	Quota            int    `json:"quota" gorm:"column:quota"`
	RequestCount     int    `json:"request_count" gorm:"column:request_count"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"column:prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens" gorm:"column:completion_tokens"`
}

func buildContainsLikePattern(input string) string {
	input = strings.TrimSpace(input)
	input = strings.ReplaceAll(input, "!", "!!")
	input = strings.ReplaceAll(input, "%", "!%")
	input = strings.ReplaceAll(input, "_", "!_")
	return "%" + input + "%"
}

// sanitizeContainsLikePattern 校验并生成带通配的模糊匹配 LIKE 模式。
// 返回值：
//   - ("", nil)：输入为空，调用方应跳过 LIKE 过滤
//   - (pattern, nil)：已转义并前后追加 %，配合 `LIKE ? ESCAPE '!'` 使用
//   - ("", error)：输入不合法（少于 2 个有效字符），错误消息可直接展示给用户
func sanitizeContainsLikePattern(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", nil
	}
	if len([]rune(trimmed)) < 2 {
		return "", errors.New("搜索关键词至少需要 2 个字符")
	}
	return buildContainsLikePattern(trimmed), nil
}

func applyAdminLogFilters(tx *gorm.DB, filters AdminLogQueryFilters, fuzzyUsername bool) *gorm.DB {
	if filters.LogType != LogTypeUnknown {
		tx = tx.Where("logs.type = ?", filters.LogType)
	}
	if filters.ModelName != "" {
		tx = tx.Where("logs.model_name like ?", filters.ModelName)
	}
	if filters.Username != "" {
		if fuzzyUsername {
			tx = tx.Where("logs.username LIKE ? ESCAPE '!'", buildContainsLikePattern(filters.Username))
		} else {
			tx = tx.Where("logs.username = ?", filters.Username)
		}
	}
	if filters.TokenName != "" {
		tx = tx.Where("logs.token_name = ?", filters.TokenName)
	}
	if filters.RequestID != "" {
		tx = tx.Where("logs.request_id = ?", filters.RequestID)
	}
	if filters.StartTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", filters.StartTimestamp)
	}
	if filters.EndTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", filters.EndTimestamp)
	}
	if filters.Channel != 0 {
		tx = tx.Where("logs.channel_id = ?", filters.Channel)
	}
	if filters.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", filters.Group)
	}
	return tx
}

func normalizeRankingOrder(order string) string {
	switch strings.ToLower(strings.TrimSpace(order)) {
	case "asc":
		return "asc"
	default:
		return "desc"
	}
}

func GetTopUsers(filters AdminLogQueryFilters, limit int, quotaOrder string, requestOrder string) ([]TopUserStat, []TopUserStat, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	quotaOrder = normalizeRankingOrder(quotaOrder)
	requestOrder = normalizeRankingOrder(requestOrder)

	baseQuery := LOG_DB.Table("logs")
	baseQuery = applyAdminLogFilters(baseQuery, filters, true)

	selectClause := "logs.user_id, logs.username, " + logGroupCol + " as user_group, " +
		"COUNT(*) as request_count, " +
		"COALESCE(SUM(quota), 0) as quota, " +
		"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) as completion_tokens"
	groupClause := "logs.user_id, logs.username, " + logGroupCol

	byQuota := make([]TopUserStat, 0, limit)
	quotaQuery := baseQuery.Session(&gorm.Session{}).
		Select(selectClause).
		Group(groupClause).
		Order("quota " + quotaOrder).
		Order("request_count " + quotaOrder).
		Order("logs.user_id asc").
		Limit(limit)
	if err := quotaQuery.Scan(&byQuota).Error; err != nil {
		return nil, nil, err
	}

	byRequests := make([]TopUserStat, 0, limit)
	requestQuery := baseQuery.Session(&gorm.Session{}).
		Select(selectClause).
		Group(groupClause).
		Order("request_count " + requestOrder).
		Order("quota " + requestOrder).
		Order("logs.user_id asc").
		Limit(limit)
	if err := requestQuery.Scan(&byRequests).Error; err != nil {
		return nil, nil, err
	}

	return byQuota, byRequests, nil
}
