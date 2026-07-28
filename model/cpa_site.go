package model

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

type CPASite struct {
	Id                     int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name                   string `json:"name" gorm:"type:varchar(128);not null;default:''"`
	Host                   string `json:"host" gorm:"type:varchar(255);not null;default:'';uniqueIndex:idx_cpa_site_host_scheme,priority:1"`
	Scheme                 string `json:"scheme" gorm:"type:varchar(16);not null;default:'https';uniqueIndex:idx_cpa_site_host_scheme,priority:2"`
	ManagementKeyEncrypted string `json:"-" gorm:"type:text;not null"`
	Status                 int    `json:"status" gorm:"not null;default:0;index:idx_cpa_site_status"`
	LastSyncedAt           int64  `json:"last_synced_at" gorm:"bigint;not null;default:0"`
	LastSyncError          string `json:"last_sync_error" gorm:"type:text;not null"`
	CachedAccounts         string `json:"-" gorm:"type:text;not null"`
	SortOrder              int    `json:"sort_order" gorm:"not null;default:0"`
	CreatedTime            int64  `json:"created_time" gorm:"bigint;not null;default:0"`
	UpdatedTime            int64  `json:"updated_time" gorm:"bigint;not null;default:0"`
}

const (
	CPASiteStatusPending = 0
	CPASiteStatusSynced  = 1
	CPASiteStatusError   = 2
)

var (
	ErrCPASiteNotFound      = errors.New("cpa_site:not_found")
	ErrCPASiteHostRequired  = errors.New("cpa_site:host_required")
	ErrCPASiteHostInvalid   = errors.New("cpa_site:host_invalid")
	ErrCPASiteNameTooLong   = errors.New("cpa_site:name_too_long")
	ErrCPASiteKeyRequired   = errors.New("cpa_site:management_key_required")
	ErrCPASiteDuplicateHost = errors.New("cpa_site:duplicate_host")
)

func (s *CPASite) TableName() string {
	return "cpa_sites"
}

func (s *CPASite) BeforeCreate(tx *gorm.DB) error {
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

func (s *CPASite) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedTime = common.GetTimestamp()
	return nil
}

func normalizeCPAHost(raw string) string {
	host := strings.TrimSpace(raw)
	lower := strings.ToLower(host)
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(lower, prefix) {
			host = host[len(prefix):]
			break
		}
	}
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(host), "/"))
}

func validateCPAHost(host string) error {
	if host == "" {
		return ErrCPASiteHostRequired
	}
	if len(host) > 255 {
		return ErrCPASiteHostInvalid
	}
	if strings.ContainsAny(host, " \t\r\n/?#@") {
		return ErrCPASiteHostInvalid
	}
	parsed, err := url.Parse("http://" + host)
	if err != nil || parsed.Host != host || parsed.Hostname() == "" || parsed.User != nil {
		return ErrCPASiteHostInvalid
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return ErrCPASiteHostInvalid
		}
	}
	if ip := net.ParseIP(parsed.Hostname()); ip == nil {
		for _, label := range strings.Split(parsed.Hostname(), ".") {
			if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
				return ErrCPASiteHostInvalid
			}
		}
	}
	return nil
}

func (s *CPASite) Normalize() {
	s.Name = strings.TrimSpace(s.Name)
	s.Host = normalizeCPAHost(s.Host)
	s.Scheme = strings.ToLower(strings.TrimSpace(s.Scheme))
	if s.Scheme != "http" && s.Scheme != "https" {
		s.Scheme = "https"
	}
}

func (s *CPASite) Validate(requireKey bool) error {
	s.Normalize()
	if utf8.RuneCountInString(s.Name) > 128 {
		return ErrCPASiteNameTooLong
	}
	if err := validateCPAHost(s.Host); err != nil {
		return err
	}
	if requireKey && strings.TrimSpace(s.ManagementKeyEncrypted) == "" {
		return ErrCPASiteKeyRequired
	}
	return nil
}

func (s *CPASite) BaseURL() string {
	scheme := strings.ToLower(strings.TrimSpace(s.Scheme))
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, strings.TrimRight(strings.TrimSpace(s.Host), "/"))
}

func cpaSiteSecretKey() []byte {
	sum := sha256.Sum256([]byte("cpa_site:" + common.CryptoSecret))
	return sum[:]
}

func EncryptCPASecret(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	block, err := aes.NewCipher(cpaSiteSecretKey())
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
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(raw), nil)), nil
}

func DecryptCPASecret(cipherText string) (string, error) {
	cipherText = strings.TrimSpace(cipherText)
	if cipherText == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(cpaSiteSecretKey())
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
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *CPASite) SetManagementKeyPlain(plain string) error {
	encrypted, err := EncryptCPASecret(plain)
	if err != nil {
		return err
	}
	s.ManagementKeyEncrypted = encrypted
	return nil
}

func (s *CPASite) DecryptManagementKey() (string, error) {
	return DecryptCPASecret(s.ManagementKeyEncrypted)
}

func MaskCPASecret(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) <= 6 {
		if raw == "" {
			return ""
		}
		return "****"
	}
	return fmt.Sprintf("%s****%s", raw[:2], raw[len(raw)-2:])
}

func ListCPASites() ([]*CPASite, error) {
	sites := make([]*CPASite, 0)
	if err := DB.Order("sort_order asc, id asc").Find(&sites).Error; err != nil {
		return nil, err
	}
	return sites, nil
}

func GetCPASiteByID(id int) (*CPASite, error) {
	if id <= 0 {
		return nil, ErrCPASiteNotFound
	}
	site := &CPASite{}
	if err := DB.First(site, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCPASiteNotFound
		}
		return nil, err
	}
	return site, nil
}

func CreateCPASite(site *CPASite) error {
	if site == nil {
		return errors.New("cpa_site:nil")
	}
	if err := site.Validate(true); err != nil {
		return err
	}
	var count int64
	if err := DB.Model(&CPASite{}).Where("host = ? AND scheme = ?", site.Host, site.Scheme).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrCPASiteDuplicateHost
	}
	return DB.Create(site).Error
}

func UpdateCPASite(site *CPASite, updateKey bool) error {
	if site == nil || site.Id <= 0 {
		return ErrCPASiteNotFound
	}
	if err := site.Validate(false); err != nil {
		return err
	}
	var count int64
	if err := DB.Model(&CPASite{}).Where("host = ? AND scheme = ? AND id <> ?", site.Host, site.Scheme, site.Id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrCPASiteDuplicateHost
	}
	updates := map[string]any{
		"name":         site.Name,
		"host":         site.Host,
		"scheme":       site.Scheme,
		"sort_order":   site.SortOrder,
		"updated_time": common.GetTimestamp(),
	}
	if updateKey {
		updates["management_key_encrypted"] = site.ManagementKeyEncrypted
	}
	return DB.Model(&CPASite{}).Where("id = ?", site.Id).Updates(updates).Error
}

func PersistCPASiteSyncContext(ctx context.Context, id int, accountsJSON string, status int, syncErr string) error {
	if id <= 0 {
		return ErrCPASiteNotFound
	}
	updates := map[string]any{
		"status":          status,
		"last_synced_at":  common.GetTimestamp(),
		"last_sync_error": strings.TrimSpace(syncErr),
		"updated_time":    common.GetTimestamp(),
	}
	if accountsJSON != "" {
		updates["cached_accounts"] = accountsJSON
	}
	return DB.WithContext(ctx).Model(&CPASite{}).Where("id = ?", id).Updates(updates).Error
}

func DeleteCPASite(id int) error {
	if id <= 0 {
		return ErrCPASiteNotFound
	}
	result := DB.Delete(&CPASite{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCPASiteNotFound
	}
	return nil
}
