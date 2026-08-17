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

func TestUpdateChannelStatusEvictsAutoDisabledMultiKeyCache(t *testing.T) {
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
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1},
		},
	}
	channelsIDM = map[int]*Channel{
		1: {
			Id:     1,
			Name:   "multi-key",
			Key:    "key-a\nkey-b",
			Group:  "default",
			Models: "gpt-5.4",
			Status: common.ChannelStatusEnabled,
			ChannelInfo: ChannelInfo{
				IsMultiKey:   true,
				MultiKeySize: 2,
			},
		},
	}
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
	channel := *channelsIDM[1]
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	if !UpdateChannelStatus(1, "key-a", common.ChannelStatusAutoDisabled, "bad key") {
		t.Fatal("expected first key status update to succeed")
	}
	if channelsIDM[1].Status != common.ChannelStatusEnabled {
		t.Fatalf("expected channel to stay enabled while one key remains enabled, got %d", channelsIDM[1].Status)
	}

	if !UpdateChannelStatus(1, "key-b", common.ChannelStatusAutoDisabled, "bad key") {
		t.Fatal("expected second key status update to succeed")
	}
	if channelsIDM[1].Status != common.ChannelStatusAutoDisabled {
		t.Fatalf("expected cached channel auto-disabled, got %d", channelsIDM[1].Status)
	}
	if got := group2model2channels["default"]["gpt-5.4"]; len(got) != 0 {
		t.Fatalf("expected auto-disabled channel evicted from routing cache, got %v", got)
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

func TestGetRandomSatisfiedChannel_MapsCodexAutoReviewToRoutingModel(t *testing.T) {
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	originGroupMap := group2model2channels
	originChannels := channelsIDM
	common.MemoryCacheEnabled = true
	group2model2channels = map[string]map[string][]int{
		"default": {
			constant.CodexAutoReviewRoutingModel: {1},
		},
	}
	priority := int64(0)
	weight := uint(100)
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Name: "codex", Type: constant.ChannelTypeCodex, Weight: &weight, Priority: &priority},
	}
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originMemoryCacheEnabled
		group2model2channels = originGroupMap
		channelsIDM = originChannels
	})

	channel, err := GetRandomSatisfiedChannel("default", constant.CodexAutoReviewModel, 0, nil, nil)
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if channel == nil {
		t.Fatal("expected channel, got nil")
	}
	if channel.Id != 1 {
		t.Fatalf("expected codex channel 1, got %d", channel.Id)
	}
}

func TestGetChannelDB_MapsCodexAutoReviewToRoutingModel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originDB := DB
	originLogDB := LOG_DB
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	DB = db
	LOG_DB = db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.MemoryCacheEnabled = originMemoryCacheEnabled
	})

	if err := db.AutoMigrate(&Channel{}, &Ability{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	channel := Channel{
		Id:     1,
		Name:   "codex",
		Key:    "sk-codex",
		Type:   constant.ChannelTypeCodex,
		Group:  "default",
		Models: constant.CodexAutoReviewRoutingModel,
		Status: common.ChannelStatusEnabled,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	ability := Ability{
		Group:     "default",
		Model:     constant.CodexAutoReviewRoutingModel,
		ChannelId: 1,
		Enabled:   true,
		Priority:  common.GetPointer[int64](0),
		Weight:    100,
	}
	if err := db.Create(&ability).Error; err != nil {
		t.Fatalf("seed ability: %v", err)
	}

	got, err := GetChannel("default", constant.CodexAutoReviewModel, 0, nil, nil)
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if got == nil {
		t.Fatal("expected channel, got nil")
	}
	if got.Id != 1 {
		t.Fatalf("expected codex channel 1, got %d", got.Id)
	}
}

func TestGetSatisfiedChannelCandidates_ReturnsSamePriorityCandidates(t *testing.T) {
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	originGroupMap := group2model2channels
	originChannels := channelsIDM
	common.MemoryCacheEnabled = true

	high := int64(10)
	low := int64(1)
	weight := uint(100)
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1, 2, 3},
		},
	}
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Name: "high-a", Weight: &weight, Priority: &high},
		2: {Id: 2, Name: "high-b", Weight: &weight, Priority: &high},
		3: {Id: 3, Name: "low", Weight: &weight, Priority: &low},
	}
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originMemoryCacheEnabled
		group2model2channels = originGroupMap
		channelsIDM = originChannels
	})

	candidates, err := GetSatisfiedChannelCandidates("default", "gpt-5.4", 0, nil, nil)
	if err != nil {
		t.Fatalf("get candidates: %v", err)
	}
	if len(candidates) != 2 || candidates[0].Id != 1 || candidates[1].Id != 2 {
		t.Fatalf("expected high-priority channels [1,2], got %#v", candidates)
	}

	candidates, err = GetSatisfiedChannelCandidates("default", "gpt-5.4", 0, nil, []int{1, 2})
	if err != nil {
		t.Fatalf("get fallback candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Id != 3 {
		t.Fatalf("expected lower-priority channel 3 after exclusions, got %#v", candidates)
	}
}
