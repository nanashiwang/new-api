package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTokenManageControllerTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true

	require.NoError(t, db.AutoMigrate(&model.Token{}))
}

func TestManageTokenBatch_RejectsPackageExhaustedEnable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTokenManageControllerTestDB(t)

	now := common.GetTimestamp()
	require.NoError(t, model.DB.Create(&model.Token{
		Id:                   11,
		UserId:               5,
		Key:                  "token_manage_package_block_key_000000000000000001",
		Name:                 "batch-enable-token",
		Status:               common.TokenStatusDisabled,
		CreatedTime:          now,
		AccessedTime:         now,
		ExpiredTime:          -1,
		RemainQuota:          50000,
		PackageEnabled:       true,
		PackageLimitQuota:    20000,
		PackageUsedQuota:     20000,
		PackagePeriod:        model.TokenPackagePeriodDaily,
		PackagePeriodMode:    model.TokenPackagePeriodModeRelative,
		PackageNextResetTime: now + 3600,
	}).Error)

	body := fmt.Sprintf(`{"ids":[%d],"action":"enable"}`, 11)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/token/manage/batch", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 5)

	ManageTokenBatch(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			SuccessCount int `json:"success_count"`
			FailedCount  int `json:"failed_count"`
			Failed       []struct {
				Message string `json:"message"`
			} `json:"failed"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, 0, resp.Data.SuccessCount)
	require.Equal(t, 1, resp.Data.FailedCount)
	require.Len(t, resp.Data.Failed, 1)
	require.Equal(t, model.ErrTokenCannotEnablePackageExhausted.Error(), resp.Data.Failed[0].Message)

	var refreshed model.Token
	require.NoError(t, model.DB.First(&refreshed, "id = ?", 11).Error)
	require.Equal(t, common.TokenStatusDisabled, refreshed.Status)
}
