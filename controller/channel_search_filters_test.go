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

func setupChannelSearchControllerTestDB(t *testing.T) {
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
	})

	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))
}

func seedChannelSearchControllerTestData(t *testing.T) {
	t.Helper()

	alphaTag := "alpha"
	betaTag := "beta"
	gammaTag := "gamma"
	priority := int64(0)
	weight := uint(10)

	channels := []*model.Channel{
		{
			Id:       1,
			Name:     "alpha-openai",
			Key:      "alpha-openai-key",
			Type:     1,
			Status:   common.ChannelStatusEnabled,
			Group:    "default,vip",
			Models:   "gpt-4o,claude-3-5-sonnet",
			Tag:      &alphaTag,
			Priority: &priority,
			Weight:   &weight,
		},
		{
			Id:       2,
			Name:     "alpha-gemini-disabled",
			Key:      "alpha-gemini-key",
			Type:     2,
			Status:   common.ChannelStatusManuallyDisabled,
			Group:    "vip",
			Models:   "gemini-1.5-pro",
			Tag:      &alphaTag,
			Priority: &priority,
			Weight:   &weight,
		},
		{
			Id:       3,
			Name:     "beta-mini",
			Key:      "beta-mini-key",
			Type:     2,
			Status:   common.ChannelStatusEnabled,
			Group:    "vip",
			Models:   "gpt-4o-mini",
			Tag:      &betaTag,
			Priority: &priority,
			Weight:   &weight,
		},
		{
			Id:       4,
			Name:     "gamma-basic",
			Key:      "gamma-basic-key",
			Type:     1,
			Status:   common.ChannelStatusEnabled,
			Group:    "default",
			Models:   "gemini-1.5-flash",
			Tag:      &gammaTag,
			Priority: &priority,
			Weight:   &weight,
		},
	}

	for _, channel := range channels {
		require.NoError(t, channel.Insert())
	}
}

func TestSearchChannels_TypeCountsIgnoreTypeFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupChannelSearchControllerTestDB(t)
	seedChannelSearchControllerTestData(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/channel/search?group=vip&model=gpt-4&status=enabled&type=1&p=1&page_size=20",
		nil,
	)

	SearchChannels(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Items      []model.Channel  `json:"items"`
			Total      int              `json:"total"`
			TypeCounts map[string]int64 `json:"type_counts"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, 1, resp.Data.Total)
	require.Equal(t, 1, resp.Data.Items[0].Type)
	require.EqualValues(t, 1, resp.Data.TypeCounts["1"])
	require.EqualValues(t, 1, resp.Data.TypeCounts["2"])
}

func TestGetAllChannels_TagModeCountsFilteredTags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupChannelSearchControllerTestDB(t)
	seedChannelSearchControllerTestData(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/channel/?tag_mode=true&status=enabled&type=1&p=1&page_size=20",
		nil,
	)

	GetAllChannels(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.Channel `json:"items"`
			Total int             `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Len(t, resp.Data.Items, 2)
	require.Equal(t, 2, resp.Data.Total)
}
