package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRequestFailureMiddlewareRecordsAuthenticatedEarlyRejections(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousEnabled, previousRedis := constant.ErrorLogEnabled, common.RedisEnabled
	model.DB, model.LOG_DB = db, db
	constant.ErrorLogEnabled = true
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		constant.ErrorLogEnabled = previousEnabled
		common.RedisEnabled = previousRedis
	})
	require.NoError(t, db.AutoMigrate(&model.Log{}, &model.User{}))
	for _, tc := range []struct {
		name, tag      string
		userID, status int
	}{
		{"quota", "relay", 42, 429}, {"no_channel", "relay", 42, 503}, {"anonymous", "relay", 0, 401}, {"dashboard", "api", 42, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Log{}).Error)
			r := gin.New()
			r.Use(RequestFailureLog(), RouteTag(tc.tag))
			r.POST("/v1/chat/completions", func(c *gin.Context) {
				if tc.userID > 0 {
					c.Set("id", tc.userID)
				}
				c.Set(common.RequestIdKey, "req-"+tc.name)
				abortWithOpenAiMessage(c, tc.status, "internal diagnostic")
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))
			require.Equal(t, tc.status, w.Code)
			var rows []model.Log
			require.NoError(t, db.Find(&rows).Error)
			if tc.userID > 0 && tc.tag == "relay" {
				require.Len(t, rows, 1)
				require.Equal(t, "req-"+tc.name, rows[0].RequestId)
			} else {
				require.Empty(t, rows)
			}
		})
	}
}
