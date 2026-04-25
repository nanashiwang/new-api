package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/QuantumNous/new-api/common"
)

type ChannelQuotaUsage struct {
	Id          int    `gorm:"primaryKey" json:"id"`
	Scope       string `gorm:"type:varchar(16);uniqueIndex:uniq_scope_period,priority:1" json:"scope"`
	ScopeKey    string `gorm:"type:varchar(255);uniqueIndex:uniq_scope_period,priority:2" json:"scope_key"`
	PeriodStart int64  `gorm:"uniqueIndex:uniq_scope_period,priority:3;index" json:"period_start"`
	PeriodEnd   int64  `gorm:"index" json:"period_end"`
	Period      string `gorm:"type:varchar(16)" json:"period"`
	UsedQuota   int64  `gorm:"default:0" json:"used_quota"`
	UsedCount   int64  `gorm:"default:0" json:"used_count"`
	Triggered   bool   `gorm:"default:false;index" json:"triggered"`
	TriggeredAt int64  `json:"triggered_at"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

func IncrChannelQuotaUsage(scope, scopeKey, period string, start, end int64, dq, dc int64) (int64, int64, error) {
	now := common.GetTimestamp()
	row := ChannelQuotaUsage{
		Scope: scope, ScopeKey: scopeKey, Period: period, PeriodStart: start, PeriodEnd: end,
		UsedQuota: dq, UsedCount: dc, CreatedAt: now, UpdatedAt: now,
	}
	err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "scope"}, {Name: "scope_key"}, {Name: "period_start"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"period_end": end,
			"period":     period,
			"used_quota": gorm.Expr("used_quota + ?", dq),
			"used_count": gorm.Expr("used_count + ?", dc),
			"updated_at": now,
		}),
	}).Create(&row).Error
	if err != nil {
		return 0, 0, err
	}
	var usage ChannelQuotaUsage
	if err := DB.Where("scope = ? AND scope_key = ? AND period_start = ?", scope, scopeKey, start).First(&usage).Error; err != nil {
		return 0, 0, err
	}
	return usage.UsedQuota, usage.UsedCount, nil
}

func MarkUsageTriggered(scope, scopeKey string, periodStart int64) (bool, error) {
	result := DB.Model(&ChannelQuotaUsage{}).
		Where("scope = ? AND scope_key = ? AND period_start = ? AND triggered = ?", scope, scopeKey, periodStart, false).
		Updates(map[string]interface{}{"triggered": true, "triggered_at": common.GetTimestamp()})
	return result.RowsAffected == 1, result.Error
}

func ListExpiredTriggeredUsages(now int64, limit int) ([]ChannelQuotaUsage, error) {
	var usages []ChannelQuotaUsage
	err := DB.Where("triggered = ? AND period_end <= ?", true, now).
		Order("period_end asc").
		Limit(limit).
		Find(&usages).Error
	return usages, err
}

func ClearTriggeredFlag(id int) error {
	return DB.Model(&ChannelQuotaUsage{}).Where("id = ?", id).Updates(map[string]interface{}{
		"triggered":    false,
		"triggered_at": 0,
		"updated_at":   common.GetTimestamp(),
	}).Error
}

func GetChannelQuotaUsage(scope, scopeKey, period string, start int64) (*ChannelQuotaUsage, error) {
	var usage ChannelQuotaUsage
	err := DB.Where("scope = ? AND scope_key = ? AND period = ? AND period_start = ?", scope, scopeKey, period, start).First(&usage).Error
	if err != nil {
		return nil, err
	}
	return &usage, nil
}
