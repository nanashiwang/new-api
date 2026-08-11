package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupWalletRedemptionControllerTest(t *testing.T) *model.User {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.RedisEnabled = originalRedisEnabled
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
	})

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Redemption{}, &model.Log{}))
	user := &model.User{
		Username:    "wallet_controller_user",
		Password:    "test-password",
		DisplayName: "wallet controller user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       1000,
		AffCode:     "wallet_controller_aff",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func TestSelfWalletRedemptionEndpoints_DeductAndReturnPrivacySafeView(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := setupWalletRedemptionControllerTest(t)

	createRecorder := httptest.NewRecorder()
	createContext, _ := gin.CreateTestContext(createRecorder)
	createContext.Request = httptest.NewRequest(http.MethodPost, "/api/user/redemptions", strings.NewReader(`{
		"quota":300,
		"request_id":"controller-wallet-request-001"
	}`))
	createContext.Request.Header.Set("Content-Type", "application/json")
	createContext.Set("id", user.Id)
	CreateSelfRedemption(createContext)

	require.Equal(t, http.StatusOK, createRecorder.Code)
	assert.NotContains(t, createRecorder.Body.String(), "used_user_id")
	var createResponse struct {
		Success bool `json:"success"`
		Data    struct {
			RemainingQuota int `json:"remaining_quota"`
			Redemption     struct {
				Key   string `json:"key"`
				Quota int    `json:"quota"`
			} `json:"redemption"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(createRecorder.Body.Bytes(), &createResponse))
	require.True(t, createResponse.Success, createRecorder.Body.String())
	assert.Equal(t, 700, createResponse.Data.RemainingQuota)
	assert.Equal(t, 300, createResponse.Data.Redemption.Quota)
	assert.NotEmpty(t, createResponse.Data.Redemption.Key)

	listRecorder := httptest.NewRecorder()
	listContext, _ := gin.CreateTestContext(listRecorder)
	listContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/redemptions/self?p=1&page_size=20", nil)
	listContext.Set("id", user.Id)
	GetSelfRedemptions(listContext)

	require.Equal(t, http.StatusOK, listRecorder.Code)
	assert.NotContains(t, listRecorder.Body.String(), "used_user_id")
	assert.Contains(t, listRecorder.Body.String(), createResponse.Data.Redemption.Key)
}
