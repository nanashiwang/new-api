package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	ContentSafetyActionWarning         = "warning"
	ContentSafetyActionCooldownStarted = "cooldown_started"
	ContentSafetyActionCooldownActive  = "cooldown_active"
	// Historical actions remain readable, but new events never produce them.
	ContentSafetyActionDisabled        = "disabled"
	ContentSafetyActionAlreadyDisabled = "already_disabled"
	ContentSafetyActionReviewRequired  = "review_required"
)

const (
	ContentSafetyLevelNormal         = "normal"
	ContentSafetyLevelWarning1       = "warning_1"
	ContentSafetyLevelWarning2       = "warning_2"
	ContentSafetyLevelCoolingOff     = "cooling_off"
	ContentSafetyLevelObserved       = "observed"
	ContentSafetyLevelFocus          = "focus"
	ContentSafetyLevelReviewPending  = "review_pending"
	ContentSafetyLevelAdminDisabled  = "admin_disabled"
	ContentSafetyLevelLegacyDisabled = "legacy_disabled"
	ContentSafetyLevelTriggered      = "triggered"
	contentSafetyWindow              = 30 * 24 * time.Hour
	contentSafetyBurstWindow         = 10 * time.Minute
)

type ContentSafetyViolation struct {
	Id                int64  `json:"id"`
	UserId            int    `json:"user_id" gorm:"index;index:idx_safety_user_created,priority:1;index:idx_safety_user_cooldown,priority:1"`
	Username          string `json:"username" gorm:"->"`
	TokenId           int    `json:"token_id" gorm:"index"`
	ChannelId         int    `json:"channel_id" gorm:"index"`
	RequestId         string `json:"request_id" gorm:"type:varchar(64);index"`
	EventKey          string `json:"-" gorm:"type:char(64);uniqueIndex"`
	ModelName         string `json:"model_name" gorm:"type:varchar(128);index"`
	ErrorType         string `json:"error_type" gorm:"type:varchar(64)"`
	ErrorCode         string `json:"error_code" gorm:"type:varchar(64);index"`
	OfficialMessage   string `json:"official_message" gorm:"type:varchar(512)"`
	FineCategory      string `json:"fine_category" gorm:"type:varchar(64);index"`
	ReasonSource      string `json:"reason_source" gorm:"type:varchar(32)"`
	ReasonConfidence  string `json:"reason_confidence" gorm:"type:varchar(16)"`
	ReasonSummary     string `json:"reason_summary" gorm:"type:varchar(512)"`
	ClassifierVersion string `json:"classifier_version" gorm:"type:varchar(32)"`
	InputHash         string `json:"-" gorm:"type:char(64)"`
	IsStream          bool   `json:"is_stream"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint;index;index:idx_safety_user_created,priority:2"`
	WindowCount       int    `json:"window_count"`
	BurstCount        int    `json:"burst_count"`
	CooldownUntil     int64  `json:"cooldown_until" gorm:"bigint;index;index:idx_safety_user_cooldown,priority:2"`
	WarningReadAt     int64  `json:"warning_read_at" gorm:"bigint;index"`
	Action            string `json:"action" gorm:"type:varchar(32);index"`
	EvidenceAvailable bool   `json:"evidence_available" gorm:"->"`
	EmailStatus       string `json:"email_status" gorm:"->"`
	EmailSource       string `json:"email_source" gorm:"->"`
}

type RecordContentSafetyViolationParams struct {
	UserId               int
	TokenId              int
	ChannelId            int
	RequestId            string
	EventKey             string
	ModelName            string
	ErrorType            string
	ErrorCode            string
	OfficialMessage      string
	FineCategory         string
	ReasonSource         string
	ReasonConfidence     string
	ReasonSummary        string
	ClassifierVersion    string
	InputHash            string
	IsStream             bool
	CreatedAt            int64
	WindowStart          int64
	BurstWindowStart     int64
	BurstThreshold       int
	CooldownSeconds      int64
	ReviewAfterCooldowns int
}

type ContentSafetyEnforcementResult struct {
	Violation  *ContentSafetyViolation
	ReviewCase *ContentSafetyReviewCase
	Duplicate  bool
	UserStatus int
	UserRole   int
	Username   string
}

type ContentSafetyState struct {
	Level            string                  `json:"level"`
	WindowCount      int                     `json:"window_count"`
	BurstCount       int                     `json:"burst_count"`
	CooldownCount    int                     `json:"cooldown_count"`
	CooldownUntil    int64                   `json:"cooldown_until"`
	ReviewCaseId     int64                   `json:"review_case_id"`
	HasUnreadWarning bool                    `json:"has_unread_warning"`
	LatestViolation  *ContentSafetyViolation `json:"latest_violation,omitempty"`
}

func contentSafetyLevelForState(user *User, state *ContentSafetyState, now int64, approvedDisable bool, legacyDisable bool) string {
	if approvedDisable && user.Status == common.UserStatusDisabled {
		return ContentSafetyLevelAdminDisabled
	}
	if legacyDisable && user.Status == common.UserStatusDisabled {
		return ContentSafetyLevelLegacyDisabled
	}
	if state.ReviewCaseId > 0 {
		return ContentSafetyLevelReviewPending
	}
	if state.CooldownUntil > now {
		return ContentSafetyLevelCoolingOff
	}
	if state.CooldownCount >= 2 {
		return ContentSafetyLevelFocus
	}
	if state.LatestViolation != nil && now-state.LatestViolation.CreatedAt < int64(contentSafetyBurstWindow.Seconds()) && state.LatestViolation.CooldownUntil == 0 {
		switch state.LatestViolation.BurstCount {
		case 1:
			return ContentSafetyLevelWarning1
		case 2:
			return ContentSafetyLevelWarning2
		}
	}
	if state.WindowCount > 0 {
		return ContentSafetyLevelObserved
	}
	return ContentSafetyLevelNormal
}

func applyUserContentSafetyFilters(tx *gorm.DB, query *gorm.DB, params UserSearchParams) *gorm.DB {
	status := strings.TrimSpace(params.ContentSafetyStatus)
	if status == "" && len(params.ContentSafetyCodes) == 0 {
		return query
	}
	if !tx.Migrator().HasTable(&ContentSafetyViolation{}) {
		if status == ContentSafetyLevelNormal && len(params.ContentSafetyCodes) == 0 {
			return query
		}
		return query.Where("1 = 0")
	}

	now := time.Now().Unix()
	cutoff := now - int64(contentSafetyWindow.Seconds())
	burstCutoff := now - int64(contentSafetyBurstWindow.Seconds())
	countSQL := "SELECT COUNT(1) FROM content_safety_violations csv WHERE csv.user_id = users.id AND csv.created_at >= ?"
	episodeSQL := "SELECT COUNT(1) FROM content_safety_violations cse WHERE cse.user_id = users.id AND cse.created_at >= ? AND cse.action = 'cooldown_started'"
	activeSQL := "SELECT COUNT(1) FROM content_safety_violations csa WHERE csa.user_id = users.id AND csa.cooldown_until > ?"
	lastCooldownSQL := "SELECT COALESCE(MAX(csl.cooldown_until), 0) FROM content_safety_violations csl WHERE csl.user_id = users.id"
	recentSQL := "SELECT COUNT(1) FROM content_safety_violations csr WHERE csr.user_id = users.id AND csr.created_at >= ? AND csr.created_at > (" + lastCooldownSQL + ")"
	hasReviewTable := tx.Migrator().HasTable(&ContentSafetyReviewCase{})
	pendingSQL := "SELECT COUNT(1) FROM content_safety_review_cases csp WHERE csp.user_id = users.id AND csp.status = 'pending'"
	approvedSQL := "SELECT COUNT(1) FROM content_safety_review_cases csa2 WHERE csa2.user_id = users.id AND csa2.status = 'approved_disable'"
	legacySQL := "SELECT COUNT(1) FROM content_safety_violations csh WHERE csh.user_id = users.id AND csh.action IN ('disabled','already_disabled')"

	switch status {
	case ContentSafetyLevelNormal:
		query = query.Where("("+countSQL+") = 0", cutoff)
	case ContentSafetyLevelTriggered:
		query = query.Where("("+countSQL+") >= 1", cutoff)
	case ContentSafetyLevelWarning1:
		condition := "(" + recentSQL + ") = 1 AND (" + activeSQL + ") = 0"
		if hasReviewTable {
			condition += " AND (" + pendingSQL + ") = 0 AND (" + approvedSQL + ") = 0"
		}
		query = query.Where(condition, burstCutoff, now)
	case ContentSafetyLevelWarning2:
		condition := "(" + recentSQL + ") = 2 AND (" + activeSQL + ") = 0"
		if hasReviewTable {
			condition += " AND (" + pendingSQL + ") = 0 AND (" + approvedSQL + ") = 0"
		}
		query = query.Where(condition, burstCutoff, now)
	case ContentSafetyLevelCoolingOff:
		query = query.Where("("+activeSQL+") >= 1", now)
	case ContentSafetyLevelObserved:
		condition := "(" + countSQL + ") >= 1 AND (" + episodeSQL + ") < 2 AND (" + activeSQL + ") = 0 AND (" + recentSQL + ") NOT IN (1, 2)"
		if hasReviewTable {
			condition += " AND (" + pendingSQL + ") = 0 AND (" + approvedSQL + ") = 0"
		}
		query = query.Where(condition, cutoff, cutoff, now, burstCutoff)
	case ContentSafetyLevelFocus:
		condition := "(" + episodeSQL + ") >= 2 AND (" + activeSQL + ") = 0"
		if hasReviewTable {
			condition += " AND (" + pendingSQL + ") = 0 AND (" + approvedSQL + ") = 0"
		}
		query = query.Where(condition, cutoff, now)
	case ContentSafetyLevelReviewPending:
		if hasReviewTable {
			query = query.Where("(" + pendingSQL + ") >= 1")
		} else {
			query = query.Where("1 = 0")
		}
	case ContentSafetyLevelAdminDisabled:
		if hasReviewTable {
			query = query.Where("("+approvedSQL+") >= 1 AND status = ?", common.UserStatusDisabled)
		} else {
			query = query.Where("1 = 0")
		}
	case ContentSafetyLevelLegacyDisabled:
		query = query.Where("("+legacySQL+") >= 1 AND status = ?", common.UserStatusDisabled)
	}

	if len(params.ContentSafetyCodes) > 0 {
		codeExists := tx.Model(&ContentSafetyViolation{}).
			Select("1").Where("content_safety_violations.user_id = users.id").
			Where("content_safety_violations.created_at >= ?", cutoff).
			Where("content_safety_violations.error_code IN ?", params.ContentSafetyCodes)
		query = query.Where("EXISTS (?)", codeExists)
	}
	return query
}

func applyUserContentSafetySort(tx *gorm.DB, query *gorm.DB, sortOrder string, fallbackOrder string) (*gorm.DB, string) {
	normalizedOrder := normalizeUserSortOrder(sortOrder)
	if normalizedOrder == "" || !tx.Migrator().HasTable(&ContentSafetyViolation{}) {
		return query, fallbackOrder
	}

	// 排序口径必须与用户列表展示的 ContentSafetyLastAt 一致：只统计最近 30 天。
	// 先聚合再 JOIN，避免在 users 主查询中直接关联多条违规记录导致重复用户、总数失真或分页漂移。
	cutoff := time.Now().Unix() - int64(contentSafetyWindow.Seconds())
	latestViolation := tx.Model(&ContentSafetyViolation{}).
		Select("user_id, MAX(created_at) AS last_at").
		Where("created_at >= ?", cutoff).
		Group("user_id")
	query = query.Joins(
		"LEFT JOIN (?) AS content_safety_sort ON content_safety_sort.user_id = users.id",
		latestViolation,
	)

	// 无近期记录的用户始终排在有记录用户之后；否则升序时 NULL/0 会占据列表顶部，
	// 与管理员“查看风控事件时间线”的目标相反。
	orderClause := "CASE WHEN content_safety_sort.last_at IS NULL THEN 1 ELSE 0 END ASC, content_safety_sort.last_at " + normalizedOrder
	if strings.TrimSpace(fallbackOrder) != "" {
		orderClause += ", " + fallbackOrder
	}
	return query, orderClause
}

func AttachUserContentSafetyMetadata(tx *gorm.DB, users []*User) error {
	if len(users) == 0 {
		return nil
	}
	for _, user := range users {
		if user != nil {
			user.ContentSafetyLevel = ContentSafetyLevelNormal
		}
	}
	if !tx.Migrator().HasTable(&ContentSafetyViolation{}) {
		return nil
	}

	userIDs := make([]int, 0, len(users))
	usersByID := make(map[int]*User, len(users))
	states := make(map[int]*ContentSafetyState, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		userIDs = append(userIDs, user.Id)
		usersByID[user.Id] = user
		states[user.Id] = &ContentSafetyState{Level: ContentSafetyLevelNormal}
	}
	if len(userIDs) == 0 {
		return nil
	}

	now := time.Now().Unix()
	cutoff := now - int64(contentSafetyWindow.Seconds())
	var counts []struct{ UserId, Count int }
	if err := tx.Model(&ContentSafetyViolation{}).Select("user_id, COUNT(1) AS count").
		Where("user_id IN ? AND created_at >= ?", userIDs, cutoff).Group("user_id").Scan(&counts).Error; err != nil {
		return err
	}
	for _, row := range counts {
		states[row.UserId].WindowCount = row.Count
	}

	var episodes []struct{ UserId, Count int }
	if err := tx.Model(&ContentSafetyViolation{}).Select("user_id, COUNT(1) AS count").
		Where("user_id IN ? AND created_at >= ? AND action = ?", userIDs, cutoff, ContentSafetyActionCooldownStarted).
		Group("user_id").Scan(&episodes).Error; err != nil {
		return err
	}
	for _, row := range episodes {
		states[row.UserId].CooldownCount = row.Count
	}

	var cooldowns []struct {
		UserId        int
		CooldownUntil int64
	}
	if err := tx.Model(&ContentSafetyViolation{}).Select("user_id, MAX(cooldown_until) AS cooldown_until").
		Where("user_id IN ?", userIDs).Group("user_id").Scan(&cooldowns).Error; err != nil {
		return err
	}
	for _, row := range cooldowns {
		states[row.UserId].CooldownUntil = row.CooldownUntil
	}

	latest := make([]ContentSafetyViolation, 0, len(userIDs))
	latestSQL := `NOT EXISTS (
		SELECT 1 FROM content_safety_violations newer
		WHERE newer.user_id = v.user_id AND newer.created_at >= ?
		AND (newer.created_at > v.created_at OR (newer.created_at = v.created_at AND newer.id > v.id))
	)`
	if err := tx.Table("content_safety_violations AS v").Where("v.user_id IN ? AND v.created_at >= ?", userIDs, cutoff).
		Where(latestSQL, cutoff).Scan(&latest).Error; err != nil {
		return err
	}
	legacyDisabled := make(map[int]bool)
	for _, violation := range latest {
		copy := violation
		state := states[violation.UserId]
		state.LatestViolation = &copy
		state.BurstCount = violation.BurstCount
		state.HasUnreadWarning = violation.WarningReadAt == 0
	}
	var legacyRows []struct{ UserId int }
	if err := tx.Model(&ContentSafetyViolation{}).Distinct("user_id").
		Where("user_id IN ? AND action IN ?", userIDs, []string{ContentSafetyActionDisabled, ContentSafetyActionAlreadyDisabled}).
		Scan(&legacyRows).Error; err != nil {
		return err
	}
	for _, row := range legacyRows {
		legacyDisabled[row.UserId] = true
	}

	approvedDisable := make(map[int]bool)
	if tx.Migrator().HasTable(&ContentSafetyReviewCase{}) {
		var pending []ContentSafetyReviewCase
		if err := tx.Where("user_id IN ? AND status = ?", userIDs, ContentSafetyReviewPending).Find(&pending).Error; err != nil {
			return err
		}
		for _, reviewCase := range pending {
			states[reviewCase.UserId].ReviewCaseId = reviewCase.Id
		}
		var approved []struct{ UserId int }
		if err := tx.Model(&ContentSafetyReviewCase{}).Distinct("user_id").
			Where("user_id IN ? AND status = ?", userIDs, ContentSafetyReviewApprovedDisable).Scan(&approved).Error; err != nil {
			return err
		}
		for _, row := range approved {
			approvedDisable[row.UserId] = true
		}
	}

	for userID, state := range states {
		user := usersByID[userID]
		state.Level = contentSafetyLevelForState(user, state, now, approvedDisable[userID], legacyDisabled[userID])
		user.ContentSafetyLevel = state.Level
		user.ContentSafetyCount = state.WindowCount
		user.ContentSafetyBurstCount = state.BurstCount
		user.ContentSafetyCooldownCount = state.CooldownCount
		user.ContentSafetyCooldownUntil = state.CooldownUntil
		user.ContentSafetyReviewCaseID = state.ReviewCaseId
		if latestViolation := state.LatestViolation; latestViolation != nil {
			user.ContentSafetyLastAt = latestViolation.CreatedAt
			user.ContentSafetyLastCode = latestViolation.ErrorCode
			user.ContentSafetyLastModel = latestViolation.ModelName
			user.ContentSafetyLastChannelID = latestViolation.ChannelId
			user.ContentSafetyLastRequestID = latestViolation.RequestId
			user.ContentSafetyLastCategory = latestViolation.FineCategory
			user.ContentSafetyReasonSource = latestViolation.ReasonSource
			user.ContentSafetyReasonConfidence = latestViolation.ReasonConfidence
			user.ContentSafetyReasonSummary = latestViolation.ReasonSummary
		}
	}
	return nil
}

func GetUserContentSafetyState(userID int) (*ContentSafetyState, error) {
	var user User
	if err := DB.Select("id", "status", "role").First(&user, userID).Error; err != nil {
		return nil, err
	}
	if err := AttachUserContentSafetyMetadata(DB, []*User{&user}); err != nil {
		return nil, err
	}
	state := &ContentSafetyState{
		Level: user.ContentSafetyLevel, WindowCount: user.ContentSafetyCount,
		BurstCount: user.ContentSafetyBurstCount, CooldownCount: user.ContentSafetyCooldownCount,
		CooldownUntil: user.ContentSafetyCooldownUntil, ReviewCaseId: user.ContentSafetyReviewCaseID,
	}
	if user.ContentSafetyLastAt > 0 {
		var latest ContentSafetyViolation
		if err := DB.Where("user_id = ? AND created_at = ?", userID, user.ContentSafetyLastAt).
			Order("id DESC").First(&latest).Error; err != nil {
			return nil, err
		}
		state.LatestViolation = &latest
		state.HasUnreadWarning = latest.WarningReadAt == 0
	}
	return state, nil
}

func GetActiveContentSafetyCooldown(userID int, now int64) (int64, error) {
	if userID <= 0 {
		return 0, nil
	}
	var cooldownUntil int64
	err := DB.Model(&ContentSafetyViolation{}).Select("COALESCE(MAX(cooldown_until), 0)").
		Where("user_id = ? AND cooldown_until > ?", userID, now).Scan(&cooldownUntil).Error
	return cooldownUntil, err
}

func AcknowledgeContentSafetyWarnings(userID int, now int64) error {
	if userID <= 0 {
		return errors.New("invalid content safety warning identity")
	}
	return DB.Model(&ContentSafetyViolation{}).Where("user_id = ? AND warning_read_at = 0", userID).
		Update("warning_read_at", now).Error
}

func RecordContentSafetyViolation(params RecordContentSafetyViolationParams) (*ContentSafetyEnforcementResult, error) {
	if params.UserId <= 0 || params.EventKey == "" || params.ErrorCode == "" {
		return nil, errors.New("invalid content safety violation identity")
	}
	if params.BurstThreshold <= 0 || params.CooldownSeconds <= 0 || params.ReviewAfterCooldowns <= 0 {
		return nil, errors.New("invalid content safety enforcement policy")
	}

	result := &ContentSafetyEnforcementResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		lockResult := tx.Model(&User{}).Where("id = ?", params.UserId).
			UpdateColumn("status", gorm.Expr("status"))
		if lockResult.Error != nil {
			return lockResult.Error
		}

		var user User
		if err := tx.Select("id", "username", "role", "status").First(&user, params.UserId).Error; err != nil {
			return err
		}
		result.UserStatus, result.UserRole, result.Username = user.Status, user.Role, user.Username

		var existing ContentSafetyViolation
		if err := tx.Where("event_key = ?", params.EventKey).First(&existing).Error; err == nil {
			result.Violation, result.Duplicate = &existing, true
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var latestCooldownUntil int64
		if err := tx.Model(&ContentSafetyViolation{}).Select("COALESCE(MAX(cooldown_until), 0)").
			Where("user_id = ?", params.UserId).Scan(&latestCooldownUntil).Error; err != nil {
			return err
		}
		burstStart := params.BurstWindowStart
		if latestCooldownUntil > burstStart {
			burstStart = latestCooldownUntil
		}

		violation := &ContentSafetyViolation{
			UserId: params.UserId, TokenId: params.TokenId, ChannelId: params.ChannelId,
			RequestId: truncateSafetyAuditValue(params.RequestId, 64), EventKey: params.EventKey,
			ModelName: truncateSafetyAuditValue(params.ModelName, 128), ErrorType: truncateSafetyAuditValue(params.ErrorType, 64),
			ErrorCode: truncateSafetyAuditValue(params.ErrorCode, 64), OfficialMessage: truncateSafetyAuditValue(params.OfficialMessage, 512),
			FineCategory: truncateSafetyAuditValue(params.FineCategory, 64), ReasonSource: truncateSafetyAuditValue(params.ReasonSource, 32),
			ReasonConfidence: truncateSafetyAuditValue(params.ReasonConfidence, 16), ReasonSummary: truncateSafetyAuditValue(params.ReasonSummary, 512),
			ClassifierVersion: truncateSafetyAuditValue(params.ClassifierVersion, 32), InputHash: truncateSafetyAuditValue(params.InputHash, 64),
			IsStream: params.IsStream, CreatedAt: params.CreatedAt, Action: ContentSafetyActionWarning,
		}
		if err := tx.Create(violation).Error; err != nil {
			return err
		}

		var windowCount, burstCount int64
		if err := tx.Model(&ContentSafetyViolation{}).Where("user_id = ? AND created_at >= ?", params.UserId, params.WindowStart).
			Count(&windowCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&ContentSafetyViolation{}).Where("user_id = ? AND created_at >= ? AND created_at <= ?", params.UserId, burstStart, params.CreatedAt).
			Count(&burstCount).Error; err != nil {
			return err
		}
		violation.WindowCount, violation.BurstCount = int(windowCount), int(burstCount)
		if latestCooldownUntil > params.CreatedAt {
			violation.Action = ContentSafetyActionCooldownActive
			violation.CooldownUntil = latestCooldownUntil
		} else if violation.BurstCount >= params.BurstThreshold {
			violation.Action = ContentSafetyActionCooldownStarted
			violation.CooldownUntil = params.CreatedAt + params.CooldownSeconds
		}
		if err := tx.Model(violation).Updates(map[string]any{
			"window_count": violation.WindowCount, "burst_count": violation.BurstCount,
			"cooldown_until": violation.CooldownUntil, "action": violation.Action,
		}).Error; err != nil {
			return err
		}

		if violation.Action == ContentSafetyActionCooldownStarted {
			var cooldownCount int64
			if err := tx.Model(&ContentSafetyViolation{}).
				Where("user_id = ? AND created_at >= ? AND action = ?", params.UserId, params.WindowStart, ContentSafetyActionCooldownStarted).
				Count(&cooldownCount).Error; err != nil {
				return err
			}
			if int(cooldownCount) >= params.ReviewAfterCooldowns {
				reviewCase, err := createPendingContentSafetyReviewCase(tx, violation, int(cooldownCount))
				if err != nil {
					return err
				}
				result.ReviewCase = reviewCase
			}
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
	base := DB.Table("content_safety_violations AS violations").Joins("LEFT JOIN users ON users.id = violations.user_id")
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
	selectClause := "violations.*, users.username AS username"
	if DB.Migrator().HasTable(&ContentSafetyEvidence{}) {
		selectClause += ", EXISTS (SELECT 1 FROM content_safety_evidences evidence WHERE evidence.violation_id = violations.id) AS evidence_available"
	}
	if DB.Migrator().HasTable(&ContentSafetyNotification{}) {
		selectClause += ", COALESCE((SELECT notification.status FROM content_safety_notifications notification WHERE notification.violation_id = violations.id ORDER BY notification.id DESC LIMIT 1), '') AS email_status"
		selectClause += ", COALESCE((SELECT notification.recipient_source FROM content_safety_notifications notification WHERE notification.violation_id = violations.id ORDER BY notification.id DESC LIMIT 1), '') AS email_source"
	}
	if err := base.Select(selectClause).Order("violations.id DESC").
		Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
