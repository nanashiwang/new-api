package middleware

import (
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

func setupIPBlacklistMiddlewareTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	originDB := model.DB
	originRedisEnabled := common.RedisEnabled
	originUsingSQLite := common.UsingSQLite
	originUsingMySQL := common.UsingMySQL
	originUsingPostgreSQL := common.UsingPostgreSQL

	model.DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	t.Cleanup(func() {
		model.DB = originDB
		common.RedisEnabled = originRedisEnabled
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
		model.InvalidateIPBlacklistCache()
	})

	require.NoError(t, db.AutoMigrate(&model.IPBlacklist{}))
	model.InvalidateIPBlacklistCache()
}

func TestIPBlacklistBlocksLoginRegisterAndRelay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupIPBlacklistMiddlewareTestDB(t)
	_, _, err := model.CreateIPBlacklist("203.0.113.0/24", "test", 0, 1)
	require.NoError(t, err)

	router := gin.New()
	router.Use(IPBlacklist())
	router.POST("/api/user/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/api/user/register", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/v1/chat/completions", func(c *gin.Context) { c.Status(http.StatusOK) })

	tests := []struct {
		name         string
		path         string
		wantOpenAI   bool
		wantContains string
	}{
		{name: "login", path: "/api/user/login", wantContains: `"success":false`},
		{name: "register", path: "/api/user/register", wantContains: `"success":false`},
		{name: "relay", path: "/v1/chat/completions", wantOpenAI: true, wantContains: `"error"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			req.RemoteAddr = "203.0.113.99:12345"

			router.ServeHTTP(recorder, req)

			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Contains(t, strings.ReplaceAll(recorder.Body.String(), " ", ""), tt.wantContains)
		})
	}
}

func TestIPBlacklistAllowsUnmatchedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupIPBlacklistMiddlewareTestDB(t)
	_, _, err := model.CreateIPBlacklist("2001:db8::/64", "test", 0, 1)
	require.NoError(t, err)

	router := gin.New()
	router.Use(IPBlacklist())
	router.GET("/api/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	req.RemoteAddr = "203.0.113.99:12345"

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
}
