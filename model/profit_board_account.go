package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Sentinel errors for i18n translation (upstream account)
var (
	ErrProfitBoardAccountTypeUnsupported = errors.New("profit_board:account_type_unsupported")
	ErrProfitBoardAccountNameEmpty       = errors.New("profit_board:account_name_empty")
	ErrProfitBoardAccountInvalid         = errors.New("profit_board:account_invalid")
	ErrProfitBoardAccountTokenEmpty      = errors.New("profit_board:account_token_empty")
	ErrProfitBoardAccountEmailEmpty      = errors.New("profit_board:account_email_empty")
	ErrProfitBoardAccountPasswordEmpty   = errors.New("profit_board:account_password_empty")
)

const profitBoardUpstreamAccountSnapshotComboID = "wallet"

var profitBoardLowBalanceAutoDisableOnce sync.Once

type ProfitBoardUpstreamAccount struct {
	Id                              int     `json:"id"`
	Name                            string  `json:"name" gorm:"type:varchar(128);not null"`
	Remark                          string  `json:"remark,omitempty" gorm:"type:text"`
	AccountType                     string  `json:"account_type" gorm:"type:varchar(24);index;not null"`
	BaseURL                         string  `json:"base_url" gorm:"type:varchar(255);not null"`
	UserID                          int     `json:"user_id" gorm:"index;not null"`
	Email                           string  `json:"email,omitempty" gorm:"type:varchar(255);index"`
	AccessToken                     string  `json:"access_token,omitempty" gorm:"-"`
	AccessTokenMasked               string  `json:"access_token_masked,omitempty" gorm:"-"`
	AccessTokenEncrypted            string  `json:"-" gorm:"type:text;not null"`
	Password                        string  `json:"password,omitempty" gorm:"-"`
	PasswordMasked                  string  `json:"password_masked,omitempty" gorm:"-"`
	PasswordEncrypted               string  `json:"-" gorm:"type:text"`
	Enabled                         bool    `json:"enabled" gorm:"default:true"`
	ResourceDisplayMode             string  `json:"resource_display_mode" gorm:"type:varchar(24);default:both"`
	LowBalanceThresholdUSD          float64 `json:"low_balance_threshold_usd" gorm:"type:decimal(18,6);default:0"`
	LowBalanceAutoDisableEnabled    bool    `json:"low_balance_auto_disable_enabled" gorm:"default:false"`
	LowBalanceCheckIntervalSeconds  int     `json:"low_balance_check_interval_seconds" gorm:"type:int;default:300"`
	LowBalanceLastCheckedAt         int64   `json:"low_balance_last_checked_at" gorm:"bigint;default:0"`
	LowBalanceLastAutoDisabledAt    int64   `json:"low_balance_last_auto_disabled_at" gorm:"bigint;default:0"`
	LowBalanceLastAutoDisabledCount int     `json:"low_balance_last_auto_disabled_count" gorm:"type:int;default:0"`
	CreatedAt                       int64   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt                       int64   `json:"updated_at" gorm:"bigint;index"`
}

type ProfitBoardUpstreamAccountOption struct {
	Id                              int     `json:"id"`
	Name                            string  `json:"name"`
	Remark                          string  `json:"remark,omitempty"`
	AccountType                     string  `json:"account_type"`
	BaseURL                         string  `json:"base_url"`
	UserID                          int     `json:"user_id"`
	Email                           string  `json:"email,omitempty"`
	Enabled                         bool    `json:"enabled"`
	ResourceDisplayMode             string  `json:"resource_display_mode"`
	AccessTokenMasked               string  `json:"access_token_masked,omitempty"`
	PasswordMasked                  string  `json:"password_masked,omitempty"`
	Status                          string  `json:"status,omitempty"`
	ErrorMessage                    string  `json:"error_message,omitempty"`
	LastSyncedAt                    int64   `json:"last_synced_at"`
	LastSuccessAt                   int64   `json:"last_success_at"`
	WalletBalanceUSD                float64 `json:"wallet_balance_usd"`
	WalletQuotaUSD                  float64 `json:"wallet_quota_usd"`
	WalletUsedTotalUSD              float64 `json:"wallet_used_total_usd"`
	WalletUsedQuotaUSD              float64 `json:"wallet_used_quota_usd"`
	PeriodUsedUSD                   float64 `json:"period_used_usd"`
	SubscriptionRemainingUSD        float64 `json:"subscription_remaining_quota_usd"`
	SubscriptionTotalQuotaUSD       float64 `json:"subscription_total_quota_usd"`
	SubscriptionUsedQuotaUSD        float64 `json:"subscription_used_quota_usd"`
	SubscriptionCount               int     `json:"subscription_count"`
	SubscriptionEarliestExpireAt    int64   `json:"subscription_earliest_expire_at"`
	HasSubscriptionData             bool    `json:"has_subscription_data"`
	SubscriptionHasUnlimited        bool    `json:"subscription_has_unlimited"`
	ObservedCostUSD                 float64 `json:"observed_cost_usd"`
	RemoteQuotaPerUnit              float64 `json:"remote_quota_per_unit"`
	QuotaPerUnitMismatch            bool    `json:"quota_per_unit_mismatch"`
	LowBalanceThresholdUSD          float64 `json:"low_balance_threshold_usd"`
	LowBalanceAlert                 bool    `json:"low_balance_alert"`
	LowBalanceAutoDisableEnabled    bool    `json:"low_balance_auto_disable_enabled"`
	LowBalanceCheckIntervalSeconds  int     `json:"low_balance_check_interval_seconds"`
	LowBalanceLastCheckedAt         int64   `json:"low_balance_last_checked_at"`
	LowBalanceLastAutoDisabledAt    int64   `json:"low_balance_last_auto_disabled_at"`
	LowBalanceLastAutoDisabledCount int     `json:"low_balance_last_auto_disabled_count"`
	BaselineReady                   bool    `json:"baseline_ready"`
	SnapshotCount                   int     `json:"snapshot_count"`
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
	account.Email = strings.TrimSpace(account.Email)
	account.AccessToken = strings.TrimSpace(account.AccessToken)
	account.Password = strings.TrimSpace(account.Password)
	if account.UserID < 0 {
		account.UserID = 0
	}
	if account.LowBalanceThresholdUSD < 0 {
		account.LowBalanceThresholdUSD = 0
	}
	if account.LowBalanceCheckIntervalSeconds < 60 {
		account.LowBalanceCheckIntervalSeconds = 300
	}
	account.ResourceDisplayMode = normalizeProfitBoardUpstreamAccountResourceDisplayMode(account.ResourceDisplayMode)
	if account.AccountType == ProfitBoardUpstreamAccountTypeSub2API {
		account.UserID = 0
		account.ResourceDisplayMode = ProfitBoardResourceDisplayWallet
	}
	return account
}

func validateProfitBoardUpstreamAccount(account ProfitBoardUpstreamAccount, requireSecret bool) error {
	account = normalizeProfitBoardUpstreamAccount(account)
	if account.Name == "" {
		return ErrProfitBoardAccountNameEmpty
	}
	switch account.AccountType {
	case ProfitBoardUpstreamAccountTypeNewAPI:
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
	case ProfitBoardUpstreamAccountTypeSub2API:
		if account.Email == "" {
			return ErrProfitBoardAccountEmailEmpty
		}
		config := ProfitBoardRemoteObserverConfig{
			Enabled:           true,
			AccountType:       ProfitBoardUpstreamAccountTypeSub2API,
			BaseURL:           account.BaseURL,
			Email:             account.Email,
			Password:          account.Password,
			PasswordEncrypted: account.PasswordEncrypted,
		}
		if !requireSecret {
			config.Password = ""
		}
		if err := validateProfitBoardRemoteObserverConfig(config); err != nil {
			if errors.Is(err, ErrProfitBoardRemoteMissingPassword) {
				return ErrProfitBoardAccountPasswordEmpty
			}
			return err
		}
	default:
		return ErrProfitBoardAccountTypeUnsupported
	}
	return nil
}

func (a ProfitBoardUpstreamAccount) remoteObserverConfig() ProfitBoardRemoteObserverConfig {
	a = normalizeProfitBoardUpstreamAccount(a)
	config := ProfitBoardRemoteObserverConfig{
		Enabled:           a.Enabled,
		AccountType:       a.AccountType,
		BaseURL:           a.BaseURL,
		UserID:            a.UserID,
		Email:             a.Email,
		Password:          a.Password,
		PasswordEncrypted: a.PasswordEncrypted,
	}
	if a.AccountType == ProfitBoardUpstreamAccountTypeNewAPI {
		config.AccessToken = a.AccessToken
		config.AccessTokenEncrypted = a.AccessTokenEncrypted
	}
	return normalizeProfitBoardRemoteObserverConfig(config)
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
		Id:                              account.Id,
		Name:                            account.Name,
		Remark:                          account.Remark,
		AccountType:                     account.AccountType,
		BaseURL:                         account.BaseURL,
		UserID:                          account.UserID,
		Email:                           account.Email,
		Enabled:                         account.Enabled,
		ResourceDisplayMode:             account.ResourceDisplayMode,
		AccessTokenMasked:               maskProfitBoardRemoteSecret(statefulProfitBoardUpstreamToken(account)),
		PasswordMasked:                  maskProfitBoardRemoteSecret(statefulProfitBoardUpstreamPassword(account)),
		Status:                          state.Status,
		ErrorMessage:                    state.ErrorMessage,
		LastSyncedAt:                    state.LastSyncedAt,
		LastSuccessAt:                   state.LastSuccessAt,
		WalletBalanceUSD:                state.WalletBalanceUSD,
		WalletQuotaUSD:                  state.WalletQuotaUSD,
		WalletUsedTotalUSD:              state.WalletUsedTotalUSD,
		WalletUsedQuotaUSD:              state.WalletUsedQuotaUSD,
		PeriodUsedUSD:                   state.PeriodUsedUSD,
		SubscriptionRemainingUSD:        state.SubscriptionRemainingUSD,
		SubscriptionTotalQuotaUSD:       state.SubscriptionTotalQuotaUSD,
		SubscriptionUsedQuotaUSD:        state.SubscriptionUsedQuotaUSD,
		SubscriptionCount:               state.SubscriptionCount,
		SubscriptionEarliestExpireAt:    state.SubscriptionEarliestExpireAt,
		HasSubscriptionData:             state.HasSubscriptionData,
		SubscriptionHasUnlimited:        state.SubscriptionHasUnlimited,
		ObservedCostUSD:                 state.PeriodUsedUSD,
		RemoteQuotaPerUnit:              state.RemoteQuotaPerUnit,
		QuotaPerUnitMismatch:            state.QuotaPerUnitMismatch,
		LowBalanceThresholdUSD:          threshold,
		LowBalanceAlert:                 lowBalanceAlert,
		LowBalanceAutoDisableEnabled:    account.LowBalanceAutoDisableEnabled,
		LowBalanceCheckIntervalSeconds:  account.LowBalanceCheckIntervalSeconds,
		LowBalanceLastCheckedAt:         account.LowBalanceLastCheckedAt,
		LowBalanceLastAutoDisabledAt:    account.LowBalanceLastAutoDisabledAt,
		LowBalanceLastAutoDisabledCount: account.LowBalanceLastAutoDisabledCount,
		BaselineReady:                   state.BaselineReady,
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

func statefulProfitBoardUpstreamPassword(account ProfitBoardUpstreamAccount) string {
	if account.Password != "" {
		return account.Password
	}
	if account.PasswordEncrypted == "" {
		return ""
	}
	decrypted, err := decryptProfitBoardRemoteSecret(account.PasswordEncrypted)
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
		existing = normalizeProfitBoardUpstreamAccount(existing)
		if account.AccessTokenEncrypted == "" {
			account.AccessTokenEncrypted = existing.AccessTokenEncrypted
		}
		if account.PasswordEncrypted == "" {
			account.PasswordEncrypted = existing.PasswordEncrypted
		}
	}
	requireSecret := account.Id == 0 ||
		(account.AccountType == ProfitBoardUpstreamAccountTypeNewAPI && (account.AccessToken != "" || account.AccessTokenEncrypted == "")) ||
		(account.AccountType == ProfitBoardUpstreamAccountTypeSub2API && (account.Password != "" || account.PasswordEncrypted == ""))
	if err := validateProfitBoardUpstreamAccount(account, requireSecret); err != nil {
		return nil, err
	}
	if err := prepareProfitBoardUpstreamAccountSecrets(&account, existing); err != nil {
		return nil, err
	}
	account.AccessToken = ""
	account.Password = ""
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
		existing.Email = account.Email
		existing.AccessTokenEncrypted = account.AccessTokenEncrypted
		existing.PasswordEncrypted = account.PasswordEncrypted
		existing.Enabled = account.Enabled
		existing.ResourceDisplayMode = account.ResourceDisplayMode
		existing.LowBalanceThresholdUSD = account.LowBalanceThresholdUSD
		existing.LowBalanceAutoDisableEnabled = account.LowBalanceAutoDisableEnabled
		existing.LowBalanceCheckIntervalSeconds = account.LowBalanceCheckIntervalSeconds
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

func prepareProfitBoardUpstreamAccountSecrets(account *ProfitBoardUpstreamAccount, existing ProfitBoardUpstreamAccount) error {
	switch account.AccountType {
	case ProfitBoardUpstreamAccountTypeNewAPI:
		account.PasswordEncrypted = ""
		account.Email = ""
		switch {
		case account.AccessToken != "":
			encrypted, err := encryptProfitBoardRemoteSecret(account.AccessToken)
			if err != nil {
				return err
			}
			account.AccessTokenEncrypted = encrypted
		case existing.AccountType == ProfitBoardUpstreamAccountTypeNewAPI && existing.AccessTokenEncrypted != "":
			account.AccessTokenEncrypted = existing.AccessTokenEncrypted
		default:
			return ErrProfitBoardAccountTokenEmpty
		}
	case ProfitBoardUpstreamAccountTypeSub2API:
		account.AccessTokenEncrypted = ""
		account.UserID = 0
		account.ResourceDisplayMode = ProfitBoardResourceDisplayWallet
		switch {
		case account.Password != "":
			encrypted, err := encryptProfitBoardRemoteSecret(account.Password)
			if err != nil {
				return err
			}
			account.PasswordEncrypted = encrypted
		case existing.AccountType == ProfitBoardUpstreamAccountTypeSub2API && existing.PasswordEncrypted != "":
			account.PasswordEncrypted = existing.PasswordEncrypted
		default:
			return ErrProfitBoardAccountPasswordEmpty
		}
	default:
		return ErrProfitBoardAccountTypeUnsupported
	}
	return nil
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

func syncProfitBoardUpstreamAccountForLowBalanceCheck(account ProfitBoardUpstreamAccount) (*ProfitBoardUpstreamAccountOption, error) {
	account = normalizeProfitBoardUpstreamAccount(account)
	config := account.remoteObserverConfig()
	config.Enabled = true
	signature := profitBoardUpstreamAccountSnapshotSignature(account.Id)
	batch := profitBoardUpstreamAccountSnapshotBatch(account)
	latestAny, latestSuccess, err := syncProfitBoardRemoteObserverSnapshotWithMinInterval(
		signature,
		batch,
		config,
		false,
		int64(account.LowBalanceCheckIntervalSeconds),
	)
	if err != nil {
		return nil, err
	}
	state := buildProfitBoardRemoteObserverState(signature, batch, config, latestAny, latestSuccess, 0)
	option := buildProfitBoardUpstreamAccountOption(account, state)
	return &option, nil
}

type profitBoardLowBalanceBoundChannel struct {
	Id             int
	Name           string
	Type           int
	Status         int
	MatchedBatches []string
}

type ProfitBoardLowBalanceAutoDisableResult struct {
	AccountID         int     `json:"account_id"`
	AccountName       string  `json:"account_name"`
	WalletBalanceUSD  float64 `json:"wallet_balance_usd"`
	ThresholdUSD      float64 `json:"threshold_usd"`
	DisabledCount     int     `json:"disabled_count"`
	CheckedAt         int64   `json:"checked_at"`
	Skipped           bool    `json:"skipped"`
	SkipReason        string  `json:"skip_reason,omitempty"`
	MatchedChannelIDs []int   `json:"matched_channel_ids,omitempty"`
}

func accountDueForLowBalanceCheck(account ProfitBoardUpstreamAccount, now int64, force bool) bool {
	if force {
		return true
	}
	interval := account.LowBalanceCheckIntervalSeconds
	if interval < 60 {
		interval = 300
	}
	return account.LowBalanceLastCheckedAt <= 0 || now-account.LowBalanceLastCheckedAt >= int64(interval)
}

func autoDisableProfitBoardLowBalanceChannel(channelId int, reason string) (bool, error) {
	channel, err := GetChannelById(channelId, true)
	if err != nil {
		return false, err
	}
	if channel.Status != common.ChannelStatusEnabled {
		return false, nil
	}
	info := channel.GetOtherInfo()
	info["status_reason"] = reason
	info["status_time"] = common.GetTimestamp()
	channel.SetOtherInfo(info)
	channel.Status = common.ChannelStatusAutoDisabled
	if err := channel.SaveWithoutKey(); err != nil {
		return false, err
	}
	if err := UpdateAbilityStatus(channel.Id, false); err != nil {
		common.SysLog(fmt.Sprintf("failed to update ability status: channel_id=%d, error=%v", channel.Id, err))
	}
	if common.MemoryCacheEnabled {
		CacheUpdateChannelStatus(channel.Id, common.ChannelStatusAutoDisabled)
	}
	return true, nil
}

func collectProfitBoardLowBalanceBoundChannels(accountID int) (map[int]*profitBoardLowBalanceBoundChannel, error) {
	records := make([]ProfitBoardConfig, 0)
	if err := DB.Find(&records).Error; err != nil {
		return nil, err
	}

	result := make(map[int]*profitBoardLowBalanceBoundChannel)
	for _, record := range records {
		payload, err := payloadFromProfitBoardConfigRecord(record)
		if err != nil {
			return nil, err
		}
		if payload == nil || len(payload.Batches) == 0 {
			continue
		}
		comboConfigs := normalizeProfitBoardComboConfigs(payload.Batches, payload.ComboConfigs, payload.SharedSite, payload.Site, payload.Upstream)
		comboByID := make(map[string]ProfitBoardComboPricingConfig, len(comboConfigs))
		for _, combo := range comboConfigs {
			comboByID[combo.ComboId] = combo
		}
		for _, batch := range payload.Batches {
			combo := comboByID[batch.Id]
			if combo.UpstreamMode != ProfitBoardUpstreamModeWallet || combo.UpstreamAccountID != accountID {
				continue
			}
			resolved, _, err := resolveProfitBoardBatch(batch, false)
			if err != nil {
				return nil, err
			}
			for _, channel := range resolved.ResolvedChannels {
				current := result[channel.Id]
				if current == nil {
					current = &profitBoardLowBalanceBoundChannel{
						Id:             channel.Id,
						Name:           channel.Name,
						Status:         channel.Status,
						MatchedBatches: make([]string, 0, 1),
					}
					result[channel.Id] = current
				}
				current.MatchedBatches = append(current.MatchedBatches, resolved.Name)
			}
		}
	}

	if len(result) == 0 {
		return result, nil
	}
	ids := make([]int, 0, len(result))
	for id := range result {
		ids = append(ids, id)
	}
	channels := make([]Channel, 0, len(ids))
	if err := DB.Select("id, name, type, status, auto_ban, channel_info").Where("id IN ?", ids).Find(&channels).Error; err != nil {
		return nil, err
	}
	for _, channel := range channels {
		if current := result[channel.Id]; current != nil {
			current.Name = channel.Name
			current.Type = channel.Type
			current.Status = channel.Status
		}
	}
	return result, nil
}

func checkProfitBoardUpstreamAccountLowBalance(accountID int, force bool) (*ProfitBoardLowBalanceAutoDisableResult, error) {
	account, err := getProfitBoardUpstreamAccountByID(accountID)
	if err != nil {
		return nil, err
	}
	normalizedAccount := normalizeProfitBoardUpstreamAccount(*account)
	account = &normalizedAccount
	now := common.GetTimestamp()
	result := &ProfitBoardLowBalanceAutoDisableResult{
		AccountID:         account.Id,
		AccountName:       account.Name,
		ThresholdUSD:      roundProfitBoardAmount(account.LowBalanceThresholdUSD),
		CheckedAt:         now,
		Skipped:           true,
		MatchedChannelIDs: make([]int, 0),
	}
	if !account.Enabled {
		result.SkipReason = "account_disabled"
		return result, nil
	}
	if !account.LowBalanceAutoDisableEnabled {
		result.SkipReason = "auto_disable_disabled"
		return result, nil
	}
	if account.LowBalanceThresholdUSD <= 0 {
		result.SkipReason = "threshold_disabled"
		return result, nil
	}
	if !accountDueForLowBalanceCheck(*account, now, force) {
		result.SkipReason = "not_due"
		return result, nil
	}

	option, err := syncProfitBoardUpstreamAccountForLowBalanceCheck(*account)
	if err != nil {
		_ = DB.Model(&ProfitBoardUpstreamAccount{}).Where("id = ?", account.Id).Update("low_balance_last_checked_at", now).Error
		return nil, err
	}
	result.WalletBalanceUSD = option.WalletBalanceUSD
	result.Skipped = false
	threshold := roundProfitBoardAmount(account.LowBalanceThresholdUSD)
	if threshold <= 0 || option.WalletBalanceUSD > threshold {
		updates := map[string]any{
			"low_balance_last_checked_at":          now,
			"low_balance_last_auto_disabled_count": 0,
		}
		if err := DB.Model(&ProfitBoardUpstreamAccount{}).Where("id = ?", account.Id).Updates(updates).Error; err != nil {
			return nil, err
		}
		return result, nil
	}

	channels, err := collectProfitBoardLowBalanceBoundChannels(account.Id)
	if err != nil {
		return nil, err
	}
	reason := fmt.Sprintf("收益看板上游账户「%s」钱包余额 $%.6f 低于阈值 $%.6f", account.Name, option.WalletBalanceUSD, threshold)
	disabledCount := 0
	for _, channel := range channels {
		result.MatchedChannelIDs = append(result.MatchedChannelIDs, channel.Id)
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}
		disabled, disableErr := autoDisableProfitBoardLowBalanceChannel(channel.Id, reason)
		if disableErr != nil {
			return nil, disableErr
		}
		if disabled {
			disabledCount++
		}
	}
	sort.Ints(result.MatchedChannelIDs)
	result.DisabledCount = disabledCount
	updates := map[string]any{
		"low_balance_last_checked_at":          now,
		"low_balance_last_auto_disabled_count": disabledCount,
	}
	if disabledCount > 0 {
		updates["low_balance_last_auto_disabled_at"] = now
	}
	if err := DB.Model(&ProfitBoardUpstreamAccount{}).Where("id = ?", account.Id).Updates(updates).Error; err != nil {
		return nil, err
	}
	if disabledCount > 0 {
		common.SysLog(fmt.Sprintf("profit board low balance disabled %d channels for account %d", disabledCount, account.Id))
	}
	return result, nil
}

func RunProfitBoardLowBalanceAutoDisableCheck(force bool) ([]ProfitBoardLowBalanceAutoDisableResult, error) {
	accounts, err := listProfitBoardUpstreamAccounts()
	if err != nil {
		return nil, err
	}
	results := make([]ProfitBoardLowBalanceAutoDisableResult, 0, len(accounts))
	for _, account := range accounts {
		if !account.LowBalanceAutoDisableEnabled {
			continue
		}
		result, checkErr := checkProfitBoardUpstreamAccountLowBalance(account.Id, force)
		if checkErr != nil {
			return results, checkErr
		}
		if result != nil {
			results = append(results, *result)
		}
	}
	return results, nil
}

func StartProfitBoardLowBalanceAutoDisableTask() {
	profitBoardLowBalanceAutoDisableOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		go func() {
			common.SysLog("profit board low balance auto-disable task started")
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()

			for range ticker.C {
				if _, err := RunProfitBoardLowBalanceAutoDisableCheck(false); err != nil {
					common.SysError("profit board low balance auto-disable failed: " + err.Error())
				}
			}
		}()
	})
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
			deltaQuota, warnings := profitBoardRemoteSnapshotDeltaForConfig(config, snapshots[index-1], snapshots[index])
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
				if config.AccountType == ProfitBoardUpstreamAccountTypeSub2API {
					if snapshots[index].WalletQuota <= snapshots[index-1].WalletQuota {
						allRolledBack = false
						break
					}
					continue
				}
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
			deltaQuota, deltaWarnings := profitBoardRemoteSnapshotDeltaForConfig(config, snapshots[index-1], snapshots[index])
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
