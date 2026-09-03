package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	forumSSOStartPath = "/api/forum/sso/start"
	forumSSOTicketTTL = 2 * time.Minute
)

// ForumSSOStart is the only new-api entry point used by the Answer user-center
// plugin. It derives the user from the existing session, never from query
// parameters, then redirects to the one configured Answer callback.
func ForumSSOStart(c *gin.Context) {
	callback, secret, err := forumSSOConfig()
	if err != nil {
		common.SysError("forum SSO is not configured: " + err.Error())
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "论坛登录暂不可用"})
		return
	}

	session := sessions.Default(c)
	id, ok := sessionInt(session.Get("id"))
	if !ok || id <= 0 {
		loginURL := "/login?next=" + url.QueryEscape(forumSSOStartPath)
		c.Redirect(http.StatusFound, loginURL)
		return
	}
	status, _ := sessionInt(session.Get("status"))
	if status == common.UserStatusDisabled {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "用户已被封禁"})
		return
	}
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil || user.Id <= 0 || user.Status == common.UserStatusDisabled {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "用户不可用"})
		return
	}

	nonce, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "生成登录凭证失败"})
		return
	}
	ticket := forumSSOTicket{
		UserID:      strconv.Itoa(user.Id),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		Avatar:      "",
		Timestamp:   time.Now().Unix(),
		Nonce:       nonce,
	}
	ticket.Signature = ticket.sign(secret)
	target, err := url.Parse(callback)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "论坛回调地址无效"})
		return
	}
	query := target.Query()
	query.Set("user_id", ticket.UserID)
	query.Set("username", ticket.Username)
	query.Set("display_name", ticket.DisplayName)
	query.Set("email", ticket.Email)
	query.Set("avatar", ticket.Avatar)
	query.Set("timestamp", strconv.FormatInt(ticket.Timestamp, 10))
	query.Set("nonce", ticket.Nonce)
	query.Set("signature", ticket.Signature)
	target.RawQuery = query.Encode()
	c.Redirect(http.StatusFound, target.String())
}

type forumSSOTicket struct {
	UserID      string
	Username    string
	DisplayName string
	Email       string
	Avatar      string
	Timestamp   int64
	Nonce       string
	Signature   string
}

func (t forumSSOTicket) sign(secret string) string {
	return signHMAC(secret, strings.Join([]string{
		t.UserID, t.Username, t.DisplayName, t.Email, t.Avatar,
		strconv.FormatInt(t.Timestamp, 10), t.Nonce,
	}, "\n"))
}

func signHMAC(secret, payload string) string {
	// Use the same standard-library HMAC implementation as Pulse's verifier.
	// Keeping this helper local avoids coupling new-api to the Pulse module.
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func forumSSOConfig() (string, string, error) {
	callback := strings.TrimSpace(os.Getenv("PULSE_FORUM_SSO_CALLBACK_URL"))
	secret := strings.TrimSpace(os.Getenv("PULSE_FORUM_SSO_SECRET"))
	if callback == "" || secret == "" {
		return "", "", errors.New("PULSE_FORUM_SSO_CALLBACK_URL and PULSE_FORUM_SSO_SECRET are required")
	}
	target, err := url.Parse(callback)
	if err != nil || target.Scheme != "https" || target.Host == "" || target.Fragment != "" {
		return "", "", errors.New("PULSE_FORUM_SSO_CALLBACK_URL must be an https URL without fragment")
	}
	return callback, secret, nil
}

func sessionInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), typed == float64(int(typed))
	case string:
		parsed, err := strconv.Atoi(typed)
		return parsed, err == nil
	default:
		return 0, false
	}
}

// GrantPulseBenefit receives the idempotent quota grant from Pulse.
func GrantPulseBenefit(c *gin.Context) {
	var request model.PulseBenefitGrantRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	result, err := model.GrantPulseBenefit(request)
	if err != nil {
		writePulseBenefitError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// QueryPulseBenefit is deliberately read-only and is safe to call after a
// network timeout. It accepts the POST shape used by Pulse and the documented
// GET form for operators and probes.
func QueryPulseBenefit(c *gin.Context) {
	sourceRef := strings.TrimSpace(c.Param("source_ref"))
	if sourceRef == "" && c.Request.Method == http.MethodPost {
		var request struct {
			SourceRef string `json:"source_ref"`
		}
		if err := common.DecodeJson(c.Request.Body, &request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
			return
		}
		sourceRef = strings.TrimSpace(request.SourceRef)
	}
	result, err := model.QueryPulseBenefit(sourceRef)
	if err != nil {
		if errors.Is(err, model.ErrPulseBenefitNotFound) {
			c.JSON(http.StatusNotFound, result)
			return
		}
		writePulseBenefitError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// RollbackPulseBenefit creates a reversal for the original source_ref. It
// never accepts a replacement source reference and remains safe to retry.
func RollbackPulseBenefit(c *gin.Context) {
	var request struct {
		SourceRef string `json:"source_ref"`
		Reason    string `json:"reason"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	result, err := model.RollbackPulseBenefit(request.SourceRef, request.Reason)
	if err != nil {
		writePulseBenefitError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func writePulseBenefitError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrPulseBenefitConflict):
		c.JSON(http.StatusConflict, gin.H{"success": false, "code": "payload_conflict", "message": "source_ref 对应的请求内容不一致"})
	case errors.Is(err, model.ErrPulseBenefitNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "not_found", "message": "奖励不存在"})
	default:
		if strings.HasPrefix(err.Error(), "invalid pulse benefit") ||
			strings.HasPrefix(err.Error(), "pulse grant_id") ||
			strings.HasPrefix(err.Error(), "pulse benefit source_ref") ||
			strings.HasPrefix(err.Error(), "pulse benefit rollback requires") {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
			return
		}
		common.SysError("pulse benefit request failed: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "奖励处理失败"})
	}
}
