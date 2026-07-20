package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSelectResponsesAutoContinueChannelAllowsSameTagSibling(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originDB := model.DB
	originLogDB := model.LOG_DB
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	model.DB = db
	model.LOG_DB = db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = originDB
		model.LOG_DB = originLogDB
		common.MemoryCacheEnabled = originMemoryCacheEnabled
	})

	if err := db.AutoMigrate(&model.Channel{}, &model.Ability{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	tag := "shared-tag"
	channels := []model.Channel{
		{Id: 1, Name: "failed", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Tag: &tag},
		{Id: 2, Name: "healthy-sibling", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Tag: &tag},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}
	abilities := []model.Ability{
		{Group: "vip", Model: "gpt-5.6-sol", ChannelId: 1, Enabled: true},
		{Group: "vip", Model: "gpt-5.6-sol", ChannelId: 2, Enabled: true},
	}
	if err := db.Create(&abilities).Error; err != nil {
		t.Fatalf("seed abilities: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	channel, ok := selectResponsesAutoContinueChannel(ctx, &relaycommon.RelayInfo{
		TokenGroup:      "vip",
		OriginModelName: "gpt-5.6-sol",
	}, 1)
	if !ok || channel == nil || channel.Id != 2 {
		t.Fatalf("expected same-tag sibling channel 2, got %#v", channel)
	}
}
