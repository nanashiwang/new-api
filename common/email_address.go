package common

import "strings"

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
