package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetPulseOptions(c *gin.Context) {
	cfg := common.GetPulseConfig()
	public := make(map[string]string)
	secrets := make(map[string]bool)
	for key, value := range cfg {
		if common.IsPulseSecretKey(key) {
			secrets[key] = strings.TrimSpace(value) != ""
		} else {
			public[key] = value
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{
		"config": public, "secrets": secrets, "quota_per_unit": common.QuotaPerUnit,
		"redis_ready": common.RedisEnabled && common.RDB != nil,
	}})
}

func UpdatePulseOptions(c *gin.Context) {
	var update model.PulseConfigUpdate
	if err := common.DecodeJson(http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10), &update); err != nil || (update.Config == nil && len(update.ClearSecrets) == 0) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 Pulse 配置参数"})
		return
	}
	if err := model.UpdatePulseConfig(update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	// Never include old/new values: both the submitted and effective settings
	// can contain environment-managed secrets.
	model.RecordLogWithAdminInfo(c.GetInt("id"), model.LogTypeManage, "管理员更新 Meta Pulse 对接配置", map[string]interface{}{
		"admin_id": c.GetInt("id"), "admin_username": c.GetString("username"),
	})
	GetPulseOptions(c)
}
