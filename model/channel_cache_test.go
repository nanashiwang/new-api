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

	channel, err := GetRandomSatisfiedChannel("default", "gpt-5.4", 0, nil, nil)
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

func TestGetRandomSatisfiedChannelFallsBackWhenTotalWeightIsNonPositive(t *testing.T) {
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	originGroupMap := group2model2channels
	originChannels := channelsIDM
	common.MemoryCacheEnabled = true
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1, 2},
		},
	}
	maxWeight := ^uint(0)
	priority := int64(0)
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Name: "overflow-a", Weight: &maxWeight, Priority: &priority},
		2: {Id: 2, Name: "overflow-b", Weight: &maxWeight, Priority: &priority},
	}
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originMemoryCacheEnabled
		group2model2channels = originGroupMap
		channelsIDM = originChannels
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetRandomSatisfiedChannel should not panic when totalWeight is non-positive: %v", r)
		}
	}()

	channel, err := GetRandomSatisfiedChannel("default", "gpt-5.4", 0, nil, nil)
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if channel == nil {
		t.Fatal("expected channel, got nil")
	}
	if channel.Id != 1 && channel.Id != 2 {
		t.Fatalf("expected one of the fallback channels, got %d", channel.Id)
	}
}

func TestGetRandomSatisfiedChannel_RespectsAllowedChannels(t *testing.T) {
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	originGroupMap := group2model2channels
	originChannels := channelsIDM
	common.MemoryCacheEnabled = true
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1, 2},
		},
	}
	priority := int64(0)
	weight := uint(100)
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Name: "allowed", Weight: &weight, Priority: &priority},
		2: {Id: 2, Name: "blocked", Weight: &weight, Priority: &priority},
	}
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originMemoryCacheEnabled
		group2model2channels = originGroupMap
		channelsIDM = originChannels
	})

	channel, err := GetRandomSatisfiedChannel("default", "gpt-5.4", 0, []int{1}, nil)
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if channel == nil {
		t.Fatal("expected channel, got nil")
	}
	if channel.Id != 1 {
		t.Fatalf("expected allowed channel 1, got %d", channel.Id)
	}
}
