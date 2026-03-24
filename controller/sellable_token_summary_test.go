package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSellableTokenSummaryControllerTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true

	require.NoError(t, db.AutoMigrate(&model.Token{}, &model.SellableTokenIssuance{}, &model.SellableTokenProduct{}))
}

func TestSanitizeSellableTokenSummaryItems_ProjectsUnlimitedAndPackageUsage(t *testing.T) {
	items := sanitizeSellableTokenSummaryItems([]*model.Token{
		{
			Id:                     1,
			Name:                   "unlimited-package-token",
			UnlimitedQuota:         true,
			PackageEnabled:         true,
			PackageLimitQuota:      1000,
			PackageUsedQuota:       250,
			SellableTokenProductId: 7,
		},
		nil,
	})

	if len(items) != 1 {
		t.Fatalf("expected 1 summary item, got %d", len(items))
	}
	if !items[0].UnlimitedQuota {
		t.Fatal("expected unlimited_quota to be projected")
	}
	if items[0].PackageUsedQuota != 250 {
		t.Fatalf("expected package_used_quota=250, got=%d", items[0].PackageUsedQuota)
	}
}

func TestAdminGetUserSellableTokenSummary_RefreshesExpiredCycleUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupSellableTokenSummaryControllerTestDB(t)

	now := common.GetTimestamp()
	require.NoError(t, model.DB.Create(&model.Token{
		Id:                     1,
		UserId:                 7,
		Key:                    "sellable_summary_reset_key_00000000000000000001",
		Name:                   "summary-token",
		SourceType:             model.TokenSourceTypeSellableToken,
		Status:                 common.TokenStatusEnabled,
		CreatedTime:            now,
		AccessedTime:           now,
		ExpiredTime:            -1,
		RemainQuota:            50000,
		PackageEnabled:         true,
		PackageLimitQuota:      20000,
		PackageUsedQuota:       12000,
		PackagePeriod:          model.TokenPackagePeriodDaily,
		PackagePeriodMode:      model.TokenPackagePeriodModeNatural,
		PackageNextResetTime:   now - 60,
		SellableTokenProductId: 2,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "7"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/7/sellable-token/summary", nil)

	AdminGetUserSellableTokenSummary(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Tokens []sellableTokenSummaryItem `json:"tokens"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Len(t, resp.Data.Tokens, 1)
	require.Equal(t, 0, resp.Data.Tokens[0].PackageUsedQuota)

	var refreshed model.Token
	require.NoError(t, model.DB.First(&refreshed, "id = ?", 1).Error)
	require.Equal(t, 0, refreshed.PackageUsedQuota)
	require.Greater(t, refreshed.PackageNextResetTime, now)
}

func TestManageUserSellableToken_RejectsPackageExhaustedEnable(t *testing.T) {
	setupSellableTokenSummaryControllerTestDB(t)

	now := common.GetTimestamp()
	require.NoError(t, model.DB.Create(&model.Token{
		Id:                     2,
		UserId:                 9,
		Key:                    "sellable_enable_block_key_00000000000000000001",
		Name:                   "sellable-enable-token",
		SourceType:             model.TokenSourceTypeSellableToken,
		Status:                 common.TokenStatusDisabled,
		CreatedTime:            now,
		AccessedTime:           now,
		ExpiredTime:            -1,
		RemainQuota:            50000,
		PackageEnabled:         true,
		PackageLimitQuota:      20000,
		PackageUsedQuota:       20000,
		PackagePeriod:          model.TokenPackagePeriodDaily,
		PackagePeriodMode:      model.TokenPackagePeriodModeRelative,
		PackageNextResetTime:   now + 3600,
		SellableTokenProductId: 3,
	}).Error)

	msg, err := manageUserSellableToken(9, 2, "enable")
	require.ErrorIs(t, err, model.ErrTokenCannotEnablePackageExhausted)
	require.Empty(t, msg)

	var refreshed model.Token
	require.NoError(t, model.DB.First(&refreshed, "id = ?", 2).Error)
	require.Equal(t, common.TokenStatusDisabled, refreshed.Status)
}
