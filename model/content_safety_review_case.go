package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	ContentSafetyReviewPending         = "pending"
	ContentSafetyReviewApprovedDisable = "approved_disable"
	ContentSafetyReviewDismissed       = "dismissed"
	ContentSafetyReviewObserving       = "observing"
)

type ContentSafetyReviewCase struct {
	Id                  int64  `json:"id"`
	UserId              int    `json:"user_id" gorm:"index;index:idx_safety_review_user_status,priority:1"`
	Username            string `json:"username" gorm:"->"`
	Status              string `json:"status" gorm:"type:varchar(32);index;index:idx_safety_review_user_status,priority:2"`
	TriggerViolationId  int64  `json:"trigger_violation_id" gorm:"uniqueIndex"`
	WindowEventCount    int    `json:"window_event_count"`
	WindowCooldownCount int    `json:"window_cooldown_count"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint;index"`
	ReviewedAt          int64  `json:"reviewed_at" gorm:"bigint"`
	ReviewerId          int    `json:"reviewer_id" gorm:"index"`
	ReviewNote          string `json:"review_note" gorm:"type:varchar(512)"`
}

type ContentSafetyReviewCaseQuery struct {
	UserId int
	Status string
}

func createPendingContentSafetyReviewCase(tx *gorm.DB, violation *ContentSafetyViolation, cooldownCount int) (*ContentSafetyReviewCase, error) {
	if tx == nil || violation == nil {
		return nil, errors.New("invalid content safety review case")
	}
	var existing ContentSafetyReviewCase
	err := tx.Where("user_id = ? AND status = ?", violation.UserId, ContentSafetyReviewPending).
		Order("id DESC").First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	created := &ContentSafetyReviewCase{
		UserId:              violation.UserId,
		Status:              ContentSafetyReviewPending,
		TriggerViolationId:  violation.Id,
		WindowEventCount:    violation.WindowCount,
		WindowCooldownCount: cooldownCount,
		CreatedAt:           violation.CreatedAt,
	}
	if err = tx.Create(created).Error; err != nil {
		return nil, err
	}
	return created, nil
}

func GetContentSafetyReviewCases(query ContentSafetyReviewCaseQuery, pageInfo *common.PageInfo) ([]ContentSafetyReviewCase, int64, error) {
	if pageInfo == nil {
		pageInfo = &common.PageInfo{Page: 1, PageSize: common.ItemsPerPage}
	}
	base := DB.Table("content_safety_review_cases AS cases").
		Joins("LEFT JOIN users ON users.id = cases.user_id")
	if query.UserId > 0 {
		base = base.Where("cases.user_id = ?", query.UserId)
	}
	if status := strings.TrimSpace(query.Status); status != "" {
		base = base.Where("cases.status = ?", status)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]ContentSafetyReviewCase, 0, pageInfo.GetPageSize())
	if err := base.Select("cases.*, users.username AS username").
		Order("cases.id DESC").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).
		Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func ResolveContentSafetyReviewCase(caseID int64, reviewerID int, resolution string, note string) (*ContentSafetyReviewCase, bool, error) {
	if caseID <= 0 || reviewerID <= 0 {
		return nil, false, errors.New("invalid content safety review identity")
	}
	resolution = strings.ToLower(strings.TrimSpace(resolution))
	switch resolution {
	case ContentSafetyReviewApprovedDisable, ContentSafetyReviewDismissed, ContentSafetyReviewObserving:
	default:
		return nil, false, errors.New("invalid content safety review resolution")
	}
	note = truncateSafetyAuditValue(note, 512)
	if resolution == ContentSafetyReviewApprovedDisable && note == "" {
		return nil, false, errors.New("content safety disable approval requires a review note")
	}

	var resolved ContentSafetyReviewCase
	userDisabled := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		caseLock := tx.Model(&ContentSafetyReviewCase{}).Where("id = ?", caseID).
			UpdateColumn("status", gorm.Expr("status"))
		if caseLock.Error != nil {
			return caseLock.Error
		}
		if err := tx.Where("id = ?", caseID).First(&resolved).Error; err != nil {
			return err
		}
		if resolved.Status != ContentSafetyReviewPending {
			return errors.New("content safety review case already resolved")
		}
		lockResult := tx.Model(&User{}).Where("id = ?", resolved.UserId).
			UpdateColumn("status", gorm.Expr("status"))
		if lockResult.Error != nil {
			return lockResult.Error
		}
		if resolution == ContentSafetyReviewApprovedDisable {
			if err := tx.Model(&User{}).Where("id = ?", resolved.UserId).
				UpdateColumn("status", common.UserStatusDisabled).Error; err != nil {
				return err
			}
			userDisabled = true
		}
		resolved.Status = resolution
		resolved.ReviewedAt = time.Now().Unix()
		resolved.ReviewerId = reviewerID
		resolved.ReviewNote = note
		return tx.Model(&resolved).Updates(map[string]any{
			"status":      resolved.Status,
			"reviewed_at": resolved.ReviewedAt,
			"reviewer_id": resolved.ReviewerId,
			"review_note": resolved.ReviewNote,
		}).Error
	})
	if err != nil {
		return nil, false, err
	}
	return &resolved, userDisabled, nil
}
