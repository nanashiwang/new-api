package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChannelUsageValidatesRangeAndSort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, query := range []string{"", "start_time=x&end_time=200", "start_time=200&end_time=100", "start_time=1&end_time=2592002", "start_time=100&end_time=99999999999", "start_time=100&end_time=200&usage_order=invalid", "start_time=100&end_time=200&page_size=-1", "start_time=100&end_time=200&p=-1", "start_time=100&end_time=200&p=9223372036854775807"} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/channel/usage?"+query, nil)
		GetChannelUsage(c)
		var body struct{ Success bool }
		require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
		require.False(t, body.Success)
	}
}

func TestChannelUsageFiltersAndOmitsKey(t *testing.T) {
	setupChannelSearchControllerTestDB(t)
	seedChannelSearchControllerTestData(t)
	require.NoError(t, model.LOG_DB.AutoMigrate(&model.Log{}))
	require.NoError(t, model.LOG_DB.Create(&[]model.Log{
		{ChannelId: 1, CreatedAt: 10101, Type: model.LogTypeConsume, Quota: 50, Group: "vip", ModelName: "gpt-4o"},
		{ChannelId: 3, CreatedAt: 10101, Type: model.LogTypeConsume, Quota: 80, Group: "vip", ModelName: "gpt-4o-mini"},
		{ChannelId: 3, CreatedAt: 10101, Type: model.LogTypeConsume, Quota: 900, Group: "default", ModelName: "gpt-4o-mini"},
	}).Error)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/channel/usage?start_time=10000&end_time=11000&model=gpt&group=vip&status=enabled&p=1&page_size=1", nil)
	GetChannelUsage(c)
	var body struct {
		Success bool
		Data    struct {
			Items   []model.ChannelUsageItem
			Total   int
			Summary model.ChannelUsage `json:"usage_summary"`
			Start   int64              `json:"start_time"`
			End     int64              `json:"end_time"`
		}
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	require.True(t, body.Success, w.Body.String())
	require.Equal(t, 2, body.Data.Total)
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, 3, body.Data.Items[0].Id)
	require.Empty(t, body.Data.Items[0].Key)
	require.NotContains(t, w.Body.String(), "beta-mini-key")
	require.EqualValues(t, 130, body.Data.Summary.TotalQuota)
	require.EqualValues(t, 10000, body.Data.Start)
	require.EqualValues(t, 11000, body.Data.End)
}
