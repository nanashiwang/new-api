package controller

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const (
	registerRiskWindowSeconds          int64 = 24 * 60 * 60
	registerRiskSameIPMaxCount               = 2
	registerRiskSameIPv4C24MaxCount          = 5
	registerRiskSameUAEmailDomainCount       = 3
)

var errRegisterRiskBlocked = errors.New("注册环境异常，请稍后再试或联系管理员")

func enforceUserRegisterRisk(c *gin.Context, user *model.User) error {
	if c == nil || c.Request == nil || user == nil || model.DB == nil {
		return nil
	}

	cutoff := time.Now().Unix() - registerRiskWindowSeconds
	clientIP := strings.TrimSpace(c.ClientIP())
	userAgent := truncateRegisterAuditValue(c.Request.UserAgent(), registerUserAgentMaxLength)
	emailDomain := normalizeRegisterEmailDomain(user.Email)

	if clientIP != "" {
		count, err := countRecentRegisteredUsers(cutoff, "register_ip = ?", clientIP)
		if err != nil {
			return err
		}
		if count >= registerRiskSameIPMaxCount {
			logRegisterRiskBlock("same_ip", count, clientIP, emailDomain, userAgent)
			return errRegisterRiskBlocked
		}

		if pattern, ok := ipv4C24LikePattern(clientIP); ok {
			count, err = countRecentRegisteredUsers(cutoff, "register_ip LIKE ?", pattern)
			if err != nil {
				return err
			}
			if count >= registerRiskSameIPv4C24MaxCount {
				logRegisterRiskBlock("same_ipv4_c24", count, clientIP, emailDomain, userAgent)
				return errRegisterRiskBlocked
			}
		}
	}

	if userAgent != "" && emailDomain != "" {
		count, err := countRecentRegisteredUsers(
			cutoff,
			"register_user_agent = ? AND LOWER(email) LIKE ?",
			userAgent,
			"%@"+emailDomain,
		)
		if err != nil {
			return err
		}
		if count >= registerRiskSameUAEmailDomainCount {
			logRegisterRiskBlock("same_ua_email_domain", count, clientIP, emailDomain, userAgent)
			return errRegisterRiskBlocked
		}
	}

	return nil
}

func countRecentRegisteredUsers(cutoff int64, query any, args ...any) (int64, error) {
	var count int64
	err := model.DB.Unscoped().
		Model(&model.User{}).
		Where("created_at >= ?", cutoff).
		Where(query, args...).
		Count(&count).Error
	return count, err
}

func normalizeRegisterEmailDomain(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return strings.TrimSpace(email[at+1:])
}

func ipv4C24LikePattern(ip string) (string, bool) {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return "", false
	}
	v4 := parsed.To4()
	if v4 == nil {
		return "", false
	}
	return fmt.Sprintf("%d.%d.%d.%%", v4[0], v4[1], v4[2]), true
}

func logRegisterRiskBlock(rule string, count int64, ip string, emailDomain string, userAgent string) {
	common.SysLog(fmt.Sprintf(
		"registration risk blocked: rule=%s count=%d ip=%s email_domain=%s ua=%s",
		rule,
		count,
		ip,
		emailDomain,
		truncateRegisterAuditValue(userAgent, 120),
	))
}
