package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestBuildImagePlaygroundOriginPrefersRequestHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://nan.meta-api.vip"
	defer func() {
		system_setting.ServerAddress = originalServerAddress
	}()

	req := httptest.NewRequest(http.MethodPost, "http://internal/api/image-playground/session", nil)
	req.Header.Set("X-Forwarded-Host", "cn.meta-api.vip")
	req.Header.Set("X-Forwarded-Proto", "https")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	if got := buildImagePlaygroundOrigin(c); got != "https://cn.meta-api.vip" {
		t.Fatalf("buildImagePlaygroundOrigin() = %q, want %q", got, "https://cn.meta-api.vip")
	}
}

func TestBuildImagePlaygroundOriginPrefersTrustedOriginHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://nan.meta-api.vip"
	defer func() {
		system_setting.ServerAddress = originalServerAddress
	}()

	req := httptest.NewRequest(http.MethodPost, "http://nan.meta-api.vip/api/image-playground/session", nil)
	req.Header.Set("Origin", "https://cn.meta-api.vip")
	req.Header.Set("X-Forwarded-Proto", "https")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	if got := buildImagePlaygroundOrigin(c); got != "https://cn.meta-api.vip" {
		t.Fatalf("buildImagePlaygroundOrigin() = %q, want %q", got, "https://cn.meta-api.vip")
	}
}

func TestBuildImagePlaygroundOriginRejectsUntrustedOriginHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://nan.meta-api.vip"
	defer func() {
		system_setting.ServerAddress = originalServerAddress
	}()

	req := httptest.NewRequest(http.MethodPost, "http://nan.meta-api.vip/api/image-playground/session", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	if got := buildImagePlaygroundOrigin(c); got != "https://nan.meta-api.vip" {
		t.Fatalf("buildImagePlaygroundOrigin() = %q, want %q", got, "https://nan.meta-api.vip")
	}
}

func TestBuildImagePlaygroundOriginFallsBackToServerAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://nan.meta-api.vip/"
	defer func() {
		system_setting.ServerAddress = originalServerAddress
	}()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Host = ""
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	if got := buildImagePlaygroundOrigin(c); got != "https://nan.meta-api.vip" {
		t.Fatalf("buildImagePlaygroundOrigin() = %q, want %q", got, "https://nan.meta-api.vip")
	}
}

func TestBuildImagePlaygroundLaunchURLForcesGalleryImagesMode(t *testing.T) {
	launchURL := buildImagePlaygroundLaunchURL("https://cn.meta-api.vip", "test-key")
	parsed, err := url.Parse(launchURL)
	if err != nil {
		t.Fatalf("parse launch url: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "cn.meta-api.vip" || parsed.Path != "/image-playground/" {
		t.Fatalf("unexpected launch url: %s", launchURL)
	}
	query := parsed.Query()
	if query.Get("apiMode") != "images" {
		t.Fatalf("apiMode = %q, want images", query.Get("apiMode"))
	}
	if query.Get("appMode") != "gallery" {
		t.Fatalf("appMode = %q, want gallery", query.Get("appMode"))
	}
	if query.Get("model") != imagePlaygroundDefaultModel {
		t.Fatalf("model = %q, want %s", query.Get("model"), imagePlaygroundDefaultModel)
	}
}

func setupImagePlaygroundGroupTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false

	if err := db.AutoMigrate(&model.User{}, &model.Token{}, &model.Ability{}, &model.Channel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if err := setting.UpdateUserUsableGroupsByJSONString(`{"default":"默认分组","vip":"VIP","team":"团队"}`); err != nil {
		t.Fatalf("update user usable groups: %v", err)
	}
	if err := setting.UpdateAutoGroupsByJsonString(`["default","vip"]`); err != nil {
		t.Fatalf("update auto groups: %v", err)
	}
	if err := ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"team":1}`); err != nil {
		t.Fatalf("update group ratio: %v", err)
	}
}

func TestResolveImagePlaygroundTokenGroupPrefersModelEnabledGroup(t *testing.T) {
	setupImagePlaygroundGroupTestDB(t)

	if err := model.DB.Create(&model.User{
		Id:       1,
		Username: "image-user",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		AffCode:  "aff-image-user",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := model.DB.Create(&model.Channel{
		Id:     10,
		Name:   "vip-image",
		Key:    "sk-image",
		Group:  "vip",
		Models: imagePlaygroundDefaultModel,
		Status: common.ChannelStatusEnabled,
	}).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	if err := model.DB.Create(&model.Ability{
		Group:     "vip",
		Model:     imagePlaygroundDefaultModel,
		ChannelId: 10,
		Enabled:   true,
	}).Error; err != nil {
		t.Fatalf("seed ability: %v", err)
	}

	group, err := resolveImagePlaygroundTokenGroup(1)
	if err != nil {
		t.Fatalf("resolve image playground group: %v", err)
	}
	if group != "vip" {
		t.Fatalf("resolveImagePlaygroundTokenGroup() = %q, want vip", group)
	}
}

func TestResolveImagePlaygroundTokenGroupFallsBackToUserGroup(t *testing.T) {
	setupImagePlaygroundGroupTestDB(t)

	if err := model.DB.Create(&model.User{
		Id:       1,
		Username: "image-user",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		AffCode:  "aff-image-user",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	group, err := resolveImagePlaygroundTokenGroup(1)
	if err != nil {
		t.Fatalf("resolve image playground group: %v", err)
	}
	if group != "default" {
		t.Fatalf("resolveImagePlaygroundTokenGroup() = %q, want default", group)
	}
}
