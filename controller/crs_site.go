package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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

	dashData, refreshErr := service.RefreshCRSSite(site)
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

	type siteDetail struct {
		crsSiteVO
		Dashboard *service.CRSDashboardData `json:"dashboard,omitempty"`
	}

	details := make([]siteDetail, 0, len(sites))
	for _, s := range sites {
		dash, _ := service.GetSiteDashboardFromCache(s)
		details = append(details, siteDetail{
			crsSiteVO: siteToVO(s),
			Dashboard: dash,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"aggregate": agg,
		"sites":     details,
	})
}
