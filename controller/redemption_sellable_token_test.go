package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRedemptionControllerTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false

	require.NoError(t, db.AutoMigrate(&model.Redemption{}))
}

func requireSellableRedemptionDisabled(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	require.False(t, body.Success)
	require.Contains(t, body.Message, "可售令牌兑换码功能已下线")
}

func TestAddRedemption_RejectsSellableTokenBenefit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupRedemptionControllerTestDB(t)

	body := `{
		"name":"可售令牌码",
		"count":1,
		"benefit_type":"sellable_token",
		"sellable_token_product_id":1
	}`

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/redemption", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 1)

	AddRedemption(ctx)
	requireSellableRedemptionDisabled(t, recorder)

	var count int64
	require.NoError(t, model.DB.Model(&model.Redemption{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestUpdateRedemption_RejectsSellableTokenBenefit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupRedemptionControllerTestDB(t)

	redemption := &model.Redemption{
		UserId:      1,
		Key:         common.GetUUID(),
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "待更新兑换码",
		BenefitType: model.RedemptionBenefitTypeQuota,
		Quota:       100,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, redemption.Insert())

	body := fmt.Sprintf(`{
		"id":%d,
		"name":"待更新兑换码",
		"benefit_type":"sellable_token",
		"sellable_token_product_id":1,
		"expired_time":0
	}`, redemption.Id)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/redemption", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateRedemption(ctx)
	requireSellableRedemptionDisabled(t, recorder)

	var refreshed model.Redemption
	require.NoError(t, model.DB.First(&refreshed, "id = ?", redemption.Id).Error)
	require.Equal(t, model.RedemptionBenefitTypeQuota, refreshed.BenefitType)
	require.Zero(t, refreshed.SellableTokenProductId)
}

func TestAddRedemption_MarksAdminFundingSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupRedemptionControllerTestDB(t)

	body := `{
		"name":"管理员余额码",
		"count":1,
		"benefit_type":"quota",
		"quota":100
	}`
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/redemption", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 1)

	AddRedemption(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, recorder.Body.String())

	var redemption model.Redemption
	require.NoError(t, model.DB.First(&redemption).Error)
	require.Equal(t, model.RedemptionFundingSourceAdmin, redemption.FundingSource)
}

func TestAdminCannotUpdateOrDeleteWalletFundedRedemption(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupRedemptionControllerTestDB(t)

	redemption := &model.Redemption{
		UserId:        10,
		Key:           common.GetUUID(),
		Status:        common.RedemptionCodeStatusEnabled,
		Name:          "用户钱包兑换码",
		BenefitType:   model.RedemptionBenefitTypeQuota,
		Quota:         100,
		CreatedTime:   common.GetTimestamp(),
		FundingSource: model.RedemptionFundingSourceWallet,
	}
	require.NoError(t, redemption.Insert())

	updateBody := fmt.Sprintf(`{"id":%d,"status":%d}`, redemption.Id, common.RedemptionCodeStatusDisabled)
	updateRecorder := httptest.NewRecorder()
	updateCtx, _ := gin.CreateTestContext(updateRecorder)
	updateCtx.Request = httptest.NewRequest(http.MethodPut, "/api/redemption?status_only=1", strings.NewReader(updateBody))
	updateCtx.Request.Header.Set("Content-Type", "application/json")
	UpdateRedemption(updateCtx)
	require.Contains(t, updateRecorder.Body.String(), "不允许由管理员编辑或禁用")

	deleteRecorder := httptest.NewRecorder()
	deleteCtx, _ := gin.CreateTestContext(deleteRecorder)
	deleteCtx.Request = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/redemption/%d", redemption.Id), nil)
	deleteCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(redemption.Id)}}
	DeleteRedemption(deleteCtx)
	require.Contains(t, deleteRecorder.Body.String(), model.ErrWalletFundedRedemptionImmutable.Error())

	var persisted model.Redemption
	require.NoError(t, model.DB.First(&persisted, redemption.Id).Error)
	require.Equal(t, common.RedemptionCodeStatusEnabled, persisted.Status)
}
