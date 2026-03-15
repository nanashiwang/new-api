package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInitChannelCache_UsesAbilitiesAsSourceOfTruth(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originDB := DB
	originLogDB := LOG_DB
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	originGroupMap := group2model2channels
	originChannels := channelsIDM
	DB = db
	LOG_DB = db
	common.MemoryCacheEnabled = true
	group2model2channels = nil
	channelsIDM = nil
	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.MemoryCacheEnabled = originMemoryCacheEnabled
		group2model2channels = originGroupMap
		channelsIDM = originChannels
	})

	if err := db.AutoMigrate(&Channel{}, &Ability{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	channels := []Channel{
		{Id: 10, Name: "wrong-codex", Key: "sk-codex", Type: constant.ChannelTypeCodex, Group: "default", Models: "gpt-5.4", Status: common.ChannelStatusEnabled},
		{Id: 11, Name: "right-openai", Key: "sk-openai", Group: "default", Models: "gpt-5.4", Status: common.ChannelStatusEnabled},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	abilities := []Ability{
		{Group: "default", Model: "gpt-5.4", ChannelId: 11, Enabled: true, Priority: common.GetPointer[int64](0), Weight: 100},
	}
	if err := db.Create(&abilities).Error; err != nil {
		t.Fatalf("seed abilities: %v", err)
	}

	InitChannelCache()

	channel, err := GetRandomSatisfiedChannel("default", "gpt-5.4", 0, nil)
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if channel == nil {
		t.Fatal("expected channel, got nil")
	}
	if channel.Id != 11 {
		t.Fatalf("expected ability-backed channel 11, got %d", channel.Id)
	}
}
