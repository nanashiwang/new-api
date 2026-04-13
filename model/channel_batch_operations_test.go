package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupChannelBatchTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originDB := DB
	originLogDB := LOG_DB
	DB = db
	LOG_DB = db
	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
	})

	if err := db.AutoMigrate(&Channel{}, &Ability{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	return db
}

func TestBatchSetChannelTag_RebuildsAbilitiesWithNewTag(t *testing.T) {
	db := setupChannelBatchTestDB(t)

	priority := int64(7)
	weight := uint(30)
	channels := []Channel{
		{Id: 1, Name: "ch-1", Key: "key-1", Group: "default,vip", Models: "gpt-4o,gpt-4o", Status: common.ChannelStatusEnabled, Priority: &priority, Weight: &weight, Tag: common.GetPointer[string]("old-tag")},
		{Id: 2, Name: "ch-2", Key: "key-2", Group: "default", Models: "claude-3-5-sonnet", Status: common.ChannelStatusEnabled, Priority: &priority, Weight: &weight, Tag: common.GetPointer[string]("old-tag")},
		{Id: 3, Name: "ch-3", Key: "key-3", Group: "team", Models: "gemini-2.5-pro", Status: common.ChannelStatusEnabled, Priority: &priority, Weight: &weight, Tag: common.GetPointer[string]("keep-tag")},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	abilities := []Ability{
		{Group: "default", Model: "gpt-4o", ChannelId: 1, Enabled: true, Priority: &priority, Weight: weight, Tag: common.GetPointer[string]("old-tag")},
		{Group: "vip", Model: "gpt-4o", ChannelId: 1, Enabled: true, Priority: &priority, Weight: weight, Tag: common.GetPointer[string]("old-tag")},
		{Group: "default", Model: "claude-3-5-sonnet", ChannelId: 2, Enabled: true, Priority: &priority, Weight: weight, Tag: common.GetPointer[string]("old-tag")},
		{Group: "team", Model: "gemini-2.5-pro", ChannelId: 3, Enabled: true, Priority: &priority, Weight: weight, Tag: common.GetPointer[string]("keep-tag")},
	}
	if err := db.Create(&abilities).Error; err != nil {
		t.Fatalf("seed abilities: %v", err)
	}

	newTag := common.GetPointer[string]("new-tag")
	if err := BatchSetChannelTag([]int{1, 2}, newTag); err != nil {
		t.Fatalf("BatchSetChannelTag: %v", err)
	}

	updatedChannels, err := GetChannelsByIds([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("GetChannelsByIds: %v", err)
	}
	for _, channel := range updatedChannels {
		switch channel.Id {
		case 1, 2:
			if channel.GetTag() != "new-tag" {
				t.Fatalf("expected channel %d tag=new-tag, got %q", channel.Id, channel.GetTag())
			}
		case 3:
			if channel.GetTag() != "keep-tag" {
				t.Fatalf("expected channel 3 tag unchanged, got %q", channel.GetTag())
			}
		}
	}

	var rebuilt []Ability
	if err := db.Where("channel_id IN ?", []int{1, 2}).Order("channel_id, model, `group`").Find(&rebuilt).Error; err != nil {
		t.Fatalf("query rebuilt abilities: %v", err)
	}
	if len(rebuilt) != 3 {
		t.Fatalf("expected 3 rebuilt abilities, got %d", len(rebuilt))
	}
	for _, ability := range rebuilt {
		if ability.Tag == nil || *ability.Tag != "new-tag" {
			t.Fatalf("expected rebuilt ability tag=new-tag, got %#v", ability.Tag)
		}
	}

	var untouched Ability
	if err := db.First(&untouched, "channel_id = ?", 3).Error; err != nil {
		t.Fatalf("query untouched ability: %v", err)
	}
	if untouched.Tag == nil || *untouched.Tag != "keep-tag" {
		t.Fatalf("expected untouched ability tag=keep-tag, got %#v", untouched.Tag)
	}
}

func TestUpdateChannelClientRestrictionByTag_MergesExistingSettings(t *testing.T) {
	db := setupChannelBatchTestDB(t)

	setting1, err := common.Marshal(dto.ChannelSettings{
		Proxy:        "http://proxy-a",
		SystemPrompt: "keep-a",
	})
	if err != nil {
		t.Fatalf("marshal setting1: %v", err)
	}
	setting2, err := common.Marshal(dto.ChannelSettings{
		ForceFormat:  true,
		SystemPrompt: "keep-b",
	})
	if err != nil {
		t.Fatalf("marshal setting2: %v", err)
	}

	targetTag := "new-tag"
	channels := []Channel{
		{Id: 11, Name: "client-1", Key: "key-11", Tag: &targetTag, Setting: common.GetPointer[string](string(setting1))},
		{Id: 12, Name: "client-2", Key: "key-12", Tag: &targetTag, Setting: common.GetPointer[string](string(setting2))},
		{Id: 13, Name: "client-3", Key: "key-13", Tag: &targetTag, Setting: common.GetPointer[string]("{bad-json}")},
		{Id: 14, Name: "client-4", Key: "key-14", Tag: common.GetPointer[string]("old-tag"), Setting: common.GetPointer[string](`{"proxy":"http://proxy-old"}`)},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	mode := "allowlist"
	if err := UpdateChannelClientRestrictionByTag("old-tag", &targetTag, &mode, []string{" codex-cli ", "cursor", "codex-cli"}); err != nil {
		t.Fatalf("UpdateChannelClientRestrictionByTag: %v", err)
	}

	var updated []*Channel
	if err := db.Where("id IN ?", []int{11, 12, 13, 14}).Order("id").Find(&updated).Error; err != nil {
		t.Fatalf("query updated channels: %v", err)
	}

	for _, channel := range updated[:3] {
		if channel.Setting == nil || *channel.Setting == "" {
			t.Fatalf("expected merged setting for channel %d", channel.Id)
		}
		var setting dto.ChannelSettings
		if err := common.Unmarshal([]byte(*channel.Setting), &setting); err != nil {
			t.Fatalf("unmarshal merged setting for channel %d: %v", channel.Id, err)
		}
		if setting.ClientRestrictionMode != dto.ClientRestrictionModeAllowlist {
			t.Fatalf("expected allowlist mode for channel %d, got %q", channel.Id, setting.ClientRestrictionMode)
		}
		if len(setting.ClientRestrictionClients) != 2 || setting.ClientRestrictionClients[0] != "codex-cli" || setting.ClientRestrictionClients[1] != "cursor" {
			t.Fatalf("unexpected clients for channel %d: %#v", channel.Id, setting.ClientRestrictionClients)
		}
	}

	var settingA dto.ChannelSettings
	if err := common.Unmarshal([]byte(*updated[0].Setting), &settingA); err != nil {
		t.Fatalf("unmarshal channel 11 setting: %v", err)
	}
	if settingA.Proxy != "http://proxy-a" || settingA.SystemPrompt != "keep-a" {
		t.Fatalf("expected channel 11 fields preserved, got %#v", settingA)
	}

	var settingB dto.ChannelSettings
	if err := common.Unmarshal([]byte(*updated[1].Setting), &settingB); err != nil {
		t.Fatalf("unmarshal channel 12 setting: %v", err)
	}
	if !settingB.ForceFormat || settingB.SystemPrompt != "keep-b" {
		t.Fatalf("expected channel 12 fields preserved, got %#v", settingB)
	}

	if updated[3].Setting == nil || *updated[3].Setting != `{"proxy":"http://proxy-old"}` {
		t.Fatalf("expected old-tag channel unchanged, got %#v", updated[3].Setting)
	}
}

func TestEditChannelByTag_UpdatesAutoBanWhenProvided(t *testing.T) {
	db := setupChannelBatchTestDB(t)

	autoBanOn := 1
	autoBanOff := 0
	channels := []Channel{
		{Id: 21, Name: "tag-1", Key: "key-21", Tag: common.GetPointer[string]("target-tag"), AutoBan: &autoBanOn},
		{Id: 22, Name: "tag-2", Key: "key-22", Tag: common.GetPointer[string]("target-tag"), AutoBan: &autoBanOn},
		{Id: 23, Name: "other", Key: "key-23", Tag: common.GetPointer[string]("other-tag"), AutoBan: &autoBanOn},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	if err := EditChannelByTag("target-tag", nil, nil, nil, nil, nil, nil, &autoBanOff, nil, nil); err != nil {
		t.Fatalf("EditChannelByTag disable auto ban: %v", err)
	}

	updated, err := GetChannelsByIds([]int{21, 22, 23})
	if err != nil {
		t.Fatalf("GetChannelsByIds: %v", err)
	}

	for _, channel := range updated {
		switch channel.Id {
		case 21, 22:
			if channel.AutoBan == nil || *channel.AutoBan != autoBanOff {
				t.Fatalf("expected channel %d auto_ban=%d, got %#v", channel.Id, autoBanOff, channel.AutoBan)
			}
		case 23:
			if channel.AutoBan == nil || *channel.AutoBan != autoBanOn {
				t.Fatalf("expected other tag channel unchanged, got %#v", channel.AutoBan)
			}
		}
	}

	if err := EditChannelByTag("target-tag", nil, nil, nil, nil, nil, nil, &autoBanOn, nil, nil); err != nil {
		t.Fatalf("EditChannelByTag enable auto ban: %v", err)
	}

	updated, err = GetChannelsByIds([]int{21, 22})
	if err != nil {
		t.Fatalf("GetChannelsByIds after enable: %v", err)
	}
	for _, channel := range updated {
		if channel.AutoBan == nil || *channel.AutoBan != autoBanOn {
			t.Fatalf("expected channel %d auto_ban restored to %d, got %#v", channel.Id, autoBanOn, channel.AutoBan)
		}
	}
}

func TestEditChannelByTag_LeavesAutoBanUnchangedWhenNil(t *testing.T) {
	db := setupChannelBatchTestDB(t)

	autoBanOn := 1
	autoBanOff := 0
	channels := []Channel{
		{Id: 31, Name: "tag-1", Key: "key-31", Tag: common.GetPointer[string]("target-tag"), AutoBan: &autoBanOn},
		{Id: 32, Name: "tag-2", Key: "key-32", Tag: common.GetPointer[string]("target-tag"), AutoBan: &autoBanOff},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	priority := int64(99)
	if err := EditChannelByTag("target-tag", nil, nil, nil, nil, &priority, nil, nil, nil, nil); err != nil {
		t.Fatalf("EditChannelByTag with nil auto ban: %v", err)
	}

	updated, err := GetChannelsByIds([]int{31, 32})
	if err != nil {
		t.Fatalf("GetChannelsByIds: %v", err)
	}

	if updated[0].AutoBan == nil || *updated[0].AutoBan != autoBanOn {
		t.Fatalf("expected channel 31 auto_ban unchanged, got %#v", updated[0].AutoBan)
	}
	if updated[1].AutoBan == nil || *updated[1].AutoBan != autoBanOff {
		t.Fatalf("expected channel 32 auto_ban unchanged, got %#v", updated[1].AutoBan)
	}
	if updated[0].Priority == nil || *updated[0].Priority != priority {
		t.Fatalf("expected unrelated update to still apply, got %#v", updated[0].Priority)
	}
}
