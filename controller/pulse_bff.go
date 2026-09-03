package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	pulseBFFRole             = "new-api"
	pulseBFFSummaryPath      = "/v1/internal/me/summary"
	pulseBFFRewardsPath      = "/v1/internal/me/rewards"
	pulseBFFMaxResponseBytes = 2 << 20
	pulseBFFUpstreamTimeout  = 3 * time.Second
	pulseHeaderUserID        = "X-Pulse-User-Id"
	pulseHeaderRole          = "X-Pulse-Role"
	pulseHeaderTimestamp     = "X-Pulse-Timestamp"
	pulseHeaderNonce         = "X-Pulse-Nonce"
	pulseHeaderSignature     = "X-Pulse-Signature"
)

var pulseBFFHTTPClient = &http.Client{Timeout: pulseBFFUpstreamTimeout}

type pulseHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// GetPulseSummary proxies the authenticated user's read-only Pulse summary.
// User identity comes exclusively from middleware.UserAuth's context.
func GetPulseSummary(c *gin.Context) {
	if len(c.Request.URL.Query()) != 0 {
		writePulseBFFBadRequest(c, "summary 不支持查询参数")
		return
	}
	proxyPulseRead(c, pulseBFFSummaryPath, nil, pulseBFFHTTPClient)
}

// GetPulseRewards proxies the authenticated user's reward history. Only limit
// is accepted; user_id and all other browser-controlled selectors are rejected.
func GetPulseRewards(c *gin.Context) {
	query := c.Request.URL.Query()
	for key := range query {
		if key != "limit" {
			writePulseBFFBadRequest(c, "不支持的查询参数")
			return
		}
	}
	upstreamQuery := make(url.Values)
	if values, ok := query["limit"]; ok {
		if len(values) != 1 {
			writePulseBFFBadRequest(c, "limit 参数无效")
			return
		}
		limit, err := strconv.Atoi(values[0])
		if err != nil || limit < 1 || limit > 100 {
			writePulseBFFBadRequest(c, "limit 必须是 1 到 100 的整数")
			return
		}
		upstreamQuery.Set("limit", strconv.Itoa(limit))
	}
	proxyPulseRead(c, pulseBFFRewardsPath, upstreamQuery, pulseBFFHTTPClient)
}

func proxyPulseRead(c *gin.Context, upstreamPath string, query url.Values, client pulseHTTPDoer) {
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	baseURL, secret, err := pulseBFFConfig()
	if err != nil {
		common.SysError("Pulse BFF 配置无效：" + err.Error())
		writePulseBFFUnavailable(c)
		return
	}

	target := *baseURL
	target.Path = strings.TrimRight(baseURL.Path, "/") + upstreamPath
	target.RawPath = ""
	target.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		writePulseBFFUnavailable(c)
		return
	}
	user := strconv.Itoa(userID)
	timestamp := time.Now().Unix()
	nonce, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		common.SysError("生成 Pulse BFF nonce 失败：" + err.Error())
		writePulseBFFUnavailable(c)
		return
	}
	canonical := pulseBFFCanonicalPayload(req.Method, req.URL.EscapedPath(), user, pulseBFFRole, timestamp, nonce, nil)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	req.Header.Set("Accept", "application/json")
	req.Header.Set(pulseHeaderUserID, user)
	req.Header.Set(pulseHeaderRole, pulseBFFRole)
	req.Header.Set(pulseHeaderTimestamp, strconv.FormatInt(timestamp, 10))
	req.Header.Set(pulseHeaderNonce, nonce)
	req.Header.Set(pulseHeaderSignature, hex.EncodeToString(mac.Sum(nil)))

	response, err := client.Do(req)
	if err != nil {
		common.SysError("Pulse BFF 请求失败：" + err.Error())
		writePulseBFFUnavailable(c)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, pulseBFFMaxResponseBytes+1))
	if err != nil || len(body) > pulseBFFMaxResponseBytes {
		common.SysError("Pulse BFF 响应读取失败或超过上限")
		writePulseBFFUnavailable(c)
		return
	}
	if response.StatusCode != http.StatusOK || !common.Valid(body) {
		common.SysError(fmt.Sprintf("Pulse BFF 上游响应无效：status=%d", response.StatusCode))
		writePulseBFFUnavailable(c)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}

func pulseBFFConfig() (*url.URL, string, error) {
	rawURL := strings.TrimSpace(os.Getenv("PULSE_INTERNAL_URL"))
	secret := strings.TrimSpace(os.Getenv("PULSE_USER_BFF_HMAC_SECRET"))
	if rawURL == "" || secret == "" {
		return nil, "", errors.New("PULSE_INTERNAL_URL and PULSE_USER_BFF_HMAC_SECRET are required")
	}
	baseURL, err := url.Parse(rawURL)
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, "", errors.New("PULSE_INTERNAL_URL must be an http(s) URL without credentials, query, or fragment")
	}
	return baseURL, secret, nil
}

func pulseBFFCanonicalPayload(method, path, userID, role string, timestamp int64, nonce string, body []byte) string {
	digest := sha256.Sum256(body)
	return strings.Join([]string{
		strings.ToUpper(strings.TrimSpace(method)), path, userID, role,
		strconv.FormatInt(timestamp, 10), nonce, hex.EncodeToString(digest[:]),
	}, "\n")
}

func writePulseBFFBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": message})
}

func writePulseBFFUnavailable(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Pulse 服务暂不可用"})
}
