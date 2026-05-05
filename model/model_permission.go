package model

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	ModelPermissionScopeAll = iota
	ModelPermissionScopeAdminOnly
	ModelPermissionScopeCommonOnly
	ModelPermissionScopeNone
)

const modelPermissionCacheTTL = time.Minute

type ModelPermission struct {
	Id              int    `json:"id"`
	ModelName       string `json:"model_name" gorm:"size:128;not null;index"`
	NameRule        int    `json:"name_rule" gorm:"default:0"`
	VisibilityScope int    `json:"visibility_scope" gorm:"default:0"`
	CallScope       int    `json:"call_scope" gorm:"default:0"`
	CreatedTime     int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime     int64  `json:"updated_time" gorm:"bigint"`
}

type ModelAccess struct {
	VisibilityScope int
	CallScope       int
	HasRule         bool
}

type modelPermissionRule struct {
	Id              int
	ModelName       string
	NameRule        int
	VisibilityScope int
	CallScope       int
}

var modelPermissionCache = struct {
	sync.RWMutex
	rules    []modelPermissionRule
	loadedAt time.Time
}{}

func NormalizeModelNameRule(rule int) int {
	switch rule {
	case NameRuleExact, NameRulePrefix, NameRuleSuffix, NameRuleContains:
		return rule
	default:
		return NameRuleExact
	}
}

func NormalizeModelPermissionScope(scope int) int {
	switch scope {
	case ModelPermissionScopeAll, ModelPermissionScopeAdminOnly, ModelPermissionScopeCommonOnly, ModelPermissionScopeNone:
		return scope
	default:
		return ModelPermissionScopeAll
	}
}

func (permission *ModelPermission) normalize() {
	permission.ModelName = strings.TrimSpace(permission.ModelName)
	permission.NameRule = NormalizeModelNameRule(permission.NameRule)
	permission.VisibilityScope = NormalizeModelPermissionScope(permission.VisibilityScope)
	permission.CallScope = NormalizeModelPermissionScope(permission.CallScope)
}

func (permission *ModelPermission) Insert() error {
	permission.normalize()
	now := common.GetTimestamp()
	permission.CreatedTime = now
	permission.UpdatedTime = now
	if err := DB.Create(permission).Error; err != nil {
		return err
	}
	InvalidateModelPermissionCache()
	return nil
}

func (permission *ModelPermission) Update() error {
	permission.normalize()
	permission.UpdatedTime = common.GetTimestamp()
	err := DB.Model(&ModelPermission{}).Where("id = ?", permission.Id).
		Select("model_name", "name_rule", "visibility_scope", "call_scope", "updated_time").
		Updates(permission).Error
	if err != nil {
		return err
	}
	InvalidateModelPermissionCache()
	return nil
}

func DeleteModelPermission(id int) error {
	if err := DB.Delete(&ModelPermission{}, id).Error; err != nil {
		return err
	}
	InvalidateModelPermissionCache()
	return nil
}

func IsModelPermissionDuplicated(id int, modelName string, nameRule int) (bool, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return false, nil
	}
	var count int64
	err := DB.Model(&ModelPermission{}).
		Where("model_name = ? AND name_rule = ? AND id <> ?", modelName, NormalizeModelNameRule(nameRule), id).
		Count(&count).Error
	return count > 0, err
}

func GetAllModelPermissions() ([]ModelPermission, error) {
	permissions := make([]ModelPermission, 0)
	err := DB.Order("id ASC").Find(&permissions).Error
	return permissions, err
}

func InvalidateModelPermissionCache() {
	modelPermissionCache.Lock()
	modelPermissionCache.rules = nil
	modelPermissionCache.loadedAt = time.Time{}
	modelPermissionCache.Unlock()
}

func ensureModelPermissionCache() {
	modelPermissionCache.RLock()
	if time.Since(modelPermissionCache.loadedAt) < modelPermissionCacheTTL && modelPermissionCache.rules != nil {
		modelPermissionCache.RUnlock()
		return
	}
	modelPermissionCache.RUnlock()

	modelPermissionCache.Lock()
	defer modelPermissionCache.Unlock()
	if time.Since(modelPermissionCache.loadedAt) < modelPermissionCacheTTL && modelPermissionCache.rules != nil {
		return
	}

	rules := make([]modelPermissionRule, 0)
	if DB != nil {
		err := DB.Model(&ModelPermission{}).
			Select("id", "model_name", "name_rule", "visibility_scope", "call_scope").
			Order("id ASC").
			Find(&rules).Error
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to load model permission rules: %v", err))
			rules = make([]modelPermissionRule, 0)
		}
	}
	modelPermissionCache.rules = rules
	modelPermissionCache.loadedAt = time.Now()
}

func betterModelPermissionRule(candidate modelPermissionRule, current *modelPermissionRule) bool {
	if current == nil {
		return true
	}
	if len(candidate.ModelName) != len(current.ModelName) {
		return len(candidate.ModelName) > len(current.ModelName)
	}
	return candidate.Id < current.Id
}

func matchModelPermissionRule(modelName string) (modelPermissionRule, bool) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return modelPermissionRule{}, false
	}

	ensureModelPermissionCache()

	modelPermissionCache.RLock()
	defer modelPermissionCache.RUnlock()

	var exact *modelPermissionRule
	var prefix *modelPermissionRule
	var suffix *modelPermissionRule
	var contains *modelPermissionRule

	for i := range modelPermissionCache.rules {
		rule := modelPermissionCache.rules[i]
		rule.ModelName = strings.TrimSpace(rule.ModelName)
		if rule.ModelName == "" {
			continue
		}
		switch NormalizeModelNameRule(rule.NameRule) {
		case NameRuleExact:
			if rule.ModelName == modelName && betterModelPermissionRule(rule, exact) {
				r := rule
				exact = &r
			}
		case NameRulePrefix:
			if strings.HasPrefix(modelName, rule.ModelName) && betterModelPermissionRule(rule, prefix) {
				r := rule
				prefix = &r
			}
		case NameRuleSuffix:
			if strings.HasSuffix(modelName, rule.ModelName) && betterModelPermissionRule(rule, suffix) {
				r := rule
				suffix = &r
			}
		case NameRuleContains:
			if strings.Contains(modelName, rule.ModelName) && betterModelPermissionRule(rule, contains) {
				r := rule
				contains = &r
			}
		}
	}

	switch {
	case exact != nil:
		return *exact, true
	case prefix != nil:
		return *prefix, true
	case suffix != nil:
		return *suffix, true
	case contains != nil:
		return *contains, true
	default:
		return modelPermissionRule{}, false
	}
}

func GetModelAccess(modelName string) ModelAccess {
	rule, ok := matchModelPermissionRule(modelName)
	if !ok {
		return ModelAccess{
			VisibilityScope: ModelPermissionScopeAll,
			CallScope:       ModelPermissionScopeAll,
			HasRule:         false,
		}
	}
	return ModelAccess{
		VisibilityScope: NormalizeModelPermissionScope(rule.VisibilityScope),
		CallScope:       NormalizeModelPermissionScope(rule.CallScope),
		HasRule:         true,
	}
}

func permissionScopeAllowsVisibility(scope int, role int) bool {
	switch NormalizeModelPermissionScope(scope) {
	case ModelPermissionScopeAll:
		return true
	case ModelPermissionScopeAdminOnly:
		return role >= common.RoleAdminUser
	case ModelPermissionScopeCommonOnly:
		return role < common.RoleAdminUser
	case ModelPermissionScopeNone:
		return false
	default:
		return true
	}
}

func permissionScopeAllowsCall(scope int, role int) bool {
	switch NormalizeModelPermissionScope(scope) {
	case ModelPermissionScopeAll:
		return true
	case ModelPermissionScopeAdminOnly:
		return role >= common.RoleAdminUser
	case ModelPermissionScopeCommonOnly:
		return role == common.RoleCommonUser
	case ModelPermissionScopeNone:
		return false
	default:
		return true
	}
}

func IsModelPermissionScopeCallableByRole(scope int, role int) bool {
	return permissionScopeAllowsCall(scope, role)
}

func IsModelVisibleToRole(modelName string, role int) bool {
	access := GetModelAccess(modelName)
	return permissionScopeAllowsVisibility(access.VisibilityScope, role)
}

func IsModelCallableByRole(modelName string, role int) bool {
	access := GetModelAccess(modelName)
	return permissionScopeAllowsCall(access.CallScope, role)
}

func FilterModelsByVisibility(models []string, role int) []string {
	filtered := make([]string, 0, len(models))
	for _, modelName := range models {
		if IsModelVisibleToRole(modelName, role) {
			filtered = append(filtered, modelName)
		}
	}
	return filtered
}

func FilterPricingByVisibility(pricing []Pricing, role int) []Pricing {
	filtered := make([]Pricing, 0, len(pricing))
	for _, item := range pricing {
		if IsModelVisibleToRole(item.ModelName, role) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
