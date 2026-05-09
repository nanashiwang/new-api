package model

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const ipBlacklistCacheTTL = 30 * time.Second
const ipBlacklistReasonMaxLength = 255

var (
	ErrInvalidIPBlacklistRule = errors.New("无效的 IP 或 CIDR")
	ErrIPBlacklistNotFound    = errors.New("IP 黑名单规则不存在")
)

type IPBlacklist struct {
	Id           int    `json:"id"`
	IP           string `json:"ip" gorm:"type:varchar(128);column:ip;index"`
	CIDR         string `json:"cidr" gorm:"type:varchar(128);column:cidr;uniqueIndex"`
	IPVersion    int    `json:"ip_version" gorm:"type:int;column:ip_version;index"`
	Reason       string `json:"reason" gorm:"type:varchar(255);column:reason"`
	SourceUserId int    `json:"source_user_id" gorm:"type:int;column:source_user_id;index"`
	CreatedBy    int    `json:"created_by" gorm:"type:int;column:created_by;index"`
	CreatedAt    int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt    int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

type CompiledIPBlacklistRule struct {
	Id      int
	CIDR    string
	Network *net.IPNet
}

type IPBlacklistBatchFailure struct {
	UserID  int    `json:"user_id"`
	IP      string `json:"ip"`
	Message string `json:"message"`
}

type IPBlacklistBatchResult struct {
	CreatedCount  int                       `json:"created_count"`
	ExistingCount int                       `json:"existing_count"`
	SkippedCount  int                       `json:"skipped_count"`
	FailedCount   int                       `json:"failed_count"`
	Items         []*IPBlacklist            `json:"items"`
	Failed        []IPBlacklistBatchFailure `json:"failed"`
}

var ipBlacklistCache = struct {
	sync.RWMutex
	rules     []CompiledIPBlacklistRule
	expiresAt time.Time
}{}

func NormalizeIPBlacklistRule(input string) (string, int, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return "", 0, ErrInvalidIPBlacklistRule
	}

	if ip, network, err := net.ParseCIDR(raw); err == nil && network != nil {
		_, bits := network.Mask.Size()
		switch bits {
		case 32:
			ip4 := ip.To4()
			if ip4 == nil {
				return "", 0, ErrInvalidIPBlacklistRule
			}
			network.IP = ip4.Mask(network.Mask)
			return network.String(), 4, nil
		case 128:
			ip16 := ip.To16()
			if ip16 == nil {
				return "", 0, ErrInvalidIPBlacklistRule
			}
			network.IP = ip16.Mask(network.Mask)
			return network.String(), 6, nil
		default:
			return "", 0, ErrInvalidIPBlacklistRule
		}
	}

	ip := net.ParseIP(raw)
	if ip == nil {
		return "", 0, ErrInvalidIPBlacklistRule
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.String() + "/32", 4, nil
	}
	ip16 := ip.To16()
	if ip16 == nil {
		return "", 0, ErrInvalidIPBlacklistRule
	}
	return ip16.String() + "/128", 6, nil
}

func DoesIPBlacklistRuleMatchIP(ruleInput string, ipInput string) (bool, error) {
	cidr, _, err := NormalizeIPBlacklistRule(ruleInput)
	if err != nil {
		return false, err
	}
	ip := net.ParseIP(strings.TrimSpace(ipInput))
	if ip == nil {
		return false, ErrInvalidIPBlacklistRule
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	return network.Contains(ip), nil
}

func InvalidateIPBlacklistCache() {
	ipBlacklistCache.Lock()
	defer ipBlacklistCache.Unlock()
	ipBlacklistCache.rules = nil
	ipBlacklistCache.expiresAt = time.Time{}
}

func GetIPBlacklistByCIDR(cidr string) (*IPBlacklist, error) {
	var item IPBlacklist
	result := DB.Where("cidr = ?", cidr).Limit(1).Find(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func GetIPBlacklistByID(id int) (*IPBlacklist, error) {
	if id <= 0 {
		return nil, ErrIPBlacklistNotFound
	}
	var item IPBlacklist
	result := DB.Where("id = ?", id).Limit(1).Find(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrIPBlacklistNotFound
	}
	return &item, nil
}

func CreateIPBlacklist(ipInput string, reason string, sourceUserID int, createdBy int) (*IPBlacklist, bool, error) {
	raw := strings.TrimSpace(ipInput)
	cidr, version, err := NormalizeIPBlacklistRule(raw)
	if err != nil {
		return nil, false, err
	}

	existing, err := GetIPBlacklistByCIDR(cidr)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	item := &IPBlacklist{
		IP:           raw,
		CIDR:         cidr,
		IPVersion:    version,
		Reason:       truncateIPBlacklistReason(reason),
		SourceUserId: sourceUserID,
		CreatedBy:    createdBy,
	}
	if err := DB.Create(item).Error; err != nil {
		if existing, findErr := GetIPBlacklistByCIDR(cidr); findErr == nil {
			return existing, false, nil
		}
		return nil, false, err
	}
	InvalidateIPBlacklistCache()
	return item, true, nil
}

func DeleteIPBlacklistByID(id int) error {
	if id <= 0 {
		return ErrIPBlacklistNotFound
	}
	result := DB.Delete(&IPBlacklist{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrIPBlacklistNotFound
	}
	InvalidateIPBlacklistCache()
	return nil
}

func SearchIPBlacklists(keyword string, pageInfo *common.PageInfo) ([]*IPBlacklist, int64, error) {
	if pageInfo == nil {
		pageInfo = &common.PageInfo{Page: 1, PageSize: common.ItemsPerPage}
	}
	query := DB.Model(&IPBlacklist{})
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("ip LIKE ? OR cidr LIKE ? OR reason LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*IPBlacklist
	if err := query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func BatchCreateIPBlacklistFromUsers(userIDs []int, reason string, createdBy int) (*IPBlacklistBatchResult, error) {
	result := &IPBlacklistBatchResult{
		Items:  make([]*IPBlacklist, 0),
		Failed: make([]IPBlacklistBatchFailure, 0),
	}
	if len(userIDs) == 0 {
		return result, nil
	}

	seenIDs := make(map[int]struct{}, len(userIDs))
	ids := make([]int, 0, len(userIDs))
	for _, id := range userIDs {
		if id <= 0 {
			result.Failed = append(result.Failed, IPBlacklistBatchFailure{
				UserID:  id,
				Message: "用户 ID 无效",
			})
			continue
		}
		if _, ok := seenIDs[id]; ok {
			continue
		}
		seenIDs[id] = struct{}{}
		ids = append(ids, id)
	}

	var users []User
	if len(ids) > 0 {
		if err := DB.Unscoped().Select("id", "register_ip").Where("id IN ?", ids).Find(&users).Error; err != nil {
			return nil, err
		}
	}

	userMap := make(map[int]User, len(users))
	for _, user := range users {
		userMap[user.Id] = user
	}

	seenCIDR := make(map[string]struct{}, len(users))
	for _, id := range ids {
		user, ok := userMap[id]
		if !ok {
			result.Failed = append(result.Failed, IPBlacklistBatchFailure{
				UserID:  id,
				Message: "用户不存在",
			})
			continue
		}
		ip := strings.TrimSpace(user.RegisterIP)
		if ip == "" {
			result.SkippedCount++
			continue
		}
		cidr, _, err := NormalizeIPBlacklistRule(ip)
		if err != nil {
			result.Failed = append(result.Failed, IPBlacklistBatchFailure{
				UserID:  id,
				IP:      ip,
				Message: err.Error(),
			})
			continue
		}
		if _, ok := seenCIDR[cidr]; ok {
			result.SkippedCount++
			continue
		}
		seenCIDR[cidr] = struct{}{}

		item, created, err := CreateIPBlacklist(ip, reason, id, createdBy)
		if err != nil {
			result.Failed = append(result.Failed, IPBlacklistBatchFailure{
				UserID:  id,
				IP:      ip,
				Message: err.Error(),
			})
			continue
		}
		if created {
			result.CreatedCount++
		} else {
			result.ExistingCount++
		}
		result.Items = append(result.Items, item)
	}
	result.FailedCount = len(result.Failed)
	return result, nil
}

func FindUserRegisterIPMatch(userIDs []int, clientIP string) (string, error) {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return "", nil
	}
	parsedClientIP := net.ParseIP(clientIP)
	if parsedClientIP == nil {
		return "", ErrInvalidIPBlacklistRule
	}

	seenIDs := make(map[int]struct{}, len(userIDs))
	ids := make([]int, 0, len(userIDs))
	for _, id := range userIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seenIDs[id]; ok {
			continue
		}
		seenIDs[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "", nil
	}

	var users []User
	if err := DB.Unscoped().Select("id", "register_ip").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return "", err
	}
	for _, user := range users {
		ip := strings.TrimSpace(user.RegisterIP)
		if ip == "" {
			continue
		}
		match, err := DoesIPBlacklistRuleMatchIP(ip, parsedClientIP.String())
		if err != nil {
			continue
		}
		if match {
			return ip, nil
		}
	}
	return "", nil
}

func FindIPBlacklistMatch(ipInput string) (*IPBlacklist, error) {
	ip := net.ParseIP(strings.TrimSpace(ipInput))
	if ip == nil {
		return nil, ErrInvalidIPBlacklistRule
	}
	rules, err := getIPBlacklistRules()
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if rule.Network.Contains(ip) {
			return &IPBlacklist{Id: rule.Id, CIDR: rule.CIDR}, nil
		}
	}
	return nil, nil
}

func getIPBlacklistRules() ([]CompiledIPBlacklistRule, error) {
	now := time.Now()
	ipBlacklistCache.RLock()
	if now.Before(ipBlacklistCache.expiresAt) {
		rules := append([]CompiledIPBlacklistRule(nil), ipBlacklistCache.rules...)
		ipBlacklistCache.RUnlock()
		return rules, nil
	}
	ipBlacklistCache.RUnlock()

	ipBlacklistCache.Lock()
	defer ipBlacklistCache.Unlock()
	if now.Before(ipBlacklistCache.expiresAt) {
		return append([]CompiledIPBlacklistRule(nil), ipBlacklistCache.rules...), nil
	}

	if DB == nil {
		ipBlacklistCache.rules = nil
		ipBlacklistCache.expiresAt = now.Add(ipBlacklistCacheTTL)
		return nil, nil
	}

	var items []IPBlacklist
	if err := DB.Select("id", "cidr").Find(&items).Error; err != nil {
		if len(ipBlacklistCache.rules) > 0 {
			ipBlacklistCache.expiresAt = now.Add(ipBlacklistCacheTTL)
			return append([]CompiledIPBlacklistRule(nil), ipBlacklistCache.rules...), nil
		}
		return nil, err
	}

	rules := make([]CompiledIPBlacklistRule, 0, len(items))
	for _, item := range items {
		_, network, err := net.ParseCIDR(item.CIDR)
		if err != nil {
			common.SysError(fmt.Sprintf("invalid ip blacklist cidr id=%d cidr=%q: %s", item.Id, item.CIDR, err.Error()))
			continue
		}
		rules = append(rules, CompiledIPBlacklistRule{
			Id:      item.Id,
			CIDR:    item.CIDR,
			Network: network,
		})
	}
	ipBlacklistCache.rules = rules
	ipBlacklistCache.expiresAt = now.Add(ipBlacklistCacheTTL)
	return append([]CompiledIPBlacklistRule(nil), rules...), nil
}

func truncateIPBlacklistReason(reason string) string {
	reason = strings.TrimSpace(reason)
	runes := []rune(reason)
	if len(runes) <= ipBlacklistReasonMaxLength {
		return reason
	}
	return string(runes[:ipBlacklistReasonMaxLength])
}
