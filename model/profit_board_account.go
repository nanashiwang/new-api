package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Sentinel errors for i18n translation (upstream account)
var (
	ErrProfitBoardAccountTypeUnsupported = errors.New("profit_board:account_type_unsupported")
	ErrProfitBoardAccountNameEmpty       = errors.New("profit_board:account_name_empty")
	ErrProfitBoardAccountInvalid         = errors.New("profit_board:account_invalid")
	ErrProfitBoardAccountTokenEmpty      = errors.New("profit_board:account_token_empty")
)

const profitBoardUpstreamAccountSnapshotComboID = "wallet"

type ProfitBoardUpstreamAccount struct {
	Id                     int     `json:"id"`
	Name                   string  `json:"name" gorm:"type:varchar(128);not null"`
	Remark                 string  `json:"remark,omitempty" gorm:"type:text"`
	AccountType            string  `json:"account_type" gorm:"type:varchar(24);index;not null"`
	BaseURL                string  `json:"base_url" gorm:"type:varchar(255);not null"`
	UserID                 int     `json:"user_id" gorm:"index;not null"`
	AccessToken            string  `json:"access_token,omitempty" gorm:"-"`
	AccessTokenMasked      string  `json:"access_token_masked,omitempty" gorm:"-"`
	AccessTokenEncrypted   string  `json:"-" gorm:"type:text;not null"`
	Enabled                bool    `json:"enabled" gorm:"default:true"`
	ResourceDisplayMode    string  `json:"resource_display_mode" gorm:"type:varchar(24);default:both"`
	LowBalanceThresholdUSD float64 `json:"low_balance_threshold_usd" gorm:"type:decimal(18,6);default:0"`
	CreatedAt              int64   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt              int64   `json:"updated_at" gorm:"bigint;index"`
}

type ProfitBoardUpstreamAccountOption struct {
	Id                           int     `json:"id"`
	Name                         string  `json:"name"`
	Remark                       string  `json:"remark,omitempty"`
	AccountType                  string  `json:"account_type"`
	BaseURL                      string  `json:"base_url"`
	UserID                       int     `json:"user_id"`
	Enabled                      bool    `json:"enabled"`
	ResourceDisplayMode          string  `json:"resource_display_mode"`
	AccessTokenMasked            string  `json:"access_token_masked,omitempty"`
	Status                       string  `json:"status,omitempty"`
	ErrorMessage                 string  `json:"error_message,omitempty"`
	LastSyncedAt                 int64   `json:"last_synced_at"`
	LastSuccessAt                int64   `json:"last_success_at"`
	WalletBalanceUSD             float64 `json:"wallet_balance_usd"`
	WalletQuotaUSD               float64 `json:"wallet_quota_usd"`
	WalletUsedTotalUSD           float64 `json:"wallet_used_total_usd"`
	WalletUsedQuotaUSD           float64 `json:"wallet_used_quota_usd"`
	PeriodUsedUSD                float64 `json:"period_used_usd"`
	SubscriptionRemainingUSD     float64 `json:"subscription_remaining_quota_usd"`
	SubscriptionTotalQuotaUSD    float64 `json:"subscription_total_quota_usd"`
	SubscriptionUsedQuotaUSD     float64 `json:"subscription_used_quota_usd"`
	SubscriptionCount            int     `json:"subscription_count"`
	SubscriptionEarliestExpireAt int64   `json:"subscription_earliest_expire_at"`
	HasSubscriptionData          bool    `json:"has_subscription_data"`
	SubscriptionHasUnlimited     bool    `json:"subscription_has_unlimited"`
	ObservedCostUSD              float64 `json:"observed_cost_usd"`
	RemoteQuotaPerUnit           float64 `json:"remote_quota_per_unit"`
	QuotaPerUnitMismatch         bool    `json:"quota_per_unit_mismatch"`
	LowBalanceThresholdUSD       float64 `json:"low_balance_threshold_usd"`
	LowBalanceAlert              bool    `json:"low_balance_alert"`
	BaselineReady                bool    `json:"baseline_ready"`
	SnapshotCount                int     `json:"snapshot_count"`
}

type profitBoardUpstreamAccountObservedAggregate struct {
	TotalCostUSD  float64
	BucketCostUSD map[int64]float64
	BucketLabels  map[int64]string
	Points        []profitBoardUpstreamAccountObservedPoint
	State         ProfitBoardUpstreamAccountOption
	Warnings      []string
}

type profitBoardUpstreamAccountObservedPoint struct {
	SyncedAt int64
	CostUSD  float64
}

type profitBoardUpstreamSubscriptionSummary struct {
	RemainingUSD     float64
	TotalUSD         float64
	UsedUSD          float64
	Count            int
	EarliestExpireAt int64
	HasData          bool
	HasUnlimited     bool
	Details          []ProfitBoardUpstreamAccountSubscription
}

func summarizeProfitBoardUpstreamSubscriptions(subscriptions []ProfitBoardRemoteSubscriptionSnapshot) profitBoardUpstreamSubscriptionSummary {
	summary := profitBoardUpstreamSubscriptionSummary{
		Details: make([]ProfitBoardUpstreamAccountSubscription, 0, len(subscriptions)),
	}
	if len(subscriptions) == 0 {
		return summary
	}
	summary.HasData = true
	summary.Count = len(subscriptions)
	for _, item := range subscriptions {
		usedUSD := profitBoardQuotaToUSD(item.AmountUsed)
		totalUSD := 0.0
		remainingUSD := 0.0
		hasUnlimited := item.AmountTotal <= 0
		if hasUnlimited {
			summary.HasUnlimited = true
		} else {
			totalUSD = profitBoardQuotaToUSD(item.AmountTotal)
			remainingQuota := item.AmountTotal - item.AmountUsed
			if remainingQuota < 0 {
				remainingQuota = 0
			}
			remainingUSD = profitBoardQuotaToUSD(remainingQuota)
			summary.TotalUSD += totalUSD
			summary.RemainingUSD += remainingUSD
		}
		summary.UsedUSD += usedUSD
		if item.EndTime > 0 && (summary.EarliestExpireAt == 0 || item.EndTime < summary.EarliestExpireAt) {
			summary.EarliestExpireAt = item.EndTime
		}
		summary.Details = append(summary.Details, ProfitBoardUpstreamAccountSubscription{
			SubscriptionID:    item.SubscriptionID,
			PlanID:            item.PlanID,
			TotalQuotaUSD:     roundProfitBoardAmount(totalUSD),
			UsedQuotaUSD:      roundProfitBoardAmount(usedUSD),
			RemainingQuotaUSD: roundProfitBoardAmount(remainingUSD),
			HasUnlimited:      hasUnlimited,
			LastResetTime:     item.LastResetTime,
			NextResetTime:     item.NextResetTime,
			StartTime:         item.StartTime,
			EndTime:           item.EndTime,
			Status:            item.Status,
		})
	}
	summary.TotalUSD = roundProfitBoardAmount(summary.TotalUSD)
	summary.UsedUSD = roundProfitBoardAmount(summary.UsedUSD)
	summary.RemainingUSD = roundProfitBoardAmount(summary.RemainingUSD)
	sort.Slice(summary.Details, func(i, j int) bool {
		left := summary.Details[i]
		right := summary.Details[j]
		if left.EndTime == 0 && right.EndTime != 0 {
			return false
		}
		if left.EndTime != 0 && right.EndTime == 0 {
			return true
		}
		if left.EndTime != right.EndTime {
			return left.EndTime < right.EndTime
		}
		return left.SubscriptionID < right.SubscriptionID
	})
	return summary
}

func (a *ProfitBoardUpstreamAccount) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if a.CreatedAt <= 0 {
		a.CreatedAt = now
	}
	if a.UpdatedAt <= 0 {
		a.UpdatedAt = now
	}
	return nil
}

func (a *ProfitBoardUpstreamAccount) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = common.GetTimestamp()
	return nil
}

func normalizeProfitBoardUpstreamAccountResourceDisplayMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ProfitBoardResourceDisplayWallet:
		return ProfitBoardResourceDisplayWallet
	case ProfitBoardResourceDisplaySubscription:
		return ProfitBoardResourceDisplaySubscription
	default:
		return ProfitBoardResourceDisplayBoth
	}
}

func normalizeProfitBoardUpstreamAccount(account ProfitBoardUpstreamAccount) ProfitBoardUpstreamAccount {
	account.Name = strings.TrimSpace(account.Name)
	account.Remark = strings.TrimSpace(account.Remark)
	account.AccountType = strings.ToLower(strings.TrimSpace(account.AccountType))
	if account.AccountType == "" {
		account.AccountType = ProfitBoardUpstreamAccountTypeNewAPI
	}
	account.BaseURL = strings.TrimRight(strings.TrimSpace(account.BaseURL), "/")
	account.AccessToken = strings.TrimSpace(account.AccessToken)
	if account.UserID < 0 {
		account.UserID = 0
	}
	if account.LowBalanceThresholdUSD < 0 {
		account.LowBalanceThresholdUSD = 0
	}
	account.ResourceDisplayMode = normalizeProfitBoardUpstreamAccountResourceDisplayMode(account.ResourceDisplayMode)
	return account
}

func validateProfitBoardUpstreamAccount(account ProfitBoardUpstreamAccount, requireSecret bool) error {
	account = normalizeProfitBoardUpstreamAccount(account)
	if account.AccountType != ProfitBoardUpstreamAccountTypeNewAPI {
		return ErrProfitBoardAccountTypeUnsupported
	}
	if account.Name == "" {
		return ErrProfitBoardAccountNameEmpty
	}
	config := ProfitBoardRemoteObserverConfig{
		Enabled:              true,
		BaseURL:              account.BaseURL,
		UserID:               account.UserID,
		AccessToken:          account.AccessToken,
		AccessTokenEncrypted: account.AccessTokenEncrypted,
	}
	if !requireSecret {
		config.AccessToken = ""
	}
	if err := validateProfitBoardRemoteObserverConfig(config); err != nil {
		return err
	}
	return nil
}

func (a ProfitBoardUpstreamAccount) remoteObserverConfig() ProfitBoardRemoteObserverConfig {
	return normalizeProfitBoardRemoteObserverConfig(ProfitBoardRemoteObserverConfig{
		Enabled:              a.Enabled,
		BaseURL:              a.BaseURL,
		UserID:               a.UserID,
		AccessToken:          a.AccessToken,
		AccessTokenEncrypted: a.AccessTokenEncrypted,
	})
}

func profitBoardUpstreamAccountSnapshotSignature(accountID int) string {
	return fmt.Sprintf("profit_board_account:%d", accountID)
}

func profitBoardUpstreamAccountSnapshotBatch(account ProfitBoardUpstreamAccount) ProfitBoardBatchInfo {
	return ProfitBoardBatchInfo{
		Id:   profitBoardUpstreamAccountSnapshotComboID,
		Name: account.Name,
	}
}

func buildProfitBoardUpstreamAccountOption(
	account ProfitBoardUpstreamAccount,
	state ProfitBoardRemoteObserverState,
) ProfitBoardUpstreamAccountOption {
	account = normalizeProfitBoardUpstreamAccount(account)
	threshold := roundProfitBoardAmount(account.LowBalanceThresholdUSD)
	lowBalanceAlert := threshold > 0 && state.WalletBalanceUSD <= threshold
	return ProfitBoardUpstreamAccountOption{
		Id:                           account.Id,
		Name:                         account.Name,
		Remark:                       account.Remark,
		AccountType:                  account.AccountType,
		BaseURL:                      account.BaseURL,
		UserID:                       account.UserID,
		Enabled:                      account.Enabled,
		ResourceDisplayMode:          account.ResourceDisplayMode,
		AccessTokenMasked:            maskProfitBoardRemoteSecret(statefulProfitBoardUpstreamToken(account)),
		Status:                       state.Status,
		ErrorMessage:                 state.ErrorMessage,
		LastSyncedAt:                 state.LastSyncedAt,
		LastSuccessAt:                state.LastSuccessAt,
		WalletBalanceUSD:             state.WalletBalanceUSD,
		WalletQuotaUSD:               state.WalletQuotaUSD,
		WalletUsedTotalUSD:           state.WalletUsedTotalUSD,
		WalletUsedQuotaUSD:           state.WalletUsedQuotaUSD,
		PeriodUsedUSD:                state.PeriodUsedUSD,
		SubscriptionRemainingUSD:     state.SubscriptionRemainingUSD,
		SubscriptionTotalQuotaUSD:    state.SubscriptionTotalQuotaUSD,
		SubscriptionUsedQuotaUSD:     state.SubscriptionUsedQuotaUSD,
		SubscriptionCount:            state.SubscriptionCount,
		SubscriptionEarliestExpireAt: state.SubscriptionEarliestExpireAt,
		HasSubscriptionData:          state.HasSubscriptionData,
		SubscriptionHasUnlimited:     state.SubscriptionHasUnlimited,
		ObservedCostUSD:              state.PeriodUsedUSD,
		RemoteQuotaPerUnit:           state.RemoteQuotaPerUnit,
		QuotaPerUnitMismatch:         state.QuotaPerUnitMismatch,
		LowBalanceThresholdUSD:       threshold,
		LowBalanceAlert:              lowBalanceAlert,
		BaselineReady:                state.BaselineReady,
	}
}

func statefulProfitBoardUpstreamToken(account ProfitBoardUpstreamAccount) string {
	if account.AccessToken != "" {
		return account.AccessToken
	}
	if account.AccessTokenEncrypted == "" {
		return ""
	}
	decrypted, err := decryptProfitBoardRemoteSecret(account.AccessTokenEncrypted)
	if err != nil {
		return ""
	}
	return decrypted
}

func getProfitBoardUpstreamAccountByID(id int) (*ProfitBoardUpstreamAccount, error) {
	if id <= 0 {
		return nil, ErrProfitBoardAccountInvalid
	}
	account := &ProfitBoardUpstreamAccount{}
	if err := DB.First(account, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return account, nil
}

func listProfitBoardUpstreamAccounts() ([]ProfitBoardUpstreamAccount, error) {
	accounts := make([]ProfitBoardUpstreamAccount, 0)
	if err := DB.Order("updated_at desc, id desc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func buildProfitBoardUpstreamAccountState(account ProfitBoardUpstreamAccount, observedCostUSD float64) (ProfitBoardRemoteObserverState, error) {
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	config := account.remoteObserverConfig()
	batch := profitBoardUpstreamAccountSnapshotBatch(account)
	latestAny, err := getLatestProfitBoardRemoteSnapshot(signature, profitBoardUpstreamAccountSnapshotComboID)
	if err != nil {
		return ProfitBoardRemoteObserverState{}, err
	}
	latestSuccess, err := getLatestProfitBoardRemoteSuccessSnapshot(signature, profitBoardUpstreamAccountSnapshotComboID, profitBoardRemoteObserverConfigHash(config))
	if err != nil {
		return ProfitBoardRemoteObserverState{}, err
	}
	return buildProfitBoardRemoteObserverState(signature, batch, config, latestAny, latestSuccess, observedCostUSD), nil
}

func GetProfitBoardUpstreamAccountOptions() ([]ProfitBoardUpstreamAccountOption, error) {
	accounts, err := listProfitBoardUpstreamAccounts()
	if err != nil {
		return nil, err
	}
	options := make([]ProfitBoardUpstreamAccountOption, 0, len(accounts))
	for _, account := range accounts {
		state, stateErr := buildProfitBoardUpstreamAccountState(account, 0)
		if stateErr != nil {
			return nil, stateErr
		}
		options = append(options, buildProfitBoardUpstreamAccountOption(account, state))
	}
	sortProfitBoardUpstreamAccountOptions(options)
	return options, nil
}

func SaveProfitBoardUpstreamAccount(account ProfitBoardUpstreamAccount) (*ProfitBoardUpstreamAccountOption, error) {
	account = normalizeProfitBoardUpstreamAccount(account)
	var existing ProfitBoardUpstreamAccount
	if account.Id > 0 {
		if err := DB.First(&existing, "id = ?", account.Id).Error; err != nil {
			return nil, err
		}
		if account.AccessTokenEncrypted == "" {
			account.AccessTokenEncrypted = existing.AccessTokenEncrypted
		}
	}
	requireSecret := account.Id == 0 || account.AccessToken != "" || account.AccessTokenEncrypted == ""
	if err := validateProfitBoardUpstreamAccount(account, requireSecret); err != nil {
		return nil, err
	}
	switch {
	case account.AccessToken != "":
		encrypted, err := encryptProfitBoardRemoteSecret(account.AccessToken)
		if err != nil {
			return nil, err
		}
		account.AccessTokenEncrypted = encrypted
	case existing.AccessTokenEncrypted != "":
		account.AccessTokenEncrypted = existing.AccessTokenEncrypted
	default:
		return nil, ErrProfitBoardAccountTokenEmpty
	}
	account.AccessToken = ""
	if account.Id == 0 {
		if err := DB.Create(&account).Error; err != nil {
			return nil, err
		}
	} else {
		existing.Name = account.Name
		existing.Remark = account.Remark
		existing.AccountType = account.AccountType
		existing.BaseURL = account.BaseURL
		existing.UserID = account.UserID
		existing.AccessTokenEncrypted = account.AccessTokenEncrypted
		existing.Enabled = account.Enabled
		existing.ResourceDisplayMode = account.ResourceDisplayMode
		existing.LowBalanceThresholdUSD = account.LowBalanceThresholdUSD
		if err := DB.Save(&existing).Error; err != nil {
			return nil, err
		}
		account = existing
	}
	state, err := buildProfitBoardUpstreamAccountState(account, 0)
	if err != nil {
		return nil, err
	}
	option := buildProfitBoardUpstreamAccountOption(account, state)
	return &option, nil
}

func DeleteProfitBoardUpstreamAccount(id int) error {
	account, err := getProfitBoardUpstreamAccountByID(id)
	if err != nil {
		return err
	}
	inUse, err := profitBoardUpstreamAccountInUse(id)
	if err != nil {
		return err
	}
	if inUse {
		return ErrProfitBoardAccountInUse
	}
	if err := DB.Delete(account).Error; err != nil {
		return err
	}
	return nil
}

func profitBoardUpstreamAccountInUse(id int) (bool, error) {
	records := make([]ProfitBoardConfig, 0)
	if err := DB.Find(&records).Error; err != nil {
		return false, err
	}
	for _, record := range records {
		payload, err := payloadFromProfitBoardConfigRecord(record)
		if err != nil {
			return false, err
		}
		if payload == nil {
			continue
		}
		comboConfigs := normalizeProfitBoardComboConfigs(payload.Batches, payload.ComboConfigs, payload.SharedSite, payload.Site, payload.Upstream)
		for _, comboConfig := range comboConfigs {
			if comboConfig.UpstreamMode != ProfitBoardUpstreamModeWallet {
				continue
			}
			if comboConfig.UpstreamAccountID == id {
				return true, nil
			}
		}
	}
	return false, nil
}

func SyncProfitBoardUpstreamAccount(id int, force bool) (*ProfitBoardUpstreamAccountOption, error) {
	account, err := getProfitBoardUpstreamAccountByID(id)
	if err != nil {
		return nil, err
	}
	config := account.remoteObserverConfig()
	config.Enabled = true
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	batch := profitBoardUpstreamAccountSnapshotBatch(*account)
	latestAny, latestSuccess, err := syncProfitBoardRemoteObserverSnapshot(signature, batch, config, force)
	if err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	aggregate, aggregateErr := collectProfitBoardUpstreamAccountObservedAggregate(
		id,
		now-7*24*60*60,
		now,
		"day",
		0,
		false,
	)
	if aggregateErr != nil {
		return nil, aggregateErr
	}
	state := buildProfitBoardRemoteObserverState(
		signature,
		batch,
		config,
		latestAny,
		latestSuccess,
		aggregate.TotalCostUSD,
	)
	option := buildProfitBoardUpstreamAccountOption(*account, state)
	return &option, nil
}

func SyncAllProfitBoardUpstreamAccounts(force bool) ([]ProfitBoardUpstreamAccountOption, error) {
	accounts, err := listProfitBoardUpstreamAccounts()
	if err != nil {
		return nil, err
	}
	options := make([]ProfitBoardUpstreamAccountOption, 0, len(accounts))
	for _, account := range accounts {
		if account.Enabled {
			option, syncErr := SyncProfitBoardUpstreamAccount(account.Id, force)
			if syncErr != nil {
				return nil, syncErr
			}
			options = append(options, *option)
			continue
		}
		state, stateErr := buildProfitBoardUpstreamAccountState(account, 0)
		if stateErr != nil {
			return nil, stateErr
		}
		options = append(options, buildProfitBoardUpstreamAccountOption(account, state))
	}
	sortProfitBoardUpstreamAccountOptions(options)
	return options, nil
}

func collectProfitBoardUpstreamAccountObservedAggregate(accountID int, startTimestamp int64, endTimestamp int64, granularity string, customIntervalMinutes int, forceSync bool) (*profitBoardUpstreamAccountObservedAggregate, error) {
	account, err := getProfitBoardUpstreamAccountByID(accountID)
	if err != nil {
		return nil, err
	}
	config := account.remoteObserverConfig()
	config.Enabled = true
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	batch := profitBoardUpstreamAccountSnapshotBatch(*account)
	latestAny, latestSuccess, err := syncProfitBoardRemoteObserverSnapshot(signature, batch, config, forceSync)
	if err != nil {
		return nil, err
	}
	configHash := profitBoardRemoteObserverConfigHash(config)
	if configHash == "" && latestSuccess != nil {
		configHash = latestSuccess.ConfigHash
	}
	effectiveEndTimestamp := endTimestamp
	if latestSuccess != nil && latestSuccess.SyncedAt > effectiveEndTimestamp {
		effectiveEndTimestamp = latestSuccess.SyncedAt
	}
	snapshots, err := listProfitBoardRemoteSuccessSnapshots(signature, profitBoardUpstreamAccountSnapshotComboID, configHash, startTimestamp, effectiveEndTimestamp)
	if err != nil {
		return nil, err
	}
	aggregate := &profitBoardUpstreamAccountObservedAggregate{
		BucketCostUSD: make(map[int64]float64),
		BucketLabels:  make(map[int64]string),
		Points:        make([]profitBoardUpstreamAccountObservedPoint, 0),
		Warnings:      make([]string, 0),
	}
	totalCostQuota := int64(0)
	if len(snapshots) == 0 {
		if configHash == "" {
			aggregate.Warnings = append(aggregate.Warnings,
				fmt.Sprintf("%s：未找到有效的远端快照，请确认账户凭证（Base URL / User ID / Access Token）是否正确配置", account.Name))
		} else {
			aggregate.Warnings = append(aggregate.Warnings,
				fmt.Sprintf("%s：在所选时间范围内没有成功同步的远端快照。首次使用钱包观测模式需要至少一次手动同步，此后系统会自动定期同步", account.Name))
		}
	} else if len(snapshots) == 1 {
		aggregate.Warnings = append(aggregate.Warnings,
			fmt.Sprintf("%s：当前仅有 1 个成功快照，需要至少 2 个快照才能计算额度消耗差值。请再次手动同步或等待自动同步", account.Name))
	}
	if len(snapshots) >= 2 {
		for index := 1; index < len(snapshots); index++ {
			deltaQuota, warnings := profitBoardRemoteSnapshotDelta(snapshots[index-1], snapshots[index])
			for _, warning := range warnings {
				aggregate.Warnings = append(aggregate.Warnings, fmt.Sprintf("%s：%s", account.Name, warning))
			}
			if deltaQuota <= 0 {
				continue
			}
			totalCostQuota += deltaQuota
			costUSD := float64(deltaQuota) / common.QuotaPerUnit
			bucketTimestamp, bucketLabel := buildProfitBoardBucket(snapshots[index].SyncedAt, granularity, customIntervalMinutes)
			aggregate.BucketCostUSD[bucketTimestamp] += costUSD
			aggregate.BucketLabels[bucketTimestamp] = bucketLabel
			aggregate.Points = append(aggregate.Points, profitBoardUpstreamAccountObservedPoint{
				SyncedAt: snapshots[index].SyncedAt,
				CostUSD:  costUSD,
			})
		}
		if totalCostQuota == 0 {
			allRolledBack := true
			for index := 1; index < len(snapshots); index++ {
				if snapshots[index].WalletUsedQuota >= snapshots[index-1].WalletUsedQuota {
					allRolledBack = false
					break
				}
			}
			if allRolledBack {
				aggregate.Warnings = append(aggregate.Warnings,
					fmt.Sprintf("%s：远端钱包已用额度在所有快照间均出现回退（可能因为远端账户充值或额度重置），当前时间段观测成本按 0 处理", account.Name))
			} else {
				aggregate.Warnings = append(aggregate.Warnings,
					fmt.Sprintf("%s：快照间额度差值均为 0，说明所选时间范围内远端钱包没有新的额度消耗", account.Name))
			}
		}
	}
	aggregate.TotalCostUSD = roundProfitBoardAmount(float64(totalCostQuota) / common.QuotaPerUnit)
	state := buildProfitBoardRemoteObserverState(signature, batch, config, latestAny, latestSuccess, aggregate.TotalCostUSD)
	aggregate.State = buildProfitBoardUpstreamAccountOption(*account, state)
	aggregate.State.SnapshotCount = len(snapshots)
	if state.QuotaPerUnitMismatch {
		aggregate.Warnings = append(aggregate.Warnings, fmt.Sprintf("%s 的远端 quota_per_unit 与本站不同，当前仍按本站额度口径换算金额", account.Name))
	}
	if state.Status == profitBoardRemoteObserverStatusFailed && state.ErrorMessage != "" {
		aggregate.Warnings = append(aggregate.Warnings, fmt.Sprintf("%s 同步失败：%s", account.Name, state.ErrorMessage))
	}
	aggregate.Warnings = uniqueProfitBoardWarnings(aggregate.Warnings)
	return aggregate, nil
}

func GetProfitBoardUpstreamAccountTrend(id int, startTimestamp int64, endTimestamp int64, granularity string, customIntervalMinutes int) (*ProfitBoardUpstreamAccountTrend, error) {
	now := common.GetTimestamp()
	if endTimestamp <= 0 {
		endTimestamp = now
	}
	if startTimestamp <= 0 || startTimestamp >= endTimestamp {
		startTimestamp = endTimestamp - 7*24*60*60
	}
	if strings.TrimSpace(granularity) == "" {
		granularity = "day"
	}
	if customIntervalMinutes <= 0 {
		customIntervalMinutes = 15
	}
	account, err := getProfitBoardUpstreamAccountByID(id)
	if err != nil {
		return nil, err
	}
	config := account.remoteObserverConfig()
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	configHash := profitBoardRemoteObserverConfigHash(config)
	if configHash == "" {
		latestSuccess, latestErr := getLatestProfitBoardRemoteSuccessSnapshot(signature, profitBoardUpstreamAccountSnapshotComboID, "")
		if latestErr != nil {
			return nil, latestErr
		}
		if latestSuccess != nil {
			configHash = latestSuccess.ConfigHash
		}
	}
	snapshots, err := listProfitBoardRemoteSuccessSnapshots(signature, profitBoardUpstreamAccountSnapshotComboID, configHash, startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	if len(snapshots) < 2 {
		snapshots, err = listProfitBoardRemoteSuccessSnapshots(signature, profitBoardUpstreamAccountSnapshotComboID, configHash, 0, endTimestamp)
		if err != nil {
			return nil, err
		}
	}
	bucketCostUSD := make(map[int64]float64)
	bucketLabels := make(map[int64]string)
	warnings := make([]string, 0)
	totalCostUSD := 0.0
	if len(snapshots) >= 2 {
		for index := 1; index < len(snapshots); index++ {
			deltaQuota, deltaWarnings := profitBoardRemoteSnapshotDelta(snapshots[index-1], snapshots[index])
			for _, warning := range deltaWarnings {
				warnings = append(warnings, fmt.Sprintf("%s：%s", account.Name, warning))
			}
			if deltaQuota <= 0 {
				continue
			}
			periodUsedUSD := float64(deltaQuota) / common.QuotaPerUnit
			totalCostUSD += periodUsedUSD
			bucketTimestamp, bucketLabel := buildProfitBoardBucket(snapshots[index].SyncedAt, granularity, customIntervalMinutes)
			bucketCostUSD[bucketTimestamp] += periodUsedUSD
			bucketLabels[bucketTimestamp] = bucketLabel
		}
	}
	state, err := buildProfitBoardUpstreamAccountState(*account, roundProfitBoardAmount(totalCostUSD))
	if err != nil {
		return nil, err
	}
	option := buildProfitBoardUpstreamAccountOption(*account, state)
	subscriptionSummary := summarizeProfitBoardUpstreamSubscriptions(nil)
	if latestSuccess, latestErr := getLatestProfitBoardRemoteSuccessSnapshot(signature, profitBoardUpstreamAccountSnapshotComboID, configHash); latestErr != nil {
		return nil, latestErr
	} else if latestSuccess != nil {
		subscriptionSummary = summarizeProfitBoardUpstreamSubscriptions(parseProfitBoardRemoteSubscriptions(latestSuccess.SubscriptionStates))
	}
	points := make([]ProfitBoardUpstreamAccountTrendPoint, 0, len(bucketCostUSD))
	for bucketTimestamp, periodUsedUSD := range bucketCostUSD {
		points = append(points, ProfitBoardUpstreamAccountTrendPoint{
			Bucket:          bucketLabels[bucketTimestamp],
			BucketTimestamp: bucketTimestamp,
			PeriodUsedUSD:   roundProfitBoardAmount(periodUsedUSD),
		})
	}
	sort.Slice(points, func(i, j int) bool {
		return points[i].BucketTimestamp < points[j].BucketTimestamp
	})
	return &ProfitBoardUpstreamAccountTrend{
		Account:               option,
		Points:                points,
		Subscriptions:         subscriptionSummary.Details,
		StartTimestamp:        startTimestamp,
		EndTimestamp:          endTimestamp,
		Granularity:           granularity,
		CustomIntervalMinutes: customIntervalMinutes,
		Warnings:              uniqueProfitBoardWarnings(warnings),
	}, nil
}

func sortProfitBoardUpstreamAccountOptions(options []ProfitBoardUpstreamAccountOption) {
	sort.Slice(options, func(i, j int) bool {
		if options[i].Enabled != options[j].Enabled {
			return options[i].Enabled
		}
		return options[i].Name < options[j].Name
	})
}

func findOrCreateProfitBoardUpstreamAccountByRemoteConfig(config ProfitBoardRemoteObserverConfig) (*ProfitBoardUpstreamAccount, error) {
	config = normalizeProfitBoardRemoteObserverConfig(config)
	if !profitBoardRemoteObserverConfigured(config) {
		return nil, nil
	}
	targetHash := profitBoardRemoteObserverConfigHash(config)
	candidates := make([]ProfitBoardUpstreamAccount, 0)
	if err := DB.Where("account_type = ? AND base_url = ? AND user_id = ?", ProfitBoardUpstreamAccountTypeNewAPI, config.BaseURL, config.UserID).Find(&candidates).Error; err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		if profitBoardRemoteObserverConfigHash(candidate.remoteObserverConfig()) == targetHash {
			return &candidate, nil
		}
	}
	account := ProfitBoardUpstreamAccount{
		Name:                 fmt.Sprintf("迁移账户 %d", common.GetTimestamp()),
		Remark:               "自动从旧版收益看板配置迁移",
		AccountType:          ProfitBoardUpstreamAccountTypeNewAPI,
		BaseURL:              config.BaseURL,
		UserID:               config.UserID,
		AccessTokenEncrypted: config.AccessTokenEncrypted,
		Enabled:              config.Enabled,
	}
	if account.AccessTokenEncrypted == "" && config.AccessToken != "" {
		encrypted, err := encryptProfitBoardRemoteSecret(config.AccessToken)
		if err != nil {
			return nil, err
		}
		account.AccessTokenEncrypted = encrypted
	}
	if err := DB.Create(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func migrateProfitBoardLegacyWalletAccount(payload *ProfitBoardConfigPayload) error {
	if payload == nil {
		return nil
	}
	payload.Upstream = normalizeProfitBoardPricingConfig(payload.Upstream, false)
	payload.ComboConfigs = normalizeProfitBoardComboConfigs(payload.Batches, payload.ComboConfigs, payload.SharedSite, payload.Site, payload.Upstream)
	needsLegacyMigration := payload.Upstream.UpstreamMode == ProfitBoardUpstreamModeWallet
	for index := range payload.ComboConfigs {
		if payload.ComboConfigs[index].UpstreamMode == ProfitBoardUpstreamModeWallet && payload.ComboConfigs[index].UpstreamAccountID <= 0 {
			needsLegacyMigration = true
			break
		}
	}
	if !needsLegacyMigration {
		return nil
	}
	if payload.Upstream.UpstreamMode == ProfitBoardUpstreamModeWallet && payload.Upstream.UpstreamAccountID > 0 {
		for index := range payload.ComboConfigs {
			if payload.ComboConfigs[index].UpstreamMode == ProfitBoardUpstreamModeWallet && payload.ComboConfigs[index].UpstreamAccountID <= 0 {
				payload.ComboConfigs[index].UpstreamAccountID = payload.Upstream.UpstreamAccountID
			}
		}
		return nil
	}
	for _, combo := range payload.ComboConfigs {
		account, err := findOrCreateProfitBoardUpstreamAccountByRemoteConfig(combo.RemoteObserver)
		if err != nil {
			return err
		}
		if account != nil {
			payload.Upstream.UpstreamAccountID = account.Id
			for index := range payload.ComboConfigs {
				if payload.ComboConfigs[index].UpstreamMode == ProfitBoardUpstreamModeWallet && payload.ComboConfigs[index].UpstreamAccountID <= 0 {
					payload.ComboConfigs[index].UpstreamAccountID = account.Id
				}
			}
			return nil
		}
	}
	return nil
}
