package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	ContentSafetyNotificationPending = "pending"
	ContentSafetyNotificationSending = "sending"
	ContentSafetyNotificationSent    = "sent"
	ContentSafetyNotificationFailed  = "failed"
	ContentSafetyNotificationSkipped = "skipped"
)

type ContentSafetyEvidence struct {
	Id           int64  `json:"id"`
	ViolationId  int64  `json:"violation_id" gorm:"uniqueIndex"`
	UserId       int    `json:"user_id" gorm:"index"`
	Version      string `json:"version" gorm:"type:varchar(32)"`
	Ciphertext   []byte `json:"-"`
	Nonce        []byte `json:"-"`
	EvidenceHash string `json:"-" gorm:"type:char(64)"`
	RoleSummary  string `json:"role_summary" gorm:"type:varchar(128)"`
	SizeBytes    int    `json:"size_bytes"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index"`
	ExpiresAt    int64  `json:"expires_at" gorm:"bigint;index"`
}

type ContentSafetyNotification struct {
	Id              int64  `json:"id"`
	ViolationId     int64  `json:"violation_id" gorm:"index"`
	UserId          int    `json:"user_id" gorm:"index;index:idx_safety_notification_user_created,priority:1"`
	DeliveryKey     string `json:"-" gorm:"type:char(64);uniqueIndex"`
	Kind            string `json:"kind" gorm:"type:varchar(32)"`
	Recipient       string `json:"-" gorm:"type:varchar(254)"`
	RecipientSource string `json:"recipient_source" gorm:"type:varchar(16)"`
	TemplateVersion string `json:"template_version" gorm:"type:varchar(32)"`
	Status          string `json:"status" gorm:"type:varchar(16);index"`
	Attempts        int    `json:"attempts"`
	LastError       string `json:"last_error" gorm:"type:varchar(256)"`
	CreatedAt       int64  `json:"created_at" gorm:"bigint;index;index:idx_safety_notification_user_created,priority:2"`
	UpdatedAt       int64  `json:"updated_at" gorm:"bigint"`
	SentAt          int64  `json:"sent_at" gorm:"bigint"`
}

func GetContentSafetyNotificationIdentity(userID int) (email, username string, err error) {
	var user User
	err = DB.Select("email", "username").First(&user, userID).Error
	return user.Email, user.Username, err
}

func CreateContentSafetyEvidence(evidence *ContentSafetyEvidence) error {
	if evidence == nil || evidence.ViolationId <= 0 || evidence.UserId <= 0 || len(evidence.Ciphertext) == 0 {
		return errors.New("invalid content safety evidence")
	}
	return DB.Create(evidence).Error
}

func GetContentSafetyEvidenceByViolation(violationID int64) (*ContentSafetyEvidence, *ContentSafetyViolation, error) {
	var evidence ContentSafetyEvidence
	if err := DB.Where("violation_id = ?", violationID).First(&evidence).Error; err != nil {
		return nil, nil, err
	}
	var violation ContentSafetyViolation
	if err := DB.First(&violation, violationID).Error; err != nil {
		return nil, nil, err
	}
	return &evidence, &violation, nil
}

func CreateContentSafetyNotification(notification *ContentSafetyNotification, since int64, limit int64) (bool, error) {
	if notification == nil || notification.DeliveryKey == "" || notification.ViolationId <= 0 || notification.UserId <= 0 {
		return false, errors.New("invalid content safety notification")
	}
	if limit <= 0 {
		return false, errors.New("invalid content safety notification limit")
	}
	created := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		lock := tx.Model(&User{}).Where("id = ?", notification.UserId).UpdateColumn("status", gorm.Expr("status"))
		if lock.Error != nil {
			return lock.Error
		}
		var existing int64
		if err := tx.Model(&ContentSafetyNotification{}).Where("delivery_key = ?", notification.DeliveryKey).Count(&existing).Error; err != nil || existing > 0 {
			return err
		}
		var recent int64
		if err := tx.Model(&ContentSafetyNotification{}).
			Where("user_id = ? AND created_at >= ? AND status IN ?", notification.UserId, since,
				[]string{ContentSafetyNotificationPending, ContentSafetyNotificationSending, ContentSafetyNotificationSent}).Count(&recent).Error; err != nil {
			return err
		}
		if recent >= limit {
			return nil
		}
		if err := tx.Create(notification).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

func ClaimContentSafetyNotification(id int64, now int64) (bool, error) {
	result := DB.Model(&ContentSafetyNotification{}).
		Where("id = ? AND status IN ?", id, []string{ContentSafetyNotificationPending, ContentSafetyNotificationFailed}).
		Updates(map[string]any{"status": ContentSafetyNotificationSending, "attempts": gorm.Expr("attempts + 1"), "updated_at": now})
	return result.RowsAffected == 1, result.Error
}

func FinishContentSafetyNotification(id int64, status, lastError string, now int64) error {
	updates := map[string]any{"status": status, "last_error": truncateSafetyAuditValue(lastError, 256), "updated_at": now}
	if status == ContentSafetyNotificationSent {
		updates["sent_at"] = now
	}
	return DB.Model(&ContentSafetyNotification{}).Where("id = ?", id).Updates(updates).Error
}

func GetRetryableContentSafetyNotifications(before int64, limit int) ([]ContentSafetyNotification, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	items := make([]ContentSafetyNotification, 0, limit)
	err := DB.Where("attempts < ? AND ((status = ?) OR (status = ? AND updated_at <= ?))", 3,
		ContentSafetyNotificationPending, ContentSafetyNotificationFailed, before).
		Order("id ASC").Limit(limit).Find(&items).Error
	return items, err
}

func RecoverStaleContentSafetyNotifications(before int64) error {
	return DB.Model(&ContentSafetyNotification{}).
		Where("status = ? AND updated_at <= ?", ContentSafetyNotificationSending, before).
		Updates(map[string]any{"status": ContentSafetyNotificationFailed, "last_error": "delivery interrupted"}).Error
}

func GetContentSafetyViolationByID(id int64) (*ContentSafetyViolation, error) {
	var violation ContentSafetyViolation
	err := DB.First(&violation, id).Error
	return &violation, err
}

func DeleteExpiredContentSafetyEvidence(now time.Time) error {
	return DB.Where("expires_at > 0 AND expires_at <= ?", now.Unix()).Delete(&ContentSafetyEvidence{}).Error
}
