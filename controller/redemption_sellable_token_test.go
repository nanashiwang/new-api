package controller

import (
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

func setupRedemptionControllerTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false

	require.NoError(t, db.AutoMigrate(&model.Redemption{}, &model.SellableTokenProduct{}))
}

func createControllerSellableTokenProduct(t *testing.T, name string) *model.SellableTokenProduct {
	t.Helper()
	product := &model.SellableTokenProduct{
		Name:       name,
		Status:     model.SellableTokenProductStatusEnabled,
		PriceQuota: 0,
		TotalQuota: 1000,
	}
	require.NoError(t, model.CreateSellableTokenProduct(product))
	return product
}

func TestAddRedemption_PersistsSellableTokenProductID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupRedemptionControllerTestDB(t)

	product := createControllerSellableTokenProduct(t, "控制器商品")
	body := fmt.Sprintf(`{
		"name":"可售令牌码",
		"count":1,
		"benefit_type":"sellable_token",
		"sellable_token_product_id":%d
	}`, product.Id)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/redemption", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 1)

	AddRedemption(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)

	var redemption model.Redemption
	require.NoError(t, model.DB.First(&redemption).Error)
	require.Equal(t, model.RedemptionBenefitTypeSellableToken, redemption.BenefitType)
	require.Equal(t, product.Id, redemption.SellableTokenProductId)
}

func TestUpdateRedemption_PersistsSellableTokenProductID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupRedemptionControllerTestDB(t)

	firstProduct := createControllerSellableTokenProduct(t, "原商品")
	secondProduct := createControllerSellableTokenProduct(t, "新商品")
	redemption := &model.Redemption{
		UserId:                 1,
		Key:                    common.GetUUID(),
		Status:                 common.RedemptionCodeStatusEnabled,
		Name:                   "待更新兑换码",
		BenefitType:            model.RedemptionBenefitTypeSellableToken,
		SellableTokenProductId: firstProduct.Id,
		CreatedTime:            common.GetTimestamp(),
	}
	require.NoError(t, redemption.Insert())

	body := fmt.Sprintf(`{
		"id":%d,
		"name":"待更新兑换码",
		"benefit_type":"sellable_token",
		"sellable_token_product_id":%d,
		"expired_time":0
	}`, redemption.Id, secondProduct.Id)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/redemption", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateRedemption(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)

	var refreshed model.Redemption
	require.NoError(t, model.DB.First(&refreshed, "id = ?", redemption.Id).Error)
	require.Equal(t, secondProduct.Id, refreshed.SellableTokenProductId)
}
