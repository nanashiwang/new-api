package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUnavailableHealthGroupsIncludesExhaustedAutoOnlyWhenConfigured(t *testing.T) {
	previousDB := model.DB
	previousGroups := setting.UserUsableGroups2JSONString()
	previousAuto := setting.GetAutoGroups()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
		_ = setting.UpdateUserUsableGroupsByJSONString(previousGroups)
		data, _ := common.Marshal(previousAuto)
		_ = setting.UpdateAutoGroupsByJsonString(string(data))
	})
	require.NoError(t, db.AutoMigrate(&model.Ability{}))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"a":"A","b":"B","unsupported":"unsupported"}`))
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["a","b","unsupported","private"]`))
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "a", Model: "m", ChannelId: 1, Enabled: false},
		{Group: "b", Model: "m", ChannelId: 2, Enabled: false},
		{Group: "private", Model: "m", ChannelId: 3, Enabled: false},
	}).Error)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(c, constant.ContextKeyUserGroup, "a")
	require.Equal(t, []string{"a", "b"}, unavailableHealthGroups(c, "m", "auto", "auto"))
	require.Equal(t, []string{"b"}, unavailableHealthGroups(c, "m", "auto", "b"))
	require.Empty(t, unavailableHealthGroups(c, "typo", "auto", "auto"))
	common.SetContextKey(c, constant.ContextKeyTokenChannelLimitEnabled, true)
	common.SetContextKey(c, constant.ContextKeyTokenChannelLimit, map[int]bool{})
	require.Empty(t, unavailableHealthGroups(c, "m", "auto", "auto"))
}
