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
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"gorm.io/gorm"
)

// Sentinel errors for i18n translation (remote observer)
var (
	ErrProfitBoardRemoteMissingURL   = errors.New("profit_board:remote_missing_url")
	ErrProfitBoardRemoteMissingUID   = errors.New("profit_board:remote_missing_uid")
	ErrProfitBoardRemoteMissingToken = errors.New("profit_board:remote_missing_token")
	ErrProfitBoardRemoteTokenEmpty   = errors.New("profit_board:remote_token_empty")
	ErrProfitBoardRemoteNotNewAPI    = errors.New("profit_board:remote_not_newapi")
	ErrProfitBoardRemoteURLInvalid   = errors.New("profit_board:remote_url_invalid")
	ErrProfitBoardRemoteRequestFail  = errors.New("profit_board:remote_request_fail")
)

const (
	profitBoardRemoteObserverStatusDisabled      = "disabled"
	profitBoardRemoteObserverStatusNotConfigured = "not_configured"
	profitBoardRemoteObserverStatusReady         = "ready"
	profitBoardRemoteObserverStatusNeedsBaseline = "needs_baseline"
	profitBoardRemoteObserverStatusFailed        = "failed"

	profitBoardRemoteSnapshotStatusSuccess = "success"
	profitBoardRemoteSnapshotStatusFailed  = "failed"

	profitBoardRemoteSyncMinIntervalSeconds = 300
)

type ProfitBoardRemoteSnapshot struct {
	Id                 int     `json:"id"`
	SelectionSignature string  `json:"selection_signature" gorm:"type:varchar(255);index:idx_profit_board_remote_snapshot_combo_time,priority:1;index:idx_profit_board_remote_snapshot_combo_hash_time,priority:1"`
	ComboId            string  `json:"combo_id" gorm:"type:varchar(64);index:idx_profit_board_remote_snapshot_combo_time,priority:2;index:idx_profit_board_remote_snapshot_combo_hash_time,priority:2"`
	ConfigHash         string  `json:"config_hash" gorm:"type:varchar(64);index:idx_profit_board_remote_snapshot_combo_hash_time,priority:3"`
	Status             string  `json:"status" gorm:"type:varchar(16);index"`
	ErrorMessage       string  `json:"error_message,omitempty" gorm:"type:text"`
	RemoteQuotaPerUnit float64 `json:"remote_quota_per_unit" gorm:"type:decimal(18,6);default:0"`
	WalletQuota        int64   `json:"wallet_quota" gorm:"type:bigint;default:0"`
	WalletUsedQuota    int64   `json:"wallet_used_quota" gorm:"type:bigint;default:0"`
	SubscriptionStates string  `json:"subscription_states,omitempty" gorm:"type:text"`
	SyncedAt           int64   `json:"synced_at" gorm:"bigint;index:idx_profit_board_remote_snapshot_combo_time,priority:3"`
	CreatedAt          int64   `json:"created_at" gorm:"bigint;index"`
}

func (s *ProfitBoardRemoteSnapshot) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if s.CreatedAt <= 0 {
		s.CreatedAt = now
	}
	if s.SyncedAt <= 0 {
		s.SyncedAt = now
	}
	return nil
}

type profitBoardRemoteStatusData struct {
	QN           string  `json:"_qn"`
	QuotaPerUnit float64 `json:"quota_per_unit"`
}

type profitBoardRemoteUserSelfData struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Quota       int64  `json:"quota"`
	UsedQuota   int64  `json:"used_quota"`
}

type profitBoardRemoteSubscriptionItem struct {
	Subscription ProfitBoardRemoteSubscriptionSnapshot `json:"subscription"`
}

type profitBoardRemoteSubscriptionSelfData struct {
	Subscriptions []profitBoardRemoteSubscriptionItem `json:"subscriptions"`
}

type profitBoardRemoteEnvelope[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type profitBoardRemoteFetchResult struct {
	RemoteQuotaPerUnit float64
	WalletQuota        int64
	WalletUsedQuota    int64
	Subscriptions      []ProfitBoardRemoteSubscriptionSnapshot
}

type profitBoardRemoteAggregate struct {
	TotalCostUSD float64
	BatchCostUSD map[string]float64
	Timeseries   []ProfitBoardTimeseriesPoint
	States       []ProfitBoardRemoteObserverState
	Warnings     []string
}

func normalizeProfitBoardRemoteObserverConfig(config ProfitBoardRemoteObserverConfig) ProfitBoardRemoteObserverConfig {
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.AccessToken = strings.TrimSpace(config.AccessToken)
	config.AccessTokenMasked = strings.TrimSpace(config.AccessTokenMasked)
	config.AccessTokenEncrypted = strings.TrimSpace(config.AccessTokenEncrypted)
	if config.UserID < 0 {
		config.UserID = 0
	}
	return config
}

func profitBoardHasEnabledRemoteObserver(comboConfigs []ProfitBoardComboPricingConfig) bool {
	for _, comboConfig := range comboConfigs {
		if normalizeProfitBoardRemoteObserverConfig(comboConfig.RemoteObserver).Enabled {
			return true
		}
	}
	return false
}

func profitBoardRemoteObserverConfigured(config ProfitBoardRemoteObserverConfig) bool {
	return strings.TrimSpace(config.BaseURL) != "" && config.UserID > 0 &&
		(strings.TrimSpace(config.AccessToken) != "" || strings.TrimSpace(config.AccessTokenEncrypted) != "")
}

func validateProfitBoardRemoteObserverConfig(config ProfitBoardRemoteObserverConfig) error {
	config = normalizeProfitBoardRemoteObserverConfig(config)
	if !config.Enabled {
		return nil
	}
	if config.BaseURL == "" {
		return ErrProfitBoardRemoteMissingURL
	}
	if config.UserID <= 0 {
		return ErrProfitBoardRemoteMissingUID
	}
	if strings.TrimSpace(config.AccessToken) == "" && strings.TrimSpace(config.AccessTokenEncrypted) == "" {
		return ErrProfitBoardRemoteMissingToken
	}
	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(
		config.BaseURL,
		fetchSetting.EnableSSRFProtection,
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		fetchSetting.AllowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
	); err != nil {
		return fmt.Errorf("%w: %v", ErrProfitBoardRemoteURLInvalid, err)
	}
	return nil
}

func profitBoardRemoteSecretKey() []byte {
	sum := sha256.Sum256([]byte(common.CryptoSecret))
	return sum[:]
}

func encryptProfitBoardRemoteSecret(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	block, err := aes.NewCipher(profitBoardRemoteSecretKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nonce, nonce, []byte(raw), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptProfitBoardRemoteSecret(cipherText string) (string, error) {
	cipherText = strings.TrimSpace(cipherText)
	if cipherText == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(profitBoardRemoteSecretKey())
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

func maskProfitBoardRemoteSecret(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= 8 {
		return "****"
	}
	return fmt.Sprintf("%s****%s", raw[:4], raw[len(raw)-4:])
}

func stripProfitBoardRemoteObserverSecret(config ProfitBoardRemoteObserverConfig) ProfitBoardRemoteObserverConfig {
	config = normalizeProfitBoardRemoteObserverConfig(config)
	if config.AccessTokenMasked == "" && config.AccessTokenEncrypted != "" {
		if decrypted, err := decryptProfitBoardRemoteSecret(config.AccessTokenEncrypted); err == nil {
			config.AccessTokenMasked = maskProfitBoardRemoteSecret(decrypted)
		}
	}
	config.AccessToken = ""
	config.AccessTokenEncrypted = ""
	return config
}

func stripProfitBoardRemoteObserverSecrets(comboConfigs []ProfitBoardComboPricingConfig) []ProfitBoardComboPricingConfig {
	stripped := make([]ProfitBoardComboPricingConfig, 0, len(comboConfigs))
	for _, config := range comboConfigs {
		current := config
		current.RemoteObserver = stripProfitBoardRemoteObserverSecret(current.RemoteObserver)
		stripped = append(stripped, current)
	}
	return stripped
}

func profitBoardPersistedComboConfigMap(signature string) (map[string]ProfitBoardComboPricingConfig, error) {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return map[string]ProfitBoardComboPricingConfig{}, nil
	}
	record := &ProfitBoardConfig{}
	if err := DB.Where("selection_signature = ?", signature).First(record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]ProfitBoardComboPricingConfig{}, nil
		}
		return nil, err
	}
	persistedSite := profitBoardPersistedSiteConfig{}
	if err := common.UnmarshalJsonStr(record.SiteConfig, &persistedSite); err != nil {
		return nil, err
	}
	configMap := make(map[string]ProfitBoardComboPricingConfig, len(persistedSite.ComboConfigs))
	for _, config := range persistedSite.ComboConfigs {
		comboID := strings.TrimSpace(config.ComboId)
		if comboID == "" {
			continue
		}
		configMap[comboID] = config
	}
	return configMap, nil
}

func profitBoardHasPersistedSiteConfig(persistedSite profitBoardPersistedSiteConfig) bool {
	return len(persistedSite.ComboConfigs) > 0 ||
		len(persistedSite.SharedSite.ModelNames) > 0 ||
		persistedSite.LegacySite.PricingMode != "" ||
		persistedSite.LegacySite.Group != "" ||
		len(persistedSite.LegacySite.ModelNames) > 0
}

func payloadFromProfitBoardConfigRecord(record ProfitBoardConfig) (*ProfitBoardConfigPayload, error) {
	defaultUpstream := normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
		CostSource: ProfitBoardCostSourceManualOnly,
	}, false)
	defaultSite := normalizeProfitBoardPricingConfig(ProfitBoardTokenPricingConfig{
		PricingMode: ProfitBoardSitePricingManual,
	}, true)

	payload := &ProfitBoardConfigPayload{
		Batches:  parseProfitBoardConfigBatches(record.SelectionValues),
		Upstream: defaultUpstream,
		Site:     defaultSite,
	}
	if len(payload.Batches) == 0 {
		return payload, nil
	}

	if err := common.UnmarshalJsonStr(record.UpstreamConfig, &payload.Upstream); err != nil {
		return nil, err
	}
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)

	persistedSite := profitBoardPersistedSiteConfig{}
	if err := common.UnmarshalJsonStr(record.SiteConfig, &persistedSite); err == nil && profitBoardHasPersistedSiteConfig(persistedSite) {
		payload.Site = normalizeProfitBoardPricingConfig(persistedSite.LegacySite, true)
		payload.SharedSite = normalizeProfitBoardSharedSiteConfig(persistedSite.SharedSite, payload.Site)
		payload.ComboConfigs = normalizeProfitBoardComboConfigs(payload.Batches, persistedSite.ComboConfigs, payload.SharedSite, payload.Site, payload.Upstream)
		return payload, nil
	}

	if err := common.UnmarshalJsonStr(record.SiteConfig, &payload.Site); err != nil {
		return nil, err
	}
	payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
	payload.SharedSite = normalizeProfitBoardSharedSiteConfig(ProfitBoardSharedSitePricingConfig{}, payload.Site)
	payload.ComboConfigs = normalizeProfitBoardComboConfigs(payload.Batches, nil, payload.SharedSite, payload.Site, payload.Upstream)
	return payload, nil
}

func prepareProfitBoardRemoteObserverConfigsForStorage(signature string, comboConfigs []ProfitBoardComboPricingConfig) ([]ProfitBoardComboPricingConfig, error) {
	persistedMap, err := profitBoardPersistedComboConfigMap(signature)
	if err != nil {
		return nil, err
	}
	prepared := make([]ProfitBoardComboPricingConfig, 0, len(comboConfigs))
	for _, comboConfig := range comboConfigs {
		current := comboConfig
		current.RemoteObserver = normalizeProfitBoardRemoteObserverConfig(current.RemoteObserver)
		persisted := normalizeProfitBoardRemoteObserverConfig(persistedMap[current.ComboId].RemoteObserver)
		switch {
		case current.RemoteObserver.AccessToken != "":
			encrypted, encErr := encryptProfitBoardRemoteSecret(current.RemoteObserver.AccessToken)
			if encErr != nil {
				return nil, encErr
			}
			current.RemoteObserver.AccessTokenEncrypted = encrypted
		case current.RemoteObserver.AccessTokenEncrypted != "":
		case persisted.AccessTokenEncrypted != "":
			current.RemoteObserver.AccessTokenEncrypted = persisted.AccessTokenEncrypted
		}
		current.RemoteObserver.AccessTokenMasked = ""
		current.RemoteObserver.AccessToken = ""
		if !current.RemoteObserver.Enabled && current.RemoteObserver.BaseURL == "" && current.RemoteObserver.UserID == 0 {
			current.RemoteObserver.AccessTokenEncrypted = ""
		}
		prepared = append(prepared, current)
	}
	return prepared, nil
}

func hydrateProfitBoardRemoteObserverSecrets(signature string, comboConfigs []ProfitBoardComboPricingConfig) []ProfitBoardComboPricingConfig {
	persistedMap, err := profitBoardPersistedComboConfigMap(signature)
	if err != nil {
		return comboConfigs
	}
	hydrated := make([]ProfitBoardComboPricingConfig, 0, len(comboConfigs))
	for _, comboConfig := range comboConfigs {
		current := comboConfig
		current.RemoteObserver = normalizeProfitBoardRemoteObserverConfig(current.RemoteObserver)
		if current.RemoteObserver.AccessToken == "" && current.RemoteObserver.AccessTokenEncrypted == "" {
			current.RemoteObserver.AccessTokenEncrypted = normalizeProfitBoardRemoteObserverConfig(persistedMap[current.ComboId].RemoteObserver).AccessTokenEncrypted
		}
		hydrated = append(hydrated, current)
	}
	return hydrated
}

func profitBoardRemoteObserverConfigHash(config ProfitBoardRemoteObserverConfig) string {
	config = normalizeProfitBoardRemoteObserverConfig(config)
	token := strings.TrimSpace(config.AccessToken)
	if token == "" && config.AccessTokenEncrypted != "" {
		if decrypted, err := decryptProfitBoardRemoteSecret(config.AccessTokenEncrypted); err == nil {
			token = strings.TrimSpace(decrypted)
		}
	}
	if token == "" {
		return ""
	}
	return common.GenerateHMAC(strings.ToLower(config.BaseURL) + "|" + fmt.Sprintf("%d", config.UserID) + "|" + token)
}

func profitBoardQuotaToUSD(quota int64) float64 {
	if quota <= 0 || common.QuotaPerUnit <= 0 {
		return 0
	}
	return roundProfitBoardAmount(float64(quota) / common.QuotaPerUnit)
}

func newProfitBoardRemoteHTTPClient() *http.Client {
	transport := &http.Transport{}
	if common.TLSInsecureSkipVerify {
		transport.TLSClientConfig = common.InsecureTLSConfig
	}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			fetchSetting := system_setting.GetFetchSetting()
			if err := common.ValidateURLWithFetchSetting(
				req.URL.String(),
				fetchSetting.EnableSSRFProtection,
				fetchSetting.AllowPrivateIp,
				fetchSetting.DomainFilterMode,
				fetchSetting.IpFilterMode,
				fetchSetting.DomainList,
				fetchSetting.IpList,
				fetchSetting.AllowedPorts,
				fetchSetting.ApplyIPFilterForDomain,
			); err != nil {
				return err
			}
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}
	if common.RelayTimeout > 0 {
		client.Timeout = time.Duration(common.RelayTimeout) * time.Second
	} else {
		client.Timeout = 30 * time.Second
	}
	return client
}

func profitBoardRemoteRequest[T any](client *http.Client, remoteConfig ProfitBoardRemoteObserverConfig, path string, target *T) error {
	token := strings.TrimSpace(remoteConfig.AccessToken)
	if token == "" && remoteConfig.AccessTokenEncrypted != "" {
		decrypted, err := decryptProfitBoardRemoteSecret(remoteConfig.AccessTokenEncrypted)
		if err != nil {
			return err
		}
		token = strings.TrimSpace(decrypted)
	}
	if token == "" {
		return ErrProfitBoardRemoteTokenEmpty
	}
	url := strings.TrimRight(remoteConfig.BaseURL, "/") + path
	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(
		url,
		fetchSetting.EnableSSRFProtection,
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		fetchSetting.AllowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
	); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = "Bearer " + token
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("New-Api-User", fmt.Sprintf("%d", remoteConfig.UserID))
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w (%d): %s", ErrProfitBoardRemoteRequestFail, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	envelope := profitBoardRemoteEnvelope[T]{}
	if err := common.Unmarshal(body, &envelope); err != nil {
		return err
	}
	if !envelope.Success {
		if envelope.Message == "" {
			envelope.Message = "远端返回失败"
		}
		return errors.New(envelope.Message)
	}
	*target = envelope.Data
	return nil
}

func fetchProfitBoardRemoteObserver(remoteConfig ProfitBoardRemoteObserverConfig) (*profitBoardRemoteFetchResult, error) {
	if err := validateProfitBoardRemoteObserverConfig(remoteConfig); err != nil {
		return nil, err
	}
	client := newProfitBoardRemoteHTTPClient()

	statusData := profitBoardRemoteStatusData{}
	if err := profitBoardRemoteRequest(client, remoteConfig, "/api/status", &statusData); err != nil {
		return nil, err
	}
	if strings.TrimSpace(statusData.QN) != "new-api" {
		return nil, ErrProfitBoardRemoteNotNewAPI
	}

	selfData := profitBoardRemoteUserSelfData{}
	if err := profitBoardRemoteRequest(client, remoteConfig, "/api/user/self", &selfData); err != nil {
		return nil, err
	}

	subData := profitBoardRemoteSubscriptionSelfData{}
	if err := profitBoardRemoteRequest(client, remoteConfig, "/api/subscription/self", &subData); err != nil {
		return nil, err
	}

	subscriptions := make([]ProfitBoardRemoteSubscriptionSnapshot, 0, len(subData.Subscriptions))
	for _, item := range subData.Subscriptions {
		if item.Subscription.SubscriptionID <= 0 {
			continue
		}
		subscriptions = append(subscriptions, item.Subscription)
	}
	sort.Slice(subscriptions, func(i, j int) bool {
		if subscriptions[i].EndTime == subscriptions[j].EndTime {
			return subscriptions[i].SubscriptionID < subscriptions[j].SubscriptionID
		}
		return subscriptions[i].EndTime < subscriptions[j].EndTime
	})

	return &profitBoardRemoteFetchResult{
		RemoteQuotaPerUnit: statusData.QuotaPerUnit,
		WalletQuota:        selfData.Quota,
		WalletUsedQuota:    selfData.UsedQuota,
		Subscriptions:      subscriptions,
	}, nil
}

func buildProfitBoardRemoteSnapshot(selectionSignature string, comboID string, config ProfitBoardRemoteObserverConfig, fetchResult *profitBoardRemoteFetchResult, status string, err error) ProfitBoardRemoteSnapshot {
	snapshot := ProfitBoardRemoteSnapshot{
		SelectionSignature: selectionSignature,
		ComboId:            comboID,
		ConfigHash:         profitBoardRemoteObserverConfigHash(config),
		Status:             status,
		SyncedAt:           common.GetTimestamp(),
		CreatedAt:          common.GetTimestamp(),
	}
	if err != nil {
		snapshot.ErrorMessage = err.Error()
		return snapshot
	}
	if fetchResult == nil {
		return snapshot
	}
	snapshot.RemoteQuotaPerUnit = fetchResult.RemoteQuotaPerUnit
	snapshot.WalletQuota = fetchResult.WalletQuota
	snapshot.WalletUsedQuota = fetchResult.WalletUsedQuota
	if payload, marshalErr := common.Marshal(fetchResult.Subscriptions); marshalErr == nil {
		snapshot.SubscriptionStates = string(payload)
	}
	return snapshot
}

func parseProfitBoardRemoteSubscriptions(raw string) []ProfitBoardRemoteSubscriptionSnapshot {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []ProfitBoardRemoteSubscriptionSnapshot{}
	}
	subscriptions := make([]ProfitBoardRemoteSubscriptionSnapshot, 0)
	if err := common.UnmarshalJsonStr(raw, &subscriptions); err != nil {
		return []ProfitBoardRemoteSubscriptionSnapshot{}
	}
	return subscriptions
}

func getLatestProfitBoardRemoteSnapshot(selectionSignature string, comboID string) (*ProfitBoardRemoteSnapshot, error) {
	snapshot := &ProfitBoardRemoteSnapshot{}
	if err := DB.Where("selection_signature = ? AND combo_id = ?", selectionSignature, comboID).
		Order("synced_at desc, id desc").
		First(snapshot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return snapshot, nil
}

func getLatestProfitBoardRemoteSuccessSnapshot(selectionSignature string, comboID string, configHash string) (*ProfitBoardRemoteSnapshot, error) {
	snapshot := &ProfitBoardRemoteSnapshot{}
	query := DB.Where("selection_signature = ? AND combo_id = ? AND status = ?", selectionSignature, comboID, profitBoardRemoteSnapshotStatusSuccess)
	if configHash != "" {
		query = query.Where("config_hash = ?", configHash)
	}
	if err := query.Order("synced_at desc, id desc").First(snapshot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return snapshot, nil
}

func countProfitBoardRemoteSuccessSnapshots(selectionSignature string, comboID string, configHash string) (int64, error) {
	var count int64
	query := DB.Model(&ProfitBoardRemoteSnapshot{}).
		Where("selection_signature = ? AND combo_id = ? AND status = ?", selectionSignature, comboID, profitBoardRemoteSnapshotStatusSuccess)
	if configHash != "" {
		query = query.Where("config_hash = ?", configHash)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func listProfitBoardRemoteSuccessSnapshots(selectionSignature string, comboID string, configHash string, startTimestamp int64, endTimestamp int64) ([]ProfitBoardRemoteSnapshot, error) {
	snapshots := make([]ProfitBoardRemoteSnapshot, 0)
	if configHash == "" {
		return snapshots, nil
	}
	if startTimestamp > 0 {
		previous := ProfitBoardRemoteSnapshot{}
		if err := DB.Where("selection_signature = ? AND combo_id = ? AND config_hash = ? AND status = ? AND synced_at < ?",
			selectionSignature, comboID, configHash, profitBoardRemoteSnapshotStatusSuccess, startTimestamp).
			Order("synced_at desc, id desc").
			Limit(1).
			Find(&previous).Error; err != nil {
			return nil, err
		}
		if previous.Id > 0 {
			snapshots = append(snapshots, previous)
		}
	}
	current := make([]ProfitBoardRemoteSnapshot, 0)
	query := DB.Where("selection_signature = ? AND combo_id = ? AND config_hash = ? AND status = ?",
		selectionSignature, comboID, configHash, profitBoardRemoteSnapshotStatusSuccess)
	if startTimestamp > 0 {
		query = query.Where("synced_at >= ?", startTimestamp)
	}
	if endTimestamp > 0 {
		query = query.Where("synced_at <= ?", endTimestamp)
	}
	if err := query.Order("synced_at asc, id asc").Find(&current).Error; err != nil {
		return nil, err
	}
	snapshots = append(snapshots, current...)
	return snapshots, nil
}

func profitBoardRemoteObserverNeedsSync(latest *ProfitBoardRemoteSnapshot, configHash string, force bool) bool {
	if force || latest == nil {
		return true
	}
	if latest.ConfigHash != configHash {
		return true
	}
	if latest.Status != profitBoardRemoteSnapshotStatusSuccess {
		return true
	}
	return common.GetTimestamp()-latest.SyncedAt >= profitBoardRemoteSyncMinIntervalSeconds
}

func syncProfitBoardRemoteObserverSnapshot(selectionSignature string, batch ProfitBoardBatchInfo, config ProfitBoardRemoteObserverConfig, force bool) (*ProfitBoardRemoteSnapshot, *ProfitBoardRemoteSnapshot, error) {
	config = normalizeProfitBoardRemoteObserverConfig(config)
	latestAny, err := getLatestProfitBoardRemoteSnapshot(selectionSignature, batch.Id)
	if err != nil {
		return nil, nil, err
	}
	configHash := profitBoardRemoteObserverConfigHash(config)
	if !profitBoardRemoteObserverNeedsSync(latestAny, configHash, force) {
		latestSuccess, successErr := getLatestProfitBoardRemoteSuccessSnapshot(selectionSignature, batch.Id, configHash)
		return latestAny, latestSuccess, successErr
	}

	fetchResult, fetchErr := fetchProfitBoardRemoteObserver(config)
	snapshot := buildProfitBoardRemoteSnapshot(selectionSignature, batch.Id, config, fetchResult, profitBoardRemoteSnapshotStatusSuccess, nil)
	if fetchErr != nil {
		snapshot = buildProfitBoardRemoteSnapshot(selectionSignature, batch.Id, config, nil, profitBoardRemoteSnapshotStatusFailed, fetchErr)
	}
	if err := DB.Create(&snapshot).Error; err != nil {
		return nil, nil, err
	}
	latestAny = &snapshot
	latestSuccess, successErr := getLatestProfitBoardRemoteSuccessSnapshot(selectionSignature, batch.Id, configHash)
	if successErr != nil {
		return latestAny, nil, successErr
	}
	return latestAny, latestSuccess, nil
}

func summarizeProfitBoardRemoteSubscriptions(subscriptions []ProfitBoardRemoteSubscriptionSnapshot) (int64, int64) {
	total := int64(0)
	used := int64(0)
	unlimited := false
	for _, item := range subscriptions {
		used += item.AmountUsed
		if item.AmountTotal <= 0 {
			unlimited = true
			continue
		}
		total += item.AmountTotal
	}
	if unlimited {
		return 0, used
	}
	return total, used
}

func buildProfitBoardRemoteObserverState(selectionSignature string, batch ProfitBoardBatchInfo, config ProfitBoardRemoteObserverConfig, latestAny *ProfitBoardRemoteSnapshot, latestSuccess *ProfitBoardRemoteSnapshot, observedCostUSD float64) ProfitBoardRemoteObserverState {
	state := ProfitBoardRemoteObserverState{
		BatchId:         batch.Id,
		BatchName:       batch.Name,
		Enabled:         config.Enabled,
		Configured:      profitBoardRemoteObserverConfigured(config),
		Status:          profitBoardRemoteObserverStatusDisabled,
		PeriodUsedUSD:   roundProfitBoardAmount(observedCostUSD),
		ObservedCostUSD: roundProfitBoardAmount(observedCostUSD),
	}
	switch {
	case !config.Enabled:
		state.Status = profitBoardRemoteObserverStatusDisabled
		return state
	case !state.Configured:
		state.Status = profitBoardRemoteObserverStatusNotConfigured
		return state
	}
	if latestAny != nil {
		state.LastSyncedAt = latestAny.SyncedAt
		if latestAny.Status == profitBoardRemoteSnapshotStatusFailed {
			state.Status = profitBoardRemoteObserverStatusFailed
			state.ErrorMessage = latestAny.ErrorMessage
		}
	}
	if latestSuccess != nil {
		state.LastSuccessAt = latestSuccess.SyncedAt
		state.RemoteQuotaPerUnit = latestSuccess.RemoteQuotaPerUnit
		state.QuotaPerUnitMismatch = latestSuccess.RemoteQuotaPerUnit > 0 && latestSuccess.RemoteQuotaPerUnit != common.QuotaPerUnit
		state.WalletBalanceUSD = profitBoardQuotaToUSD(latestSuccess.WalletQuota)
		state.WalletQuotaUSD = state.WalletBalanceUSD
		state.WalletUsedTotalUSD = profitBoardQuotaToUSD(latestSuccess.WalletUsedQuota)
		state.WalletUsedQuotaUSD = state.WalletUsedTotalUSD
		subTotal, subUsed := summarizeProfitBoardRemoteSubscriptions(parseProfitBoardRemoteSubscriptions(latestSuccess.SubscriptionStates))
		state.SubscriptionTotalQuotaUSD = profitBoardQuotaToUSD(subTotal)
		state.SubscriptionUsedQuotaUSD = profitBoardQuotaToUSD(subUsed)
		if count, err := countProfitBoardRemoteSuccessSnapshots(selectionSignature, batch.Id, latestSuccess.ConfigHash); err == nil && count >= 2 {
			state.BaselineReady = true
		}
		if state.Status != profitBoardRemoteObserverStatusFailed {
			if state.BaselineReady {
				state.Status = profitBoardRemoteObserverStatusReady
			} else {
				state.Status = profitBoardRemoteObserverStatusNeedsBaseline
			}
		}
	}
	if latestAny == nil && latestSuccess == nil {
		state.Status = profitBoardRemoteObserverStatusNeedsBaseline
	}
	return state
}

func profitBoardRemoteSubscriptionDelta(prev ProfitBoardRemoteSubscriptionSnapshot, curr ProfitBoardRemoteSubscriptionSnapshot) (int64, bool) {
	if curr.AmountUsed >= prev.AmountUsed {
		return curr.AmountUsed - prev.AmountUsed, false
	}
	resetDetected := curr.LastResetTime > prev.LastResetTime ||
		(curr.NextResetTime > 0 && curr.NextResetTime != prev.NextResetTime) ||
		(curr.EndTime > 0 && curr.EndTime > prev.EndTime) ||
		(curr.AmountTotal > 0 && curr.AmountTotal != prev.AmountTotal)
	if resetDetected {
		return curr.AmountUsed, false
	}
	return 0, true
}

func profitBoardRemoteSnapshotDelta(prev ProfitBoardRemoteSnapshot, curr ProfitBoardRemoteSnapshot) (int64, []string) {
	warnings := make([]string, 0)
	totalDelta := int64(0)
	if curr.WalletUsedQuota >= prev.WalletUsedQuota {
		totalDelta += curr.WalletUsedQuota - prev.WalletUsedQuota
	} else {
		warnings = append(warnings, "远端钱包已用额度出现回退，当前时间段的钱包观测成本已按 0 处理")
	}

	prevSubs := make(map[int]ProfitBoardRemoteSubscriptionSnapshot)
	for _, item := range parseProfitBoardRemoteSubscriptions(prev.SubscriptionStates) {
		prevSubs[item.SubscriptionID] = item
	}
	for _, currentSub := range parseProfitBoardRemoteSubscriptions(curr.SubscriptionStates) {
		previousSub, ok := prevSubs[currentSub.SubscriptionID]
		if !ok {
			continue
		}
		delta, anomaly := profitBoardRemoteSubscriptionDelta(previousSub, currentSub)
		if anomaly {
			warnings = append(warnings, fmt.Sprintf("远端订阅 #%d 已用额度出现异常回退，当前时间段的订阅观测成本已按 0 处理", currentSub.SubscriptionID))
			continue
		}
		totalDelta += delta
	}
	return totalDelta, warnings
}

func uniqueProfitBoardWarnings(items []string) []string {
	if len(items) == 0 {
		return items
	}
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func collectProfitBoardRemoteObserverAggregate(signature string, batches []ProfitBoardBatchInfo, comboConfigs []ProfitBoardComboPricingConfig, startTimestamp int64, endTimestamp int64, granularity string, customIntervalMinutes int, forceSync bool, includeTimeseries bool) (*profitBoardRemoteAggregate, error) {
	aggregate := &profitBoardRemoteAggregate{
		BatchCostUSD: make(map[string]float64),
		Timeseries:   make([]ProfitBoardTimeseriesPoint, 0),
		States:       make([]ProfitBoardRemoteObserverState, 0, len(batches)),
		Warnings:     make([]string, 0),
	}
	comboMap := make(map[string]ProfitBoardComboPricingConfig, len(comboConfigs))
	for _, config := range comboConfigs {
		comboMap[strings.TrimSpace(config.ComboId)] = config
	}
	bucketMap := make(map[string]*ProfitBoardTimeseriesPoint)
	for _, batch := range batches {
		comboConfig, ok := comboMap[batch.Id]
		if !ok {
			aggregate.States = append(aggregate.States, ProfitBoardRemoteObserverState{
				BatchId:   batch.Id,
				BatchName: batch.Name,
				Status:    profitBoardRemoteObserverStatusDisabled,
			})
			continue
		}
		remoteConfig := normalizeProfitBoardRemoteObserverConfig(comboConfig.RemoteObserver)
		if !remoteConfig.Enabled || !profitBoardRemoteObserverConfigured(remoteConfig) {
			aggregate.States = append(aggregate.States, buildProfitBoardRemoteObserverState(signature, batch, remoteConfig, nil, nil, 0))
			continue
		}
		latestAny, latestSuccess, err := syncProfitBoardRemoteObserverSnapshot(signature, batch, remoteConfig, forceSync)
		if err != nil {
			return nil, err
		}
		configHash := profitBoardRemoteObserverConfigHash(remoteConfig)
		snapshots, err := listProfitBoardRemoteSuccessSnapshots(signature, batch.Id, configHash, startTimestamp, endTimestamp)
		if err != nil {
			return nil, err
		}
		batchCostQuota := int64(0)
		if len(snapshots) >= 2 {
			for index := 1; index < len(snapshots); index++ {
				deltaQuota, warnings := profitBoardRemoteSnapshotDelta(snapshots[index-1], snapshots[index])
				for _, warning := range warnings {
					aggregate.Warnings = append(aggregate.Warnings, fmt.Sprintf("%s：%s", batch.Name, warning))
				}
				if deltaQuota <= 0 {
					continue
				}
				batchCostQuota += deltaQuota
				if includeTimeseries {
					bucketTimestamp, bucketLabel := buildProfitBoardBucket(snapshots[index].SyncedAt, granularity, customIntervalMinutes)
					key := fmt.Sprintf("%s:%d", batch.Id, bucketTimestamp)
					point, ok := bucketMap[key]
					if !ok {
						point = &ProfitBoardTimeseriesPoint{
							BatchId:         batch.Id,
							BatchName:       batch.Name,
							Bucket:          bucketLabel,
							BucketTimestamp: bucketTimestamp,
						}
						bucketMap[key] = point
					}
					point.RemoteObservedCostUSD += float64(deltaQuota) / common.QuotaPerUnit
				}
			}
		}
		batchCostUSD := roundProfitBoardAmount(float64(batchCostQuota) / common.QuotaPerUnit)
		aggregate.TotalCostUSD += batchCostUSD
		aggregate.BatchCostUSD[batch.Id] = batchCostUSD
		state := buildProfitBoardRemoteObserverState(signature, batch, remoteConfig, latestAny, latestSuccess, batchCostUSD)
		aggregate.States = append(aggregate.States, state)
		if state.QuotaPerUnitMismatch {
			aggregate.Warnings = append(aggregate.Warnings, fmt.Sprintf("%s 的远端 quota_per_unit 与本站不同，当前仍按本站额度口径换算金额", batch.Name))
		}
		if state.Status == profitBoardRemoteObserverStatusFailed && state.ErrorMessage != "" {
			aggregate.Warnings = append(aggregate.Warnings, fmt.Sprintf("%s 远端额度同步失败：%s", batch.Name, state.ErrorMessage))
		}
	}
	for _, point := range bucketMap {
		point.RemoteObservedCostUSD = roundProfitBoardAmount(point.RemoteObservedCostUSD)
		aggregate.Timeseries = append(aggregate.Timeseries, *point)
	}
	sort.Slice(aggregate.Timeseries, func(i, j int) bool {
		if aggregate.Timeseries[i].BucketTimestamp == aggregate.Timeseries[j].BucketTimestamp {
			return aggregate.Timeseries[i].BatchName < aggregate.Timeseries[j].BatchName
		}
		return aggregate.Timeseries[i].BucketTimestamp < aggregate.Timeseries[j].BucketTimestamp
	})
	sort.Slice(aggregate.States, func(i, j int) bool {
		return aggregate.States[i].BatchName < aggregate.States[j].BatchName
	})
	aggregate.TotalCostUSD = roundProfitBoardAmount(aggregate.TotalCostUSD)
	aggregate.Warnings = uniqueProfitBoardWarnings(aggregate.Warnings)
	return aggregate, nil
}

func SyncProfitBoardRemoteObservers(payload ProfitBoardConfigPayload, force bool) ([]ProfitBoardRemoteObserverState, error) {
	normalizedBatches, signature, _, err := normalizeProfitBoardBatches(payload.Batches, payload.Selection)
	if err != nil {
		return nil, err
	}
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
	payload.SharedSite = normalizeProfitBoardSharedSiteConfig(payload.SharedSite, payload.Site)
	payload.ComboConfigs = normalizeProfitBoardComboConfigs(normalizedBatches, payload.ComboConfigs, payload.SharedSite, payload.Site, payload.Upstream)
	payload.ComboConfigs = hydrateProfitBoardRemoteObserverSecrets(signature, payload.ComboConfigs)
	if err := validateProfitBoardComboConfigs(payload.ComboConfigs); err != nil {
		return nil, err
	}
	resolvedBatches, err := resolveProfitBoardBatches(normalizedBatches)
	if err != nil {
		return nil, err
	}
	aggregate, err := collectProfitBoardRemoteObserverAggregate(signature, resolvedBatches, payload.ComboConfigs, 0, common.GetTimestamp(), "day", 0, force, false)
	if err != nil {
		return nil, err
	}
	return aggregate.States, nil
}
