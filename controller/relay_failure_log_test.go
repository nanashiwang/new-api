package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRelayValidationFailureReachesUserLog(t *testing.T) {
	db := useControllerCapacityTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}, &model.User{}))
	previous, previousRedis := constant.ErrorLogEnabled, common.RedisEnabled
	constant.ErrorLogEnabled = true
	common.RedisEnabled = false
	t.Cleanup(func() { constant.ErrorLogEnabled = previous; common.RedisEnabled = previousRedis })
	r := gin.New()
	r.Use(middleware.RequestFailureLog(), middleware.RouteTag("relay"))
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", 7311)
		c.Set(common.RequestIdKey, "req-invalid-json")
		Relay(c, types.RelayFormatOpenAI)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.GreaterOrEqual(t, w.Code, 400)
	var row model.Log
	require.NoError(t, db.Where("request_id = ?", "req-invalid-json").First(&row).Error)
	require.Equal(t, model.LogTypeError, row.Type)
	require.Contains(t, row.Other, "invalid_request")
}
