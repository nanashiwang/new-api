package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const crsClientTimeout = 15 * time.Second

// CRSDashboardData 是 CRS /admin/dashboard 的核心数据结构。
type CRSDashboardData struct {
	Overview        CRSOverview        `json:"overview"`
	RecentActivity  CRSRecentActivity  `json:"recentActivity"`
	RealtimeMetrics CRSRealtimeMetrics `json:"realtimeMetrics"`
	SystemHealth    CRSSystemHealth    `json:"systemHealth"`
}

type CRSAccountsByPlatform map[string]CRSPlatformStat

type CRSPlatformStat struct {
	Total       int `json:"total"`
	Normal      int `json:"normal"`
	Abnormal    int `json:"abnormal"`
	Paused      int `json:"paused"`
	RateLimited int `json:"rateLimited"`
}

type CRSOverview struct {
	TotalApiKeys          int                   `json:"totalApiKeys"`
	ActiveApiKeys         int                   `json:"activeApiKeys"`
	TotalAccounts         int                   `json:"totalAccounts"`
	NormalAccounts        int                   `json:"normalAccounts"`
	AbnormalAccounts      int                   `json:"abnormalAccounts"`
	PausedAccounts        int                   `json:"pausedAccounts"`
	RateLimitedAccounts   int                   `json:"rateLimitedAccounts"`
	AccountsByPlatform    CRSAccountsByPlatform `json:"accountsByPlatform"`
	TotalTokensUsed       int64                 `json:"totalTokensUsed"`
	TotalRequestsUsed     int64                 `json:"totalRequestsUsed"`
	TotalInputTokensUsed  int64                 `json:"totalInputTokensUsed"`
	TotalOutputTokensUsed int64                 `json:"totalOutputTokensUsed"`
}

type CRSRecentActivity struct {
	ApiKeysCreatedToday    int   `json:"apiKeysCreatedToday"`
	RequestsToday          int64 `json:"requestsToday"`
	TokensToday            int64 `json:"tokensToday"`
	InputTokensToday       int64 `json:"inputTokensToday"`
	OutputTokensToday      int64 `json:"outputTokensToday"`
	CacheCreateTokensToday int64 `json:"cacheCreateTokensToday"`
	CacheReadTokensToday   int64 `json:"cacheReadTokensToday"`
}

type CRSRealtimeMetrics struct {
	RPM           float64 `json:"rpm"`
	TPM           float64 `json:"tpm"`
	WindowMinutes int     `json:"windowMinutes"`
}

type CRSSystemHealth struct {
	RedisConnected        bool    `json:"redisConnected"`
	ClaudeAccountsHealthy bool    `json:"claudeAccountsHealthy"`
	GeminiAccountsHealthy bool    `json:"geminiAccountsHealthy"`
	DroidAccountsHealthy  bool    `json:"droidAccountsHealthy"`
	Uptime                float64 `json:"uptime"`
}

type crsLoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type crsLoginResp struct {
	Success   bool   `json:"success"`
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expiresIn"`
	Message   string `json:"message"`
}

type crsDashboardResp struct {
	Success bool             `json:"success"`
	Message string           `json:"message"`
	Data    CRSDashboardData `json:"data"`
}

func newCRSHTTPClient() *http.Client {
	transport := &http.Transport{}
	if common.TLSInsecureSkipVerify {
		transport.TLSClientConfig = common.InsecureTLSConfig
	}
	return &http.Client{
		Transport: transport,
		Timeout:   crsClientTimeout,
	}
}

func validateCRSURL(rawURL string) error {
	fetchSetting := system_setting.GetFetchSetting()
	return common.ValidateURLWithFetchSetting(
		rawURL,
		fetchSetting.EnableSSRFProtection,
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		fetchSetting.AllowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
	)
}

// LoginCRS 用 username/password 调用 CRS 的 /web/auth/login 端点，返回 session token 及过期时长（秒）。
func LoginCRS(baseURL, username, password string) (token string, expiresIn int64, err error) {
	loginURL := strings.TrimRight(baseURL, "/") + "/web/auth/login"
	if urlErr := validateCRSURL(loginURL); urlErr != nil {
		return "", 0, fmt.Errorf("%w: %v", model.ErrCRSSiteHostInvalid, urlErr)
	}

	body, jsonErr := common.Marshal(crsLoginReq{Username: username, Password: password})
	if jsonErr != nil {
		return "", 0, jsonErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), crsClientTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := newCRSHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %v", model.ErrCRSSiteRequestFailure, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	var loginResp crsLoginResp
	if err := common.Unmarshal(raw, &loginResp); err != nil {
		return "", 0, fmt.Errorf("crs_site:login_parse_failed: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := loginResp.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return "", 0, fmt.Errorf("crs_site:login_failed: %s", msg)
	}

	if !loginResp.Success || strings.TrimSpace(loginResp.Token) == "" {
		msg := loginResp.Message
		if msg == "" {
			msg = "empty token"
		}
		return "", 0, fmt.Errorf("crs_site:login_failed: %s", msg)
	}

	return loginResp.Token, loginResp.ExpiresIn, nil
}

// FetchCRSDashboard 调用 CRS 的 /admin/dashboard 端点，返回原始 JSON bytes 及解析后的数据。
func FetchCRSDashboard(baseURL, token string) ([]byte, *CRSDashboardData, error) {
	dashURL := strings.TrimRight(baseURL, "/") + "/admin/dashboard"
	if urlErr := validateCRSURL(dashURL); urlErr != nil {
		return nil, nil, fmt.Errorf("%w: %v", model.ErrCRSSiteHostInvalid, urlErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), crsClientTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dashURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := newCRSHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", model.ErrCRSSiteRequestFailure, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("%w (HTTP %d): %s",
			model.ErrCRSSiteRequestFailure, resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var dashResp crsDashboardResp
	if parseErr := common.Unmarshal(raw, &dashResp); parseErr != nil {
		return nil, nil, fmt.Errorf("crs_site:dashboard_parse_failed: %v", parseErr)
	}
	if !dashResp.Success {
		msg := dashResp.Message
		if msg == "" {
			msg = "server returned success=false"
		}
		return nil, nil, fmt.Errorf("crs_site:dashboard_failed: %s", msg)
	}

	return raw, &dashResp.Data, nil
}

// RefreshCRSSite 完整刷新流程：确保 token 有效 → 拉取 dashboard → 持久化结果。
// 返回最新解析的 CRSDashboardData（供调用方即时返回给前端）。
func RefreshCRSSite(site *model.CRSSite) (*CRSDashboardData, error) {
	if site == nil {
		return nil, errors.New("crs_site:nil")
	}

	baseURL := site.BaseURL()
	if baseURL == "" {
		return nil, model.ErrCRSSiteHostRequired
	}

	token, err := resolveOrRenewCRSToken(site)
	if err != nil {
		syncErr := err.Error()
		_ = model.PersistCRSSiteStats(site.Id, "", 0, "", model.CRSSiteStatusError, syncErr)
		return nil, err
	}

	rawBytes, dashData, err := FetchCRSDashboard(baseURL, token)
	if err != nil {
		syncErr := err.Error()
		_ = model.PersistCRSSiteStats(site.Id, "", 0, "", model.CRSSiteStatusError, syncErr)
		return nil, err
	}

	rawStr := string(rawBytes)
	_ = model.PersistCRSSiteStats(site.Id, "", 0, rawStr, model.CRSSiteStatusSynced, "")
	return dashData, nil
}

// resolveOrRenewCRSToken 从缓存取 token（至少还有 60 秒有效期），过期则重新登录。
func resolveOrRenewCRSToken(site *model.CRSSite) (string, error) {
	now := common.GetTimestamp()
	if site.TokenExpiresAt > now+60 {
		if tok, err := site.DecryptToken(); err == nil && tok != "" {
			return tok, nil
		}
	}

	password, err := site.DecryptPassword()
	if err != nil || password == "" {
		return "", model.ErrCRSSitePassRequired
	}

	token, expiresIn, err := LoginCRS(site.BaseURL(), site.Username, password)
	if err != nil {
		return "", err
	}

	var expiresAt int64
	if expiresIn > 0 {
		expiresAt = now + expiresIn
	} else {
		expiresAt = now + 86400
	}

	encrypted, encErr := model.EncryptCRSSecret(token)
	if encErr != nil {
		return token, nil // 加密失败不阻断，只是下次仍会重新登录
	}
	_ = model.PersistCRSSiteStats(site.Id, encrypted, expiresAt, "", 0, "")
	site.TokenEncrypted = encrypted
	site.TokenExpiresAt = expiresAt
	return token, nil
}

// RefreshAllCRSSites 并发刷新所有站点，返回每个站点 ID 和错误（nil 表示成功）。
func RefreshAllCRSSites() map[int]error {
	sites, err := model.ListCRSSites()
	results := make(map[int]error, len(sites))
	if err != nil {
		return results
	}
	type pair struct {
		id  int
		err error
	}
	ch := make(chan pair, len(sites))
	for _, s := range sites {
		go func(site *model.CRSSite) {
			_, refreshErr := RefreshCRSSite(site)
			ch <- pair{id: site.Id, err: refreshErr}
		}(s)
	}
	for range sites {
		p := <-ch
		results[p.id] = p.err
	}
	return results
}

// AggregateCRSStats 从所有已同步站点的缓存 stats 聚合出汇总数据。
func AggregateCRSStats(sites []*model.CRSSite) *CRSDashboardData {
	agg := &CRSDashboardData{
		Overview: CRSOverview{
			AccountsByPlatform: make(CRSAccountsByPlatform),
		},
	}
	for _, site := range sites {
		if site.Status != model.CRSSiteStatusSynced || strings.TrimSpace(site.CachedStats) == "" {
			continue
		}
		var resp crsDashboardResp
		if err := common.UnmarshalJsonStr(site.CachedStats, &resp); err != nil || !resp.Success {
			continue
		}
		d := resp.Data
		o := d.Overview
		agg.Overview.TotalApiKeys += o.TotalApiKeys
		agg.Overview.ActiveApiKeys += o.ActiveApiKeys
		agg.Overview.TotalAccounts += o.TotalAccounts
		agg.Overview.NormalAccounts += o.NormalAccounts
		agg.Overview.AbnormalAccounts += o.AbnormalAccounts
		agg.Overview.PausedAccounts += o.PausedAccounts
		agg.Overview.RateLimitedAccounts += o.RateLimitedAccounts
		agg.Overview.TotalTokensUsed += o.TotalTokensUsed
		agg.Overview.TotalRequestsUsed += o.TotalRequestsUsed
		agg.Overview.TotalInputTokensUsed += o.TotalInputTokensUsed
		agg.Overview.TotalOutputTokensUsed += o.TotalOutputTokensUsed
		for platform, stat := range o.AccountsByPlatform {
			cur := agg.Overview.AccountsByPlatform[platform]
			cur.Total += stat.Total
			cur.Normal += stat.Normal
			cur.Abnormal += stat.Abnormal
			cur.Paused += stat.Paused
			cur.RateLimited += stat.RateLimited
			agg.Overview.AccountsByPlatform[platform] = cur
		}
		r := d.RecentActivity
		agg.RecentActivity.RequestsToday += r.RequestsToday
		agg.RecentActivity.TokensToday += r.TokensToday
		agg.RecentActivity.InputTokensToday += r.InputTokensToday
		agg.RecentActivity.OutputTokensToday += r.OutputTokensToday
		agg.RecentActivity.CacheCreateTokensToday += r.CacheCreateTokensToday
		agg.RecentActivity.CacheReadTokensToday += r.CacheReadTokensToday
		rt := d.RealtimeMetrics
		agg.RealtimeMetrics.RPM += rt.RPM
		agg.RealtimeMetrics.TPM += rt.TPM
		if rt.WindowMinutes > 0 && agg.RealtimeMetrics.WindowMinutes == 0 {
			agg.RealtimeMetrics.WindowMinutes = rt.WindowMinutes
		}
	}
	return agg
}

// GetSiteDashboardFromCache 从缓存读取单个站点的 dashboard 数据。
func GetSiteDashboardFromCache(site *model.CRSSite) (*CRSDashboardData, error) {
	if strings.TrimSpace(site.CachedStats) == "" {
		return nil, nil
	}
	var resp crsDashboardResp
	if err := common.UnmarshalJsonStr(site.CachedStats, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, nil
	}
	return &resp.Data, nil
}
