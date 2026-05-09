package controller

import (
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

func setupIPBlacklistControllerTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	originDB := model.DB
	originLogDB := model.LOG_DB
	originRedisEnabled := common.RedisEnabled
	originUsingSQLite := common.UsingSQLite
	originUsingMySQL := common.UsingMySQL
	originUsingPostgreSQL := common.UsingPostgreSQL

	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	t.Cleanup(func() {
		model.DB = originDB
		model.LOG_DB = originLogDB
		common.RedisEnabled = originRedisEnabled
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
		model.InvalidateIPBlacklistCache()
	})

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.IPBlacklist{}, &model.Log{}))
	model.InvalidateIPBlacklistCache()
}

func newIPBlacklistControllerTestContext(t *testing.T, method string, target string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var payload string
	if body != nil {
		data, err := common.Marshal(body)
		require.NoError(t, err)
		payload = string(data)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, target, strings.NewReader(payload))
	req.RemoteAddr = "203.0.113.5:12345"
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("id", 100)
	return ctx, recorder
}

func TestCreateIPBlacklistRequiresCurrentIPConfirmation(t *testing.T) {
	setupIPBlacklistControllerTestDB(t)

	ctx, recorder := newIPBlacklistControllerTestContext(t, http.MethodPost, "/api/user/ip-blacklist", gin.H{
		"ip": "203.0.113.0/24",
	})
	CreateIPBlacklist(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "current_ip_requires_confirmation")

	var count int64
	require.NoError(t, model.DB.Model(&model.IPBlacklist{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestCreateIPBlacklistWithConfirmationCreatesRule(t *testing.T) {
	setupIPBlacklistControllerTestDB(t)

	ctx, recorder := newIPBlacklistControllerTestContext(t, http.MethodPost, "/api/user/ip-blacklist", gin.H{
		"ip":                 "203.0.113.0/24",
		"confirm_current_ip": true,
	})
	CreateIPBlacklist(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)

	var item model.IPBlacklist
	require.NoError(t, model.DB.First(&item).Error)
	require.Equal(t, "203.0.113.0/24", item.CIDR)
	require.Equal(t, 100, item.CreatedBy)

	var logCount int64
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("type = ?", model.LogTypeManage).Count(&logCount).Error)
	require.Equal(t, int64(1), logCount)
}

func TestBatchCreateIPBlacklistRequiresCurrentIPConfirmation(t *testing.T) {
	setupIPBlacklistControllerTestDB(t)
	user := model.User{
		Username:   "batch_ip_user",
		Password:   "password123",
		Role:       common.RoleCommonUser,
		Status:     common.UserStatusEnabled,
		RegisterIP: "203.0.113.5",
	}
	require.NoError(t, model.DB.Create(&user).Error)

	ctx, recorder := newIPBlacklistControllerTestContext(t, http.MethodPost, "/api/user/ip-blacklist/batch", gin.H{
		"user_ids": []int{user.Id},
	})
	BatchCreateIPBlacklist(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "current_ip_requires_confirmation")
}

func TestGetAndDeleteIPBlacklist(t *testing.T) {
	setupIPBlacklistControllerTestDB(t)
	item, created, err := model.CreateIPBlacklist("2001:db8::1", "manual", 0, 100)
	require.NoError(t, err)
	require.True(t, created)

	ctx, recorder := newIPBlacklistControllerTestContext(t, http.MethodGet, "/api/user/ip-blacklist?keyword=2001:db8", nil)
	GetIPBlacklists(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Contains(t, recorder.Body.String(), "2001:db8::1/128")

	ctx, recorder = newIPBlacklistControllerTestContext(t, http.MethodDelete, "/api/user/ip-blacklist/"+strconv.Itoa(item.Id), nil)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(item.Id)}}
	DeleteIPBlacklist(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)

	_, err = model.GetIPBlacklistByID(item.Id)
	require.ErrorIs(t, err, model.ErrIPBlacklistNotFound)
}
