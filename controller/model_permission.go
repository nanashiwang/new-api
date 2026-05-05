package controller

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func getRequestRole(c *gin.Context) int {
	if role := common.GetContextKeyInt(c, constant.ContextKeyUserRole); role != common.RoleGuestUser {
		return role
	}
	if role := c.GetInt("role"); role != common.RoleGuestUser {
		return role
	}
	if userId := c.GetInt("id"); userId > 0 {
		if userCache, err := model.GetUserCache(userId); err == nil {
			return userCache.Role
		}
	}
	return common.RoleGuestUser
}

func addModelPermissionCandidate(candidates map[string]struct{}, modelName string) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return
	}
	candidates[modelName] = struct{}{}
}

func addModelPermissionCandidatesFromFloatMap(candidates map[string]struct{}, items map[string]float64) {
	for modelName := range items {
		addModelPermissionCandidate(candidates, modelName)
	}
}

func addModelPermissionCandidatesFromOption(candidates map[string]struct{}, optionKey string) {
	common.OptionMapRWMutex.RLock()
	raw := common.Interface2String(common.OptionMap[optionKey])
	common.OptionMapRWMutex.RUnlock()
	collectModelNamesFromOptionValue(raw, candidates)
}

func collectModelPermissionCandidates() ([]string, error) {
	candidateSet := make(map[string]struct{})
	for _, modelName := range model.GetEnabledModels() {
		addModelPermissionCandidate(candidateSet, modelName)
	}

	addModelPermissionCandidatesFromFloatMap(candidateSet, ratio_setting.GetModelRatioCopy())
	addModelPermissionCandidatesFromFloatMap(candidateSet, ratio_setting.GetModelPriceCopy())
	addModelPermissionCandidatesFromFloatMap(candidateSet, ratio_setting.GetCompletionRatioCopy())
	addModelPermissionCandidatesFromFloatMap(candidateSet, ratio_setting.GetCacheRatioCopy())
	addModelPermissionCandidatesFromFloatMap(candidateSet, ratio_setting.GetCreateCacheRatioCopy())

	for _, optionKey := range []string{"ImageRatio", "AudioRatio", "AudioCompletionRatio"} {
		addModelPermissionCandidatesFromOption(candidateSet, optionKey)
	}

	modelNames := make([]string, 0)
	if err := model.DB.Model(&model.Model{}).Pluck("model_name", &modelNames).Error; err != nil {
		return nil, err
	}
	for _, modelName := range modelNames {
		addModelPermissionCandidate(candidateSet, modelName)
	}

	permissions, err := model.GetAllModelPermissions()
	if err != nil {
		return nil, err
	}
	for _, permission := range permissions {
		addModelPermissionCandidate(candidateSet, permission.ModelName)
	}

	candidates := make([]string, 0, len(candidateSet))
	for modelName := range candidateSet {
		candidates = append(candidates, modelName)
	}
	sort.Strings(candidates)
	return candidates, nil
}

func validateModelPermission(permission *model.ModelPermission) error {
	if permission == nil {
		return fmt.Errorf("权限配置不能为空")
	}
	permission.ModelName = strings.TrimSpace(permission.ModelName)
	if permission.ModelName == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	permission.NameRule = model.NormalizeModelNameRule(permission.NameRule)
	permission.VisibilityScope = model.NormalizeModelPermissionScope(permission.VisibilityScope)
	permission.CallScope = model.NormalizeModelPermissionScope(permission.CallScope)
	duplicated, err := model.IsModelPermissionDuplicated(permission.Id, permission.ModelName, permission.NameRule)
	if err != nil {
		return err
	}
	if duplicated {
		return fmt.Errorf("同名模型和命名规则的权限配置已存在")
	}
	return nil
}

func ListModelPermissions(c *gin.Context) {
	permissions, err := model.GetAllModelPermissions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	candidates, err := collectModelPermissionCandidates()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items":      permissions,
		"candidates": candidates,
	})
}

func GetModelPermissionCandidates(c *gin.Context) {
	candidates, err := collectModelPermissionCandidates()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, candidates)
}

func CreateModelPermission(c *gin.Context) {
	var permission model.ModelPermission
	if err := common.DecodeJson(c.Request.Body, &permission); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := validateModelPermission(&permission); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := permission.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, permission)
}

func UpdateModelPermission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, fmt.Errorf("id 无效"))
		return
	}
	var permission model.ModelPermission
	if err := common.DecodeJson(c.Request.Body, &permission); err != nil {
		common.ApiError(c, err)
		return
	}
	permission.Id = id
	if err := validateModelPermission(&permission); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := permission.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, permission)
}

func DeleteModelPermission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, fmt.Errorf("id 无效"))
		return
	}
	if err := model.DeleteModelPermission(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
