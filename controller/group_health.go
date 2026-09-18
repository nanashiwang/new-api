package controller

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	grouphealth "github.com/QuantumNous/new-api/pkg/group_health"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

func visibleHealthGroups(userGroup string, requested string) []string {
	active := ratio_setting.GetGroupRatioCopy()
	groups := make([]string, 0)
	for group := range service.GetUserUsableGroups(userGroup) {
		if group == "" || group == "auto" || (requested != "" && requested != group) {
			continue
		}
		if _, exists := active[group]; exists {
			groups = append(groups, group)
		}
	}
	sort.Strings(groups)
	return groups
}

func GetGroupHealth(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "请先登录"})
		return
	}
	// Read current membership from DB rather than trusting query parameters or
	// a possibly stale session/cache when deciding visibility.
	user, err := model.GetUserById(userID, false)
	if err != nil {
		common.SysError("failed to resolve group health permissions: " + err.Error())
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "分组健康数据暂不可用"})
		return
	}
	if user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "用户已被禁用"})
		return
	}
	modelName := strings.TrimSpace(c.Query("model"))
	requested := strings.TrimSpace(c.Query("group"))
	if len(modelName) > 255 || len(requested) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "查询参数无效"})
		return
	}
	groups := visibleHealthGroups(user.Group, requested)
	permissionName := ratio_setting.FormatMatchingModelName(modelName)
	if modelName != "" && (!model.IsModelCallableByRole(permissionName, user.Role) || !model.IsModelVisibleToRole(permissionName, user.Role)) {
		groups = nil
	}
	result, err := grouphealth.Query(c.Request.Context(), groups, modelName, user.Role, time.Now())
	if err != nil {
		common.SysError("failed to query group health: " + err.Error())
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "分组健康数据暂不可用"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
