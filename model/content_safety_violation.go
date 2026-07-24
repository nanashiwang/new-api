package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	ContentSafetyActionWarning         = "warning"
	ContentSafetyActionDisabled        = "disabled"
	ContentSafetyActionAlreadyDisabled = "already_disabled"
	ContentSafetyActionReviewRequired  = "review_required"
)

type ContentSafetyViolation struct {
	Id          int64  `json:"id"`
	UserId      int    `json:"user_id" gorm:"index;index:idx_safety_user_created,priority:1"`
	Username    string `json:"username" gorm:"->"`
	TokenId     int    `json:"token_id" gorm:"index"`
	ChannelId   int    `json:"channel_id" gorm:"index"`
	RequestId   string `json:"request_id" gorm:"type:varchar(64);index"`
	EventKey    string `json:"-" gorm:"type:char(64);uniqueIndex"`
	ModelName   string `json:"model_name" gorm:"type:varchar(128);index"`
	ErrorType   string `json:"error_type" gorm:"type:varchar(64)"`
	ErrorCode   string `json:"error_code" gorm:"type:varchar(64);index"`
	InputHash   string `json:"-" gorm:"type:char(64)"`
	IsStream    bool   `json:"is_stream"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index;index:idx_safety_user_created,priority:2"`
	WindowCount int    `json:"window_count"`
	Action      string `json:"action" gorm:"type:varchar(32);index"`
}

type RecordContentSafetyViolationParams struct {
	UserId       int
	TokenId      int
	ChannelId    int
	RequestId    string
	EventKey     string
	ModelName    string
	ErrorType    string
	ErrorCode    string
	InputHash    string
	IsStream     bool
	CreatedAt    int64
	WindowStart  int64
	DisableAfter int
}

type ContentSafetyEnforcementResult struct {
	Violation  *ContentSafetyViolation
	Duplicate  bool
	UserStatus int
	UserRole   int
	Username   string
}

func RecordContentSafetyViolation(params RecordContentSafetyViolationParams) (*ContentSafetyEnforcementResult, error) {
	if params.UserId <= 0 || params.EventKey == "" || params.ErrorCode == "" {
		return nil, errors.New("invalid content safety violation identity")
	}
	if params.DisableAfter <= 0 {
		return nil, errors.New("invalid content safety disable threshold")
	}

	result := &ContentSafetyEnforcementResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		// A harmless row update serializes enforcement for one user on SQLite,
		// MySQL, and PostgreSQL, preventing concurrent fourth events from escaping.
		lockResult := tx.Model(&User{}).
			Where("id = ?", params.UserId).
			UpdateColumn("status", gorm.Expr("status"))
		if lockResult.Error != nil {
			return lockResult.Error
		}

		var user User
		if err := tx.Select("id", "username", "role", "status").First(&user, params.UserId).Error; err != nil {
			return err
		}
		result.UserStatus = user.Status
		result.UserRole = user.Role
		result.Username = user.Username

		var existing ContentSafetyViolation
		if err := tx.Where("event_key = ?", params.EventKey).First(&existing).Error; err == nil {
			result.Violation = &existing
			result.Duplicate = true
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		violation := &ContentSafetyViolation{
			UserId:    params.UserId,
			TokenId:   params.TokenId,
			ChannelId: params.ChannelId,
			RequestId: truncateSafetyAuditValue(params.RequestId, 64),
			EventKey:  params.EventKey,
			ModelName: truncateSafetyAuditValue(params.ModelName, 128),
			ErrorType: truncateSafetyAuditValue(params.ErrorType, 64),
			ErrorCode: truncateSafetyAuditValue(params.ErrorCode, 64),
			InputHash: params.InputHash,
			IsStream:  params.IsStream,
			CreatedAt: params.CreatedAt,
		}
		if err := tx.Create(violation).Error; err != nil {
			return err
		}

		var windowCount int64
		if err := tx.Model(&ContentSafetyViolation{}).
			Where("user_id = ? AND created_at >= ?", params.UserId, params.WindowStart).
			Count(&windowCount).Error; err != nil {
			return err
		}
		violation.WindowCount = int(windowCount)

		violation.Action = ContentSafetyActionWarning
		if violation.WindowCount >= params.DisableAfter {
			if user.Role >= common.RoleAdminUser {
				violation.Action = ContentSafetyActionReviewRequired
			} else if user.Status == common.UserStatusDisabled {
				violation.Action = ContentSafetyActionAlreadyDisabled
			} else {
				if err := tx.Model(&User{}).Where("id = ?", user.Id).
					UpdateColumn("status", common.UserStatusDisabled).Error; err != nil {
					return err
				}
				user.Status = common.UserStatusDisabled
				result.UserStatus = user.Status
				violation.Action = ContentSafetyActionDisabled
			}
		}

		if err := tx.Model(violation).Updates(map[string]any{
			"window_count": violation.WindowCount,
			"action":       violation.Action,
		}).Error; err != nil {
			return err
		}
		violation.Username = user.Username
		result.Violation = violation
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func truncateSafetyAuditValue(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

type ContentSafetyViolationQuery struct {
	UserId         int
	Username       string
	ErrorCode      string
	Action         string
	RequestId      string
	StartTimestamp int64
	EndTimestamp   int64
}

func GetContentSafetyViolations(query ContentSafetyViolationQuery, pageInfo *common.PageInfo) ([]ContentSafetyViolation, int64, error) {
	if pageInfo == nil {
		pageInfo = &common.PageInfo{Page: 1, PageSize: common.ItemsPerPage}
	}
	base := DB.Table("content_safety_violations AS violations").
		Joins("LEFT JOIN users ON users.id = violations.user_id")
	if query.UserId > 0 {
		base = base.Where("violations.user_id = ?", query.UserId)
	}
	if query.Username != "" {
		base = base.Where("users.username = ?", strings.TrimSpace(query.Username))
	}
	if query.ErrorCode != "" {
		base = base.Where("violations.error_code = ?", strings.ToLower(strings.TrimSpace(query.ErrorCode)))
	}
	if query.Action != "" {
		base = base.Where("violations.action = ?", strings.ToLower(strings.TrimSpace(query.Action)))
	}
	if query.RequestId != "" {
		base = base.Where("violations.request_id = ?", strings.TrimSpace(query.RequestId))
	}
	if query.StartTimestamp > 0 {
		base = base.Where("violations.created_at >= ?", query.StartTimestamp)
	}
	if query.EndTimestamp > 0 {
		base = base.Where("violations.created_at <= ?", query.EndTimestamp)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]ContentSafetyViolation, 0, pageInfo.GetPageSize())
	if err := base.Select("violations.*, users.username AS username").
		Order("violations.id DESC").
		Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).
		Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
