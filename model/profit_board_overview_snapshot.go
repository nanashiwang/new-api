package model

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	profitBoardOverviewSnapshotStatusReady  = "ready"
	profitBoardOverviewSnapshotStatusFailed = "failed"
)

type ProfitBoardOverviewSnapshot struct {
	Id                  int                       `json:"id"`
	SelectionSignature  string                    `json:"selection_signature" gorm:"type:varchar(255);index:idx_profit_board_overview_snapshot_signature_updated,priority:1"`
	ConfigHash          string                    `json:"config_hash" gorm:"type:varchar(64);uniqueIndex;not null"`
	DependencyWatermark string                    `json:"dependency_watermark" gorm:"type:text"`
	Status              string                    `json:"status" gorm:"type:varchar(16);index"`
	ErrorMessage        string                    `json:"error_message,omitempty" gorm:"type:text"`
	Report              ProfitBoardSnapshotReport `json:"report,omitempty"`
	GeneratedAt         int64                     `json:"generated_at" gorm:"bigint;index"`
	UpdatedAt           int64                     `json:"updated_at" gorm:"bigint;index:idx_profit_board_overview_snapshot_signature_updated,priority:2"`
}

type ProfitBoardSnapshotReport string

func (ProfitBoardSnapshotReport) GormDataType() string {
	return "text"
}

func (ProfitBoardSnapshotReport) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db != nil && db.Dialector.Name() == "mysql" {
		return "mediumtext"
	}
	return "text"
}

type profitBoardOverviewSnapshotMeta struct {
	Payload             ProfitBoardConfigPayload
	Signature           string
	ConfigHash          string
	DependencyWatermark string
}

func buildProfitBoardWalletObserverConfigWatermark(comboPricingMap map[string]profitBoardResolvedComboPricing) string {
	accountIDs := make([]int, 0)
	seen := make(map[int]struct{})
	for _, comboPricing := range comboPricingMap {
		accountID := comboPricing.UpstreamAccountID
		if !profitBoardComboUsesWalletObserver(comboPricing) || accountID <= 0 {
			continue
		}
		if _, ok := seen[accountID]; ok {
			continue
		}
		seen[accountID] = struct{}{}
		accountIDs = append(accountIDs, accountID)
	}
	sort.Ints(accountIDs)
	parts := make([]string, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		account, err := getProfitBoardUpstreamAccountByID(accountID)
		if err != nil {
			parts = append(parts, fmt.Sprintf("%d:missing", accountID))
			continue
		}
		configHash := profitBoardRemoteObserverConfigHash(account.remoteObserverConfig())
		parts = append(parts, fmt.Sprintf("%d:%t:%d:%s", accountID, account.Enabled, account.UpdatedAt, configHash))
	}
	return strings.Join(parts, ",")
}

func buildProfitBoardOverviewSnapshotConfigHash(payload ProfitBoardConfigPayload, batchFingerprint string) string {
	payload.ComboConfigs = stripProfitBoardRemoteObserverSecrets(payload.ComboConfigs)
	payload.Selection = ProfitBoardSelection{}
	payloadBytes, err := common.Marshal(struct {
		Batches          []ProfitBoardBatch                 `json:"batches"`
		SharedSite       ProfitBoardSharedSitePricingConfig `json:"shared_site"`
		ComboConfigs     []ProfitBoardComboPricingConfig    `json:"combo_configs"`
		ExcludedUserIDs  []int                              `json:"excluded_user_ids"`
		Upstream         ProfitBoardTokenPricingConfig      `json:"upstream"`
		Site             ProfitBoardTokenPricingConfig      `json:"site"`
		BatchFingerprint string                             `json:"batch_fingerprint,omitempty"`
	}{
		Batches:          payload.Batches,
		SharedSite:       payload.SharedSite,
		ComboConfigs:     payload.ComboConfigs,
		ExcludedUserIDs:  payload.ExcludedUserIDs,
		Upstream:         payload.Upstream,
		Site:             payload.Site,
		BatchFingerprint: batchFingerprint,
	})
	if err != nil {
		return ""
	}
	hash := sha1.Sum(payloadBytes)
	return hex.EncodeToString(hash[:])
}

func buildProfitBoardRemoteObserverSnapshotWatermark(signature string, batches []ProfitBoardBatchInfo, comboConfigs []ProfitBoardComboPricingConfig) string {
	if strings.TrimSpace(signature) == "" || !profitBoardHasEnabledRemoteObserver(comboConfigs) {
		return ""
	}
	comboMap := make(map[string]ProfitBoardComboPricingConfig, len(comboConfigs))
	for _, config := range comboConfigs {
		comboMap[strings.TrimSpace(config.ComboId)] = config
	}
	parts := make([]string, 0, len(batches))
	for _, batch := range batches {
		comboConfig, ok := comboMap[batch.Id]
		if !ok {
			continue
		}
		remoteConfig := normalizeProfitBoardRemoteObserverConfig(comboConfig.RemoteObserver)
		if !remoteConfig.Enabled || !profitBoardRemoteObserverConfigured(remoteConfig) {
			continue
		}
		configHash := profitBoardRemoteObserverConfigHash(remoteConfig)
		latestAny, anyErr := getLatestProfitBoardRemoteSnapshot(signature, batch.Id)
		if anyErr != nil {
			continue
		}
		latestSuccess, successErr := getLatestProfitBoardRemoteSuccessSnapshot(signature, batch.Id, configHash)
		if successErr != nil {
			continue
		}
		anyPart := "0:0:"
		if latestAny != nil {
			anyPart = fmt.Sprintf("%d:%d:%s", latestAny.Id, latestAny.SyncedAt, latestAny.Status)
		}
		successPart := "0:0"
		if latestSuccess != nil {
			successPart = fmt.Sprintf("%d:%d", latestSuccess.Id, latestSuccess.SyncedAt)
		}
		parts = append(parts, fmt.Sprintf("%s:%s:%s:%s", batch.Id, configHash, anyPart, successPart))
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

func prepareProfitBoardOverviewSnapshotMeta(payload ProfitBoardConfigPayload) (*profitBoardOverviewSnapshotMeta, error) {
	normalizedBatches, signature, _, err := normalizeProfitBoardBatches(payload.Batches, payload.Selection)
	if err != nil {
		return nil, err
	}
	payload.Batches = normalizedBatches
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	payload.Site = normalizeProfitBoardPricingConfig(payload.Site, true)
	payload.SharedSite = normalizeProfitBoardSharedSiteConfig(payload.SharedSite, payload.Site)
	payload.ExcludedUserIDs = normalizeProfitBoardExcludedUserIDs(payload.ExcludedUserIDs)
	payload.ComboConfigs = normalizeProfitBoardComboConfigs(normalizedBatches, payload.ComboConfigs, payload.SharedSite, payload.Site, payload.Upstream)
	payload.ComboConfigs = hydrateProfitBoardRemoteObserverSecrets(signature, payload.ComboConfigs)
	if err = validateProfitBoardPricingConfig(payload.Upstream, false); err != nil {
		return nil, err
	}
	if err = validateProfitBoardPricingConfig(payload.Site, true); err != nil {
		return nil, err
	}
	if err = validateProfitBoardComboConfigs(payload.ComboConfigs); err != nil {
		return nil, err
	}
	resolvedBatches, resolvedBatchWarnings, err := resolveProfitBoardBatches(normalizedBatches)
	if err != nil {
		return nil, err
	}
	resolvedBatchFingerprint := buildProfitBoardResolvedBatchFingerprint(resolvedBatches, resolvedBatchWarnings)
	configHash := buildProfitBoardOverviewSnapshotConfigHash(payload, resolvedBatchFingerprint)
	if configHash == "" {
		return nil, errors.New("profit board overview snapshot config hash is empty")
	}

	query := ProfitBoardQuery{
		Batches:         normalizedBatches,
		SharedSite:      payload.SharedSite,
		ComboConfigs:    payload.ComboConfigs,
		ExcludedUserIDs: payload.ExcludedUserIDs,
		Upstream:        payload.Upstream,
		Site:            payload.Site,
	}
	comboPricingMap := resolveProfitBoardComboPricingMap(query, resolvedBatches)
	walletWatermark, err := buildProfitBoardWalletSnapshotWatermark(comboPricingMap)
	if err != nil {
		return nil, err
	}
	walletConfigWatermark := buildProfitBoardWalletObserverConfigWatermark(comboPricingMap)
	remoteWatermark := buildProfitBoardRemoteObserverSnapshotWatermark(signature, resolvedBatches, payload.ComboConfigs)
	dependencyWatermark := strings.Join([]string{
		"aggregate:" + buildProfitBoardAggregateActivityWatermark(),
		"wallet:" + walletWatermark,
		"wallet_config:" + walletConfigWatermark,
		"remote:" + remoteWatermark,
	}, "|")

	return &profitBoardOverviewSnapshotMeta{
		Payload:             payload,
		Signature:           signature,
		ConfigHash:          configHash,
		DependencyWatermark: dependencyWatermark,
	}, nil
}

func GetProfitBoardOverviewSnapshot(payload ProfitBoardConfigPayload) (*ProfitBoardReport, bool, error) {
	meta, err := prepareProfitBoardOverviewSnapshotMeta(payload)
	if err != nil {
		return nil, false, err
	}
	snapshot := &ProfitBoardOverviewSnapshot{}
	err = DB.Where("config_hash = ? AND status = ?", meta.ConfigHash, profitBoardOverviewSnapshotStatusReady).First(snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if snapshot.DependencyWatermark != meta.DependencyWatermark {
		return nil, false, nil
	}
	report := &ProfitBoardReport{}
	if err = common.UnmarshalJsonStr(string(snapshot.Report), report); err != nil {
		return nil, false, err
	}
	return report, true, nil
}

func saveProfitBoardOverviewSnapshotWithMeta(meta *profitBoardOverviewSnapshotMeta, report *ProfitBoardReport) error {
	if report == nil {
		return nil
	}
	reportBytes, err := common.Marshal(report)
	if err != nil {
		return err
	}
	now := common.GetTimestamp()
	snapshot := &ProfitBoardOverviewSnapshot{}
	err = DB.Where("config_hash = ?", meta.ConfigHash).First(snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DB.Create(&ProfitBoardOverviewSnapshot{
			SelectionSignature:  meta.Signature,
			ConfigHash:          meta.ConfigHash,
			DependencyWatermark: meta.DependencyWatermark,
			Status:              profitBoardOverviewSnapshotStatusReady,
			Report:              ProfitBoardSnapshotReport(string(reportBytes)),
			GeneratedAt:         report.Meta.GeneratedAt,
			UpdatedAt:           now,
		}).Error
	}
	if err != nil {
		return err
	}
	return DB.Model(&ProfitBoardOverviewSnapshot{}).
		Where("config_hash = ?", meta.ConfigHash).
		Updates(map[string]any{
			"selection_signature":  meta.Signature,
			"dependency_watermark": meta.DependencyWatermark,
			"status":               profitBoardOverviewSnapshotStatusReady,
			"error_message":        "",
			"report":               string(reportBytes),
			"generated_at":         report.Meta.GeneratedAt,
			"updated_at":           now,
		}).Error
}

func saveProfitBoardOverviewSnapshot(payload ProfitBoardConfigPayload, report *ProfitBoardReport) error {
	meta, err := prepareProfitBoardOverviewSnapshotMeta(payload)
	if err != nil {
		return err
	}
	return saveProfitBoardOverviewSnapshotWithMeta(meta, report)
}

func saveProfitBoardOverviewSnapshotFailure(payload ProfitBoardConfigPayload, snapshotErr error) error {
	meta, err := prepareProfitBoardOverviewSnapshotMeta(payload)
	if err != nil {
		return err
	}
	now := common.GetTimestamp()
	message := ""
	if snapshotErr != nil {
		message = snapshotErr.Error()
	}
	snapshot := &ProfitBoardOverviewSnapshot{}
	err = DB.Where("config_hash = ?", meta.ConfigHash).First(snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DB.Create(&ProfitBoardOverviewSnapshot{
			SelectionSignature:  meta.Signature,
			ConfigHash:          meta.ConfigHash,
			DependencyWatermark: meta.DependencyWatermark,
			Status:              profitBoardOverviewSnapshotStatusFailed,
			ErrorMessage:        message,
			GeneratedAt:         now,
			UpdatedAt:           now,
		}).Error
	}
	if err != nil {
		return err
	}
	return DB.Model(&ProfitBoardOverviewSnapshot{}).
		Where("config_hash = ?", meta.ConfigHash).
		Updates(map[string]any{
			"selection_signature":  meta.Signature,
			"dependency_watermark": meta.DependencyWatermark,
			"status":               profitBoardOverviewSnapshotStatusFailed,
			"error_message":        message,
			"generated_at":         now,
			"updated_at":           now,
		}).Error
}

func SyncProfitBoardOverviewSnapshotForPayload(payload ProfitBoardConfigPayload) error {
	meta, err := prepareProfitBoardOverviewSnapshotMeta(payload)
	if err != nil {
		return err
	}
	current := &ProfitBoardOverviewSnapshot{}
	err = DB.Where("config_hash = ? AND status = ? AND dependency_watermark = ?", meta.ConfigHash, profitBoardOverviewSnapshotStatusReady, meta.DependencyWatermark).
		First(current).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	report, err := generateProfitBoardOverviewSummaryWithCache(meta.Payload, false)
	if err != nil {
		_ = saveProfitBoardOverviewSnapshotFailure(meta.Payload, err)
		return err
	}
	return saveProfitBoardOverviewSnapshotWithMeta(meta, report)
}

func SyncProfitBoardOverviewSnapshots() error {
	records := make([]ProfitBoardConfig, 0)
	if err := DB.Order("updated_at desc").Find(&records).Error; err != nil {
		return err
	}
	var firstErr error
	for _, record := range records {
		payload, err := payloadFromProfitBoardConfigRecord(record)
		if err != nil {
			common.SysError("profit board overview snapshot parse config failed: " + err.Error())
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if payload == nil || len(payload.Batches) == 0 {
			continue
		}
		if err = SyncProfitBoardOverviewSnapshotForPayload(*payload); err != nil {
			common.SysError("profit board overview snapshot sync failed: " + err.Error())
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
