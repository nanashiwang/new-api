package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/gin-gonic/gin"
)

const registerUserAgentMaxLength = 255

func applyUserRegisterAudit(c *gin.Context, user *model.User, source string) {
	if user == nil {
		return
	}
	user.RegisterSource = model.NormalizeUserRegisterSource(source)
	if c == nil || c.Request == nil {
		return
	}
	user.RegisterIP = truncateRegisterAuditValue(c.ClientIP(), 64)
	user.RegisterUserAgent = truncateRegisterAuditValue(c.Request.UserAgent(), registerUserAgentMaxLength)
}

func oauthRegisterSource(provider oauth.Provider) string {
	if genericProvider, ok := provider.(*oauth.GenericOAuthProvider); ok {
		config := genericProvider.GetConfig()
		if config != nil && strings.TrimSpace(config.Slug) != "" {
			return model.UserRegisterSourceCustomOAuthPrefix + config.Slug
		}
		return model.UserRegisterSourceCustomOAuthPrefix + "unknown"
	}

	switch strings.ToLower(strings.ReplaceAll(provider.GetName(), " ", "")) {
	case "github":
		return model.UserRegisterSourceGitHub
	case "discord":
		return model.UserRegisterSourceDiscord
	case "oidc":
		return model.UserRegisterSourceOIDC
	case "linuxdo":
		return model.UserRegisterSourceLinuxDO
	default:
		return model.UserRegisterSourceUnknown
	}
}

func truncateRegisterAuditValue(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	if maxLength <= 0 || value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	return string(runes[:maxLength])
}
