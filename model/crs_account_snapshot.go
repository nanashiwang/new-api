package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type CRSAccountSnapshot struct {
	Id                        int     `json:"id" gorm:"primaryKey;autoIncrement"`
	SiteID                    int     `json:"site_id" gorm:"not null;index:idx_crs_account_site;uniqueIndex:idx_crs_account_unique,priority:1"`
	RemoteAccountID           string  `json:"remote_account_id" gorm:"type:varchar(191);not null;uniqueIndex:idx_crs_account_unique,priority:2;index:idx_crs_account_remote"`
	Platform                  string  `json:"platform" gorm:"type:varchar(64);not null;index:idx_crs_account_platform"`
	Name                      string  `json:"name" gorm:"type:varchar(255);not null;default:'';index:idx_crs_account_name"`
	Description               string  `json:"description" gorm:"type:text;not null"`
	AccountType               string  `json:"account_type" gorm:"type:varchar(64);not null;default:''"`
	AuthType                  string  `json:"auth_type" gorm:"type:varchar(64);not null;default:''"`
	Status                    string  `json:"status" gorm:"type:varchar(64);not null;default:'';index:idx_crs_account_status"`
	ErrorMessage              string  `json:"error_message" gorm:"type:text;not null"`
	IsActive                  bool    `json:"is_active" gorm:"not null;default:false;index:idx_crs_account_active"`
	Schedulable               bool    `json:"schedulable" gorm:"not null;default:false;index:idx_crs_account_schedulable"`
	Priority                  int     `json:"priority" gorm:"not null;default:0"`
	RateLimited               bool    `json:"rate_limited" gorm:"not null;default:false;index:idx_crs_account_rate_limited"`
	RateLimitMinutesRemaining int     `json:"rate_limit_minutes_remaining" gorm:"not null;default:0"`
	RateLimitResetAt          string  `json:"rate_limit_reset_at" gorm:"type:varchar(64);not null;default:''"`
	SessionWindowActive       bool    `json:"session_window_active" gorm:"not null;default:false"`
	SessionWindowStatus       string  `json:"session_window_status" gorm:"type:varchar(64);not null;default:''"`
	SessionWindowProgress     float64 `json:"session_window_progress" gorm:"not null;default:0"`
	SessionWindowRemaining    string  `json:"session_window_remaining" gorm:"type:varchar(128);not null;default:''"`
	SessionWindowEndAt        string  `json:"session_window_end_at" gorm:"type:varchar(64);not null;default:''"`
	UsageWindowsJSON          string  `json:"usage_windows_json" gorm:"type:text"`
	SubscriptionPlan          string  `json:"subscription_plan" gorm:"type:varchar(64);not null;default:''"`
	SubscriptionInfo          string  `json:"subscription_info" gorm:"type:text;not null"`
	QuotaJSON                 string  `json:"quota_json" gorm:"type:text;not null"`
	QuotaUnlimited            bool    `json:"quota_unlimited" gorm:"not null;default:false"`
	QuotaTotal                float64 `json:"quota_total" gorm:"not null;default:0"`
	QuotaUsed                 float64 `json:"quota_used" gorm:"not null;default:0"`
	QuotaRemaining            float64 `json:"quota_remaining" gorm:"not null;default:0"`
	QuotaPercentage           float64 `json:"quota_percentage" gorm:"not null;default:0"`
	QuotaResetAt              string  `json:"quota_reset_at" gorm:"type:varchar(64);not null;default:''"`
	BalanceAmount             float64 `json:"balance_amount" gorm:"not null;default:0"`
	BalanceCurrency           string  `json:"balance_currency" gorm:"type:varchar(16);not null;default:''"`
	UsageDailyRequests        int64   `json:"usage_daily_requests" gorm:"bigint;not null;default:0"`
	UsageTotalRequests        int64   `json:"usage_total_requests" gorm:"bigint;not null;default:0"`
	UsageDailyTokens          int64   `json:"usage_daily_tokens" gorm:"bigint;not null;default:0"`
	UsageTotalTokens          int64   `json:"usage_total_tokens" gorm:"bigint;not null;default:0"`
	UsageRPM                  float64 `json:"usage_rpm" gorm:"not null;default:0"`
	UsageTPM                  float64 `json:"usage_tpm" gorm:"not null;default:0"`
	UsageDailyCost            float64 `json:"usage_daily_cost" gorm:"not null;default:0"`
	RawAccount                string  `json:"raw_account" gorm:"type:text;not null"`
	RawBalance                string  `json:"raw_balance" gorm:"type:text;not null"`
	SyncError                 string  `json:"sync_error" gorm:"type:text;not null"`
	LastSyncedAt              int64   `json:"last_synced_at" gorm:"bigint;not null;default:0;index:idx_crs_account_synced_at"`
	CreatedTime               int64   `json:"created_time" gorm:"bigint;not null;default:0"`
	UpdatedTime               int64   `json:"updated_time" gorm:"bigint;not null;default:0"`
}

type CRSUsageWindow struct {
	Key           string  `json:"key"`
	Label         string  `json:"label"`
	Progress      float64 `json:"progress"`
	RemainingText string  `json:"remaining_text"`
	ResetAt       string  `json:"reset_at"`
	Tone          string  `json:"tone"`
	Source        string  `json:"source"`
}

type CRSAccountSnapshotQuery struct {
	SiteID     int
	Platform   string
	Status     string
	Keyword    string
	QuotaState string
	Page       int
	PageSize   int
}

func (s *CRSAccountSnapshot) TableName() string {
	return "crs_account_snapshots"
}

func (s *CRSAccountSnapshot) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if s.CreatedTime <= 0 {
		s.CreatedTime = now
	}
	if s.UpdatedTime <= 0 {
		s.UpdatedTime = now
	}
	if strings.TrimSpace(s.UsageWindowsJSON) == "" {
		s.UsageWindowsJSON = "[]"
	}
	return nil
}

func (s *CRSAccountSnapshot) BeforeUpdate(tx *gorm.DB) error {
	if strings.TrimSpace(s.UsageWindowsJSON) == "" {
		s.UsageWindowsJSON = "[]"
	}
	s.UpdatedTime = common.GetTimestamp()
	return nil
}

func ReplaceCRSAccountSnapshots(siteID int, snapshots []*CRSAccountSnapshot) error {
	if siteID <= 0 {
		return errors.New("crs_account_snapshot:invalid_site_id")
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("site_id = ?", siteID).Delete(&CRSAccountSnapshot{}).Error; err != nil {
			return err
		}
		if len(snapshots) == 0 {
			return nil
		}
		for _, snapshot := range snapshots {
			if snapshot == nil {
				continue
			}
			snapshot.SiteID = siteID
		}
		return tx.Create(&snapshots).Error
	})
}

func ListCRSAccountSnapshotsBySite(siteID int) ([]*CRSAccountSnapshot, error) {
	result := make([]*CRSAccountSnapshot, 0)
	if siteID <= 0 {
		return result, nil
	}
	err := DB.Where("site_id = ?", siteID).
		Order("rate_limited DESC").
		Order("quota_remaining ASC").
		Order("name ASC").
		Find(&result).Error
	return result, err
}

func ListAllCRSAccountSnapshots() ([]*CRSAccountSnapshot, error) {
	result := make([]*CRSAccountSnapshot, 0)
	err := DB.Order("rate_limited DESC").
		Order("quota_remaining ASC").
		Order("updated_time DESC").
		Find(&result).Error
	return result, err
}

func QueryCRSAccountSnapshots(query CRSAccountSnapshotQuery) ([]*CRSAccountSnapshot, int64, error) {
	result := make([]*CRSAccountSnapshot, 0)
	db := DB.Model(&CRSAccountSnapshot{})

	if query.SiteID > 0 {
		db = db.Where("site_id = ?", query.SiteID)
	}
	if platform := strings.TrimSpace(query.Platform); platform != "" {
		db = db.Where("platform = ?", platform)
	}
	if status := strings.TrimSpace(query.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where(
			"name LIKE ? OR remote_account_id LIKE ? OR subscription_plan LIKE ?",
			like,
			like,
			like,
		)
	}
	switch strings.TrimSpace(query.QuotaState) {
	case "low":
		db = db.Where("quota_unlimited = ? AND quota_remaining > 0 AND quota_remaining <= ?", false, 10)
	case "empty":
		db = db.Where("quota_unlimited = ? AND quota_remaining <= ?", false, 0)
	case "unlimited":
		db = db.Where("quota_unlimited = ?", true)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return result, 0, err
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	err := db.Order("rate_limited DESC").
		Order("quota_remaining ASC").
		Order("updated_time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&result).Error
	return result, total, err
}
