package model

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const tagPolicyCacheTTL = 60 * time.Second

type ChannelTagPolicy struct {
	Id          int    `gorm:"primaryKey" json:"id"`
	Tag         string `gorm:"uniqueIndex;type:varchar(255);not null" json:"tag"`
	QuotaPolicy string `gorm:"type:text" json:"quota_policy"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type tagPolicyCacheEntry struct {
	policy dto.QuotaPolicy
	found  bool
	expire time.Time
}

var tagPolicyCache sync.Map

func InvalidateTagPolicyCache(tag string) {
	tagPolicyCache.Delete(strings.TrimSpace(tag))
}

func GetTagPolicy(tag string) (dto.QuotaPolicy, bool, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return dto.QuotaPolicy{}, false, nil
	}
	if value, ok := tagPolicyCache.Load(tag); ok {
		entry := value.(tagPolicyCacheEntry)
		if time.Now().Before(entry.expire) {
			return entry.policy, entry.found, nil
		}
		tagPolicyCache.Delete(tag)
	}

	var row ChannelTagPolicy
	err := DB.Where("tag = ?", tag).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		tagPolicyCache.Store(tag, tagPolicyCacheEntry{found: false, expire: time.Now().Add(tagPolicyCacheTTL)})
		return dto.QuotaPolicy{}, false, nil
	}
	if err != nil {
		return dto.QuotaPolicy{}, false, err
	}
	var policy dto.QuotaPolicy
	if row.QuotaPolicy != "" {
		if err := common.UnmarshalJsonStr(row.QuotaPolicy, &policy); err != nil {
			return dto.QuotaPolicy{}, false, err
		}
	}
	tagPolicyCache.Store(tag, tagPolicyCacheEntry{policy: policy, found: true, expire: time.Now().Add(tagPolicyCacheTTL)})
	return policy, true, nil
}

func normalizeTagQuotaPolicyAnchor(tag string, policy dto.QuotaPolicy) dto.QuotaPolicy {
	if !policy.Enabled {
		policy.AnchorTime = 0
		return policy
	}
	oldPolicy, found, err := GetTagPolicy(tag)
	if err == nil && found && quotaPolicyAnchorReusable(oldPolicy, policy) {
		policy.AnchorTime = oldPolicy.AnchorTime
		return policy
	}
	policy.AnchorTime = common.GetTimestamp()
	return policy
}

func quotaPolicyAnchorReusable(oldPolicy, newPolicy dto.QuotaPolicy) bool {
	return oldPolicy.Enabled &&
		oldPolicy.AnchorTime > 0 &&
		oldPolicy.Period == newPolicy.Period &&
		oldPolicy.QuotaLimit == newPolicy.QuotaLimit &&
		oldPolicy.CountLimit == newPolicy.CountLimit
}

func UpsertTagPolicy(tag string, policy dto.QuotaPolicy) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return errors.New("tag cannot be empty")
	}
	if err := policy.Validate(); err != nil {
		return err
	}
	policy = normalizeTagQuotaPolicyAnchor(tag, policy)
	data, err := common.Marshal(policy)
	if err != nil {
		return err
	}
	now := common.GetTimestamp()
	row := ChannelTagPolicy{Tag: tag, QuotaPolicy: string(data), CreatedAt: now, UpdatedAt: now}
	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tag"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quota_policy": string(data),
			"updated_at":   now,
		}),
	}).Create(&row).Error
	InvalidateTagPolicyCache(tag)
	return err
}

func DeleteTagPolicy(tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return errors.New("tag cannot be empty")
	}
	err := DB.Where("tag = ?", tag).Delete(&ChannelTagPolicy{}).Error
	InvalidateTagPolicyCache(tag)
	return err
}
