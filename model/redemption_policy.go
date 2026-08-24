package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const WalletRedemptionPolicyBundleOptionKey = "WalletRedemptionPolicyBundle"

var walletRedemptionPolicyMutex sync.Mutex

type WalletRedemptionPolicy struct {
	DailyCreateLimit               int `json:"daily_create_limit"`
	MinimumQuota                   int `json:"minimum_quota"`
	ActiveLimit                    int `json:"active_limit"`
	DailyQuotaLimit                int `json:"daily_quota_limit"`
	ReviewDistinctCreatorThreshold int `json:"review_distinct_creator_threshold"`
	ReviewSmallQuotaLimit          int `json:"review_small_quota_limit"`
}

var walletRedemptionPolicyOptionFields = map[string]func(*WalletRedemptionPolicy) *int{
	"WalletRedemptionDailyCreateLimit":               func(p *WalletRedemptionPolicy) *int { return &p.DailyCreateLimit },
	"WalletRedemptionMinimumQuota":                   func(p *WalletRedemptionPolicy) *int { return &p.MinimumQuota },
	"WalletRedemptionActiveLimit":                    func(p *WalletRedemptionPolicy) *int { return &p.ActiveLimit },
	"WalletRedemptionDailyQuotaLimit":                func(p *WalletRedemptionPolicy) *int { return &p.DailyQuotaLimit },
	"WalletRedemptionReviewDistinctCreatorThreshold": func(p *WalletRedemptionPolicy) *int { return &p.ReviewDistinctCreatorThreshold },
	"WalletRedemptionReviewSmallQuotaLimit":          func(p *WalletRedemptionPolicy) *int { return &p.ReviewSmallQuotaLimit },
}

func isWalletRedemptionPolicyField(key string) bool {
	_, ok := walletRedemptionPolicyOptionFields[key]
	return ok
}

func currentWalletRedemptionPolicy() WalletRedemptionPolicy {
	return WalletRedemptionPolicy{
		DailyCreateLimit:               common.WalletRedemptionDailyCreateLimit,
		MinimumQuota:                   common.WalletRedemptionMinimumQuota,
		ActiveLimit:                    common.WalletRedemptionActiveLimit,
		DailyQuotaLimit:                common.WalletRedemptionDailyQuotaLimit,
		ReviewDistinctCreatorThreshold: common.WalletRedemptionReviewDistinctCreatorThreshold,
		ReviewSmallQuotaLimit:          common.WalletRedemptionReviewSmallQuotaLimit,
	}
}

func GetWalletRedemptionPolicy() WalletRedemptionPolicy {
	walletRedemptionPolicyMutex.Lock()
	defer walletRedemptionPolicyMutex.Unlock()
	return currentWalletRedemptionPolicy()
}

func validateWalletRedemptionPolicy(policy WalletRedemptionPolicy) error {
	values := []int{
		policy.DailyCreateLimit, policy.MinimumQuota, policy.ActiveLimit,
		policy.DailyQuotaLimit, policy.ReviewDistinctCreatorThreshold,
		policy.ReviewSmallQuotaLimit,
	}
	for _, value := range values {
		if value < 0 {
			return errors.New("wallet redemption policy values must be non-negative")
		}
	}
	if policy.ReviewDistinctCreatorThreshold == 1 {
		return errors.New("redemption review distinct creator threshold must be 0 or at least 2")
	}
	if policy.ReviewDistinctCreatorThreshold > 0 && policy.ReviewSmallQuotaLimit > 0 &&
		policy.ReviewSmallQuotaLimit < policy.MinimumQuota {
		return errors.New("redemption review small quota limit cannot be below minimum code quota")
	}
	for _, value := range []int{policy.MinimumQuota, policy.DailyQuotaLimit, policy.ReviewSmallQuotaLimit} {
		if value > 0 {
			if _, err := walletRedemptionQuotaFromUnits(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func ParseWalletRedemptionPolicy(raw string) (WalletRedemptionPolicy, error) {
	policy := WalletRedemptionPolicy{}
	if err := common.UnmarshalJsonStr(raw, &policy); err != nil {
		return WalletRedemptionPolicy{}, fmt.Errorf("兑换码策略必须是 JSON 对象: %w", err)
	}
	return policy, validateWalletRedemptionPolicy(policy)
}

func validateWalletRedemptionPolicyField(key, value string) error {
	setter, ok := walletRedemptionPolicyOptionFields[key]
	if !ok {
		return nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("%s must be an integer", key)
	}
	policy := currentWalletRedemptionPolicy()
	*setter(&policy) = parsed
	return validateWalletRedemptionPolicy(policy)
}

func publishWalletRedemptionPolicy(policy WalletRedemptionPolicy) {
	common.WalletRedemptionDailyCreateLimit = policy.DailyCreateLimit
	common.WalletRedemptionMinimumQuota = policy.MinimumQuota
	common.WalletRedemptionActiveLimit = policy.ActiveLimit
	common.WalletRedemptionDailyQuotaLimit = policy.DailyQuotaLimit
	common.WalletRedemptionReviewDistinctCreatorThreshold = policy.ReviewDistinctCreatorThreshold
	common.WalletRedemptionReviewSmallQuotaLimit = policy.ReviewSmallQuotaLimit
}

func UpdateWalletRedemptionPolicyOptions(raw string) error {
	policy, err := ParseWalletRedemptionPolicy(raw)
	if err != nil {
		return err
	}
	walletRedemptionPolicyMutex.Lock()
	defer walletRedemptionPolicyMutex.Unlock()

	values := map[string]string{
		"WalletRedemptionDailyCreateLimit":               strconv.Itoa(policy.DailyCreateLimit),
		"WalletRedemptionMinimumQuota":                   strconv.Itoa(policy.MinimumQuota),
		"WalletRedemptionActiveLimit":                    strconv.Itoa(policy.ActiveLimit),
		"WalletRedemptionDailyQuotaLimit":                strconv.Itoa(policy.DailyQuotaLimit),
		"WalletRedemptionReviewDistinctCreatorThreshold": strconv.Itoa(policy.ReviewDistinctCreatorThreshold),
		"WalletRedemptionReviewSmallQuotaLimit":          strconv.Itoa(policy.ReviewSmallQuotaLimit),
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			option := Option{Key: key}
			if err := tx.FirstOrCreate(&option, Option{Key: key}).Error; err != nil {
				return err
			}
			option.Value = value
			if err := tx.Save(&option).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	for key, value := range values {
		common.OptionMap[key] = value
	}
	common.OptionMapRWMutex.Unlock()
	publishWalletRedemptionPolicy(policy)
	return nil
}
