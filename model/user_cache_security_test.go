package model

import (
	"sort"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupUserCacheSecurityTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originDB := DB
	originLogDB := LOG_DB
	originRedisEnabled := common.RedisEnabled
	originUsingSQLite := common.UsingSQLite
	originUsingMySQL := common.UsingMySQL
	originUsingPostgreSQL := common.UsingPostgreSQL

	DB = db
	LOG_DB = db
	common.RedisEnabled = true
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		common.RedisEnabled = originRedisEnabled
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
	})

	if err := db.AutoMigrate(&User{}, &Token{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	return db
}

func TestSyncUserCacheByIDUsesFreshDatabaseState(t *testing.T) {
	db := setupUserCacheSecurityTestDB(t)

	user := &User{
		Id:       1,
		Username: "alice-cache",
		Password: "secret",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		AffCode:  "cache-user",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Model(&User{}).Where("id = ?", user.Id).Update("status", common.UserStatusDisabled).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}

	originWriter := userCacheWriter
	defer func() {
		userCacheWriter = originWriter
	}()

	var cached User
	userCacheWriter = func(user User) error {
		cached = user
		return nil
	}

	if err := syncUserCacheByID(user.Id); err != nil {
		t.Fatalf("syncUserCacheByID: %v", err)
	}

	if cached.Id != user.Id {
		t.Fatalf("expected cached user id %d, got %d", user.Id, cached.Id)
	}
	if cached.Status != common.UserStatusDisabled {
		t.Fatalf("expected cached status %d, got %d", common.UserStatusDisabled, cached.Status)
	}
}

func TestInvalidateUserAndTokenCachesDeletesUserAndAllTokenEntries(t *testing.T) {
	db := setupUserCacheSecurityTestDB(t)

	user := &User{
		Id:       1,
		Username: "alice-invalidate",
		Password: "secret",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		AffCode:  "invalidate-user",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	tokens := []Token{
		{Id: 1, UserId: user.Id, Key: "token-cache-a", Name: "a", Status: common.TokenStatusEnabled},
		{Id: 2, UserId: user.Id, Key: "token-cache-b", Name: "b", Status: common.TokenStatusEnabled},
	}
	if err := db.Create(&tokens).Error; err != nil {
		t.Fatalf("seed tokens: %v", err)
	}

	originUserInvalidator := userCacheInvalidator
	originTokenInvalidator := tokenCacheInvalidator
	defer func() {
		userCacheInvalidator = originUserInvalidator
		tokenCacheInvalidator = originTokenInvalidator
	}()

	var invalidatedUserIDs []int
	var invalidatedTokenKeys []string
	userCacheInvalidator = func(userID int) error {
		invalidatedUserIDs = append(invalidatedUserIDs, userID)
		return nil
	}
	tokenCacheInvalidator = func(key string) error {
		invalidatedTokenKeys = append(invalidatedTokenKeys, key)
		return nil
	}

	if err := InvalidateUserAndTokenCaches(user.Id); err != nil {
		t.Fatalf("InvalidateUserAndTokenCaches: %v", err)
	}

	if len(invalidatedUserIDs) != 1 || invalidatedUserIDs[0] != user.Id {
		t.Fatalf("unexpected invalidated user ids: %#v", invalidatedUserIDs)
	}

	sort.Strings(invalidatedTokenKeys)
	wantKeys := []string{"token-cache-a", "token-cache-b"}
	for i, want := range wantKeys {
		if i >= len(invalidatedTokenKeys) || invalidatedTokenKeys[i] != want {
			t.Fatalf("unexpected invalidated token keys: %#v", invalidatedTokenKeys)
		}
	}
}
