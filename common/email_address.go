package common

import (
	"errors"
	"net"
	"net/mail"
	"strings"
	"unicode"
)

var ErrInvalidEmailAddress = errors.New("邮箱地址格式不正确")

func NormalizeEmailAddress(email string) string {
	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	localPart := strings.TrimSpace(parts[0])
	domainPart := strings.ToLower(strings.TrimSpace(parts[1]))
	if localPart == "" || domainPart == "" {
		return email
	}
	return localPart + "@" + domainPart
}

// ValidateEmailAddress 校验单个邮箱地址格式，不执行 DNS/MX 查询。
// 需要持久化时，调用方应先使用 NormalizeEmailAddress 统一域名大小写。
func ValidateEmailAddress(email string) error {
	email = strings.TrimSpace(email)
	if email == "" || len([]byte(email)) > 254 || strings.Count(email, "@") != 1 {
		return ErrInvalidEmailAddress
	}
	for _, r := range email {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return ErrInvalidEmailAddress
		}
	}

	parts := strings.SplitN(email, "@", 2)
	localPart, domainPart := parts[0], parts[1]
	if localPart == "" || domainPart == "" || len([]byte(localPart)) > 64 || len([]byte(domainPart)) > 253 {
		return ErrInvalidEmailAddress
	}
	if !isValidEmailDomain(domainPart) {
		return ErrInvalidEmailAddress
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return ErrInvalidEmailAddress
	}
	return nil
}

func isValidEmailDomain(domain string) bool {
	if strings.HasPrefix(domain, "[") || strings.HasSuffix(domain, "]") {
		if !strings.HasPrefix(domain, "[") || !strings.HasSuffix(domain, "]") || len(domain) <= 2 {
			return false
		}
		literal := strings.TrimSuffix(strings.TrimPrefix(domain, "["), "]")
		if strings.HasPrefix(strings.ToLower(literal), "ipv6:") {
			literal = literal[len("IPv6:"):]
		}
		return net.ParseIP(literal) != nil
	}
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if label == "" || len([]byte(label)) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, r := range label {
			if r != '-' && !unicode.IsLetter(r) && !unicode.IsNumber(r) {
				return false
			}
		}
	}
	return true
}

func NormalizeEmailDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

func EmailDomainInWhitelist(domain string, whitelist []string) bool {
	domain = NormalizeEmailDomain(domain)
	if domain == "" {
		return false
	}
	for _, allowedDomain := range whitelist {
		if domain == NormalizeEmailDomain(allowedDomain) {
			return true
		}
	}
	return false
}
