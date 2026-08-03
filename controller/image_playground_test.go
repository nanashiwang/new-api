package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
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
	launchURL := buildImagePlaygroundLaunchURL("https://cn.meta-api.vip", "test-key", true)
	parsed, err := url.Parse(launchURL)
	if err != nil {
		t.Fatalf("parse launch url: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "cn.meta-api.vip" || parsed.Path != "/image-playground/" {
		t.Fatalf("unexpected launch url: %s", launchURL)
	}
	query := parsed.Query()
	if query.Get("appMode") != "gallery" {
		t.Fatalf("appMode = %q, want gallery", query.Get("appMode"))
	}
	var settings imagePlaygroundLaunchSettings
	if err := common.Unmarshal([]byte(query.Get("settings")), &settings); err != nil {
		t.Fatalf("unmarshal launch settings: %v", err)
	}
	if settings.ActiveProfileID != "newapi-image-playground" {
		t.Fatalf("activeProfileId = %q, want newapi-image-playground", settings.ActiveProfileID)
	}
	if settings.DefaultImageModel != imagePlaygroundDefaultModel {
		t.Fatalf("defaultImageModel = %q, want %s", settings.DefaultImageModel, imagePlaygroundDefaultModel)
	}
	if settings.DefaultPlanModel != imagePlaygroundAgentModel {
		t.Fatalf("defaultPlanModel = %q, want %s", settings.DefaultPlanModel, imagePlaygroundAgentModel)
	}
	if !settings.SupportsEcommerce {
		t.Fatal("supportsEcommerce = false, want true")
	}
	if len(settings.Profiles) != 1 {
		t.Fatalf("profiles length = %d, want 1", len(settings.Profiles))
	}
	profile := settings.Profiles[0]
	if profile.APIMode != "images" {
		t.Fatalf("apiMode = %q, want images", profile.APIMode)
	}
	if profile.Model != imagePlaygroundDefaultModel {
		t.Fatalf("model = %q, want %s", profile.Model, imagePlaygroundDefaultModel)
	}
	if profile.StreamImages {
		t.Fatal("streamImages = true, want false")
	}
	if !profile.ResponseFormatB64Json {
		t.Fatal("responseFormatB64Json = false, want true")
	}
	if profile.BaseURL != "https://cn.meta-api.vip/v1" {
		t.Fatalf("baseUrl = %q, want https://cn.meta-api.vip/v1", profile.BaseURL)
	}
	if profile.APIKey != "sk-test-key" {
		t.Fatalf("apiKey = %q, want sk-test-key", profile.APIKey)
	}
}

func setupImagePlaygroundGroupTestDB(t *testing.T) {
	t.Helper()

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalUserUsableGroups := setting.UserUsableGroups2JSONString()
	originalAutoGroups := setting.AutoGroups2JsonString()
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.RedisEnabled = originalRedisEnabled
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		_ = setting.UpdateUserUsableGroupsByJSONString(originalUserUsableGroups)
		_ = setting.UpdateAutoGroupsByJsonString(originalAutoGroups)
		_ = ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio)
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
	})

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

func TestBuildImagePlaygroundGroupCandidatesExcludesRemovedDefaultGroup(t *testing.T) {
	setupImagePlaygroundGroupTestDB(t)

	if err := setting.UpdateUserUsableGroupsByJSONString(`{"vip":"VIP","team":"团队"}`); err != nil {
		t.Fatalf("update user usable groups: %v", err)
	}
	if err := setting.UpdateAutoGroupsByJsonString(`["default","team","vip"]`); err != nil {
		t.Fatalf("update auto groups: %v", err)
	}
	if err := ratio_setting.UpdateGroupRatioByJSONString(`{"vip":1,"team":1}`); err != nil {
		t.Fatalf("update group ratio: %v", err)
	}

	got := buildImagePlaygroundGroupCandidates("default")
	want := []string{"team", "vip"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildImagePlaygroundGroupCandidates() = %v, want %v", got, want)
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

	group, supportsEcommerce, err := resolveImagePlaygroundTokenGroup(1)
	if err != nil {
		t.Fatalf("resolve image playground group: %v", err)
	}
	if group != "vip" {
		t.Fatalf("resolveImagePlaygroundTokenGroup() = %q, want vip", group)
	}
	if supportsEcommerce {
		t.Fatal("supportsEcommerce = true, want false (only image model enabled)")
	}
}

func TestResolveImagePlaygroundTokenGroupPrefersImageAndAgentGroup(t *testing.T) {
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

	channels := []model.Channel{
		{Id: 10, Name: "default-image", Key: "sk-image", Group: "default", Models: imagePlaygroundDefaultModel, Status: common.ChannelStatusEnabled},
		{Id: 11, Name: "vip-image", Key: "sk-image", Group: "vip", Models: imagePlaygroundDefaultModel + "," + imagePlaygroundAgentModel, Status: common.ChannelStatusEnabled},
	}
	if err := model.DB.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}
	abilities := []model.Ability{
		{Group: "default", Model: imagePlaygroundDefaultModel, ChannelId: 10, Enabled: true},
		{Group: "vip", Model: imagePlaygroundDefaultModel, ChannelId: 11, Enabled: true},
		{Group: "vip", Model: imagePlaygroundAgentModel, ChannelId: 11, Enabled: true},
	}
	if err := model.DB.Create(&abilities).Error; err != nil {
		t.Fatalf("seed abilities: %v", err)
	}

	group, supportsEcommerce, err := resolveImagePlaygroundTokenGroup(1)
	if err != nil {
		t.Fatalf("resolve image playground group: %v", err)
	}
	if group != "vip" {
		t.Fatalf("resolveImagePlaygroundTokenGroup() = %q, want vip", group)
	}
	if !supportsEcommerce {
		t.Fatal("supportsEcommerce = false, want true (vip supports both models)")
	}
}

func TestResolveImagePlaygroundTokenGroupReturnsErrorWithoutCapableGroup(t *testing.T) {
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

	group, supportsEcommerce, err := resolveImagePlaygroundTokenGroup(1)
	if err == nil {
		t.Fatal("resolveImagePlaygroundTokenGroup() error = nil, want no capable group error")
	}
	if group != "" {
		t.Fatalf("resolveImagePlaygroundTokenGroup() group = %q, want empty", group)
	}
	if supportsEcommerce {
		t.Fatal("supportsEcommerce = true, want false")
	}
	if !strings.Contains(err.Error(), imagePlaygroundDefaultModel) {
		t.Fatalf("error = %q, want model name %q", err, imagePlaygroundDefaultModel)
	}
}

func TestCreateImagePlaygroundTokenUsesCapableGroupWithoutModelLimits(t *testing.T) {
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

	token, supportsEcommerce, err := createImagePlaygroundToken(1, 100)
	if err != nil {
		t.Fatalf("create image playground token: %v", err)
	}
	if token.Group != "vip" {
		t.Fatalf("image playground token group = %q, want vip", token.Group)
	}
	if token.ModelLimitsEnabled {
		t.Fatal("image playground token should not enable model limits")
	}
	if token.ModelLimits != "" {
		t.Fatalf("image playground token model limits = %q, want empty", token.ModelLimits)
	}
	if supportsEcommerce {
		t.Fatal("supportsEcommerce = true, want false (only image model enabled)")
	}
}

func TestClearImagePlaygroundTokenModelLimits(t *testing.T) {
	token := &model.Token{
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-image-2",
	}

	if !clearImagePlaygroundTokenModelLimits(token) {
		t.Fatal("clearImagePlaygroundTokenModelLimits() should report change")
	}
	if token.ModelLimitsEnabled {
		t.Fatal("model limits should be disabled")
	}
	if token.ModelLimits != "" {
		t.Fatalf("model limits = %q, want empty", token.ModelLimits)
	}
	if clearImagePlaygroundTokenModelLimits(token) {
		t.Fatal("second clear should not report change")
	}
}

func TestRefreshImagePlaygroundTokenClearsLimitsAndUpdatesGroup(t *testing.T) {
	setupImagePlaygroundGroupTestDB(t)

	// 模拟 default 已从当前分组配置中移除，但历史用户和旧令牌仍保留 default。
	if err := setting.UpdateUserUsableGroupsByJSONString(`{"vip":"VIP","team":"团队"}`); err != nil {
		t.Fatalf("update user usable groups: %v", err)
	}
	if err := setting.UpdateAutoGroupsByJsonString(`["vip"]`); err != nil {
		t.Fatalf("update auto groups: %v", err)
	}
	if err := ratio_setting.UpdateGroupRatioByJSONString(`{"vip":1,"team":1}`); err != nil {
		t.Fatalf("update group ratio: %v", err)
	}

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
		Models: imagePlaygroundDefaultModel + "," + imagePlaygroundAgentModel,
		Status: common.ChannelStatusEnabled,
	}).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	abilities := []model.Ability{
		{Group: "vip", Model: imagePlaygroundDefaultModel, ChannelId: 10, Enabled: true},
		{Group: "vip", Model: imagePlaygroundAgentModel, ChannelId: 10, Enabled: true},
	}
	if err := model.DB.Create(&abilities).Error; err != nil {
		t.Fatalf("seed abilities: %v", err)
	}

	token := &model.Token{
		Group:              "default",
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-image-2",
	}
	changed, supportsEcommerce, err := refreshImagePlaygroundToken(token, 1)
	if err != nil {
		t.Fatalf("refresh image playground token: %v", err)
	}
	if !changed {
		t.Fatal("refreshImagePlaygroundToken() should report change")
	}
	if !supportsEcommerce {
		t.Fatal("supportsEcommerce = false, want true (vip supports both models)")
	}
	if token.Group != "vip" {
		t.Fatalf("token group = %q, want vip", token.Group)
	}
	if token.ModelLimitsEnabled || token.ModelLimits != "" {
		t.Fatalf("model limits not cleared: enabled=%v limits=%q", token.ModelLimitsEnabled, token.ModelLimits)
	}
}
