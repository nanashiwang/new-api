package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/model_setting"
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

func TestImageGenerationToolCallableByRole(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	originalScope := settings.ImageGenerationToolCallPermission
	t.Cleanup(func() {
		settings.ImageGenerationToolCallPermission = originalScope
	})

	tests := []struct {
		name  string
		scope int
		role  int
		want  bool
	}{
		{"all allows admin", model.ModelPermissionScopeAll, common.RoleAdminUser, true},
		{"all allows common", model.ModelPermissionScopeAll, common.RoleCommonUser, true},
		{"admin only allows admin", model.ModelPermissionScopeAdminOnly, common.RoleAdminUser, true},
		{"admin only rejects common", model.ModelPermissionScopeAdminOnly, common.RoleCommonUser, false},
		{"common only allows common", model.ModelPermissionScopeCommonOnly, common.RoleCommonUser, true},
		{"common only rejects admin", model.ModelPermissionScopeCommonOnly, common.RoleAdminUser, false},
		{"none rejects root", model.ModelPermissionScopeNone, common.RoleRootUser, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings.ImageGenerationToolCallPermission = tt.scope
			if got := isImageGenerationToolCallableByRole(tt.role); got != tt.want {
				t.Fatalf("isImageGenerationToolCallableByRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDistributeRejectsImageGenerationToolWithoutPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := model_setting.GetGlobalSettings()
	originalScope := settings.ImageGenerationToolCallPermission
	settings.ImageGenerationToolCallPermission = model.ModelPermissionScopeAdminOnly
	t.Cleanup(func() {
		settings.ImageGenerationToolCallPermission = originalScope
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/responses",
		strings.NewReader(`{"model":"gpt-5.5","input":"draw","tools":[{"type":"image_generation"}]}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(ctx, constant.ContextKeyUserRole, common.RoleCommonUser)

	Distribute()(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
	if !ctx.IsAborted() {
		t.Fatalf("request should be aborted")
	}
}
