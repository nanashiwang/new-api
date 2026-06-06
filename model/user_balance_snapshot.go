package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const userBalanceSnapshotHour = 4

type UserBalanceSnapshot struct {
	Id                 int    `json:"id"`
	SnapshotDate       string `json:"snapshot_date" gorm:"type:varchar(10);uniqueIndex;not null"`
	SnapshotAt         int64  `json:"snapshot_at" gorm:"index;not null"`
	TotalQuota         int64  `json:"total_quota" gorm:"type:bigint;not null;default:0"`
	TotalPositiveQuota int64  `json:"total_positive_quota" gorm:"type:bigint;not null;default:0"`
	UserCount          int    `json:"user_count" gorm:"type:int;not null;default:0"`
	PositiveUserCount  int    `json:"positive_user_count" gorm:"type:int;not null;default:0"`
	NegativeUserCount  int    `json:"negative_user_count" gorm:"type:int;not null;default:0"`
	Top10Quota         int64  `json:"top10_quota" gorm:"type:bigint;not null;default:0"`
	TopUsersJson       string `json:"-" gorm:"type:text"`
	NegativeUsersJson  string `json:"-" gorm:"type:text"`
	CreatedAt          int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

type UserBalanceSnapshotUser struct {
	Id                   int    `json:"id"`
	Username             string `json:"username"`
	DisplayName          string `json:"display_name"`
	Quota                int64  `json:"quota"`
	BalanceTrendDisabled bool   `json:"balance_trend_disabled"`
}

type UserBalanceSnapshotPoint struct {
	Id                   int     `json:"id"`
	SnapshotDate         string  `json:"snapshot_date"`
	SnapshotAt           int64   `json:"snapshot_at"`
	TotalQuota           int64   `json:"total_quota"`
	TotalPositiveQuota   int64   `json:"total_positive_quota"`
	UserCount            int     `json:"user_count"`
	PositiveUserCount    int     `json:"positive_user_count"`
	NegativeUserCount    int     `json:"negative_user_count"`
	Top10Quota           int64   `json:"top10_quota"`
	Top10Share           float64 `json:"top10_share"`
	AverageQuota         float64 `json:"average_quota"`
	AveragePositiveQuota float64 `json:"average_positive_quota"`
}

type UserBalanceSnapshotReport struct {
	Snapshots     []UserBalanceSnapshotPoint `json:"snapshots"`
	Latest        *UserBalanceSnapshotPoint  `json:"latest"`
	Previous      *UserBalanceSnapshotPoint  `json:"previous"`
	DeltaQuota    int64                      `json:"delta_quota"`
	DeltaRate     float64                    `json:"delta_rate"`
	TopUsers      []UserBalanceSnapshotUser  `json:"top_users"`
	NegativeUsers []UserBalanceSnapshotUser  `json:"negative_users"`
}

type UserBalanceTrendUserSearchParams struct {
	Keyword       string
	IncludeStatus string
	StartIdx      int
	PageSize      int
}

type userBalanceAggregateRow struct {
	TotalQuota         int64
	TotalPositiveQuota int64
	UserCount          int64
	PositiveUserCount  int64
	NegativeUserCount  int64
}

var (
	userBalanceSnapshotTaskOnce    sync.Once
	userBalanceSnapshotTaskRunning atomic.Bool
)

func StartUserBalanceSnapshotTask() {
	userBalanceSnapshotTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			common.SysLog("user balance snapshot task started: schedule=04:00 local time")
			for {
				wait := durationUntilNextLocalHour(time.Now(), userBalanceSnapshotHour)
				timer := time.NewTimer(wait)
				<-timer.C
				timer.Stop()
				if _, err := SaveUserBalanceSnapshot(time.Now()); err != nil {
					common.SysError("user balance snapshot failed: " + err.Error())
				}
			}
		}()
	})
}

func durationUntilNextLocalHour(now time.Time, hour int) time.Duration {
	if hour < 0 || hour > 23 {
		hour = userBalanceSnapshotHour
	}
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !now.Before(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next.Sub(now)
}

func SaveUserBalanceSnapshot(now time.Time) (*UserBalanceSnapshot, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	if !userBalanceSnapshotTaskRunning.CompareAndSwap(false, true) {
		return nil, errors.New("user balance snapshot is running")
	}
	defer userBalanceSnapshotTaskRunning.Store(false)

	snapshotDate := now.Format("2006-01-02")
	snapshotAt := now.Unix()
	aggregate, err := queryUserBalanceAggregate()
	if err != nil {
		return nil, err
	}
	topUsers, err := queryUserBalanceTopUsers(10)
	if err != nil {
		return nil, err
	}
	negativeUsers, err := queryUserBalanceNegativeUsers(10)
	if err != nil {
		return nil, err
	}
	topUsersJson, err := common.Marshal(topUsers)
	if err != nil {
		return nil, err
	}
	negativeUsersJson, err := common.Marshal(negativeUsers)
	if err != nil {
		return nil, err
	}

	top10Quota := int64(0)
	for _, user := range topUsers {
		top10Quota += user.Quota
	}

	snapshot := &UserBalanceSnapshot{
		SnapshotDate:       snapshotDate,
		SnapshotAt:         snapshotAt,
		TotalQuota:         aggregate.TotalQuota,
		TotalPositiveQuota: aggregate.TotalPositiveQuota,
		UserCount:          int(aggregate.UserCount),
		PositiveUserCount:  int(aggregate.PositiveUserCount),
		NegativeUserCount:  int(aggregate.NegativeUserCount),
		Top10Quota:         top10Quota,
		TopUsersJson:       string(topUsersJson),
		NegativeUsersJson:  string(negativeUsersJson),
	}

	var existing UserBalanceSnapshot
	if err := DB.Where("snapshot_date = ?", snapshotDate).First(&existing).Error; err == nil {
		updates := map[string]interface{}{
			"snapshot_at":          snapshot.SnapshotAt,
			"total_quota":          snapshot.TotalQuota,
			"total_positive_quota": snapshot.TotalPositiveQuota,
			"user_count":           snapshot.UserCount,
			"positive_user_count":  snapshot.PositiveUserCount,
			"negative_user_count":  snapshot.NegativeUserCount,
			"top10_quota":          snapshot.Top10Quota,
			"top_users_json":       snapshot.TopUsersJson,
			"negative_users_json":  snapshot.NegativeUsersJson,
		}
		if err := DB.Model(&existing).Updates(updates).Error; err != nil {
			return nil, err
		}
		return GetUserBalanceSnapshotByDate(snapshotDate)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := DB.Create(snapshot).Error; err != nil {
		return nil, err
	}
	return snapshot, nil
}

func queryUserBalanceAggregate() (userBalanceAggregateRow, error) {
	var aggregate userBalanceAggregateRow
	err := DB.Model(&User{}).Where("balance_trend_disabled = ?", false).Select(`
		COALESCE(SUM(quota), 0) AS total_quota,
		COALESCE(SUM(CASE WHEN quota > 0 THEN quota ELSE 0 END), 0) AS total_positive_quota,
		COUNT(*) AS user_count,
		COALESCE(SUM(CASE WHEN quota > 0 THEN 1 ELSE 0 END), 0) AS positive_user_count,
		COALESCE(SUM(CASE WHEN quota < 0 THEN 1 ELSE 0 END), 0) AS negative_user_count
	`).Scan(&aggregate).Error
	return aggregate, err
}

func queryUserBalanceTopUsers(limit int) ([]UserBalanceSnapshotUser, error) {
	users := make([]UserBalanceSnapshotUser, 0, limit)
	err := DB.Model(&User{}).
		Select("id, username, display_name, quota").
		Where("balance_trend_disabled = ? AND quota > 0", false).
		Order("quota desc").
		Order("id asc").
		Limit(limit).
		Scan(&users).Error
	return users, err
}

func queryUserBalanceNegativeUsers(limit int) ([]UserBalanceSnapshotUser, error) {
	users := make([]UserBalanceSnapshotUser, 0, limit)
	err := DB.Model(&User{}).
		Select("id, username, display_name, quota").
		Where("balance_trend_disabled = ? AND quota < 0", false).
		Order("quota asc").
		Order("id asc").
		Limit(limit).
		Scan(&users).Error
	return users, err
}

func GetUserBalanceSnapshotByDate(snapshotDate string) (*UserBalanceSnapshot, error) {
	var snapshot UserBalanceSnapshot
	if err := DB.Where("snapshot_date = ?", snapshotDate).First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func GetUserBalanceSnapshotReport(startTime int64, endTime int64) (*UserBalanceSnapshotReport, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	if endTime <= 0 {
		endTime = time.Now().Unix()
	}
	if startTime <= 0 || startTime > endTime {
		startTime = time.Unix(endTime, 0).AddDate(0, -1, 0).Unix()
	}

	var snapshots []UserBalanceSnapshot
	if err := DB.Where("snapshot_at >= ? AND snapshot_at <= ?", startTime, endTime).
		Order("snapshot_at asc").
		Find(&snapshots).Error; err != nil {
		return nil, err
	}

	report := &UserBalanceSnapshotReport{
		Snapshots: make([]UserBalanceSnapshotPoint, 0, len(snapshots)),
	}
	for _, snapshot := range snapshots {
		point := buildUserBalanceSnapshotPoint(snapshot)
		report.Snapshots = append(report.Snapshots, point)
	}
	if len(snapshots) > 0 {
		latestSnapshot := snapshots[len(snapshots)-1]
		latest := buildUserBalanceSnapshotPoint(latestSnapshot)
		report.Latest = &latest
		report.TopUsers = decodeUserBalanceSnapshotUsers(latestSnapshot.TopUsersJson)
		report.NegativeUsers = decodeUserBalanceSnapshotUsers(latestSnapshot.NegativeUsersJson)
	}
	if len(snapshots) > 1 {
		previous := buildUserBalanceSnapshotPoint(snapshots[len(snapshots)-2])
		report.Previous = &previous
		report.DeltaQuota = report.Latest.TotalQuota - previous.TotalQuota
		if previous.TotalQuota != 0 {
			report.DeltaRate = float64(report.DeltaQuota) / float64(previous.TotalQuota)
		}
	}
	return report, nil
}

func GetUserBalanceSnapshotReportByDays(days int) (*UserBalanceSnapshotReport, error) {
	if days <= 0 {
		days = 7
	}
	if days > 365 {
		days = 365
	}
	endTime := time.Now().Unix()
	startTime := time.Unix(endTime, 0).AddDate(0, 0, -days).Unix()
	return GetUserBalanceSnapshotReport(startTime, endTime)
}

func SearchUserBalanceTrendUsers(params UserBalanceTrendUserSearchParams) ([]UserBalanceSnapshotUser, int64, error) {
	if DB == nil {
		return nil, 0, errors.New("database not initialized")
	}
	users := make([]UserBalanceSnapshotUser, 0)
	var total int64
	query := DB.Model(&User{})

	keyword := strings.TrimSpace(params.Keyword)
	if keyword != "" {
		likeCondition := "username LIKE ? OR email LIKE ? OR display_name LIKE ? OR remark LIKE ?"
		if keywordInt, parseErr := strconv.Atoi(keyword); parseErr == nil {
			query = query.Where(
				"id = ? OR "+likeCondition,
				keywordInt, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
			)
		} else {
			query = query.Where(likeCondition, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}
	}

	switch strings.TrimSpace(params.IncludeStatus) {
	case "included":
		query = query.Where("balance_trend_disabled = ?", false)
	case "excluded":
		query = query.Where("balance_trend_disabled = ?", true)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	startIdx := params.StartIdx
	if startIdx < 0 {
		startIdx = 0
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = common.ItemsPerPage
	}

	err := query.Select("id, username, display_name, quota, balance_trend_disabled").
		Order("quota desc").
		Order("id asc").
		Limit(pageSize).
		Offset(startIdx).
		Scan(&users).Error
	return users, total, err
}

func UpdateUserBalanceTrendDisabled(userId int, disabled bool) (*UserBalanceSnapshotUser, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	if err := DB.Model(&User{}).Where("id = ?", userId).Update("balance_trend_disabled", disabled).Error; err != nil {
		return nil, err
	}
	if err := syncUserCacheByID(userId); err != nil {
		return nil, err
	}
	var user UserBalanceSnapshotUser
	if err := DB.Model(&User{}).
		Select("id, username, display_name, quota, balance_trend_disabled").
		Where("id = ?", userId).
		First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func buildUserBalanceSnapshotPoint(snapshot UserBalanceSnapshot) UserBalanceSnapshotPoint {
	point := UserBalanceSnapshotPoint{
		Id:                 snapshot.Id,
		SnapshotDate:       snapshot.SnapshotDate,
		SnapshotAt:         snapshot.SnapshotAt,
		TotalQuota:         snapshot.TotalQuota,
		TotalPositiveQuota: snapshot.TotalPositiveQuota,
		UserCount:          snapshot.UserCount,
		PositiveUserCount:  snapshot.PositiveUserCount,
		NegativeUserCount:  snapshot.NegativeUserCount,
		Top10Quota:         snapshot.Top10Quota,
	}
	if snapshot.TotalPositiveQuota > 0 {
		point.Top10Share = float64(snapshot.Top10Quota) / float64(snapshot.TotalPositiveQuota)
	}
	if snapshot.UserCount > 0 {
		point.AverageQuota = float64(snapshot.TotalQuota) / float64(snapshot.UserCount)
	}
	if snapshot.PositiveUserCount > 0 {
		point.AveragePositiveQuota = float64(snapshot.TotalPositiveQuota) / float64(snapshot.PositiveUserCount)
	}
	return point
}

func decodeUserBalanceSnapshotUsers(value string) []UserBalanceSnapshotUser {
	if value == "" {
		return []UserBalanceSnapshotUser{}
	}
	var users []UserBalanceSnapshotUser
	if err := common.Unmarshal([]byte(value), &users); err != nil {
		common.SysError(fmt.Sprintf("decode user balance snapshot users failed: %s", err.Error()))
		return []UserBalanceSnapshotUser{}
	}
	return users
}
