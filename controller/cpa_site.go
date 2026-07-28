package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type cpaCreateSiteReq struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Scheme        string `json:"scheme"`
	ManagementKey string `json:"management_key"`
	SortOrder     int    `json:"sort_order"`
}

type cpaUpdateSiteReq struct {
	cpaCreateSiteReq
	ManagementKeyChange bool `json:"management_key_change"`
}

type cpaTestConnectionReq struct {
	ID            int    `json:"id"`
	Host          string `json:"host"`
	Scheme        string `json:"scheme"`
	ManagementKey string `json:"management_key"`
}

type cpaSiteVO struct {
	Id                  int    `json:"id"`
	Name                string `json:"name"`
	Host                string `json:"host"`
	Scheme              string `json:"scheme"`
	Status              int    `json:"status"`
	LastSyncedAt        int64  `json:"last_synced_at"`
	LastSyncError       string `json:"last_sync_error"`
	ManagementKeyMasked string `json:"management_key_masked"`
	SortOrder           int    `json:"sort_order"`
	CreatedTime         int64  `json:"created_time"`
	UpdatedTime         int64  `json:"updated_time"`
	AccountCount        int    `json:"account_count"`
	AvailableCount      int    `json:"available_count"`
	LimitedCount        int    `json:"limited_count"`
	AbnormalCount       int    `json:"abnormal_count"`
	DisabledCount       int    `json:"disabled_count"`
	UnknownCount        int    `json:"unknown_count"`
}

type cpaAccountVO struct {
	service.CPAAccount
	SiteID   int    `json:"site_id"`
	SiteName string `json:"site_name"`
}

func cpaError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"success": false, "message": message})
}

func cpaAccountState(account service.CPAAccount, now time.Time) string {
	status := strings.ToLower(strings.TrimSpace(account.Status))
	if account.Disabled || status == "disabled" {
		return "disabled"
	}
	if account.Unavailable || status == "pending" || status == "refreshing" {
		return "limited"
	}
	if retryAt, err := time.Parse(time.RFC3339, strings.TrimSpace(account.NextRetryAfter)); err == nil && retryAt.After(now) {
		return "limited"
	}
	switch status {
	case "error", "failed", "invalid", "unavailable":
		return "abnormal"
	case "active":
		return "available"
	default:
		return "unknown"
	}
}

func cpaSiteToVO(site *model.CPASite, accounts []service.CPAAccount) cpaSiteVO {
	if site == nil {
		return cpaSiteVO{}
	}
	masked := ""
	if key, err := site.DecryptManagementKey(); err == nil {
		masked = model.MaskCPASecret(key)
	}
	vo := cpaSiteVO{
		Id:                  site.Id,
		Name:                site.Name,
		Host:                site.Host,
		Scheme:              site.Scheme,
		Status:              site.Status,
		LastSyncedAt:        site.LastSyncedAt,
		LastSyncError:       site.LastSyncError,
		ManagementKeyMasked: masked,
		SortOrder:           site.SortOrder,
		CreatedTime:         site.CreatedTime,
		UpdatedTime:         site.UpdatedTime,
		AccountCount:        len(accounts),
	}
	now := time.Now()
	for _, account := range accounts {
		switch cpaAccountState(account, now) {
		case "available":
			vo.AvailableCount++
		case "limited":
			vo.LimitedCount++
		case "abnormal":
			vo.AbnormalCount++
		case "disabled":
			vo.DisabledCount++
		default:
			vo.UnknownCount++
		}
	}
	return vo
}

func cpaValidationError(err error) bool {
	return errors.Is(err, model.ErrCPASiteHostRequired) ||
		errors.Is(err, model.ErrCPASiteHostInvalid) ||
		errors.Is(err, model.ErrCPASiteNameTooLong) ||
		errors.Is(err, model.ErrCPASiteKeyRequired) ||
		errors.Is(err, model.ErrCPASiteDuplicateHost)
}

func GetCPAOverview(c *gin.Context) {
	sites, err := model.ListCPASites()
	if err != nil {
		cpaError(c, http.StatusInternalServerError, err.Error())
		return
	}
	siteVOs := make([]cpaSiteVO, 0, len(sites))
	accounts := make([]cpaAccountVO, 0)
	for _, site := range sites {
		cached := service.DecodeCPACachedAccounts(site.CachedAccounts)
		siteVOs = append(siteVOs, cpaSiteToVO(site, cached))
		for _, account := range cached {
			accounts = append(accounts, cpaAccountVO{CPAAccount: account, SiteID: site.Id, SiteName: cpaSiteDisplayName(site)})
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "sites": siteVOs, "accounts": accounts})
}

func cpaSiteDisplayName(site *model.CPASite) string {
	if site == nil {
		return ""
	}
	if strings.TrimSpace(site.Name) != "" {
		return strings.TrimSpace(site.Name)
	}
	return site.Host
}

func cpaSiteEndpointChanged(existing *model.CPASite, candidate *model.CPASite) bool {
	if existing == nil || candidate == nil {
		return true
	}
	existingCopy := *existing
	candidateCopy := *candidate
	existingCopy.Normalize()
	candidateCopy.Normalize()
	return existingCopy.Host != candidateCopy.Host || existingCopy.Scheme != candidateCopy.Scheme
}

func TestCPAConnection(c *gin.Context) {
	var req cpaTestConnectionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		cpaError(c, http.StatusBadRequest, err.Error())
		return
	}
	site := &model.CPASite{Host: req.Host, Scheme: req.Scheme}
	if strings.TrimSpace(req.ManagementKey) != "" {
		if err := site.SetManagementKeyPlain(req.ManagementKey); err != nil {
			cpaError(c, http.StatusBadRequest, err.Error())
			return
		}
	} else if req.ID > 0 {
		existing, err := model.GetCPASiteByID(req.ID)
		if err != nil {
			cpaError(c, http.StatusNotFound, "CPA site not found")
			return
		}
		if cpaSiteEndpointChanged(existing, site) {
			cpaError(c, http.StatusBadRequest, model.ErrCPASiteKeyRequired.Error())
			return
		}
		site.ManagementKeyEncrypted = existing.ManagementKeyEncrypted
	}
	if err := site.Validate(true); err != nil {
		cpaError(c, http.StatusBadRequest, err.Error())
		return
	}
	accounts, err := service.FetchCPAAccounts(c.Request.Context(), site)
	if err != nil {
		cpaError(c, http.StatusBadGateway, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "account_count": len(accounts)})
}

func CreateCPASite(c *gin.Context) {
	var req cpaCreateSiteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		cpaError(c, http.StatusBadRequest, err.Error())
		return
	}
	site := &model.CPASite{Name: req.Name, Host: req.Host, Scheme: req.Scheme, SortOrder: req.SortOrder}
	if err := site.SetManagementKeyPlain(req.ManagementKey); err != nil {
		cpaError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := model.CreateCPASite(site); err != nil {
		if cpaValidationError(err) {
			cpaError(c, http.StatusBadRequest, err.Error())
		} else {
			cpaError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	syncErr := service.SyncCPASiteContext(c.Request.Context(), site)
	refreshedSite, err := model.GetCPASiteByID(site.Id)
	if err != nil {
		cpaError(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := gin.H{"success": true, "data": cpaSiteToVO(refreshedSite, service.DecodeCPACachedAccounts(refreshedSite.CachedAccounts)), "sync_success": syncErr == nil}
	if syncErr != nil {
		resp["message"] = syncErr.Error()
	}
	c.JSON(http.StatusOK, resp)
}

func UpdateCPASite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		cpaError(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req cpaUpdateSiteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		cpaError(c, http.StatusBadRequest, err.Error())
		return
	}
	site, err := model.GetCPASiteByID(id)
	if err != nil {
		cpaError(c, http.StatusNotFound, "CPA site not found")
		return
	}
	existingEndpoint := &model.CPASite{Host: site.Host, Scheme: site.Scheme}
	site.Name = req.Name
	site.Host = req.Host
	site.Scheme = req.Scheme
	site.SortOrder = req.SortOrder
	endpointChanged := cpaSiteEndpointChanged(existingEndpoint, site)
	if endpointChanged && !req.ManagementKeyChange {
		cpaError(c, http.StatusBadRequest, model.ErrCPASiteKeyRequired.Error())
		return
	}
	if req.ManagementKeyChange && strings.TrimSpace(req.ManagementKey) == "" {
		cpaError(c, http.StatusBadRequest, model.ErrCPASiteKeyRequired.Error())
		return
	}
	updateKey := req.ManagementKeyChange
	if updateKey {
		if err := site.SetManagementKeyPlain(req.ManagementKey); err != nil {
			cpaError(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	if err := model.UpdateCPASite(site, updateKey); err != nil {
		if cpaValidationError(err) {
			cpaError(c, http.StatusBadRequest, err.Error())
		} else {
			cpaError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	syncErr := service.SyncCPASiteContext(c.Request.Context(), site)
	site, err = model.GetCPASiteByID(id)
	if err != nil {
		cpaError(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := gin.H{"success": true, "data": cpaSiteToVO(site, service.DecodeCPACachedAccounts(site.CachedAccounts)), "sync_success": syncErr == nil}
	if syncErr != nil {
		resp["message"] = syncErr.Error()
	}
	c.JSON(http.StatusOK, resp)
}

func RefreshCPASite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		cpaError(c, http.StatusBadRequest, "invalid id")
		return
	}
	site, err := model.GetCPASiteByID(id)
	if err != nil {
		cpaError(c, http.StatusNotFound, "CPA site not found")
		return
	}
	if err := service.SyncCPASiteContext(c.Request.Context(), site); err != nil {
		cpaError(c, http.StatusBadGateway, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteCPASite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		cpaError(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := model.DeleteCPASite(id); err != nil {
		if errors.Is(err, model.ErrCPASiteNotFound) {
			cpaError(c, http.StatusNotFound, "CPA site not found")
		} else {
			cpaError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
