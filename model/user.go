package model

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

const UserNameMaxLength = 20

// User if you add sensitive fields, don't forget to clean them in setupLogin function.
// Otherwise, the sensitive information will be saved on local storage in plain text!
type User struct {
	Id               int            `json:"id"`
	Username         string         `json:"username" gorm:"unique;index" validate:"max=20"`
	Password         string         `json:"password" gorm:"not null;" validate:"min=8,max=20"`
	OriginalPassword string         `json:"original_password" gorm:"-:all"` // this field is only for Password change verification, don't save it to database!
	DisplayName      string         `json:"display_name" gorm:"index" validate:"max=20"`
	Role             int            `json:"role" gorm:"type:int;default:1"`   // admin, common
	Status           int            `json:"status" gorm:"type:int;default:1"` // enabled, disabled
	Email            string         `json:"email" gorm:"index" validate:"max=50"`
	GitHubId         string         `json:"github_id" gorm:"column:github_id;index"`
	DiscordId        string         `json:"discord_id" gorm:"column:discord_id;index"`
	OidcId           string         `json:"oidc_id" gorm:"column:oidc_id;index"`
	WeChatId         string         `json:"wechat_id" gorm:"column:wechat_id;index"`
	TelegramId       string         `json:"telegram_id" gorm:"column:telegram_id;index"`
	VerificationCode string         `json:"verification_code" gorm:"-:all"`                                    // this field is only for Email verification, don't save it to database!
	AccessToken      *string        `json:"access_token" gorm:"type:char(32);column:access_token;uniqueIndex"` // this token is for system management
	Quota            int            `json:"quota" gorm:"type:int;default:0"`
	UsedQuota        int            `json:"used_quota" gorm:"type:int;default:0;column:used_quota"` // used quota
	RequestCount     int            `json:"request_count" gorm:"type:int;default:0;"`               // request number
	Group            string         `json:"group" gorm:"type:varchar(64);default:'default'"`
	AffCode          string         `json:"aff_code" gorm:"type:varchar(32);column:aff_code;uniqueIndex"`
	AffCount         int            `json:"aff_count" gorm:"type:int;default:0;column:aff_count"`
	AffQuota         int            `json:"aff_quota" gorm:"type:int;default:0;column:aff_quota"`           // 邀请剩余额度
	AffHistoryQuota  int            `json:"aff_history_quota" gorm:"type:int;default:0;column:aff_history"` // 邀请历史额度
	InviterId        int            `json:"inviter_id" gorm:"type:int;column:inviter_id;index"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
	LinuxDOId        string         `json:"linux_do_id" gorm:"column:linux_do_id;index"`
	Setting          string         `json:"setting" gorm:"type:text;column:setting"`
	Remark           string         `json:"remark,omitempty" gorm:"type:varchar(255)" validate:"max=255"`
	StripeCustomer   string         `json:"stripe_customer" gorm:"type:varchar(64);column:stripe_customer;index"`

	InviterUsername string `json:"inviter_username,omitempty" gorm:"-"`
	InviteeCount    int    `json:"invitee_count,omitempty" gorm:"-"`
	// 生效套餐元数据（仅用于用户管理列表响应，不落库）：
	// - HasActiveSubscription：当前是否存在生效套餐
	// - ActiveSubscriptionCount：当前生效套餐条数
	HasActiveSubscription            bool `json:"has_active_subscription" gorm:"-"`
	ActiveSubscriptionCount          int  `json:"active_subscription_count" gorm:"-"`
	PendingSubscriptionIssuanceCount int  `json:"pending_subscription_issuance_count" gorm:"-"`
	// 可售令牌元数据（仅用于用户管理列表响应，不落库）
	HasSellableToken             bool `json:"has_sellable_token" gorm:"-"`
	ActiveSellableTokenCount     int  `json:"active_sellable_token_count" gorm:"-"`
	PendingSellableIssuanceCount int  `json:"pending_sellable_issuance_count" gorm:"-"`
	// 邀请关系视图的派生字段（仅用于接口响应，不落库）：
	// 1) 直接收益：按当前配置口径（QuotaForInviter）估算每个被邀请人贡献；
	// 2) 充值返佣：从返佣台账（InviteCommissionLedger）按 invitee 聚合已结算金额；
	// 3) 总收益：直接收益 + 充值返佣。
	InviteDirectIncomeQuota       int `json:"invite_direct_income_quota" gorm:"-"`
	InviteRechargeCommissionQuota int `json:"invite_recharge_commission_quota" gorm:"-"`
	InviteTotalIncomeQuota        int `json:"invite_total_income_quota" gorm:"-"`
}

type UserSearchParams struct {
	Keyword               string
	Group                 string
	Role                  *int
	Status                *int
	InviterID             *int
	InviteeUserID         *int
	HasInviter            *bool
	HasInvitees           *bool
	HasActiveSubscription *bool
	HasSellableToken      *bool
	BalanceMin            *int
	BalanceMax            *int
	UsedBalanceMin        *int
	UsedBalanceMax        *int
	SortBy                string
	SortOrder             string
	IdSortOrder           string
	BalanceSortOrder      string
	StartIdx              int
	PageSize              int
}

type InviteIncomeSummary struct {
	DirectTotalQuota   int `json:"direct_total_quota"`
	RechargeTotalQuota int `json:"recharge_total_quota"`
	TotalQuota         int `json:"total_quota"`
}

func normalizeUserSortOrder(sortOrder string) string {
	normalizedSortOrder := strings.ToLower(strings.TrimSpace(sortOrder))
	switch normalizedSortOrder {
	case "asc", "desc":
		return normalizedSortOrder
	default:
		return ""
	}
}

func normalizeLegacyUserSort(sortBy string, sortOrder string) (string, string) {
	normalizedSortBy := strings.ToLower(strings.TrimSpace(sortBy))
	switch normalizedSortBy {
	case "quota":
	default:
		normalizedSortBy = "id"
	}

	normalizedSortOrder := strings.ToLower(strings.TrimSpace(sortOrder))
	switch normalizedSortOrder {
	case "asc":
	default:
		normalizedSortOrder = "desc"
	}

	return normalizedSortBy, normalizedSortOrder
}

func buildUserOrderClause(sortBy string, sortOrder string, idSortOrder string, balanceSortOrder string) string {
	orderClauses := make([]string, 0, 2)
	if normalized := normalizeUserSortOrder(idSortOrder); normalized != "" {
		orderClauses = append(orderClauses, "id "+normalized)
	}
	if normalized := normalizeUserSortOrder(balanceSortOrder); normalized != "" {
		orderClauses = append(orderClauses, "quota "+normalized)
	}

	// 兼容旧参数：如果新参数都为空，则回退到 sort_by/sort_order。
	if len(orderClauses) == 0 {
		normalizedSortBy, normalizedSortOrder := normalizeLegacyUserSort(sortBy, sortOrder)
		orderClauses = append(orderClauses, normalizedSortBy+" "+normalizedSortOrder)
	}

	return strings.Join(orderClauses, ", ")
}

func (user *User) ToBaseUser() *UserBase {
	cache := &UserBase{
		Id:       user.Id,
		Group:    user.Group,
		Quota:    user.Quota,
		Status:   user.Status,
		Username: user.Username,
		Setting:  user.Setting,
		Email:    user.Email,
	}
	return cache
}

func (user *User) GetAccessToken() string {
	if user.AccessToken == nil {
		return ""
	}
	return *user.AccessToken
}

func (user *User) SetAccessToken(token string) {
	user.AccessToken = &token
}

func (user *User) GetSetting() dto.UserSetting {
	setting := dto.UserSetting{}
	if user.Setting != "" {
		err := json.Unmarshal([]byte(user.Setting), &setting)
		if err != nil {
			common.SysLog("failed to unmarshal setting: " + err.Error())
		}
	}
	return setting
}

func (user *User) SetSetting(setting dto.UserSetting) {
	settingBytes, err := json.Marshal(setting)
	if err != nil {
		common.SysLog("failed to marshal setting: " + err.Error())
		return
	}
	user.Setting = string(settingBytes)
}

// 根据用户角色生成默认的边栏配置
func generateDefaultSidebarConfigForRole(userRole int) string {
	defaultConfig := map[string]interface{}{}

	// 聊天区域 - 所有用户都可以访问
	defaultConfig["chat"] = map[string]interface{}{
		"enabled":    true,
		"playground": true,
		"chat":       true,
	}

	// 控制台区域 - 所有用户都可以访问
	defaultConfig["console"] = map[string]interface{}{
		"enabled":    true,
		"detail":     true,
		"token":      true,
		"log":        true,
		"midjourney": true,
		"task":       true,
	}

	// 个人中心区域 - 所有用户都可以访问
	defaultConfig["personal"] = map[string]interface{}{
		"enabled":  true,
		"topup":    true,
		"personal": true,
	}

	// 管理员区域 - 根据角色决定
	if userRole == common.RoleAdminUser {
		// 管理员可以访问管理员区域，但不能访问系统设置
		defaultConfig["admin"] = map[string]interface{}{
			"enabled":    true,
			"channel":    true,
			"models":     true,
			"redemption": true,
			"user":       true,
			"setting":    false, // 管理员不能访问系统设置
		}
	} else if userRole == common.RoleRootUser {
		// 超级管理员可以访问所有功能
		defaultConfig["admin"] = map[string]interface{}{
			"enabled":    true,
			"channel":    true,
			"models":     true,
			"redemption": true,
			"user":       true,
			"setting":    true,
		}
	}
	// 普通用户不包含admin区域

	// 转换为JSON字符串
	configBytes, err := json.Marshal(defaultConfig)
	if err != nil {
		common.SysLog("生成默认边栏配置失败: " + err.Error())
		return ""
	}

	return string(configBytes)
}

// CheckUserExistOrDeleted check if user exist or deleted, if not exist, return false, nil, if deleted or exist, return true, nil
func CheckUserExistOrDeleted(username string, email string) (bool, error) {
	var user User

	// err := DB.Unscoped().First(&user, "username = ? or email = ?", username, email).Error
	// check email if empty
	var err error
	if email == "" {
		err = DB.Unscoped().First(&user, "username = ?", username).Error
	} else {
		err = DB.Unscoped().First(&user, "username = ? or email = ?", username, email).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// not exist, return false, nil
			return false, nil
		}
		// other error, return false, err
		return false, err
	}
	// exist, return true, nil
	return true, nil
}

func GetMaxUserId() int {
	var user User
	DB.Unscoped().Last(&user)
	return user.Id
}

func GetAllUsers(pageInfo *common.PageInfo, sortBy string, sortOrder string, idSortOrder string, balanceSortOrder string) (users []*User, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get total count within transaction
	err = tx.Unscoped().Model(&User{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	orderClause := buildUserOrderClause(sortBy, sortOrder, idSortOrder, balanceSortOrder)
	// Get paginated users within same transaction
	err = tx.Unscoped().Order(orderClause).Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Omit("password").Find(&users).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	// 当前页批量补充"生效套餐"统计，避免前端逐行请求造成 N+1。
	if err = AttachUserSubscriptionMetadata(tx, users); err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err = AttachUserSellableTokenMetadata(tx, users); err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func SearchUsers(keyword string, group string, startIdx int, num int) ([]*User, int64, error) {
	return SearchUsersWithParams(UserSearchParams{
		Keyword:   keyword,
		Group:     group,
		SortBy:    "id",
		SortOrder: "desc",
		StartIdx:  startIdx,
		PageSize:  num,
	})
}

func SearchUsersWithParams(params UserSearchParams) ([]*User, int64, error) {
	var users []*User
	var total int64
	var err error

	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 构建基础查询
	query := tx.Unscoped().Model(&User{})

	// 这里统一做输入清洗（去首尾空白），并且后续所有条件都使用 GORM 参数绑定写法，
	// 避免把用户输入直接拼接到 SQL 字符串里，从根源上规避 SQL 注入风险。
	keyword := strings.TrimSpace(params.Keyword)
	group := strings.TrimSpace(params.Group)
	if keyword != "" {
		likeCondition := "username LIKE ? OR email LIKE ? OR display_name LIKE ?"
		if keywordInt, parseErr := strconv.Atoi(keyword); parseErr == nil {
			likeCondition = "id = ? OR " + likeCondition
			query = query.Where(
				likeCondition,
				keywordInt, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
			)
		} else {
			query = query.Where(
				likeCondition,
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
			)
		}
	}

	if group != "" {
		query = query.Where(commonGroupCol+" = ?", group)
	}
	if params.Role != nil {
		query = query.Where("role = ?", *params.Role)
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}
	if params.InviterID != nil {
		query = query.Where("inviter_id = ?", *params.InviterID)
	}
	if params.HasInviter != nil {
		if *params.HasInviter {
			query = query.Where("inviter_id > 0")
		} else {
			query = query.Where("inviter_id = 0")
		}
	}
	if params.HasInvitees != nil {
		if *params.HasInvitees {
			query = query.Where("aff_count > 0")
		} else {
			query = query.Where("aff_count = 0")
		}
	}
	if params.HasActiveSubscription != nil {
		now := common.GetTimestamp()
		// 套餐筛选统一按"生效套餐"口径：status=active 且 end_time > now。
		// 使用 EXISTS/NOT EXISTS 让三种数据库都能稳定执行，且不影响主表分页结构。
		activeSubExistsSQL := "SELECT 1 FROM user_subscriptions us WHERE us.user_id = users.id AND us.status = ? AND us.end_time > ?"
		if *params.HasActiveSubscription {
			query = query.Where("EXISTS ("+activeSubExistsSQL+")", "active", now)
		} else {
			query = query.Where("NOT EXISTS ("+activeSubExistsSQL+")", "active", now)
		}
	}
	if params.HasSellableToken != nil {
		sellableExistsSQL := "SELECT 1 FROM tokens t WHERE t.user_id = users.id AND t.source_type = 'sellable_token' AND t.status = 1 AND t.deleted_at IS NULL"
		if *params.HasSellableToken {
			query = query.Where("EXISTS (" + sellableExistsSQL + ")")
		} else {
			query = query.Where("NOT EXISTS (" + sellableExistsSQL + ")")
		}
	}
	// 额度筛选基于"总额度 = quota + used_quota"。
	// 使用参数绑定占位符传值，避免把数值拼接到 SQL 字符串中。
	if params.BalanceMin != nil {
		query = query.Where("(quota + used_quota) >= ?", *params.BalanceMin)
	}
	if params.BalanceMax != nil {
		query = query.Where("(quota + used_quota) <= ?", *params.BalanceMax)
	}
	if params.UsedBalanceMin != nil {
		query = query.Where("used_quota >= ?", *params.UsedBalanceMin)
	}
	if params.UsedBalanceMax != nil {
		query = query.Where("used_quota <= ?", *params.UsedBalanceMax)
	}
	if params.InviteeUserID != nil {
		// 通过子查询拿到"被邀请人 -> 邀请人ID"，然后反查邀请人用户。
		// 子查询同样走参数绑定，避免原始 SQL 拼接带来的注入风险。
		subQuery := tx.Unscoped().
			Model(&User{}).
			Select("inviter_id").
			Where("id = ?", *params.InviteeUserID).
			Limit(1)
		query = query.Where("id = (?)", subQuery)
	}

	// 获取总数
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 对分页参数做兜底，确保极端情况下不会出现负偏移或零分页大小。
	startIdx := params.StartIdx
	if startIdx < 0 {
		startIdx = 0
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = common.ItemsPerPage
	}

	orderClause := buildUserOrderClause(params.SortBy, params.SortOrder, params.IdSortOrder, params.BalanceSortOrder)

	// 获取分页数据
	err = query.Omit("password").Order(orderClause).Limit(pageSize).Offset(startIdx).Find(&users).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = AttachUserInviteMetadata(tx, users); err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err = AttachUserSubscriptionMetadata(tx, users); err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err = AttachUserSellableTokenMetadata(tx, users); err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func AttachUserInviteMetadata(tx *gorm.DB, users []*User) error {
	if len(users) == 0 {
		return nil
	}

	inviterIDs := make([]int, 0, len(users))
	seen := make(map[int]struct{}, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		user.InviteeCount = user.AffCount
		if user.InviterId <= 0 {
			continue
		}
		if _, ok := seen[user.InviterId]; ok {
			continue
		}
		seen[user.InviterId] = struct{}{}
		inviterIDs = append(inviterIDs, user.InviterId)
	}
	if len(inviterIDs) == 0 {
		return nil
	}

	// 为当前页批量补充 inviter_username，避免前端逐行查询导致 N+1 问题。
	type inviterProjection struct {
		Id       int    `json:"id"`
		Username string `json:"username"`
	}
	var inviters []inviterProjection
	if err := tx.Unscoped().Model(&User{}).Select("id", "username").Where("id IN ?", inviterIDs).Find(&inviters).Error; err != nil {
		return err
	}
	nameByID := make(map[int]string, len(inviters))
	for _, inviter := range inviters {
		nameByID[inviter.Id] = inviter.Username
	}
	for _, user := range users {
		if user == nil || user.InviterId <= 0 {
			continue
		}
		user.InviterUsername = nameByID[user.InviterId]
	}
	return nil
}

func AttachUserSubscriptionMetadata(tx *gorm.DB, users []*User) error {
	if len(users) == 0 {
		return nil
	}
	userIDs := make([]int, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		userIDs = append(userIDs, user.Id)
		// 先写入默认值，确保"无套餐"用户也有明确响应字段。
		user.ActiveSubscriptionCount = 0
		user.HasActiveSubscription = false
		user.PendingSubscriptionIssuanceCount = 0
	}
	if len(userIDs) == 0 {
		return nil
	}

	now := common.GetTimestamp()
	type activeSubscriptionCount struct {
		UserId      int `gorm:"column:user_id"`
		ActiveCount int `gorm:"column:active_count"`
	}
	var counts []activeSubscriptionCount
	if err := tx.Model(&UserSubscription{}).
		Select("user_id, COUNT(*) AS active_count").
		Where("user_id IN ? AND status = ? AND end_time > ?", userIDs, "active", now).
		Group("user_id").
		Scan(&counts).Error; err != nil {
		return err
	}
	type pendingIssuanceCount struct {
		UserId int `gorm:"column:user_id"`
		Count  int `gorm:"column:cnt"`
	}
	var pendingCounts []pendingIssuanceCount
	if tx.Migrator().HasTable(&SubscriptionIssuance{}) {
		if err := tx.Model(&SubscriptionIssuance{}).
			Select("user_id, COUNT(*) AS cnt").
			Where("user_id IN ? AND status = ?", userIDs, SubscriptionIssuanceStatusPending).
			Group("user_id").
			Scan(&pendingCounts).Error; err != nil {
			return err
		}
	}

	countByUserID := make(map[int]int, len(counts))
	for _, row := range counts {
		countByUserID[row.UserId] = row.ActiveCount
	}
	pendingByUserID := make(map[int]int, len(pendingCounts))
	for _, row := range pendingCounts {
		pendingByUserID[row.UserId] = row.Count
	}
	for _, user := range users {
		if user == nil {
			continue
		}
		activeCount := countByUserID[user.Id]
		user.ActiveSubscriptionCount = activeCount
		user.PendingSubscriptionIssuanceCount = pendingByUserID[user.Id]
		user.HasActiveSubscription = activeCount > 0 || user.PendingSubscriptionIssuanceCount > 0
	}
	return nil
}

func AttachUserSellableTokenMetadata(tx *gorm.DB, users []*User) error {
	if len(users) == 0 {
		return nil
	}
	userIDs := make([]int, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		userIDs = append(userIDs, user.Id)
		user.HasSellableToken = false
		user.ActiveSellableTokenCount = 0
		user.PendingSellableIssuanceCount = 0
	}
	if len(userIDs) == 0 {
		return nil
	}

	type countRow struct {
		UserId int `gorm:"column:user_id"`
		Count  int `gorm:"column:cnt"`
	}
	var tokenCounts []countRow
	if tx.Migrator().HasTable(&Token{}) && tx.Migrator().HasColumn(&Token{}, "source_type") {
		if err := tx.Model(&Token{}).
			Select("user_id, COUNT(*) AS cnt").
			Where("user_id IN ? AND source_type = ?", userIDs, TokenSourceTypeSellableToken).
			Group("user_id").
			Scan(&tokenCounts).Error; err != nil {
			return err
		}
	}
	var pendingCounts []countRow
	if tx.Migrator().HasTable(&SellableTokenIssuance{}) {
		if err := tx.Model(&SellableTokenIssuance{}).
			Select("user_id, COUNT(*) AS cnt").
			Where("user_id IN ? AND status = ?", userIDs, SellableTokenIssuanceStatusPending).
			Group("user_id").
			Scan(&pendingCounts).Error; err != nil {
			return err
		}
	}

	tokenCountByUserID := make(map[int]int, len(tokenCounts))
	for _, row := range tokenCounts {
		tokenCountByUserID[row.UserId] = row.Count
	}
	pendingCountByUserID := make(map[int]int, len(pendingCounts))
	for _, row := range pendingCounts {
		pendingCountByUserID[row.UserId] = row.Count
	}
	for _, user := range users {
		if user == nil {
			continue
		}
		user.ActiveSellableTokenCount = tokenCountByUserID[user.Id]
		user.PendingSellableIssuanceCount = pendingCountByUserID[user.Id]
		user.HasSellableToken = user.ActiveSellableTokenCount > 0 || user.PendingSellableIssuanceCount > 0
	}
	return nil
}

func GetUserInviteRelations(userID int, startIdx int, pageSize int) (*User, *User, []*User, int64, *InviteIncomeSummary, error) {
	if userID <= 0 {
		return nil, nil, nil, 0, nil, errors.New("id 为空！")
	}
	if startIdx < 0 {
		startIdx = 0
	}
	if pageSize <= 0 {
		pageSize = common.ItemsPerPage
	}
	summary := &InviteIncomeSummary{}

	// 先查主用户，再查邀请人与被邀请人分页列表，方便前端一次渲染完整关系视图。
	user := &User{}
	if err := DB.Unscoped().Omit("password").First(user, "id = ?", userID).Error; err != nil {
		return nil, nil, nil, 0, nil, err
	}
	user.InviteeCount = user.AffCount

	var inviter *User
	if user.InviterId > 0 {
		inviter = &User{}
		if err := DB.Unscoped().Omit("password").First(inviter, "id = ?", user.InviterId).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil, nil, 0, nil, err
			}
			inviter = nil
		}
	}

	query := DB.Unscoped().Model(&User{}).Where("inviter_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, nil, nil, 0, nil, err
	}

	// 直接收益采用"当前配置口径"：
	// - 每个被邀请用户按 QuotaForInviter 计一次；
	// - 该口径实现简单、开销稳定，但不追溯历史配置变更。
	if common.QuotaForInviter > 0 {
		summary.DirectTotalQuota = int(total) * common.QuotaForInviter
	}
	// 充值返佣按台账汇总（仅统计 settled）：
	// - pending/skipped 不计入已到账收益；
	// - 这里按 inviter_user_id 聚合，得到"该用户作为邀请人"的总返佣。
	if err := DB.Model(&InviteCommissionLedger{}).
		Select("COALESCE(SUM(settled_quota), 0)").
		Where("inviter_user_id = ? AND status = ?", userID, InviteCommissionStatusSettled).
		Scan(&summary.RechargeTotalQuota).Error; err != nil {
		return nil, nil, nil, 0, nil, err
	}
	summary.TotalQuota = summary.DirectTotalQuota + summary.RechargeTotalQuota

	var invitees []*User
	if err := query.
		Omit("password").
		Order("id desc").
		Limit(pageSize).
		Offset(startIdx).
		Find(&invitees).Error; err != nil {
		return nil, nil, nil, 0, nil, err
	}

	rechargeByInvitee := map[int]int{}
	inviteeIDs := make([]int, 0, len(invitees))
	for _, invitee := range invitees {
		if invitee == nil || invitee.Id <= 0 {
			continue
		}
		inviteeIDs = append(inviteeIDs, invitee.Id)
	}
	if len(inviteeIDs) > 0 {
		type inviteeRechargeRow struct {
			InviteeUserID      int `gorm:"column:invitee_user_id"`
			RechargeTotalQuota int `gorm:"column:recharge_total_quota"`
		}
		var rows []inviteeRechargeRow
		// 为"当前页被邀请人"批量聚合返佣，避免前端逐行请求导致 N+1。
		if err := DB.Model(&InviteCommissionLedger{}).
			Select("invitee_user_id, COALESCE(SUM(settled_quota), 0) AS recharge_total_quota").
			Where("inviter_user_id = ? AND status = ? AND invitee_user_id IN ?", userID, InviteCommissionStatusSettled, inviteeIDs).
			Group("invitee_user_id").
			Find(&rows).Error; err != nil {
			return nil, nil, nil, 0, nil, err
		}
		for _, row := range rows {
			rechargeByInvitee[row.InviteeUserID] = row.RechargeTotalQuota
		}
	}
	for _, invitee := range invitees {
		if invitee != nil {
			invitee.InviteeCount = invitee.AffCount
			// 每个被邀请人对应一份"直接收益"（当前配置口径）。
			directIncome := 0
			if common.QuotaForInviter > 0 {
				directIncome = common.QuotaForInviter
			}
			invitee.InviteDirectIncomeQuota = directIncome
			// 充值返佣是按台账聚合后的真实已结算值。
			invitee.InviteRechargeCommissionQuota = rechargeByInvitee[invitee.Id]
			// 总收益用于前端主展示，拆分项用于 tooltip/明细展示。
			invitee.InviteTotalIncomeQuota = invitee.InviteDirectIncomeQuota + invitee.InviteRechargeCommissionQuota
		}
	}

	return user, inviter, invitees, total, summary, nil
}

type RebuildAffCountResult struct {
	UpdatedInviters int64 `json:"updated_inviters"`
	ResetRows       int64 `json:"reset_rows"`
}

func RebuildAffCount(targetUserID *int) (*RebuildAffCountResult, error) {
	// 该能力用于修复 aff_count 与真实邀请关系(inviter_id)不一致的问题。
	// 支持两种模式：
	// 1) 传入 targetUserID：仅重算单个邀请人的 aff_count。
	// 2) targetUserID 为空：全量重算所有用户的 aff_count。
	result := &RebuildAffCountResult{}
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if targetUserID != nil {
		if *targetUserID <= 0 {
			tx.Rollback()
			return nil, errors.New("user_id 无效")
		}

		var inviteeCount int64
		if err := tx.Unscoped().Model(&User{}).Where("inviter_id = ?", *targetUserID).Count(&inviteeCount).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		updateResult := tx.Unscoped().
			Model(&User{}).
			Where("id = ?", *targetUserID).
			Update("aff_count", int(inviteeCount))
		if updateResult.Error != nil {
			tx.Rollback()
			return nil, updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			tx.Rollback()
			return nil, gorm.ErrRecordNotFound
		}
		result.UpdatedInviters = 1
		if err := tx.Commit().Error; err != nil {
			return nil, err
		}
		return result, nil
	}

	// 全量模式：先按 inviter_id 聚合真实邀请数量，再回写 aff_count。
	// 这里全部使用 GORM 构造器与参数绑定，保证 SQLite/MySQL/PostgreSQL 兼容。
	type inviterCountRow struct {
		InviterID    int   `gorm:"column:inviter_id"`
		InviteeCount int64 `gorm:"column:invitee_count"`
	}
	var inviterCountRows []inviterCountRow
	if err := tx.Unscoped().
		Model(&User{}).
		Select("inviter_id, COUNT(*) AS invitee_count").
		Where("inviter_id > 0").
		Group("inviter_id").
		Find(&inviterCountRows).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 为了避免旧脏数据残留，先把非 0 的 aff_count 统一归零，再按聚合结果回填。
	resetResult := tx.Unscoped().Model(&User{}).Where("aff_count <> 0").Update("aff_count", 0)
	if resetResult.Error != nil {
		tx.Rollback()
		return nil, resetResult.Error
	}
	result.ResetRows = resetResult.RowsAffected

	var updatedInviters int64 = 0
	for _, row := range inviterCountRows {
		if row.InviterID <= 0 {
			continue
		}
		updateResult := tx.Unscoped().
			Model(&User{}).
			Where("id = ?", row.InviterID).
			Update("aff_count", int(row.InviteeCount))
		if updateResult.Error != nil {
			tx.Rollback()
			return nil, updateResult.Error
		}
		if updateResult.RowsAffected > 0 {
			updatedInviters += updateResult.RowsAffected
		}
	}
	result.UpdatedInviters = updatedInviters

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return result, nil
}

func GetUserById(id int, selectAll bool) (*User, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	user := User{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(&user, "id = ?", id).Error
	} else {
		err = DB.Omit("password").First(&user, "id = ?", id).Error
	}
	return &user, err
}

func GetUserIdByAffCode(affCode string) (int, error) {
	if affCode == "" {
		return 0, errors.New("affCode 为空！")
	}
	var user User
	err := DB.Select("id").First(&user, "aff_code = ?", affCode).Error
	return user.Id, err
}

func DeleteUserById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	user := User{Id: id}
	return user.Delete()
}

func HardDeleteUserById(id int) error {
	if id == 0 {
		return errors.New("id 为空！")
	}
	err := DB.Unscoped().Delete(&User{}, "id = ?", id).Error
	return err
}

func inviteUser(inviterId int) (err error) {
	if inviterId <= 0 {
		return errors.New("inviterId 为空！")
	}
	// 这里必须使用数据库原子自增而不是"先查后改再保存"：
	// 在高并发注册场景下，后者会出现并发覆盖（lost update），导致 aff_count 实际被少加。
	// 这正是"邀请关系查到 2 人，但直接邀请人数只有 1"这类数据漂移的常见根因。
	result := DB.Model(&User{}).
		Where("id = ?", inviterId).
		Updates(map[string]interface{}{
			"aff_count":   gorm.Expr("aff_count + 1"),
			"aff_quota":   gorm.Expr("aff_quota + ?", common.QuotaForInviter),
			"aff_history": gorm.Expr("aff_history + ?", common.QuotaForInviter),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (user *User) TransferAffQuotaToQuota(quota int) error {
	// 检查quota是否小于最小额度
	if float64(quota) < common.QuotaPerUnit {
		return fmt.Errorf("转移额度最小为%s！", logger.LogQuota(int(common.QuotaPerUnit)))
	}

	// 开始数据库事务
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback() // 确保在函数退出时事务能回滚

	// 加锁查询用户以确保数据一致性
	err := tx.Set("gorm:query_option", "FOR UPDATE").First(&user, user.Id).Error
	if err != nil {
		return err
	}

	// 再次检查用户的AffQuota是否足够
	if user.AffQuota < quota {
		return errors.New("邀请额度不足！")
	}

	// 更新用户额度
	user.AffQuota -= quota
	user.Quota += quota

	// 保存用户状态
	if err := tx.Save(user).Error; err != nil {
		return err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return err
	}
	// 用事务后的准确值覆盖 Redis 缓存，避免返回旧余额
	_ = updateUserQuotaCache(user.Id, user.Quota)
	return nil
}

func (user *User) Insert(inviterId int) error {
	var err error
	if user.Password != "" {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	user.Quota = common.QuotaForNewUser
	//user.SetAccessToken(common.GetUUID())
	user.AffCode = common.GetRandomString(4)

	// 初始化用户设置，包括默认的边栏配置
	if user.Setting == "" {
		defaultSetting := dto.UserSetting{}
		// 这里暂时不设置SidebarModules，因为需要在用户创建后根据角色设置
		user.SetSetting(defaultSetting)
	}

	result := DB.Create(user)
	if result.Error != nil {
		return result.Error
	}

	// 用户创建成功后，根据角色初始化边栏配置
	// 需要重新获取用户以确保有正确的ID和Role
	var createdUser User
	if err := DB.Where("username = ?", user.Username).First(&createdUser).Error; err == nil {
		// 生成基于角色的默认边栏配置
		defaultSidebarConfig := generateDefaultSidebarConfigForRole(createdUser.Role)
		if defaultSidebarConfig != "" {
			currentSetting := createdUser.GetSetting()
			currentSetting.SidebarModules = defaultSidebarConfig
			createdUser.SetSetting(currentSetting)
			createdUser.Update(false)
			common.SysLog(fmt.Sprintf("为新用户 %s (角色: %d) 初始化边栏配置", createdUser.Username, createdUser.Role))
		}
	}

	if common.QuotaForNewUser > 0 {
		RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("新用户注册赠送 %s", logger.LogQuota(common.QuotaForNewUser)))
	}
	if inviterId != 0 {
		if common.QuotaForInvitee > 0 {
			_ = IncreaseUserQuota(user.Id, common.QuotaForInvitee, true)
			RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("使用邀请码赠送 %s", logger.LogQuota(common.QuotaForInvitee)))
		}
		// 邀请人数统计必须独立于邀请奖励额度开关：
		// 即使 QuotaForInviter=0（不发固定邀请奖励），也要累计 aff_count，
		// 否则会出现"邀请关系已建立但邀请人数始终为 0"的错误展示。
		if err := inviteUser(inviterId); err != nil {
			// 邀请关系统计属于关键业务指标，这里显式记录错误便于排查数据不一致问题。
			common.SysError(fmt.Sprintf("更新邀请统计失败(inviter_id=%d,user_id=%d): %s", inviterId, user.Id, err.Error()))
		}
		if common.QuotaForInviter > 0 {
			//_ = IncreaseUserQuota(inviterId, common.QuotaForInviter)
			RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("邀请用户赠送 %s", logger.LogQuota(common.QuotaForInviter)))
		}
	}
	return nil
}

// InsertWithTx inserts a new user within an existing transaction.
// This is used for OAuth registration where user creation and binding need to be atomic.
// Post-creation tasks (sidebar config, logs, inviter rewards) are handled after the transaction commits.
func (user *User) InsertWithTx(tx *gorm.DB, inviterId int) error {
	var err error
	if user.Password != "" {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	user.Quota = common.QuotaForNewUser
	user.AffCode = common.GetRandomString(4)

	// 初始化用户设置
	if user.Setting == "" {
		defaultSetting := dto.UserSetting{}
		user.SetSetting(defaultSetting)
	}

	result := tx.Create(user)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// FinalizeOAuthUserCreation performs post-transaction tasks for OAuth user creation.
// This should be called after the transaction commits successfully.
func (user *User) FinalizeOAuthUserCreation(inviterId int) {
	// 用户创建成功后，根据角色初始化边栏配置
	var createdUser User
	if err := DB.Where("id = ?", user.Id).First(&createdUser).Error; err == nil {
		defaultSidebarConfig := generateDefaultSidebarConfigForRole(createdUser.Role)
		if defaultSidebarConfig != "" {
			currentSetting := createdUser.GetSetting()
			currentSetting.SidebarModules = defaultSidebarConfig
			createdUser.SetSetting(currentSetting)
			createdUser.Update(false)
			common.SysLog(fmt.Sprintf("为新用户 %s (角色: %d) 初始化边栏配置", createdUser.Username, createdUser.Role))
		}
	}

	if common.QuotaForNewUser > 0 {
		RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("新用户注册赠送 %s", logger.LogQuota(common.QuotaForNewUser)))
	}
	if inviterId != 0 {
		if common.QuotaForInvitee > 0 {
			_ = IncreaseUserQuota(user.Id, common.QuotaForInvitee, true)
			RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("使用邀请码赠送 %s", logger.LogQuota(common.QuotaForInvitee)))
		}
		// 与普通注册保持一致：邀请人数统计不依赖 QuotaForInviter。
		// 这样 OAuth 注册链路也能在"奖励为 0"时正确累计邀请人数。
		if err := inviteUser(inviterId); err != nil {
			common.SysError(fmt.Sprintf("更新邀请统计失败(inviter_id=%d,user_id=%d): %s", inviterId, user.Id, err.Error()))
		}
		if common.QuotaForInviter > 0 {
			RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("邀请用户赠送 %s", logger.LogQuota(common.QuotaForInviter)))
		}
	}
}

func (user *User) Update(updatePassword bool) error {
	var err error
	if updatePassword {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	newUser := *user
	DB.First(&user, user.Id)
	if err = DB.Model(user).Updates(newUser).Error; err != nil {
		return err
	}

	// Update cache
	return updateUserCache(*user)
}

func (user *User) Edit(updatePassword bool) error {
	var err error
	if updatePassword {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}

	newUser := *user
	updates := map[string]interface{}{
		"username":     newUser.Username,
		"display_name": newUser.DisplayName,
		"group":        newUser.Group,
		"quota":        newUser.Quota,
		"remark":       newUser.Remark,
	}
	if updatePassword {
		updates["password"] = newUser.Password
	}

	DB.First(&user, user.Id)
	if err = DB.Model(user).Updates(updates).Error; err != nil {
		return err
	}

	// Update cache
	return updateUserCache(*user)
}

func (user *User) ClearBinding(bindingType string) error {
	if user.Id == 0 {
		return errors.New("user id is empty")
	}

	bindingColumnMap := map[string]string{
		"email":    "email",
		"github":   "github_id",
		"discord":  "discord_id",
		"oidc":     "oidc_id",
		"wechat":   "wechat_id",
		"telegram": "telegram_id",
		"linuxdo":  "linux_do_id",
	}

	column, ok := bindingColumnMap[bindingType]
	if !ok {
		return errors.New("invalid binding type")
	}

	if err := DB.Model(&User{}).Where("id = ?", user.Id).Update(column, "").Error; err != nil {
		return err
	}

	if err := DB.Where("id = ?", user.Id).First(user).Error; err != nil {
		return err
	}

	return updateUserCache(*user)
}

func (user *User) Delete() error {
	if user.Id == 0 {
		return errors.New("id 为空！")
	}
	if err := DB.Delete(user).Error; err != nil {
		return err
	}

	// 清除缓存
	return invalidateUserCache(user.Id)
}

func (user *User) HardDelete() error {
	if user.Id == 0 {
		return errors.New("id 为空！")
	}
	err := DB.Unscoped().Delete(user).Error
	return err
}

// ValidateAndFill check password & user status
func (user *User) ValidateAndFill() (err error) {
	// When querying with struct, GORM will only query with non-zero fields,
	// that means if your field's value is 0, '', false or other zero values,
	// it won't be used to build query conditions
	password := user.Password
	username := strings.TrimSpace(user.Username)
	if username == "" || password == "" {
		return errors.New("用户名或密码为空")
	}
	// find buy username or email
	DB.Where("username = ? OR email = ?", username, username).First(user)
	okay := common.ValidatePasswordAndHash(password, user.Password)
	if !okay || user.Status != common.UserStatusEnabled {
		return errors.New("用户名或密码错误，或用户已被封禁")
	}
	return nil
}

func (user *User) FillUserById() error {
	if user.Id == 0 {
		return errors.New("id 为空！")
	}
	DB.Where(User{Id: user.Id}).First(user)
	return nil
}

func (user *User) FillUserByEmail() error {
	if user.Email == "" {
		return errors.New("email 为空！")
	}
	DB.Where(User{Email: user.Email}).First(user)
	return nil
}

func (user *User) FillUserByGitHubId() error {
	if user.GitHubId == "" {
		return errors.New("GitHub id 为空！")
	}
	DB.Where(User{GitHubId: user.GitHubId}).First(user)
	return nil
}

// UpdateGitHubId updates the user's GitHub ID (used for migration from login to numeric ID)
func (user *User) UpdateGitHubId(newGitHubId string) error {
	if user.Id == 0 {
		return errors.New("user id is empty")
	}
	return DB.Model(user).Update("github_id", newGitHubId).Error
}

func (user *User) FillUserByDiscordId() error {
	if user.DiscordId == "" {
		return errors.New("discord id 为空！")
	}
	DB.Where(User{DiscordId: user.DiscordId}).First(user)
	return nil
}

func (user *User) FillUserByOidcId() error {
	if user.OidcId == "" {
		return errors.New("oidc id 为空！")
	}
	DB.Where(User{OidcId: user.OidcId}).First(user)
	return nil
}

func (user *User) FillUserByWeChatId() error {
	if user.WeChatId == "" {
		return errors.New("WeChat id 为空！")
	}
	DB.Where(User{WeChatId: user.WeChatId}).First(user)
	return nil
}

func (user *User) FillUserByTelegramId() error {
	if user.TelegramId == "" {
		return errors.New("Telegram id 为空！")
	}
	err := DB.Where(User{TelegramId: user.TelegramId}).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("该 Telegram 账户未绑定")
	}
	return nil
}

func IsEmailAlreadyTaken(email string) bool {
	return DB.Unscoped().Where("email = ?", email).Find(&User{}).RowsAffected == 1
}

func IsWeChatIdAlreadyTaken(wechatId string) bool {
	return DB.Unscoped().Where("wechat_id = ?", wechatId).Find(&User{}).RowsAffected == 1
}

func IsGitHubIdAlreadyTaken(githubId string) bool {
	return DB.Unscoped().Where("github_id = ?", githubId).Find(&User{}).RowsAffected == 1
}

func IsDiscordIdAlreadyTaken(discordId string) bool {
	return DB.Unscoped().Where("discord_id = ?", discordId).Find(&User{}).RowsAffected == 1
}

func IsOidcIdAlreadyTaken(oidcId string) bool {
	return DB.Where("oidc_id = ?", oidcId).Find(&User{}).RowsAffected == 1
}

func IsTelegramIdAlreadyTaken(telegramId string) bool {
	return DB.Unscoped().Where("telegram_id = ?", telegramId).Find(&User{}).RowsAffected == 1
}

func ResetUserPasswordByEmail(email string, password string) error {
	if email == "" || password == "" {
		return errors.New("邮箱地址或密码为空！")
	}
	hashedPassword, err := common.Password2Hash(password)
	if err != nil {
		return err
	}
	err = DB.Model(&User{}).Where("email = ?", email).Update("password", hashedPassword).Error
	return err
}

func IsAdmin(userId int) bool {
	if userId == 0 {
		return false
	}
	var user User
	err := DB.Where("id = ?", userId).Select("role").Find(&user).Error
	if err != nil {
		common.SysLog("no such user " + err.Error())
		return false
	}
	return user.Role >= common.RoleAdminUser
}

//// IsUserEnabled checks user status from Redis first, falls back to DB if needed
//func IsUserEnabled(id int, fromDB bool) (status bool, err error) {
//	defer func() {
//		// Update Redis cache asynchronously on successful DB read
//		if shouldUpdateRedis(fromDB, err) {
//			gopool.Go(func() {
//				if err := updateUserStatusCache(id, status); err != nil {
//					common.SysError("failed to update user status cache: " + err.Error())
//				}
//			})
//		}
//	}()
//	if !fromDB && common.RedisEnabled {
//		// Try Redis first
//		status, err := getUserStatusCache(id)
//		if err == nil {
//			return status == common.UserStatusEnabled, nil
//		}
//		// Don't return error - fall through to DB
//	}
//	fromDB = true
//	var user User
//	err = DB.Where("id = ?", id).Select("status").Find(&user).Error
//	if err != nil {
//		return false, err
//	}
//
//	return user.Status == common.UserStatusEnabled, nil
//}

func ValidateAccessToken(token string) (user *User) {
	if token == "" {
		return nil
	}
	token = strings.Replace(token, "Bearer ", "", 1)
	user = &User{}
	if DB.Where("access_token = ?", token).First(user).RowsAffected == 1 {
		return user
	}
	return nil
}

// GetUserQuota gets quota from Redis first, falls back to DB if needed
func GetUserQuota(id int, fromDB bool) (quota int, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserQuotaCache(id, quota); err != nil {
					common.SysLog("failed to update user quota cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		quota, err := getUserQuotaCache(id)
		if err == nil {
			return quota, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select("quota").Find(&quota).Error
	if err != nil {
		return 0, err
	}

	return quota, nil
}

func GetUserUsedQuota(id int) (quota int, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("used_quota").Find(&quota).Error
	return quota, err
}

func GetUserEmail(id int) (email string, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("email").Find(&email).Error
	return email, err
}

// GetUserGroup gets group from Redis first, falls back to DB if needed
func GetUserGroup(id int, fromDB bool) (group string, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserGroupCache(id, group); err != nil {
					common.SysLog("failed to update user group cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		group, err := getUserGroupCache(id)
		if err == nil {
			return group, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select(commonGroupCol).Find(&group).Error
	if err != nil {
		return "", err
	}

	return group, nil
}

// GetUserSetting gets setting from Redis first, falls back to DB if needed
func GetUserSetting(id int, fromDB bool) (settingMap dto.UserSetting, err error) {
	var setting string
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserSettingCache(id, setting); err != nil {
					common.SysLog("failed to update user setting cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		setting, err := getUserSettingCache(id)
		if err == nil {
			return setting, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	// can be nil setting
	var safeSetting sql.NullString
	err = DB.Model(&User{}).Where("id = ?", id).Select("setting").Find(&safeSetting).Error
	if err != nil {
		return settingMap, err
	}
	if safeSetting.Valid {
		setting = safeSetting.String
	} else {
		setting = ""
	}
	userBase := &UserBase{
		Setting: setting,
	}
	return userBase.GetSetting(), nil
}

func IncreaseUserQuota(id int, quota int, db bool) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	gopool.Go(func() {
		err := cacheIncrUserQuota(id, int64(quota))
		if err != nil {
			common.SysLog("failed to increase user quota: " + err.Error())
		}
	})
	if !db && common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUserQuota, id, quota)
		return nil
	}
	return increaseUserQuota(id, quota)
}

func increaseUserQuota(id int, quota int) (err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Update("quota", gorm.Expr("quota + ?", quota)).Error
	if err != nil {
		return err
	}
	return err
}

func DecreaseUserQuota(id int, quota int) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	gopool.Go(func() {
		err := cacheDecrUserQuota(id, int64(quota))
		if err != nil {
			common.SysLog("failed to decrease user quota: " + err.Error())
		}
	})
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUserQuota, id, -quota)
		return nil
	}
	return decreaseUserQuota(id, quota)
}

func decreaseUserQuota(id int, quota int) (err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Update("quota", gorm.Expr("quota - ?", quota)).Error
	if err != nil {
		return err
	}
	return err
}

func DeltaUpdateUserQuota(id int, delta int) (err error) {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		return IncreaseUserQuota(id, delta, false)
	} else {
		return DecreaseUserQuota(id, -delta)
	}
}

//func GetRootUserEmail() (email string) {
//	DB.Model(&User{}).Where("role = ?", common.RoleRootUser).Select("email").Find(&email)
//	return email
//}

func GetRootUser() (user *User) {
	DB.Where("role = ?", common.RoleRootUser).First(&user)
	return user
}

func UpdateUserUsedQuotaAndRequestCount(id int, quota int) {
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUsedQuota, id, quota)
		addNewRecord(BatchUpdateTypeRequestCount, id, 1)
		return
	}
	updateUserUsedQuotaAndRequestCount(id, quota, 1)
}

func updateUserUsedQuotaAndRequestCount(id int, quota int, count int) {
	err := DB.Model(&User{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"used_quota":    gorm.Expr("used_quota + ?", quota),
			"request_count": gorm.Expr("request_count + ?", count),
		},
	).Error
	if err != nil {
		common.SysLog("failed to update user used quota and request count: " + err.Error())
		return
	}

	//// 更新缓存
	//if err := invalidateUserCache(id); err != nil {
	//	common.SysError("failed to invalidate user cache: " + err.Error())
	//}
}

func updateUserUsedQuota(id int, quota int) {
	err := DB.Model(&User{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"used_quota": gorm.Expr("used_quota + ?", quota),
		},
	).Error
	if err != nil {
		common.SysLog("failed to update user used quota: " + err.Error())
	}
}

// UpdateUserRequestCount 仅更新用户请求计数，不更新已使用额度。
// 用于可售令牌等仅令牌计费模式，避免令牌消费被计入用户钱包已使用额度。
func UpdateUserRequestCount(id int) {
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeRequestCount, id, 1)
		return
	}
	updateUserRequestCount(id, 1)
}

func updateUserRequestCount(id int, count int) {
	err := DB.Model(&User{}).Where("id = ?", id).Update("request_count", gorm.Expr("request_count + ?", count)).Error
	if err != nil {
		common.SysLog("failed to update user request count: " + err.Error())
	}
}

// GetUsernameById gets username from Redis first, falls back to DB if needed
func GetUsernameById(id int, fromDB bool) (username string, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserNameCache(id, username); err != nil {
					common.SysLog("failed to update user name cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		username, err := getUserNameCache(id)
		if err == nil {
			return username, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select("username").Find(&username).Error
	if err != nil {
		return "", err
	}

	return username, nil
}

func IsLinuxDOIdAlreadyTaken(linuxDOId string) bool {
	var user User
	err := DB.Unscoped().Where("linux_do_id = ?", linuxDOId).First(&user).Error
	return !errors.Is(err, gorm.ErrRecordNotFound)
}

func (user *User) FillUserByLinuxDOId() error {
	if user.LinuxDOId == "" {
		return errors.New("linux do id is empty")
	}
	err := DB.Where("linux_do_id = ?", user.LinuxDOId).First(user).Error
	return err
}

func RootUserExists() bool {
	var user User
	err := DB.Where("role = ?", common.RoleRootUser).First(&user).Error
	if err != nil {
		return false
	}
	return true
}
