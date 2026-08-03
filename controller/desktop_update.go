package controller

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetDesktopUpdateStatus(c *gin.Context) {
	settings := service.GetDesktopUpdateSettings()
	configured, tokenSource := service.DesktopUpdatePublishTokenStatus()
	manifest, manifestErr := service.GetDesktopUpdateManifestSummary()
	releases, releasesErr := service.ListDesktopUpdateReleases(settings.PublicBaseURL)
	if releasesErr != nil {
		respondDesktopUpdateError(c, releasesErr)
		return
	}
	manifestError := ""
	if manifestErr != nil && !errors.Is(manifestErr, service.ErrDesktopUpdateNotFound) {
		manifestError = manifestErr.Error()
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"settings":         settings,
			"storage":          service.GetDesktopUpdateStorageStatus(),
			"token_configured": configured,
			"token_source":     tokenSource,
			"manifest":         manifest,
			"manifest_error":   manifestError,
			"release_count":    len(releases),
		},
	})
}

func UpdateDesktopUpdateSettings(c *gin.Context) {
	const maxSettingsBytes int64 = 64 * 1024
	if c.Request.ContentLength > maxSettingsBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "配置内容过大"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSettingsBytes)
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "配置内容过大"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	var settings service.DesktopUpdateSettings
	if err = common.Unmarshal(payload, &settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	validated, err := service.ValidateDesktopUpdateSettings(settings)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if validated.Enabled {
		if err = service.EnsureDesktopUpdateStorage(); err != nil {
			respondDesktopUpdateError(c, err)
			return
		}
	}
	previous := service.GetDesktopUpdateSettings()
	encoded, err := common.Marshal(validated)
	if err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	if err = model.UpdateOption(service.DesktopUpdateSettingsOptionKey, string(encoded)); err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	if validated.PublicBaseURL != "" && validated.PublicBaseURL != previous.PublicBaseURL {
		err = service.RepublishCurrentDesktopUpdateManifest(validated.PublicBaseURL, validated.RetentionCount)
		if err != nil && !errors.Is(err, service.ErrDesktopUpdateNotFound) {
			previousEncoded, marshalErr := common.Marshal(previous)
			if marshalErr == nil {
				if rollbackErr := model.UpdateOption(service.DesktopUpdateSettingsOptionKey, string(previousEncoded)); rollbackErr != nil {
					common.SysError("failed to rollback desktop update settings: " + rollbackErr.Error())
				}
			}
			respondDesktopUpdateError(c, err)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "保存成功", "data": validated})
}

func RotateDesktopUpdatePublishToken(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	token, tokenHash, err := service.GenerateDesktopUpdatePublishToken()
	if err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	if err = model.UpdateOption(service.DesktopUpdateTokenOptionKey, tokenHash); err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "发布令牌已轮换，请立即保存",
		"data":    gin.H{"token": token},
	})
}

func ListDesktopUpdateReleases(c *gin.Context) {
	settings := service.GetDesktopUpdateSettings()
	releases, err := service.ListDesktopUpdateReleases(settings.PublicBaseURL)
	if err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": releases})
}

func UploadDesktopUpdateArtifact(c *gin.Context) {
	settings := service.GetDesktopUpdateSettings()
	maxBytes := int64(settings.MaxUploadMB) * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes+1024*1024)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || strings.Contains(strings.ToLower(err.Error()), "request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "上传文件超过大小限制"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请选择要上传的更新文件"})
		}
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	defer file.Close()
	uploaded, err := service.SaveDesktopUpdateArtifact(c.Param("version"), fileHeader.Filename, file, maxBytes)
	if err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "上传成功", "data": uploaded})
}

func PublishDesktopUpdateManifest(c *gin.Context) {
	settings := service.GetDesktopUpdateSettings()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024*1024)
	manifest, err := service.PublishDesktopUpdateManifest(c.Request.Body, settings.PublicBaseURL, settings.RetentionCount)
	if err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "版本发布成功", "data": manifest})
}

func DeleteDesktopUpdateRelease(c *gin.Context) {
	if err := service.DeleteDesktopUpdateRelease(c.Param("version")); err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "版本已删除"})
}

func ServeDesktopUpdateManifest(c *gin.Context) {
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	if !service.GetDesktopUpdateSettings().Enabled {
		c.Status(http.StatusNotFound)
		return
	}
	if _, err := service.GetDesktopUpdateManifestSummary(); err != nil {
		respondDesktopUpdatePublicError(c, err)
		return
	}
	file, info, err := service.OpenDesktopUpdateManifest()
	if err != nil {
		respondDesktopUpdatePublicError(c, err)
		return
	}
	defer file.Close()
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("ETag", desktopUpdateETag(info))
	http.ServeContent(c.Writer, c.Request, "latest.json", info.ModTime(), file)
}

func ServeDesktopDownloadCatalog(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	settings := service.GetDesktopUpdateSettings()
	if !settings.Enabled {
		c.Status(http.StatusNotFound)
		return
	}
	catalog, err := service.GetDesktopDownloadCatalog(settings.PublicBaseURL)
	if err != nil {
		respondDesktopUpdatePublicError(c, err)
		return
	}
	payload, err := common.Marshal(catalog)
	if err != nil {
		respondDesktopUpdatePublicError(c, err)
		return
	}
	etag := fmt.Sprintf(`"%x"`, sha256.Sum256(payload))
	c.Header("Cache-Control", "public, max-age=300, must-revalidate")
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Length", strconv.Itoa(len(payload)))
	c.Header("ETag", etag)
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	if c.Request.Method == http.MethodHead {
		c.Status(http.StatusOK)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

func ServeDesktopUpdateArtifact(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	if !service.GetDesktopUpdateSettings().Enabled {
		c.Status(http.StatusNotFound)
		return
	}
	file, info, err := service.OpenDesktopUpdateArtifact(c.Param("version"), c.Param("filename"))
	if err != nil {
		respondDesktopUpdatePublicError(c, err)
		return
	}
	defer file.Close()
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("ETag", desktopUpdateETag(info))
	if disposition := mime.FormatMediaType("attachment", map[string]string{"filename": info.Name()}); disposition != "" {
		c.Header("Content-Disposition", disposition)
	}
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
}

func PublishDesktopUpdateArtifact(c *gin.Context) {
	settings := service.GetDesktopUpdateSettings()
	if !settings.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": service.ErrDesktopUpdateDisabled.Error()})
		return
	}
	if !authorizeDesktopUpdatePublisher(c) {
		return
	}
	maxBytes := int64(settings.MaxUploadMB) * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	uploaded, err := service.SaveDesktopUpdateArtifact(c.Param("version"), c.Param("filename"), c.Request.Body, maxBytes)
	if err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "上传成功", "data": uploaded})
}

func PublishDesktopUpdateManifestWithToken(c *gin.Context) {
	settings := service.GetDesktopUpdateSettings()
	if !settings.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": service.ErrDesktopUpdateDisabled.Error()})
		return
	}
	if !authorizeDesktopUpdatePublisher(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024*1024)
	manifest, err := service.PublishDesktopUpdateManifest(c.Request.Body, settings.PublicBaseURL, settings.RetentionCount)
	if err != nil {
		respondDesktopUpdateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "版本发布成功", "data": manifest})
}

func authorizeDesktopUpdatePublisher(c *gin.Context) bool {
	configured, _ := service.DesktopUpdatePublishTokenStatus()
	if !configured {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "桌面更新发布令牌未配置"})
		return false
	}
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || !service.ValidateDesktopUpdatePublishToken(strings.TrimSpace(parts[1])) {
		c.Header("WWW-Authenticate", "Bearer")
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "发布令牌无效"})
		return false
	}
	return true
}

func desktopUpdateETag(info os.FileInfo) string {
	return fmt.Sprintf(`W/"%x-%x"`, info.Size(), info.ModTime().UnixNano())
}

func respondDesktopUpdatePublicError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrDesktopUpdateNotFound) {
		c.Status(http.StatusNotFound)
		return
	}
	common.SysError("desktop update public error: " + err.Error())
	c.Status(http.StatusInternalServerError)
}

func respondDesktopUpdateError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrDesktopUpdateNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrDesktopUpdateConflict), errors.Is(err, service.ErrDesktopUpdateImmutable):
		status = http.StatusConflict
	case errors.Is(err, service.ErrDesktopUpdateDisabled):
		status = http.StatusServiceUnavailable
	default:
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			status = http.StatusRequestEntityTooLarge
		}
		message := strings.ToLower(err.Error())
		if status == http.StatusInternalServerError && (strings.Contains(message, "request body too large") || strings.Contains(message, "超过")) {
			status = http.StatusRequestEntityTooLarge
		} else if strings.Contains(message, "不合法") || strings.Contains(message, "无效") || strings.Contains(message, "不是有效") || strings.Contains(message, "必须") || strings.Contains(message, "缺少") || strings.Contains(message, "不能为空") || strings.Contains(message, "尚未上传") || strings.Contains(message, "不支持") || strings.Contains(message, "请先配置") {
			status = http.StatusBadRequest
		}
	}
	if status >= 500 {
		common.SysError("desktop update error: " + err.Error())
	}
	c.JSON(status, gin.H{"success": false, "message": err.Error()})
}
