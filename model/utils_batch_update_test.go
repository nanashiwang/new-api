package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupBatchUpdateTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originDB := DB
	originLogDB := LOG_DB
	originRedisEnabled := common.RedisEnabled
	DB = db
	LOG_DB = db
	common.RedisEnabled = false
	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.RedisEnabled = originRedisEnabled
		for i := 0; i < BatchUpdateTypeCount; i++ {
			batchUpdateStores[i] = make(map[int]int)
		}
	})

	if err := db.AutoMigrate(&User{}, &Token{}, &Channel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	return db
}

func TestBatchUpdate_FlushesStoresInBulk(t *testing.T) {
	db := setupBatchUpdateTestDB(t)

	users := []User{
		{Id: 1, Username: "user-1", Password: "password1", AccessToken: common.GetPointer[string]("access-1"), AffCode: "aff-1", Quota: 100, UsedQuota: 20, RequestCount: 3},
		{Id: 2, Username: "user-2", Password: "password2", AccessToken: common.GetPointer[string]("access-2"), AffCode: "aff-2", Quota: 200, UsedQuota: 5, RequestCount: 1},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	tokens := []Token{
		{Id: 1, UserId: 1, Key: "token-key-1", RemainQuota: 1000, UsedQuota: 50},
		{Id: 2, UserId: 2, Key: "token-key-2", RemainQuota: 500, UsedQuota: 10},
	}
	if err := db.Create(&tokens).Error; err != nil {
		t.Fatalf("seed tokens: %v", err)
	}

	channels := []Channel{
		{Id: 1, Name: "channel-1", Key: "channel-key-1", UsedQuota: 10},
		{Id: 2, Name: "channel-2", Key: "channel-key-2", UsedQuota: 3},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	addNewRecord(BatchUpdateTypeUserQuota, 1, 10)
	addNewRecord(BatchUpdateTypeUserQuota, 1, -3)
	addNewRecord(BatchUpdateTypeUserQuota, 2, 5)

	addNewRecord(BatchUpdateTypeUsedQuota, 1, 20)
	addNewRecord(BatchUpdateTypeUsedQuota, 1, 30)
	addNewRecord(BatchUpdateTypeUsedQuota, 2, 7)

	addNewRecord(BatchUpdateTypeRequestCount, 1, 2)
	addNewRecord(BatchUpdateTypeRequestCount, 2, 1)

	addNewRecord(BatchUpdateTypeTokenQuota, 1, 50)
	addNewRecord(BatchUpdateTypeTokenQuota, 2, 5)

	addNewRecord(BatchUpdateTypeChannelUsedQuota, 1, 100)
	addNewRecord(BatchUpdateTypeChannelUsedQuota, 2, 7)

	batchUpdate()

	var updatedUsers []User
	if err := db.Order("id").Find(&updatedUsers).Error; err != nil {
		t.Fatalf("query users: %v", err)
	}
	if updatedUsers[0].Quota != 107 || updatedUsers[0].UsedQuota != 70 || updatedUsers[0].RequestCount != 5 {
		t.Fatalf("unexpected user 1 values: %#v", updatedUsers[0])
	}
	if updatedUsers[1].Quota != 205 || updatedUsers[1].UsedQuota != 12 || updatedUsers[1].RequestCount != 2 {
		t.Fatalf("unexpected user 2 values: %#v", updatedUsers[1])
	}

	var updatedTokens []Token
	if err := db.Order("id").Find(&updatedTokens).Error; err != nil {
		t.Fatalf("query tokens: %v", err)
	}
	if updatedTokens[0].RemainQuota != 1050 || updatedTokens[0].UsedQuota != 0 || updatedTokens[0].AccessedTime == 0 {
		t.Fatalf("unexpected token 1 values: %#v", updatedTokens[0])
	}
	if updatedTokens[1].RemainQuota != 505 || updatedTokens[1].UsedQuota != 5 || updatedTokens[1].AccessedTime == 0 {
		t.Fatalf("unexpected token 2 values: %#v", updatedTokens[1])
	}

	var updatedChannels []Channel
	if err := db.Order("id").Find(&updatedChannels).Error; err != nil {
		t.Fatalf("query channels: %v", err)
	}
	if updatedChannels[0].UsedQuota != 110 {
		t.Fatalf("unexpected channel 1 used quota: %#v", updatedChannels[0])
	}
	if updatedChannels[1].UsedQuota != 10 {
		t.Fatalf("unexpected channel 2 used quota: %#v", updatedChannels[1])
	}

	for i := 0; i < BatchUpdateTypeCount; i++ {
		if len(batchUpdateStores[i]) != 0 {
			t.Fatalf("expected store %d to be cleared, got %d entries", i, len(batchUpdateStores[i]))
		}
	}
}
