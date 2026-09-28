package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDistributeSeparatesUnknownModelFromUnavailableRoutes(t *testing.T) {
	require.NoError(t, i18n.Init())
	oldDB, oldMemory, oldRedis := model.DB, common.MemoryCacheEnabled, common.RedisEnabled
	oldGroups, oldAuto := setting.UserUsableGroups2JSONString(), setting.GetAutoGroups()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB, common.MemoryCacheEnabled, common.RedisEnabled = db, false, false
	model.InvalidateModelPermissionCache()
	t.Cleanup(func() {
		model.DB, common.MemoryCacheEnabled, common.RedisEnabled = oldDB, oldMemory, oldRedis
		model.InvalidateModelPermissionCache()
		_ = setting.UpdateUserUsableGroupsByJSONString(oldGroups)
		data, _ := common.Marshal(oldAuto)
		_ = setting.UpdateAutoGroupsByJsonString(string(data))
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	require.NoError(t, db.AutoMigrate(&model.Ability{}, &model.Channel{}, &model.ModelPermission{}))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"public":"Public","backup":"Backup"}`))
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["public","backup","private"]`))
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "public", Model: "kimi-k3", ChannelId: 1, Enabled: false},
		{Group: "backup", Model: "backup-model", ChannelId: 2, Enabled: false},
		{Group: "private", Model: "private-model", ChannelId: 3, Enabled: false},
		{Group: "public", Model: "gpt-4-gizmo-*", ChannelId: 1, Enabled: false},
	}).Error)
	for _, tc := range []struct {
		name, model, group, code string
		allowed                  []int
		status                   int
	}{
		{"unknown", "typo", "public", "model_not_found", nil, 404},
		{"disabled", "kimi-k3", "public", "no_available_channel", nil, 503},
		{"foreign_group", "private-model", "public", "model_not_found", nil, 404},
		{"normalized_alias", "gpt-4-gizmo-test", "public", "no_available_channel", nil, 503},
		{"channel_restricted", "kimi-k3", "public", "model_not_found", []int{2}, 404},
		{"empty_channel_scope", "kimi-k3", "public", "model_not_found", []int{}, 404},
		{"auto_disabled", "backup-model", "auto", "no_available_channel", nil, 503},
		{"auto_foreign", "private-model", "auto", "model_not_found", nil, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			body, err := common.Marshal(map[string]any{"model": tc.model, "messages": []any{map[string]any{"role": "user", "content": "test"}}})
			require.NoError(t, err)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))
			c.Request.Header.Set("Content-Type", "application/json")
			common.SetContextKey(c, constant.ContextKeyUsingGroup, tc.group)
			common.SetContextKey(c, constant.ContextKeyUserGroup, "public")
			common.SetContextKey(c, constant.ContextKeyUserRole, common.RoleCommonUser)
			if tc.allowed != nil {
				allowed := make(map[int]bool)
				for _, id := range tc.allowed {
					allowed[id] = true
				}
				common.SetContextKey(c, constant.ContextKeyTokenChannelLimitEnabled, true)
				common.SetContextKey(c, constant.ContextKeyTokenChannelLimit, allowed)
			}
			Distribute()(c)
			require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
			require.Contains(t, recorder.Body.String(), `"code":"`+tc.code+`"`)
		})
	}
	require.NoError(t, db.Migrator().DropTable(&model.Ability{}))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	require.True(t, hasConfiguredRequestModel(c, "kimi-k3", "public"), "database errors must remain availability errors")
}
