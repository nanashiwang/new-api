package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetChannelMonitor 获取渠道监控统计
func GetChannelMonitor(c *gin.Context) {
	// 获取查询参数
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	groupBy := c.DefaultQuery("group_by", "channel")
	username := c.Query("username")

	// 解析时间戳
	var startTime, endTime int64
	if startTimeStr != "" {
		if val, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil {
			startTime = val
		}
	}
	if endTimeStr != "" {
		if val, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil {
			endTime = val
		}
	}

	// 验证 group_by 参数
	if groupBy != "channel" && groupBy != "group" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "group_by 参数必须是 'channel' 或 'group'",
		})
		return
	}

	// 查询监控数据
	stats, err := model.GetChannelMonitorStats(startTime, endTime, groupBy, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "查询监控数据失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}
