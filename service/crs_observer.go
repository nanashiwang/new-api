package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var errCRSObserverEndpointUnsupported = errors.New("crs_observer:endpoint_unsupported")

type crsRemoteAccountEndpoint struct {
	Platform string
	Path     string
}

var crsRemoteAccountEndpoints = []crsRemoteAccountEndpoint{
	{Platform: "claude", Path: "/admin/claude-accounts"},
	{Platform: "claude-console", Path: "/admin/claude-console-accounts"},
	{Platform: "gemini", Path: "/admin/gemini-accounts"},
	{Platform: "gemini-api", Path: "/admin/gemini-api-accounts"},
	{Platform: "openai", Path: "/admin/openai-accounts"},
	{Platform: "openai-responses", Path: "/admin/openai-responses-accounts"},
	{Platform: "azure_openai", Path: "/admin/azure-openai-accounts"},
	{Platform: "bedrock", Path: "/admin/bedrock-accounts"},
	{Platform: "droid", Path: "/admin/droid-accounts"},
	{Platform: "ccr", Path: "/admin/ccr-accounts"},
}

type CRSObserverSummary struct {
	TotalSites       int `json:"total_sites"`
	SyncedSites      int `json:"synced_sites"`
	ErrorSites       int `json:"error_sites"`
	TotalAccounts    int `json:"total_accounts"`
	ActiveAccounts   int `json:"active_accounts"`
	SchedulableCount int `json:"schedulable_count"`
	RateLimitedCount int `json:"rate_limited_count"`
	LowQuotaCount    int `json:"low_quota_count"`
	EmptyQuotaCount  int `json:"empty_quota_count"`
}

func SyncCRSObserverSite(site *model.CRSSite) error {
	if site == nil {
		return errors.New("crs_observer:nil_site")
	}

	_, err := RefreshCRSSite(site)
	if err != nil {
		return err
	}

	now := common.GetTimestamp()
	token, err := resolveOrRenewCRSToken(site)
	if err != nil {
		return err
	}

	snapshots, fetchErr := fetchCRSObserverSnapshots(site, token, now)
	if replaceErr := model.ReplaceCRSAccountSnapshots(site.Id, snapshots); replaceErr != nil {
		return replaceErr
	}
	if fetchErr != nil {
		return fetchErr
	}
	return nil
}

func BuildCRSObserverSummary(sites []*model.CRSSite, accounts []*model.CRSAccountSnapshot) CRSObserverSummary {
	summary := CRSObserverSummary{
		TotalSites:    len(sites),
		TotalAccounts: len(accounts),
	}
	for _, site := range sites {
		if site == nil {
			continue
		}
		switch site.Status {
		case model.CRSSiteStatusSynced:
			summary.SyncedSites++
		case model.CRSSiteStatusError:
			summary.ErrorSites++
		}
	}
	for _, account := range accounts {
		if account == nil {
			continue
		}
		if account.IsActive {
			summary.ActiveAccounts++
		}
		if account.Schedulable {
			summary.SchedulableCount++
		}
		if account.RateLimited {
			summary.RateLimitedCount++
		}
		if !account.QuotaUnlimited {
			if account.QuotaRemaining <= 0 && account.QuotaTotal > 0 {
				summary.EmptyQuotaCount++
			} else if account.QuotaRemaining > 0 && account.QuotaRemaining <= 10 {
				summary.LowQuotaCount++
			}
		}
	}
	return summary
}

func fetchCRSObserverSnapshots(site *model.CRSSite, token string, syncedAt int64) ([]*model.CRSAccountSnapshot, error) {
	snapshots := make([]*model.CRSAccountSnapshot, 0)
	warnings := make([]string, 0)

	for _, endpoint := range crsRemoteAccountEndpoints {
		accounts, err := fetchCRSRemoteAccountList(site.BaseURL(), token, endpoint.Path)
		if err != nil {
			if errors.Is(err, errCRSObserverEndpointUnsupported) {
				continue
			}
			warnings = append(warnings, fmt.Sprintf("%s: %v", endpoint.Platform, err))
			continue
		}
		for _, account := range accounts {
			var balancePayload map[string]any
			balancePayload, err = fetchCRSRemoteAccountBalance(site.BaseURL(), token, endpoint.Platform, getStringValue(account["id"]))
			if err != nil && !errors.Is(err, errCRSObserverEndpointUnsupported) {
				balancePayload = map[string]any{
					"data": map[string]any{
						"status": "error",
						"error":  err.Error(),
					},
				}
			}

			snapshot, normalizeErr := normalizeCRSRemoteAccountSnapshot(site.Id, endpoint.Platform, account, balancePayload, syncedAt)
			if normalizeErr != nil {
				warnings = append(warnings, fmt.Sprintf("%s:%s: %v", endpoint.Platform, getStringValue(account["id"]), normalizeErr))
				continue
			}
			if err != nil && !errors.Is(err, errCRSObserverEndpointUnsupported) {
				snapshot.SyncError = err.Error()
			}
			snapshots = append(snapshots, snapshot)
		}
	}

	if len(warnings) == 0 {
		return snapshots, nil
	}
	return snapshots, errors.New(strings.Join(warnings, "; "))
}

func fetchCRSRemoteAccountList(baseURL, token, path string) ([]map[string]any, error) {
	payload, err := fetchCRSAuthJSON(baseURL, token, path)
	if err != nil {
		return nil, err
	}

	items, err := unwrapCRSDataArray(payload)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func fetchCRSRemoteAccountBalance(baseURL, token, platform, accountID string) (map[string]any, error) {
	if strings.TrimSpace(accountID) == "" {
		return nil, errors.New("crs_observer:empty_account_id")
	}
	query := neturl.Values{}
	query.Set("platform", platform)
	query.Set("queryApi", "false")
	path := fmt.Sprintf("/admin/accounts/%s/balance?%s", neturl.PathEscape(accountID), query.Encode())

	payload, err := fetchCRSAuthJSON(baseURL, token, path)
	if err != nil {
		return nil, err
	}
	data, ok := payload.(map[string]any)
	if !ok {
		return nil, errors.New("crs_observer:invalid_balance_payload")
	}
	return data, nil
}

func fetchCRSAuthJSON(baseURL, token, path string) (any, error) {
	fullURL := strings.TrimRight(baseURL, "/") + path
	if err := validateCRSURL(fullURL); err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrCRSSiteHostInvalid, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), crsClientTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := newCRSHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrCRSSiteRequestFailure, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("%w: %s", errCRSObserverEndpointUnsupported, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w (HTTP %d): %s", model.ErrCRSSiteRequestFailure, resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var payload any
	if err := common.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func unwrapCRSDataArray(payload any) ([]map[string]any, error) {
	switch typed := payload.(type) {
	case map[string]any:
		if success, ok := typed["success"].(bool); ok && !success {
			return nil, errors.New(getStringValue(typed["message"]))
		}
		if data, ok := typed["data"]; ok {
			return coerceMapArray(data)
		}
		return nil, errors.New("crs_observer:missing_data_array")
	case []any:
		return coerceMapArray(typed)
	default:
		return nil, errors.New("crs_observer:unexpected_payload")
	}
}

func coerceMapArray(raw any) ([]map[string]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, errors.New("crs_observer:data_not_array")
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		mapped, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, mapped)
	}
	return result, nil
}

func normalizeCRSRemoteAccountSnapshot(siteID int, platform string, account map[string]any, balance map[string]any, syncedAt int64) (*model.CRSAccountSnapshot, error) {
	remoteAccountID := getStringValue(account["id"])
	if remoteAccountID == "" {
		return nil, errors.New("crs_observer:missing_remote_account_id")
	}

	snapshot := &model.CRSAccountSnapshot{
		SiteID:              siteID,
		RemoteAccountID:     remoteAccountID,
		Platform:            strings.TrimSpace(platform),
		Name:                firstNonEmpty(getStringValue(account["name"]), remoteAccountID),
		Description:         getStringValue(account["description"]),
		AccountType:         getStringValue(account["accountType"]),
		AuthType:            getStringValue(account["authType"]),
		Status:              getStringValue(account["status"]),
		ErrorMessage:        getStringValue(account["errorMessage"]),
		IsActive:            getBoolValue(account["isActive"], true),
		Schedulable:         getBoolValue(account["schedulable"], true),
		Priority:            getIntValue(account["priority"]),
		RateLimitResetAt:    getStringValue(account["rateLimitResetAt"]),
		SessionWindowStatus: "",
		LastSyncedAt:        syncedAt,
	}

	usage := getMapValue(account["usage"])
	totalUsage := getMapValue(usage["total"])
	dailyUsage := getMapValue(usage["daily"])
	avgUsage := getMapValue(usage["averages"])
	snapshot.UsageDailyRequests = int64(getIntValue(dailyUsage["requests"]))
	snapshot.UsageTotalRequests = int64(getIntValue(totalUsage["requests"]))
	snapshot.UsageDailyTokens = int64(firstNonZeroInt(getIntValue(dailyUsage["allTokens"]), getIntValue(dailyUsage["tokens"])))
	snapshot.UsageTotalTokens = int64(firstNonZeroInt(getIntValue(totalUsage["allTokens"]), getIntValue(totalUsage["tokens"])))
	snapshot.UsageRPM = getFloatValue(avgUsage["rpm"])
	snapshot.UsageTPM = getFloatValue(avgUsage["tpm"])
	snapshot.UsageDailyCost = getFloatValue(dailyUsage["cost"])

	rateLimit := getMapValue(account["rateLimitStatus"])
	snapshot.RateLimited = getBoolValue(rateLimit["isRateLimited"], false)
	snapshot.RateLimitMinutesRemaining = getIntValue(rateLimit["minutesRemaining"])
	snapshot.RateLimitResetAt = firstNonEmpty(snapshot.RateLimitResetAt, getStringValue(rateLimit["rateLimitEndAt"]), getStringValue(rateLimit["resetAt"]))

	sessionWindow := getMapValue(account["sessionWindow"])
	snapshot.SessionWindowActive = getBoolValue(sessionWindow["hasActiveWindow"], false)
	snapshot.SessionWindowStatus = firstNonEmpty(getStringValue(sessionWindow["sessionWindowStatus"]), getStringValue(sessionWindow["status"]))
	snapshot.SessionWindowProgress = getFloatValue(sessionWindow["progress"])
	snapshot.SessionWindowRemaining = getStringValue(sessionWindow["remainingTime"])
	snapshot.SessionWindowEndAt = firstNonEmpty(getStringValue(sessionWindow["windowEnd"]), getStringValue(sessionWindow["windowEndAt"]))

	subscriptionInfo := getMapOrJSONValue(account["subscriptionInfo"])
	snapshot.SubscriptionPlan = firstNonEmpty(
		getStringValue(subscriptionInfo["accountType"]),
		getStringValue(subscriptionInfo["plan"]),
		getStringValue(subscriptionInfo["planName"]),
	)
	snapshot.SubscriptionInfo = marshalCRSObserverValue(subscriptionInfo)

	snapshot.QuotaTotal = getFloatValue(account["dailyQuota"])
	if snapshot.QuotaTotal > 0 && snapshot.QuotaResetAt == "" {
		snapshot.QuotaResetAt = getStringValue(account["quotaResetTime"])
	}

	data := balance
	if payload := getMapValue(balance["data"]); payload != nil {
		data = payload
	}

	quota := getMapOrJSONValue(data["quota"])
	if quota != nil {
		snapshot.QuotaJSON = marshalCRSObserverValue(quota)
		snapshot.QuotaUnlimited = getBoolValue(quota["unlimited"], false)
		snapshot.QuotaUsed = getFloatValue(quota["used"])
		snapshot.QuotaRemaining = getFloatValue(quota["remaining"])
		snapshot.QuotaPercentage = getFloatValue(quota["percentage"])
		snapshot.QuotaResetAt = firstNonEmpty(getStringValue(quota["resetAt"]), snapshot.QuotaResetAt)
		if !snapshot.QuotaUnlimited && snapshot.QuotaTotal == 0 {
			total := snapshot.QuotaUsed + snapshot.QuotaRemaining
			if total > 0 {
				snapshot.QuotaTotal = total
			}
		}
	}

	balanceBlock := getMapOrJSONValue(data["balance"])
	if balanceBlock != nil {
		snapshot.BalanceAmount = getFloatValue(balanceBlock["amount"])
		snapshot.BalanceCurrency = getStringValue(balanceBlock["currency"])
	}

	if snapshot.QuotaTotal > 0 && !snapshot.QuotaUnlimited && snapshot.QuotaRemaining == 0 && snapshot.BalanceAmount > 0 {
		snapshot.QuotaRemaining = snapshot.BalanceAmount
		snapshot.QuotaUsed = snapshot.QuotaTotal - snapshot.QuotaRemaining
		if snapshot.QuotaUsed < 0 {
			snapshot.QuotaUsed = 0
		}
		snapshot.QuotaPercentage = (snapshot.QuotaUsed / snapshot.QuotaTotal) * 100
	}

	if syncErr := firstNonEmpty(getStringValue(data["error"]), getStringValue(data["errorMessage"])); syncErr != "" {
		snapshot.SyncError = syncErr
	}

	snapshot.UsageWindowsJSON = marshalCRSUsageWindows(buildCRSUsageWindows(account, snapshot))
	snapshot.RawAccount = marshalCRSObserverValue(account)
	snapshot.RawBalance = marshalCRSObserverValue(balance)
	return snapshot, nil
}

func buildCRSUsageWindows(account map[string]any, snapshot *model.CRSAccountSnapshot) []model.CRSUsageWindow {
	if windows := buildCRSClaudeUsageWindows(getMapOrJSONValue(account["claudeUsage"])); len(windows) > 0 {
		return windows
	}
	if windows := buildCRSCodexUsageWindows(getMapOrJSONValue(account["codexUsage"])); len(windows) > 0 {
		return windows
	}
	if sessionWindow, ok := buildCRSSessionUsageWindow(getMapOrJSONValue(account["sessionWindow"])); ok {
		return []model.CRSUsageWindow{sessionWindow}
	}
	if quotaWindow, ok := buildCRSQuotaUsageWindow(snapshot); ok {
		return []model.CRSUsageWindow{quotaWindow}
	}
	return []model.CRSUsageWindow{}
}

func buildCRSClaudeUsageWindows(raw map[string]any) []model.CRSUsageWindow {
	definitions := []struct {
		field string
		key   string
		label string
	}{
		{field: "fiveHour", key: "five_hour", label: "5h"},
		{field: "sevenDay", key: "seven_day", label: "7d"},
		{field: "sevenDayOpus", key: "seven_day_opus", label: "Opus 周限"},
	}

	windows := make([]model.CRSUsageWindow, 0, len(definitions))
	for _, definition := range definitions {
		window, ok := normalizeCRSUsageWindow(
			definition.key,
			definition.label,
			"claude_usage",
			getMapOrJSONValue(raw[definition.field]),
		)
		if ok {
			windows = append(windows, window)
		}
	}
	return windows
}

func buildCRSCodexUsageWindows(raw map[string]any) []model.CRSUsageWindow {
	definitions := []struct {
		field string
		key   string
		label string
	}{
		{field: "primary", key: "primary", label: "5h"},
		{field: "secondary", key: "secondary", label: "周限"},
	}

	windows := make([]model.CRSUsageWindow, 0, len(definitions))
	for _, definition := range definitions {
		window, ok := normalizeCRSUsageWindow(
			definition.key,
			definition.label,
			"codex_usage",
			getMapOrJSONValue(raw[definition.field]),
		)
		if ok {
			windows = append(windows, window)
		}
	}
	return windows
}

func buildCRSSessionUsageWindow(raw map[string]any) (model.CRSUsageWindow, bool) {
	return normalizeCRSUsageWindow("session_window", "5h", "session_window", raw)
}

func buildCRSQuotaUsageWindow(snapshot *model.CRSAccountSnapshot) (model.CRSUsageWindow, bool) {
	if snapshot == nil {
		return model.CRSUsageWindow{}, false
	}

	progress := snapshot.QuotaPercentage
	hasProgress := false
	if progress > 0 || snapshot.QuotaTotal > 0 || snapshot.QuotaUsed > 0 {
		hasProgress = true
	}
	if !hasProgress && snapshot.QuotaTotal > 0 {
		progress = clampCRSUsageWindowProgress((snapshot.QuotaUsed / snapshot.QuotaTotal) * 100)
		hasProgress = true
	}

	remainingText := ""
	switch {
	case snapshot.QuotaRemaining > 0:
		remainingText = formatCRSUsageWindowNumber(snapshot.QuotaRemaining)
	case snapshot.BalanceAmount > 0:
		remainingText = formatCRSUsageWindowNumber(snapshot.BalanceAmount)
	}

	if !hasProgress && remainingText == "" && strings.TrimSpace(snapshot.QuotaResetAt) == "" {
		return model.CRSUsageWindow{}, false
	}

	return model.CRSUsageWindow{
		Key:           "quota",
		Label:         "额度",
		Progress:      clampCRSUsageWindowProgress(progress),
		RemainingText: remainingText,
		ResetAt:       strings.TrimSpace(snapshot.QuotaResetAt),
		Tone:          resolveCRSUsageWindowTone(progress, hasProgress, remainingText, snapshot.QuotaResetAt),
		Source:        "quota_balance",
	}, true
}

func normalizeCRSUsageWindow(key, label, source string, raw map[string]any) (model.CRSUsageWindow, bool) {
	if len(raw) == 0 {
		return model.CRSUsageWindow{}, false
	}

	progress, hasProgress := firstCRSUsageWindowNumber(raw, "utilization", "progress", "percentage")
	remainingText := firstCRSUsageWindowText(raw, "remainingText", "remainingTime", "remaining")
	if remainingText == "" {
		if value, ok := raw["remainingSeconds"]; ok {
			if seconds := getFloatValue(value); seconds > 0 {
				remainingText = formatCRSRemainingDuration(seconds)
			}
		}
	}
	resetAt := firstCRSUsageWindowText(raw, "resetsAt", "resetAt", "windowEnd", "windowEndAt")
	if !hasProgress && remainingText == "" && resetAt == "" {
		return model.CRSUsageWindow{}, false
	}

	return model.CRSUsageWindow{
		Key:           key,
		Label:         label,
		Progress:      progress,
		RemainingText: remainingText,
		ResetAt:       resetAt,
		Tone:          resolveCRSUsageWindowTone(progress, hasProgress, remainingText, resetAt),
		Source:        source,
	}, true
}

func firstCRSUsageWindowNumber(raw map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		return clampCRSUsageWindowProgress(getFloatValue(value)), true
	}
	return 0, false
}

func firstCRSUsageWindowText(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		text := getStringValue(value)
		if text != "" {
			return text
		}
	}
	return ""
}

func formatCRSRemainingDuration(seconds float64) string {
	total := int64(seconds)
	if total < 60 {
		return "不足 1 分钟"
	}
	if total < 3600 {
		return fmt.Sprintf("%d 分钟", total/60)
	}
	if total < 86400 {
		hours := total / 3600
		minutes := (total % 3600) / 60
		if minutes == 0 {
			return fmt.Sprintf("%d 小时", hours)
		}
		return fmt.Sprintf("%d 小时 %d 分钟", hours, minutes)
	}
	days := total / 86400
	hours := (total % 86400) / 3600
	if hours == 0 {
		return fmt.Sprintf("%d 天", days)
	}
	return fmt.Sprintf("%d 天 %d 小时", days, hours)
}

func resolveCRSUsageWindowTone(progress float64, hasProgress bool, remainingText, resetAt string) string {
	if !hasProgress {
		if strings.TrimSpace(remainingText) != "" {
			return "success"
		}
		if strings.TrimSpace(resetAt) != "" {
			return "info"
		}
		return "muted"
	}
	switch {
	case progress >= 90:
		return "danger"
	case progress >= 80:
		return "warning"
	case progress >= 45:
		return "info"
	default:
		return "success"
	}
}

func clampCRSUsageWindowProgress(progress float64) float64 {
	switch {
	case progress < 0:
		return 0
	case progress > 100:
		return 100
	default:
		return progress
	}
}

func formatCRSUsageWindowNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func marshalCRSUsageWindows(windows []model.CRSUsageWindow) string {
	if windows == nil {
		windows = []model.CRSUsageWindow{}
	}
	raw, err := common.Marshal(windows)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func marshalCRSObserverValue(value any) string {
	if value == nil {
		return ""
	}
	raw, err := common.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func getMapOrJSONValue(raw any) map[string]any {
	if mapped := getMapValue(raw); mapped != nil {
		return mapped
	}
	text := strings.TrimSpace(getStringValue(raw))
	if text == "" {
		return nil
	}
	parsed := make(map[string]any)
	if err := common.UnmarshalJsonStr(text, &parsed); err != nil {
		return nil
	}
	return parsed
}

func getMapValue(raw any) map[string]any {
	if mapped, ok := raw.(map[string]any); ok {
		return mapped
	}
	return nil
}

func getStringValue(raw any) string {
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case bool:
		if value {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func getBoolValue(raw any, fallback bool) bool {
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		if value == "" {
			return fallback
		}
		return strings.EqualFold(strings.TrimSpace(value), "true") || strings.TrimSpace(value) == "1"
	case float64:
		return value != 0
	case int:
		return value != 0
	default:
		return fallback
	}
}

func getIntValue(raw any) int {
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case float32:
		return int(value)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func getFloatValue(raw any) float64 {
	switch value := raw.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonZeroInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
