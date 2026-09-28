package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetChannelUsage is administrator-only. It deliberately rejects invalid ranges
// instead of silently shortening the administrator's requested accounting period.
func GetChannelUsage(c *gin.Context) {
	start, e1 := strconv.ParseInt(c.Query("start_time"), 10, 64)
	end, e2 := strconv.ParseInt(c.Query("end_time"), 10, 64)
	if e1 != nil || e2 != nil || start <= 0 || end <= start || end-start > 30*86400 || end > time.Now().Unix()+1 {
		common.ApiErrorMsg(c, "请选择有效时间范围，最长30天且不能超过当前时间")
		return
	}
	page := common.GetPageQuery(c)
	if page.GetPage() < 1 || page.GetPageSize() < 1 || page.GetPage() > int(^uint(0)>>1)/page.GetPageSize() {
		common.ApiErrorMsg(c, "无效的分页参数")
		return
	}
	order := c.DefaultQuery("usage_order", "desc")
	if order != "asc" && order != "desc" {
		common.ApiErrorMsg(c, "无效的用量排序")
		return
	}
	category, err := model.ParseChannelCategory(c.Query("category"))
	if err != nil {
		common.ApiErrorMsg(c, "无效的渠道分类")
		return
	}
	group := strings.TrimSpace(c.Query("group"))
	if group == "null" {
		group = ""
	}
	modelName := strings.TrimSpace(c.Query("model"))
	channels, err := model.SearchChannelsWithFilters(c.Query("keyword"), "", modelName, true, parseStatusFilter(c.Query("status")), -1)
	if err != nil {
		common.ApiErrorMsg(c, "加载渠道失败")
		return
	}
	categoryCounts := model.CountChannelCategories(channels)
	if group != "" {
		channels, err = model.SearchChannelsWithFilters(c.Query("keyword"), group, modelName, true, parseStatusFilter(c.Query("status")), -1)
		if err != nil {
			common.ApiErrorMsg(c, "加载渠道失败")
			return
		}
	}
	channels = model.FilterChannelsByCategory(channels, category)
	snapshot, cached, err := model.GetChannelUsage(start, end, group, modelName)
	if err != nil {
		common.SysError("加载渠道用量失败: " + err.Error())
		common.ApiErrorMsg(c, "加载渠道用量失败")
		return
	}
	deleted, err := model.GetDeletedChannelUsage(snapshot.Rows)
	if err != nil {
		common.ApiErrorMsg(c, "加载渠道用量失败")
		return
	}
	items, total, summary := model.PaginateChannelUsage(channels, snapshot.Rows, c.Query("tag_mode") == "true", order == "asc", page.GetStartIdx(), page.GetPageSize())
	for _, item := range items {
		populateChannelRuntimeState(item.Channel)
		clearChannelInfo(item.Channel)
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "page": page.GetPage(), "page_size": page.GetPageSize(), "category_counts": categoryCounts,
		"usage_summary": summary, "deleted_usage": deleted, "start_time": start, "end_time": end, "as_of": snapshot.AsOf, "cached": cached, "cache_ttl_seconds": int64(model.ChannelUsageCacheTTL().Seconds())})
}
