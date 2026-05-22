package controller

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	imagePlaygroundDefaultModel    = "gpt-image-2"
	imagePlaygroundModelLimits     = "gpt-image-2,gpt-image-2-2026-04-21,gpt-image-1,gpt-image-1-mini,gpt-image-1.5,chatgpt-image-latest"
	imagePlaygroundSessionDuration = 2 * time.Hour
	imagePlaygroundRefreshWindow   = 15 * time.Minute
)

type imagePlaygroundSessionResponse struct {
	URL       string `json:"url"`
	ExpiresAt int64  `json:"expires_at"`
	Model     string `json:"model"`
}

func CreateImagePlaygroundSession(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未登录")
		return
	}

	now := common.GetTimestamp()
	if err := model.DeleteExpiredImagePlaygroundTokens(userId, now); err != nil {
		common.SysLog("failed to delete expired image playground tokens: " + err.Error())
	}

	minExpiredTime := now + int64(imagePlaygroundRefreshWindow/time.Second)
	token, err := model.GetReusableImagePlaygroundToken(userId, minExpiredTime)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiError(c, err)
			return
		}
		token, err = createImagePlaygroundToken(userId, now)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}

	origin := buildImagePlaygroundOrigin(c)
	if origin == "" {
		common.ApiErrorMsg(c, "无法识别当前站点地址，请先配置服务器地址")
		return
	}

	common.ApiSuccess(c, imagePlaygroundSessionResponse{
		URL:       buildImagePlaygroundLaunchURL(origin, token.Key),
		ExpiresAt: token.ExpiredTime,
		Model:     imagePlaygroundDefaultModel,
	})
}

func createImagePlaygroundToken(userId int, now int64) (*model.Token, error) {
	key, err := common.GenerateKey()
	if err != nil {
		return nil, err
	}
	group, err := model.GetUserGroup(userId, false)
	if err != nil {
		return nil, err
	}
	group = strings.TrimSpace(group)
	if group == "" {
		group = "default"
	}

	token := &model.Token{
		UserId:               userId,
		Key:                  key,
		Status:               common.TokenStatusEnabled,
		Name:                 "image-playground",
		SourceType:           model.TokenSourceTypeImagePlayground,
		CreatedTime:          now,
		AccessedTime:         now,
		ExpiredTime:          now + int64(imagePlaygroundSessionDuration/time.Second),
		UnlimitedQuota:       true,
		ModelLimitsEnabled:   true,
		ModelLimits:          imagePlaygroundModelLimits,
		Group:                group,
		MaxConcurrency:       2,
		WindowRequestLimit:   60,
		WindowSeconds:        60,
		PackagePeriod:        model.TokenPackagePeriodNone,
		PackagePeriodMode:    model.TokenPackagePeriodModeRelative,
		PackageNextResetTime: 0,
	}
	if err := model.ValidateTokenRuntimeLimitConfig(token); err != nil {
		return nil, err
	}
	if err := token.Insert(); err != nil {
		return nil, err
	}
	return token, nil
}

func buildImagePlaygroundOrigin(c *gin.Context) string {
	if serverAddress := strings.TrimSpace(system_setting.ServerAddress); serverAddress != "" {
		return strings.TrimRight(serverAddress, "/")
	}

	host := firstHeaderValue(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		return ""
	}

	proto := firstHeaderValue(c.GetHeader("X-Forwarded-Proto"))
	if proto == "" {
		if strings.EqualFold(c.GetHeader("X-Forwarded-Ssl"), "on") || c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	return proto + "://" + host
}

func firstHeaderValue(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	return strings.TrimSpace(parts[0])
}

func buildImagePlaygroundLaunchURL(origin string, key string) string {
	u := url.URL{
		Scheme: "http",
		Host:   "localhost",
		Path:   "/image-playground/",
	}
	if parsed, err := url.Parse(origin + "/image-playground/"); err == nil {
		u = *parsed
	}

	q := u.Query()
	q.Set("apiUrl", origin+"/v1")
	q.Set("apiKey", "sk-"+key)
	q.Set("model", imagePlaygroundDefaultModel)
	q.Set("apiMode", "images")
	u.RawQuery = q.Encode()
	return u.String()
}
