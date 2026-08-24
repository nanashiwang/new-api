package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RedemptionReviewStatusPending   = "pending"
	RedemptionReviewStatusDismissed = "dismissed"
	RedemptionReviewStatusDisabled  = "disabled"

	RedemptionReviewActionDismiss = "dismiss"
	RedemptionReviewActionDisable = "disable"
)

var (
	ErrRedemptionReviewCaseNotFound = errors.New("redemption review case not found")
	ErrRedemptionReviewResolved     = errors.New("redemption review case already resolved")
)

var walletRedemptionLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}()

type RedemptionReviewCase struct {
	Id                   int    `json:"id"`
	UserId               int    `json:"user_id" gorm:"index;uniqueIndex:idx_redemption_review_user_date,priority:1"`
	BizDate              string `json:"biz_date" gorm:"type:varchar(10);uniqueIndex:idx_redemption_review_user_date,priority:2"`
	Status               string `json:"status" gorm:"type:varchar(16);index"`
	DistinctCreatorCount int    `json:"distinct_creator_count"`
	SmallCodeCount       int    `json:"small_code_count"`
	TotalQuota           int    `json:"total_quota"`
	CreatorIds           string `json:"creator_ids" gorm:"type:text"`
	RedemptionIds        string `json:"redemption_ids" gorm:"type:text"`
	TriggerRedemptionId  int    `json:"trigger_redemption_id" gorm:"index"`
	ReviewerId           int    `json:"reviewer_id" gorm:"index"`
	ReviewNote           string `json:"review_note" gorm:"type:varchar(512)"`
	ReviewedAt           int64  `json:"reviewed_at" gorm:"bigint"`
	CreatedAt            int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt            int64  `json:"updated_at" gorm:"bigint"`
}

func (review *RedemptionReviewCase) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if review.CreatedAt <= 0 {
		review.CreatedAt = now
	}
	review.UpdatedAt = now
	if review.Status == "" {
		review.Status = RedemptionReviewStatusPending
	}
	return nil
}

func (review *RedemptionReviewCase) BeforeUpdate(_ *gorm.DB) error {
	review.UpdatedAt = common.GetTimestamp()
	return nil
}

func walletRedemptionLocalDayRange(now time.Time) (string, int64, int64) {
	local := now.In(walletRedemptionLocation)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	return start.Format("2006-01-02"), start.Unix(), start.AddDate(0, 0, 1).Unix()
}

func walletRedemptionQuotaFromUnits(units int) (int, error) {
	if units < 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) || common.QuotaPerUnit <= 0 {
		return 0, errors.New("invalid wallet redemption quota setting")
	}
	value := float64(units) * common.QuotaPerUnit
	if math.IsNaN(value) || math.IsInf(value, 0) || value >= float64(math.MaxInt) {
		return 0, errors.New("wallet redemption quota setting exceeds integer range")
	}
	return int(math.Round(value)), nil
}

func recordWalletRedemptionReviewTx(tx *gorm.DB, redemption *Redemption, redeemerUserID int) error {
	if tx == nil || redemption == nil || redeemerUserID <= 0 ||
		redemption.FundingSource != RedemptionFundingSourceWallet ||
		redemption.BenefitType != RedemptionBenefitTypeQuota ||
		redemption.UserId <= 0 || redemption.UserId == redeemerUserID {
		return nil
	}
	creatorThreshold := common.WalletRedemptionReviewDistinctCreatorThreshold
	smallQuotaUnits := common.WalletRedemptionReviewSmallQuotaLimit
	if creatorThreshold <= 0 || smallQuotaUnits <= 0 {
		return nil
	}
	smallQuotaLimit, err := walletRedemptionQuotaFromUnits(smallQuotaUnits)
	if err != nil {
		return err
	}
	bizDate, dayStart, dayEnd := walletRedemptionLocalDayRange(time.Now())

	var rows []struct {
		Id     int `gorm:"column:id"`
		UserId int `gorm:"column:user_id"`
		Quota  int `gorm:"column:quota"`
	}
	if err := tx.Unscoped().Model(&Redemption{}).
		Select("id", "user_id", "quota").
		Where("used_user_id = ? AND user_id <> ?", redeemerUserID, redeemerUserID).
		Where("funding_source = ? AND benefit_type = ? AND status = ?", RedemptionFundingSourceWallet, RedemptionBenefitTypeQuota, common.RedemptionCodeStatusUsed).
		Where("redeemed_time >= ? AND redeemed_time < ? AND quota <= ?", dayStart, dayEnd, smallQuotaLimit).
		Order("id ASC").Find(&rows).Error; err != nil {
		return err
	}
	creatorSet := make(map[int]struct{}, len(rows))
	creatorIDs := make([]int, 0, len(rows))
	redemptionIDs := make([]int, 0, len(rows))
	totalQuota := 0
	for _, row := range rows {
		if _, exists := creatorSet[row.UserId]; !exists {
			creatorSet[row.UserId] = struct{}{}
			creatorIDs = append(creatorIDs, row.UserId)
		}
		redemptionIDs = append(redemptionIDs, row.Id)
		totalQuota += row.Quota
	}
	if len(creatorSet) < creatorThreshold {
		return nil
	}
	sort.Ints(creatorIDs)

	review := RedemptionReviewCase{}
	lookup := tx.Where("user_id = ? AND biz_date = ?", redeemerUserID, bizDate).Limit(1).Find(&review)
	if lookup.Error != nil {
		return lookup.Error
	}
	updates := map[string]any{
		"distinct_creator_count": len(creatorSet),
		"small_code_count":       len(rows),
		"total_quota":            totalQuota,
		"creator_ids":            joinRedemptionReviewIDs(creatorIDs),
		"redemption_ids":         joinRedemptionReviewIDs(redemptionIDs),
		"trigger_redemption_id":  redemption.Id,
		"updated_at":             common.GetTimestamp(),
	}
	if review.Id == 0 {
		review = RedemptionReviewCase{
			UserId:               redeemerUserID,
			BizDate:              bizDate,
			Status:               RedemptionReviewStatusPending,
			DistinctCreatorCount: len(creatorSet),
			SmallCodeCount:       len(rows),
			TotalQuota:           totalQuota,
			CreatorIds:           joinRedemptionReviewIDs(creatorIDs),
			RedemptionIds:        joinRedemptionReviewIDs(redemptionIDs),
			TriggerRedemptionId:  redemption.Id,
		}
		return tx.Create(&review).Error
	}
	if review.Status == RedemptionReviewStatusDismissed && review.TriggerRedemptionId != redemption.Id {
		updates["status"] = RedemptionReviewStatusPending
		updates["reviewer_id"] = 0
		updates["review_note"] = ""
		updates["reviewed_at"] = 0
	}
	return tx.Model(&RedemptionReviewCase{}).Where("id = ?", review.Id).Updates(updates).Error
}

func joinRedemptionReviewIDs(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}

func ListRedemptionReviewCases(status string, pageInfo *common.PageInfo) ([]*RedemptionReviewCase, int64, error) {
	query := DB.Model(&RedemptionReviewCase{})
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if pageInfo == nil {
		pageInfo = &common.PageInfo{Page: 1, PageSize: common.ItemsPerPage}
	}
	var cases []*RedemptionReviewCase
	if err := query.Order("id DESC").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&cases).Error; err != nil {
		return nil, 0, err
	}
	return cases, total, nil
}

func GetRedemptionReviewCaseByID(id int) (*RedemptionReviewCase, error) {
	if id <= 0 {
		return nil, ErrRedemptionReviewCaseNotFound
	}
	var review RedemptionReviewCase
	if err := DB.First(&review, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRedemptionReviewCaseNotFound
		}
		return nil, err
	}
	return &review, nil
}

func ResolveRedemptionReviewCase(id int, reviewerID int, status string, note string) (*RedemptionReviewCase, error) {
	return resolveRedemptionReviewCase(id, reviewerID, 0, status, note, false)
}

func ResolveRedemptionReviewCaseAction(id int, reviewerID int, reviewerRole int, action string, note string) (*RedemptionReviewCase, bool, error) {
	status := RedemptionReviewStatusDismissed
	disableUser := false
	switch action {
	case RedemptionReviewActionDismiss:
	case RedemptionReviewActionDisable:
		status = RedemptionReviewStatusDisabled
		disableUser = true
	default:
		return nil, false, fmt.Errorf("invalid redemption review action: %s", action)
	}
	resolved, err := resolveRedemptionReviewCase(id, reviewerID, reviewerRole, status, note, disableUser)
	return resolved, disableUser && err == nil, err
}

func resolveRedemptionReviewCase(id int, reviewerID int, reviewerRole int, status string, note string, disableUser bool) (*RedemptionReviewCase, error) {
	if id <= 0 || reviewerID <= 0 {
		return nil, ErrRedemptionReviewCaseNotFound
	}
	if status != RedemptionReviewStatusDismissed && status != RedemptionReviewStatusDisabled {
		return nil, fmt.Errorf("invalid redemption review status: %s", status)
	}
	var resolved RedemptionReviewCase
	disabledUserID := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Where("id = ?", id)
		if !common.UsingSQLite {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := query.First(&resolved).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRedemptionReviewCaseNotFound
			}
			return err
		}
		if resolved.Status != RedemptionReviewStatusPending {
			return ErrRedemptionReviewResolved
		}
		if disableUser {
			var user User
			userQuery := tx.Unscoped().Where("id = ?", resolved.UserId)
			if !common.UsingSQLite {
				userQuery = userQuery.Clauses(clause.Locking{Strength: "UPDATE"})
			}
			if err := userQuery.First(&user).Error; err != nil {
				return err
			}
			if user.Role == common.RoleRootUser {
				return errors.New("cannot disable root user")
			}
			if reviewerRole <= user.Role && reviewerRole != common.RoleRootUser {
				return errors.New("cannot disable same or higher role user")
			}
			if err := tx.Unscoped().Model(&User{}).Where("id = ?", user.Id).
				Update("status", common.UserStatusDisabled).Error; err != nil {
				return err
			}
			disabledUserID = user.Id
		}
		resolved.Status = status
		resolved.ReviewerId = reviewerID
		resolved.ReviewNote = strings.TrimSpace(note)
		resolved.ReviewedAt = common.GetTimestamp()
		return tx.Save(&resolved).Error
	})
	if err == nil && disabledUserID > 0 {
		if cacheErr := syncUserCacheByID(disabledUserID); cacheErr != nil {
			common.SysLog(fmt.Sprintf("failed to sync disabled redemption review user cache: user_id=%d err=%v", disabledUserID, cacheErr))
		}
	}
	return &resolved, err
}
