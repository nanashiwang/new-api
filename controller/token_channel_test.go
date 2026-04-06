package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func seedTokenChannelAdminUser(t *testing.T, userID int) {
	t.Helper()
	if err := model.DB.Create(&model.User{
		Id:       userID,
		Username: "admin-user",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
		AffCode:  "aff-admin-user",
	}).Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
}

func TestGetTokenChannels_RejectsNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)
	seedTokenChannelAdminUser(t, 9)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/token/channels?group=auto&model_limits=gpt-5.2", nil)
	ctx.Set("id", 9)
	ctx.Set("role", common.RoleCommonUser)

	GetTokenChannels(ctx)

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Success {
		t.Fatal("expected non-admin request to fail")
	}
	if !strings.Contains(resp.Message, "权限不足") {
		t.Fatalf("unexpected message: %s", resp.Message)
	}
}

func TestGetTokenChannels_FiltersByGroupAndModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)
	seedTokenChannelAdminUser(t, 9)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/token/channels?group=auto&model_limits=gpt-5.2", nil)
	ctx.Set("id", 9)
	ctx.Set("role", common.RoleAdminUser)

	GetTokenChannels(ctx)

	var resp struct {
		Success bool                 `json:"success"`
		Data    []tokenChannelOption `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got body: %s", recorder.Body.String())
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 channel, got %#v", resp.Data)
	}
	if resp.Data[0].ID != 1 {
		t.Fatalf("expected channel 1, got %#v", resp.Data[0])
	}
	if !resp.Data[0].Available {
		t.Fatalf("expected channel 1 to be available: %#v", resp.Data[0])
	}
	if resp.Data[0].Tag != "shared-openai" {
		t.Fatalf("unexpected channel tag: %#v", resp.Data[0])
	}
	if len(resp.Data[0].MatchedGroups) != 2 || resp.Data[0].MatchedGroups[0] != "default" || resp.Data[0].MatchedGroups[1] != "vip" {
		t.Fatalf("unexpected matched groups: %#v", resp.Data[0].MatchedGroups)
	}
	if len(resp.Data[0].MatchedModels) != 1 || resp.Data[0].MatchedModels[0] != "gpt-5.2" {
		t.Fatalf("unexpected matched models: %#v", resp.Data[0].MatchedModels)
	}
}

func TestGetTokenChannels_WithTokenIDIncludesPersistedUnavailableChannels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)
	seedTokenChannelAdminUser(t, 9)

	if err := model.DB.Model(&model.Channel{}).
		Where("id = ?", 2).
		Update("status", common.ChannelStatusManuallyDisabled).Error; err != nil {
		t.Fatalf("disable channel 2: %v", err)
	}

	if err := model.DB.Create(&model.Token{
		Id:                   101,
		UserId:               9,
		Name:                 "token-with-stale-channel",
		Key:                  "sk-token-channel-test-101",
		Group:                "auto",
		ChannelLimitsEnabled: true,
		ChannelLimits:        "1,2",
	}).Error; err != nil {
		t.Fatalf("seed token: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/token/channels?token_id=101&group=auto&model_limits=gpt-5.2", nil)
	ctx.Set("id", 9)
	ctx.Set("role", common.RoleAdminUser)

	GetTokenChannels(ctx)

	var resp struct {
		Success bool                 `json:"success"`
		Data    []tokenChannelOption `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got body: %s", recorder.Body.String())
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 channels, got %#v", resp.Data)
	}

	optionsByID := make(map[int]tokenChannelOption, len(resp.Data))
	for _, option := range resp.Data {
		optionsByID[option.ID] = option
	}

	if !optionsByID[1].Available {
		t.Fatalf("expected channel 1 to remain available: %#v", optionsByID[1])
	}
	if optionsByID[2].Available {
		t.Fatalf("expected channel 2 to be returned as unavailable: %#v", optionsByID[2])
	}
	if optionsByID[2].Status != common.ChannelStatusManuallyDisabled {
		t.Fatalf("unexpected channel 2 status: %#v", optionsByID[2])
	}
}

func TestNormalizeTokenChannelLimitsForSave_RejectsNonAdmin(t *testing.T) {
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)

	_, _, err := normalizeTokenChannelLimitsForSave(1, common.RoleCommonUser, "default", "", false, "1", true, "")
	if err == nil {
		t.Fatal("expected non-admin normalization to fail")
	}
	if !strings.Contains(err.Error(), "仅管理员") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeTokenChannelLimitsForSave_RejectsMismatchedChannel(t *testing.T) {
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)

	_, _, err := normalizeTokenChannelLimitsForSave(1, common.RoleAdminUser, "default", "gemini-2.5-pro", true, "1", true, "")
	if err == nil {
		t.Fatal("expected mismatched channel normalization to fail")
	}
	if !strings.Contains(err.Error(), "不匹配") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeTokenChannelLimitsForSave_AllowsPersistedUnavailableChannel(t *testing.T) {
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)

	enabled, channelLimits, err := normalizeTokenChannelLimitsForSave(
		1,
		common.RoleAdminUser,
		"default",
		"gemini-2.5-pro",
		true,
		"1",
		true,
		"1",
	)
	if err != nil {
		t.Fatalf("expected persisted unavailable channel to be preserved: %v", err)
	}
	if !enabled {
		t.Fatal("expected channel limits to stay enabled")
	}
	if channelLimits != "1" {
		t.Fatalf("unexpected normalized channel limits: %s", channelLimits)
	}
}
