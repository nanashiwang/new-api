package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDistributeRejectsModelCallWithoutPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originDB := model.DB
	originLogDB := model.LOG_DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	model.InvalidateModelPermissionCache()
	t.Cleanup(func() {
		model.DB = originDB
		model.LOG_DB = originLogDB
		model.InvalidateModelPermissionCache()
	})
	if err := db.AutoMigrate(&model.ModelPermission{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	permission := model.ModelPermission{
		ModelName: "blocked-model",
		CallScope: model.ModelPermissionScopeNone,
	}
	if err := permission.Insert(); err != nil {
		t.Fatalf("insert permission: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(`{"model":"blocked-model","messages":[]}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	Distribute()(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
	if !ctx.IsAborted() {
		t.Fatalf("request should be aborted")
	}
}
