package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const profitBoardUpstreamAccountSnapshotComboID = "wallet"

type ProfitBoardUpstreamAccount struct {
	Id                   int    `json:"id"`
	Name                 string `json:"name" gorm:"type:varchar(128);not null"`
	Remark               string `json:"remark,omitempty" gorm:"type:text"`
	AccountType          string `json:"account_type" gorm:"type:varchar(24);index;not null"`
	BaseURL              string `json:"base_url" gorm:"type:varchar(255);not null"`
	UserID               int    `json:"user_id" gorm:"index;not null"`
	AccessToken          string `json:"access_token,omitempty" gorm:"-"`
	AccessTokenMasked    string `json:"access_token_masked,omitempty" gorm:"-"`
	AccessTokenEncrypted string `json:"-" gorm:"type:text;not null"`
	Enabled              bool   `json:"enabled" gorm:"default:true"`
	CreatedAt            int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt            int64  `json:"updated_at" gorm:"bigint;index"`
}

type ProfitBoardUpstreamAccountOption struct {
	Id                        int     `json:"id"`
	Name                      string  `json:"name"`
	Remark                    string  `json:"remark,omitempty"`
	AccountType               string  `json:"account_type"`
	BaseURL                   string  `json:"base_url"`
	UserID                    int     `json:"user_id"`
	Enabled                   bool    `json:"enabled"`
	AccessTokenMasked         string  `json:"access_token_masked,omitempty"`
	Status                    string  `json:"status,omitempty"`
	ErrorMessage              string  `json:"error_message,omitempty"`
	LastSyncedAt              int64   `json:"last_synced_at"`
	LastSuccessAt             int64   `json:"last_success_at"`
	WalletQuotaUSD            float64 `json:"wallet_quota_usd"`
	WalletUsedQuotaUSD        float64 `json:"wallet_used_quota_usd"`
	SubscriptionTotalQuotaUSD float64 `json:"subscription_total_quota_usd"`
	SubscriptionUsedQuotaUSD  float64 `json:"subscription_used_quota_usd"`
	ObservedCostUSD           float64 `json:"observed_cost_usd"`
	RemoteQuotaPerUnit        float64 `json:"remote_quota_per_unit"`
	QuotaPerUnitMismatch      bool    `json:"quota_per_unit_mismatch"`
	BaselineReady             bool    `json:"baseline_ready"`
}

type profitBoardUpstreamAccountObservedAggregate struct {
	TotalCostUSD  float64
	BucketCostUSD map[int64]float64
	State         ProfitBoardUpstreamAccountOption
	Warnings      []string
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
	return account
}

func validateProfitBoardUpstreamAccount(account ProfitBoardUpstreamAccount, requireSecret bool) error {
	account = normalizeProfitBoardUpstreamAccount(account)
	if account.AccountType != ProfitBoardUpstreamAccountTypeNewAPI {
		return errors.New("当前仅支持 new-api 上游账户")
	}
	if account.Name == "" {
		return errors.New("上游账户名称不能为空")
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
	return ProfitBoardUpstreamAccountOption{
		Id:                        account.Id,
		Name:                      account.Name,
		Remark:                    account.Remark,
		AccountType:               account.AccountType,
		BaseURL:                   account.BaseURL,
		UserID:                    account.UserID,
		Enabled:                   account.Enabled,
		AccessTokenMasked:         maskProfitBoardRemoteSecret(statefulProfitBoardUpstreamToken(account)),
		Status:                    state.Status,
		ErrorMessage:              state.ErrorMessage,
		LastSyncedAt:              state.LastSyncedAt,
		LastSuccessAt:             state.LastSuccessAt,
		WalletQuotaUSD:            state.WalletQuotaUSD,
		WalletUsedQuotaUSD:        state.WalletUsedQuotaUSD,
		SubscriptionTotalQuotaUSD: state.SubscriptionTotalQuotaUSD,
		SubscriptionUsedQuotaUSD:  state.SubscriptionUsedQuotaUSD,
		ObservedCostUSD:           state.ObservedCostUSD,
		RemoteQuotaPerUnit:        state.RemoteQuotaPerUnit,
		QuotaPerUnitMismatch:      state.QuotaPerUnitMismatch,
		BaselineReady:             state.BaselineReady,
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
		return nil, errors.New("无效的上游账户")
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
	sort.Slice(options, func(i, j int) bool {
		if options[i].Enabled != options[j].Enabled {
			return options[i].Enabled
		}
		return options[i].Name < options[j].Name
	})
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
		return nil, errors.New("上游 access token 不能为空")
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
	if err := DB.Delete(account).Error; err != nil {
		return err
	}
	return nil
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
	state := buildProfitBoardRemoteObserverState(signature, batch, config, latestAny, latestSuccess, 0)
	option := buildProfitBoardUpstreamAccountOption(*account, state)
	return &option, nil
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
	snapshots, err := listProfitBoardRemoteSuccessSnapshots(signature, profitBoardUpstreamAccountSnapshotComboID, configHash, startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	aggregate := &profitBoardUpstreamAccountObservedAggregate{
		BucketCostUSD: make(map[int64]float64),
		Warnings:      make([]string, 0),
	}
	totalCostQuota := int64(0)
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
			bucketTimestamp, _ := buildProfitBoardBucket(snapshots[index].SyncedAt, granularity, customIntervalMinutes)
			aggregate.BucketCostUSD[bucketTimestamp] += float64(deltaQuota) / common.QuotaPerUnit
		}
	}
	aggregate.TotalCostUSD = roundProfitBoardAmount(float64(totalCostQuota) / common.QuotaPerUnit)
	state := buildProfitBoardRemoteObserverState(signature, batch, config, latestAny, latestSuccess, aggregate.TotalCostUSD)
	aggregate.State = buildProfitBoardUpstreamAccountOption(*account, state)
	if state.QuotaPerUnitMismatch {
		aggregate.Warnings = append(aggregate.Warnings, fmt.Sprintf("%s 的远端 quota_per_unit 与本站不同，当前仍按本站额度口径换算金额", account.Name))
	}
	if state.Status == profitBoardRemoteObserverStatusFailed && state.ErrorMessage != "" {
		aggregate.Warnings = append(aggregate.Warnings, fmt.Sprintf("%s 同步失败：%s", account.Name, state.ErrorMessage))
	}
	aggregate.Warnings = uniqueProfitBoardWarnings(aggregate.Warnings)
	return aggregate, nil
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
	if payload.Upstream.UpstreamMode != ProfitBoardUpstreamModeWallet {
		switch payload.Upstream.CostSource {
		case ProfitBoardCostSourceReturnedFirst, ProfitBoardCostSourceReturnedOnly:
			payload.Upstream.UpstreamMode = ProfitBoardUpstreamModeWallet
		}
	}
	if payload.Upstream.UpstreamMode != ProfitBoardUpstreamModeWallet || payload.Upstream.UpstreamAccountID > 0 {
		return nil
	}
	for _, combo := range payload.ComboConfigs {
		account, err := findOrCreateProfitBoardUpstreamAccountByRemoteConfig(combo.RemoteObserver)
		if err != nil {
			return err
		}
		if account != nil {
			payload.Upstream.UpstreamAccountID = account.Id
			return nil
		}
	}
	return nil
}
