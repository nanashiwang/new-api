package model

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// CRSSite 表示一个 Claude Relay Service (CRS) 上游站点配置。
// 管理员可配置多个 CRS 站点，系统会使用 username/password 登录获取 token，
// 随后周期性拉取 /admin/dashboard 以展示账号概览。
type CRSSite struct {
	Id                int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name              string `json:"name" gorm:"type:varchar(128);not null;default:''"`
	Host              string `json:"host" gorm:"type:varchar(255);not null;default:'';uniqueIndex:idx_crs_site_host_scheme,priority:1"`
	Scheme            string `json:"scheme" gorm:"type:varchar(16);not null;default:'https';uniqueIndex:idx_crs_site_host_scheme,priority:2"`
	Group             string `json:"group" gorm:"column:group_name;type:varchar(128);not null;default:'';index:idx_crs_site_group"`
	Username          string `json:"-" gorm:"type:varchar(255);not null;default:''"`
	PasswordEncrypted string `json:"-" gorm:"type:text;not null;default:''"`
	TokenEncrypted    string `json:"-" gorm:"type:text;not null;default:''"`
	TokenExpiresAt    int64  `json:"token_expires_at" gorm:"bigint;not null;default:0"`
	Status            int    `json:"status" gorm:"not null;default:0;index:idx_crs_site_status"`
	LastSyncedAt      int64  `json:"last_synced_at" gorm:"bigint;not null;default:0"`
	LastSyncError     string `json:"last_sync_error" gorm:"type:text;not null;default:''"`
	CachedStats       string `json:"-" gorm:"type:text;not null;default:''"`
	SortOrder         int    `json:"sort_order" gorm:"not null;default:0"`
	CreatedTime       int64  `json:"created_time" gorm:"bigint;not null;default:0"`
	UpdatedTime       int64  `json:"updated_time" gorm:"bigint;not null;default:0"`
}

const (
	CRSSiteStatusPending = 0 // 未同步
	CRSSiteStatusSynced  = 1 // 已同步
	CRSSiteStatusError   = 2 // 同步失败
)

var (
	ErrCRSSiteNotFound       = errors.New("crs_site:not_found")
	ErrCRSSiteHostRequired   = errors.New("crs_site:host_required")
	ErrCRSSiteUserRequired   = errors.New("crs_site:username_required")
	ErrCRSSitePassRequired   = errors.New("crs_site:password_required")
	ErrCRSSiteTokenEmpty     = errors.New("crs_site:token_empty")
	ErrCRSSiteHostInvalid    = errors.New("crs_site:host_invalid")
	ErrCRSSiteDuplicateHost  = errors.New("crs_site:duplicate_host")
	ErrCRSSiteRequestFailure = errors.New("crs_site:request_failed")
)

func (s *CRSSite) TableName() string {
	return "crs_sites"
}

func (s *CRSSite) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if s.CreatedTime <= 0 {
		s.CreatedTime = now
	}
	if s.UpdatedTime <= 0 {
		s.UpdatedTime = now
	}
	if strings.TrimSpace(s.Scheme) == "" {
		s.Scheme = "https"
	}
	return nil
}

func (s *CRSSite) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedTime = common.GetTimestamp()
	if strings.TrimSpace(s.Scheme) == "" {
		s.Scheme = "https"
	}
	return nil
}

// Normalize 标准化站点字段，去除空白并补齐默认值。
func (s *CRSSite) Normalize() {
	s.Host = normalizeCRSHost(s.Host)
	s.Scheme = strings.ToLower(strings.TrimSpace(s.Scheme))
	if s.Scheme != "http" && s.Scheme != "https" {
		s.Scheme = "https"
	}
	s.Name = strings.TrimSpace(s.Name)
	s.Group = strings.TrimSpace(s.Group)
	s.Username = strings.TrimSpace(s.Username)
}

// BaseURL 组装出站点对外的 URL 前缀（不含末尾斜杠）。
func (s *CRSSite) BaseURL() string {
	scheme := strings.ToLower(strings.TrimSpace(s.Scheme))
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}
	host := strings.TrimRight(strings.TrimSpace(s.Host), "/")
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func normalizeCRSHost(raw string) string {
	host := strings.TrimSpace(raw)
	if host == "" {
		return ""
	}
	// 允许用户粘贴带 scheme 的 URL，统一剔除。
	lower := strings.ToLower(host)
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(lower, prefix) {
			host = host[len(prefix):]
			break
		}
	}
	host = strings.TrimRight(host, "/")
	return host
}

// Validate 校验字段是否合法。
func (s *CRSSite) Validate(requirePassword bool) error {
	s.Normalize()
	if s.Host == "" {
		return ErrCRSSiteHostRequired
	}
	if strings.ContainsAny(s.Host, " \t\r\n") {
		return ErrCRSSiteHostInvalid
	}
	if s.Username == "" {
		return ErrCRSSiteUserRequired
	}
	if requirePassword && strings.TrimSpace(s.PasswordEncrypted) == "" {
		return ErrCRSSitePassRequired
	}
	return nil
}

func crsSiteSecretKey() []byte {
	sum := sha256.Sum256([]byte("crs_site:" + common.CryptoSecret))
	return sum[:]
}

// EncryptCRSSecret 将明文 token/password 加密为 base64 字符串。
func EncryptCRSSecret(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	block, err := aes.NewCipher(crsSiteSecretKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nonce, nonce, []byte(raw), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// DecryptCRSSecret 解密 base64 字符串为明文。空串返回空串。
func DecryptCRSSecret(cipherText string) (string, error) {
	cipherText = strings.TrimSpace(cipherText)
	if cipherText == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(crsSiteSecretKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("cipher text too short")
	}
	nonce := raw[:gcm.NonceSize()]
	data := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// MaskCRSSecret 生成脱敏后的字符串，用于前端展示。
func MaskCRSSecret(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= 6 {
		return "****"
	}
	return fmt.Sprintf("%s****%s", raw[:2], raw[len(raw)-2:])
}

// DecryptPassword 返回明文密码。
func (s *CRSSite) DecryptPassword() (string, error) {
	return DecryptCRSSecret(s.PasswordEncrypted)
}

// DecryptToken 返回明文 session token。
func (s *CRSSite) DecryptToken() (string, error) {
	return DecryptCRSSecret(s.TokenEncrypted)
}

// SetPasswordPlain 设置新密码并加密。
func (s *CRSSite) SetPasswordPlain(plain string) error {
	encrypted, err := EncryptCRSSecret(plain)
	if err != nil {
		return err
	}
	s.PasswordEncrypted = encrypted
	return nil
}

// SetTokenPlain 设置 session token 并加密。
func (s *CRSSite) SetTokenPlain(plain string, expiresAt int64) error {
	encrypted, err := EncryptCRSSecret(plain)
	if err != nil {
		return err
	}
	s.TokenEncrypted = encrypted
	s.TokenExpiresAt = expiresAt
	return nil
}

// ListCRSSites 返回所有 CRS 站点，按 SortOrder, Id 排序。
func ListCRSSites() ([]*CRSSite, error) {
	sites := make([]*CRSSite, 0)
	if err := DB.Order("sort_order asc, id asc").Find(&sites).Error; err != nil {
		return nil, err
	}
	return sites, nil
}

// GetCRSSiteByID 按主键查询。
func GetCRSSiteByID(id int) (*CRSSite, error) {
	if id <= 0 {
		return nil, ErrCRSSiteNotFound
	}
	site := &CRSSite{}
	if err := DB.First(site, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCRSSiteNotFound
		}
		return nil, err
	}
	return site, nil
}

// CreateCRSSite 插入新站点，会校验 Host 唯一性。
func CreateCRSSite(site *CRSSite) error {
	if site == nil {
		return errors.New("crs_site:nil")
	}
	site.Normalize()
	if err := site.Validate(true); err != nil {
		return err
	}
	var count int64
	if err := DB.Model(&CRSSite{}).
		Where("host = ? AND scheme = ?", site.Host, site.Scheme).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrCRSSiteDuplicateHost
	}
	return DB.Create(site).Error
}

// UpdateCRSSite 更新可编辑字段。
func UpdateCRSSite(site *CRSSite, updatePassword bool) error {
	if site == nil || site.Id <= 0 {
		return ErrCRSSiteNotFound
	}
	site.Normalize()
	if err := site.Validate(false); err != nil {
		return err
	}
	// host/scheme 冲突检测（排除自己）。
	var count int64
	if err := DB.Model(&CRSSite{}).
		Where("host = ? AND scheme = ? AND id <> ?", site.Host, site.Scheme, site.Id).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrCRSSiteDuplicateHost
	}
	updates := map[string]any{
		"name":         site.Name,
		"host":         site.Host,
		"scheme":       site.Scheme,
		"group_name":   site.Group,
		"username":     site.Username,
		"sort_order":   site.SortOrder,
		"updated_time": common.GetTimestamp(),
	}
	if updatePassword {
		updates["password_encrypted"] = site.PasswordEncrypted
		// 密码变更后清空旧 token，强制重新登录。
		updates["token_encrypted"] = ""
		updates["token_expires_at"] = 0
	}
	return DB.Model(&CRSSite{}).Where("id = ?", site.Id).Updates(updates).Error
}

// PersistCRSSiteStats 将最新一次 dashboard 结果、token、状态写入。
func PersistCRSSiteStats(id int, tokenEncrypted string, tokenExpiresAt int64, stats string, status int, syncErr string) error {
	if id <= 0 {
		return ErrCRSSiteNotFound
	}
	now := common.GetTimestamp()
	updates := map[string]any{
		"status":          status,
		"cached_stats":    stats,
		"last_synced_at":  now,
		"last_sync_error": syncErr,
		"updated_time":    now,
	}
	if tokenEncrypted != "" {
		updates["token_encrypted"] = tokenEncrypted
		updates["token_expires_at"] = tokenExpiresAt
	}
	return DB.Model(&CRSSite{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteCRSSite 删除指定站点。
func DeleteCRSSite(id int) error {
	if id <= 0 {
		return ErrCRSSiteNotFound
	}
	return DB.Delete(&CRSSite{}, id).Error
}

// SiteBriefToken 返回脱敏后的展示值（前端使用）。
func (s *CRSSite) SiteBriefToken() string {
	if strings.TrimSpace(s.TokenEncrypted) == "" {
		return ""
	}
	plain, err := s.DecryptToken()
	if err != nil || plain == "" {
		return "****"
	}
	return MaskCRSSecret(plain)
}
