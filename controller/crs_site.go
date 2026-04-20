package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ---------- DTO ----------

type crsCreateSiteReq struct {
	Name     string `json:"name"`
	Host     string `json:"host" binding:"required"`
	Scheme   string `json:"scheme"`
	Group    string `json:"group"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type crsUpdateSiteReq struct {
	Name           string `json:"name"`
	Host           string `json:"host" binding:"required"`
	Scheme         string `json:"scheme"`
	Group          string `json:"group"`
	Username       string `json:"username" binding:"required"`
	Password       string `json:"password"`
	PasswordChange bool   `json:"password_change"`
}

// crsSiteVO 是返回给前端的站点视图对象（不含敏感字段）。
type crsSiteVO struct {
	Id            int    `json:"id"`
	Name          string `json:"name"`
	Host          string `json:"host"`
	Scheme        string `json:"scheme"`
	Group         string `json:"group"`
	Username      string `json:"username"`
	Status        int    `json:"status"`
	LastSyncedAt  int64  `json:"last_synced_at"`
	LastSyncError string `json:"last_sync_error"`
	TokenMasked   string `json:"token_masked"`
	SortOrder     int    `json:"sort_order"`
	CreatedTime   int64  `json:"created_time"`
	UpdatedTime   int64  `json:"updated_time"`
}

type crsAccountVO struct {
	Id                        int                    `json:"id"`
	SiteID                    int                    `json:"site_id"`
	SiteName                  string                 `json:"site_name"`
	RemoteAccountID           string                 `json:"remote_account_id"`
	Platform                  string                 `json:"platform"`
	Name                      string                 `json:"name"`
	Description               string                 `json:"description"`
	AccountType               string                 `json:"account_type"`
	AuthType                  string                 `json:"auth_type"`
	Status                    string                 `json:"status"`
	ErrorMessage              string                 `json:"error_message"`
	IsActive                  bool                   `json:"is_active"`
	Schedulable               bool                   `json:"schedulable"`
	Priority                  int                    `json:"priority"`
	RateLimited               bool                   `json:"rate_limited"`
	RateLimitMinutesRemaining int                    `json:"rate_limit_minutes_remaining"`
	RateLimitResetAt          string                 `json:"rate_limit_reset_at"`
	SessionWindowActive       bool                   `json:"session_window_active"`
	SessionWindowStatus       string                 `json:"session_window_status"`
	SessionWindowProgress     float64                `json:"session_window_progress"`
	SessionWindowRemaining    string                 `json:"session_window_remaining"`
	SessionWindowEndAt        string                 `json:"session_window_end_at"`
	UsageWindows              []model.CRSUsageWindow `json:"usage_windows"`
	SubscriptionPlan          string                 `json:"subscription_plan"`
	SubscriptionInfo          map[string]any         `json:"subscription_info,omitempty"`
	Quota                     map[string]any         `json:"quota,omitempty"`
	QuotaUnlimited            bool                   `json:"quota_unlimited"`
	QuotaTotal                float64                `json:"quota_total"`
	QuotaUsed                 float64                `json:"quota_used"`
	QuotaRemaining            float64                `json:"quota_remaining"`
	QuotaPercentage           float64                `json:"quota_percentage"`
	QuotaResetAt              string                 `json:"quota_reset_at"`
	BalanceAmount             float64                `json:"balance_amount"`
	BalanceCurrency           string                 `json:"balance_currency"`
	UsageDailyRequests        int64                  `json:"usage_daily_requests"`
	UsageTotalRequests        int64                  `json:"usage_total_requests"`
	UsageDailyTokens          int64                  `json:"usage_daily_tokens"`
	UsageTotalTokens          int64                  `json:"usage_total_tokens"`
	UsageRPM                  float64                `json:"usage_rpm"`
	UsageTPM                  float64                `json:"usage_tpm"`
	SyncError                 string                 `json:"sync_error"`
	LastSyncedAt              int64                  `json:"last_synced_at"`
	UpdatedTime               int64                  `json:"updated_time"`
}

func siteToVO(s *model.CRSSite) crsSiteVO {
	return crsSiteVO{
		Id:            s.Id,
		Name:          s.Name,
		Host:          s.Host,
		Scheme:        s.Scheme,
		Group:         s.Group,
		Username:      s.Username,
		Status:        s.Status,
		LastSyncedAt:  s.LastSyncedAt,
		LastSyncError: s.LastSyncError,
		TokenMasked:   s.SiteBriefToken(),
		SortOrder:     s.SortOrder,
		CreatedTime:   s.CreatedTime,
		UpdatedTime:   s.UpdatedTime,
	}
}

func siteDisplayName(s *model.CRSSite) string {
	if s == nil {
		return ""
	}
	if strings.TrimSpace(s.Name) != "" {
		return strings.TrimSpace(s.Name)
	}
	return strings.TrimSpace(s.Host)
}

func decodeJSONMap(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	result := make(map[string]any)
	if err := common.UnmarshalJsonStr(raw, &result); err != nil {
		return nil
	}
	return result
}

func decodeCRSUsageWindows(raw string) []model.CRSUsageWindow {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []model.CRSUsageWindow{}
	}
	windows := make([]model.CRSUsageWindow, 0)
	if err := common.UnmarshalJsonStr(raw, &windows); err != nil {
		return []model.CRSUsageWindow{}
	}
	return windows
}

func accountSnapshotToVO(snapshot *model.CRSAccountSnapshot, siteName string) crsAccountVO {
	return crsAccountVO{
		Id:                        snapshot.Id,
		SiteID:                    snapshot.SiteID,
		SiteName:                  siteName,
		RemoteAccountID:           snapshot.RemoteAccountID,
		Platform:                  snapshot.Platform,
		Name:                      snapshot.Name,
		Description:               snapshot.Description,
		AccountType:               snapshot.AccountType,
		AuthType:                  snapshot.AuthType,
		Status:                    snapshot.Status,
		ErrorMessage:              snapshot.ErrorMessage,
		IsActive:                  snapshot.IsActive,
		Schedulable:               snapshot.Schedulable,
		Priority:                  snapshot.Priority,
		RateLimited:               snapshot.RateLimited,
		RateLimitMinutesRemaining: snapshot.RateLimitMinutesRemaining,
		RateLimitResetAt:          snapshot.RateLimitResetAt,
		SessionWindowActive:       snapshot.SessionWindowActive,
		SessionWindowStatus:       snapshot.SessionWindowStatus,
		SessionWindowProgress:     snapshot.SessionWindowProgress,
		SessionWindowRemaining:    snapshot.SessionWindowRemaining,
		SessionWindowEndAt:        snapshot.SessionWindowEndAt,
		UsageWindows:              decodeCRSUsageWindows(snapshot.UsageWindowsJSON),
		SubscriptionPlan:          snapshot.SubscriptionPlan,
		SubscriptionInfo:          decodeJSONMap(snapshot.SubscriptionInfo),
		Quota:                     decodeJSONMap(snapshot.QuotaJSON),
		QuotaUnlimited:            snapshot.QuotaUnlimited,
		QuotaTotal:                snapshot.QuotaTotal,
		QuotaUsed:                 snapshot.QuotaUsed,
		QuotaRemaining:            snapshot.QuotaRemaining,
		QuotaPercentage:           snapshot.QuotaPercentage,
		QuotaResetAt:              snapshot.QuotaResetAt,
		BalanceAmount:             snapshot.BalanceAmount,
		BalanceCurrency:           snapshot.BalanceCurrency,
		UsageDailyRequests:        snapshot.UsageDailyRequests,
		UsageTotalRequests:        snapshot.UsageTotalRequests,
		UsageDailyTokens:          snapshot.UsageDailyTokens,
		UsageTotalTokens:          snapshot.UsageTotalTokens,
		UsageRPM:                  snapshot.UsageRPM,
		UsageTPM:                  snapshot.UsageTPM,
		SyncError:                 snapshot.SyncError,
		LastSyncedAt:              snapshot.LastSyncedAt,
		UpdatedTime:               snapshot.UpdatedTime,
	}
}

func crsError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"success": false, "message": msg})
}

func crsSiteNotFound(err error) bool {
	return errors.Is(err, model.ErrCRSSiteNotFound)
}

// ---------- Handlers ----------

// GetCRSSites GET /api/crs/sites
func GetCRSSites(c *gin.Context) {
	sites, err := model.ListCRSSites()
	if err != nil {
		crsError(c, http.StatusInternalServerError, err.Error())
		return
	}
	vos := make([]crsSiteVO, 0, len(sites))
	for _, s := range sites {
		vos = append(vos, siteToVO(s))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": vos})
}

// CreateCRSSite POST /api/crs/sites
func CreateCRSSite(c *gin.Context) {
	var req crsCreateSiteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		crsError(c, http.StatusBadRequest, err.Error())
		return
	}

	site := &model.CRSSite{
		Name:     req.Name,
		Host:     req.Host,
		Scheme:   req.Scheme,
		Group:    req.Group,
		Username: req.Username,
	}
	site.Normalize()

	if err := site.SetPasswordPlain(req.Password); err != nil {
		crsError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := model.CreateCRSSite(site); err != nil {
		switch {
		case errors.Is(err, model.ErrCRSSiteHostRequired),
			errors.Is(err, model.ErrCRSSiteUserRequired),
			errors.Is(err, model.ErrCRSSitePassRequired),
			errors.Is(err, model.ErrCRSSiteHostInvalid),
			errors.Is(err, model.ErrCRSSiteDuplicateHost):
			crsError(c, http.StatusBadRequest, err.Error())
		default:
			crsError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": siteToVO(site)})
}

// UpdateCRSSite PUT /api/crs/sites/:id
func UpdateCRSSite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		crsError(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req crsUpdateSiteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		crsError(c, http.StatusBadRequest, err.Error())
		return
	}

	existing, err := model.GetCRSSiteByID(id)
	if err != nil {
		if crsSiteNotFound(err) {
			crsError(c, http.StatusNotFound, "site not found")
		} else {
			crsError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	existing.Name = req.Name
	existing.Host = req.Host
	existing.Scheme = req.Scheme
	existing.Group = req.Group
	existing.Username = req.Username

	updatePassword := req.PasswordChange && strings.TrimSpace(req.Password) != ""
	if updatePassword {
		if err := existing.SetPasswordPlain(req.Password); err != nil {
			crsError(c, http.StatusBadRequest, err.Error())
			return
		}
	}

	if err := model.UpdateCRSSite(existing, updatePassword); err != nil {
		switch {
		case errors.Is(err, model.ErrCRSSiteHostRequired),
			errors.Is(err, model.ErrCRSSiteUserRequired),
			errors.Is(err, model.ErrCRSSiteHostInvalid),
			errors.Is(err, model.ErrCRSSiteDuplicateHost):
			crsError(c, http.StatusBadRequest, err.Error())
		case crsSiteNotFound(err):
			crsError(c, http.StatusNotFound, "site not found")
		default:
			crsError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": siteToVO(existing)})
}

// DeleteCRSSite DELETE /api/crs/sites/:id
func DeleteCRSSite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		crsError(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := model.DeleteCRSSite(id); err != nil {
		if crsSiteNotFound(err) {
			crsError(c, http.StatusNotFound, "site not found")
		} else {
			crsError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RefreshCRSSiteByID POST /api/crs/sites/:id/refresh
func RefreshCRSSiteByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		crsError(c, http.StatusBadRequest, "invalid id")
		return
	}

	site, err := model.GetCRSSiteByID(id)
	if err != nil {
		if crsSiteNotFound(err) {
			crsError(c, http.StatusNotFound, "site not found")
		} else {
			crsError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	refreshErr := service.SyncCRSObserverSite(site)
	site, _ = model.GetCRSSiteByID(id)
	dashData, _ := service.GetSiteDashboardFromCache(site)
	resp := gin.H{
		"success": refreshErr == nil,
		"site":    siteToVO(site),
	}
	if refreshErr != nil {
		resp["message"] = refreshErr.Error()
		resp["dashboard"] = nil
	} else {
		resp["dashboard"] = dashData
	}
	c.JSON(http.StatusOK, resp)
}

// RefreshAllCRSSites POST /api/crs/refresh_all
func RefreshAllCRSSites(c *gin.Context) {
	results := service.RefreshAllCRSSites()
	summary := make([]gin.H, 0, len(results))
	for id, err := range results {
		item := gin.H{"id": id, "success": err == nil}
		if err != nil {
			item["error"] = err.Error()
		}
		summary = append(summary, item)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

// GetCRSOverview GET /api/crs/overview
// 返回汇总后的统计数据 + 所有站点的 VO 列表（含 CachedStats 用于各站点明细）。
func GetCRSOverview(c *gin.Context) {
	sites, err := model.ListCRSSites()
	if err != nil {
		crsError(c, http.StatusInternalServerError, err.Error())
		return
	}

	agg := service.AggregateCRSStats(sites)
	accounts, err := model.ListAllCRSAccountSnapshots()
	if err != nil {
		crsError(c, http.StatusInternalServerError, err.Error())
		return
	}
	observerSummary := service.BuildCRSObserverSummary(sites, accounts)

	type siteDetail struct {
		crsSiteVO
		Dashboard        *service.CRSDashboardData `json:"dashboard,omitempty"`
		AccountCount     int                       `json:"account_count"`
		RateLimitedCount int                       `json:"rate_limited_count"`
		LowQuotaCount    int                       `json:"low_quota_count"`
		EmptyQuotaCount  int                       `json:"empty_quota_count"`
	}

	accountStatsBySite := make(map[int]gin.H)
	for _, account := range accounts {
		if account == nil {
			continue
		}
		stat, ok := accountStatsBySite[account.SiteID]
		if !ok {
			stat = gin.H{
				"total":        0,
				"rate_limited": 0,
				"low_quota":    0,
				"empty_quota":  0,
			}
		}
		stat["total"] = stat["total"].(int) + 1
		if account.RateLimited {
			stat["rate_limited"] = stat["rate_limited"].(int) + 1
		}
		if !account.QuotaUnlimited && account.QuotaTotal > 0 {
			if account.QuotaRemaining <= 0 {
				stat["empty_quota"] = stat["empty_quota"].(int) + 1
			} else if account.QuotaRemaining <= 10 {
				stat["low_quota"] = stat["low_quota"].(int) + 1
			}
		}
		accountStatsBySite[account.SiteID] = stat
	}

	details := make([]siteDetail, 0, len(sites))
	for _, s := range sites {
		dash, _ := service.GetSiteDashboardFromCache(s)
		stat := accountStatsBySite[s.Id]
		accountCount := 0
		rateLimitedCount := 0
		lowQuotaCount := 0
		emptyQuotaCount := 0
		if stat != nil {
			accountCount = stat["total"].(int)
			rateLimitedCount = stat["rate_limited"].(int)
			lowQuotaCount = stat["low_quota"].(int)
			emptyQuotaCount = stat["empty_quota"].(int)
		}
		details = append(details, siteDetail{
			crsSiteVO:        siteToVO(s),
			Dashboard:        dash,
			AccountCount:     accountCount,
			RateLimitedCount: rateLimitedCount,
			LowQuotaCount:    lowQuotaCount,
			EmptyQuotaCount:  emptyQuotaCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"aggregate": agg,
		"observer":  observerSummary,
		"sites":     details,
	})
}

// GetCRSAccounts GET /api/crs/accounts
func GetCRSAccounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	siteID, _ := strconv.Atoi(c.DefaultQuery("site_id", "0"))

	rows, total, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:     siteID,
		Platform:   c.Query("platform"),
		Status:     c.Query("status"),
		Keyword:    c.Query("keyword"),
		QuotaState: c.Query("quota_state"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		crsError(c, http.StatusInternalServerError, err.Error())
		return
	}

	sites, err := model.ListCRSSites()
	if err != nil {
		crsError(c, http.StatusInternalServerError, err.Error())
		return
	}
	siteNames := make(map[int]string, len(sites))
	for _, site := range sites {
		siteNames[site.Id] = siteDisplayName(site)
	}

	data := make([]crsAccountVO, 0, len(rows))
	for _, row := range rows {
		data = append(data, accountSnapshotToVO(row, siteNames[row.SiteID]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      data,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetCRSSiteAccounts GET /api/crs/sites/:id/accounts
func GetCRSSiteAccounts(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		crsError(c, http.StatusBadRequest, "invalid id")
		return
	}

	site, err := model.GetCRSSiteByID(id)
	if err != nil {
		if crsSiteNotFound(err) {
			crsError(c, http.StatusNotFound, "site not found")
		} else {
			crsError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	accounts, err := model.ListCRSAccountSnapshotsBySite(id)
	if err != nil {
		crsError(c, http.StatusInternalServerError, err.Error())
		return
	}
	dashboard, _ := service.GetSiteDashboardFromCache(site)
	data := make([]crsAccountVO, 0, len(accounts))
	for _, row := range accounts {
		data = append(data, accountSnapshotToVO(row, siteDisplayName(site)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"site":      siteToVO(site),
		"dashboard": dashboard,
		"observer":  service.BuildCRSObserverSummary([]*model.CRSSite{site}, accounts),
		"accounts":  data,
	})
}
