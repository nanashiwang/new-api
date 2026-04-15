package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupLogQueryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originLogDB := LOG_DB
	LOG_DB = db
	t.Cleanup(func() {
		LOG_DB = originLogDB
	})

	if err := db.AutoMigrate(&Log{}); err != nil {
		t.Fatalf("migrate logs: %v", err)
	}

	return db
}

func seedLogQueryFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()

	logs := []Log{
		{UserId: 1, Username: "alice", Type: LogTypeConsume, ModelName: "gpt-4o", Quota: 100, PromptTokens: 10, CompletionTokens: 5, Group: "default", ChannelId: 1, RequestId: "req-1", CreatedAt: 100},
		{UserId: 1, Username: "alice", Type: LogTypeConsume, ModelName: "gpt-4o", Quota: 40, PromptTokens: 3, CompletionTokens: 2, Group: "default", ChannelId: 1, RequestId: "req-2", CreatedAt: 101},
		{UserId: 2, Username: "alice-admin", Type: LogTypeConsume, ModelName: "gpt-4o", Quota: 300, PromptTokens: 20, CompletionTokens: 10, Group: "vip", ChannelId: 2, RequestId: "req-3", CreatedAt: 102},
		{UserId: 3, Username: "bob", Type: LogTypeConsume, ModelName: "gpt-4o", Quota: 80, PromptTokens: 8, CompletionTokens: 4, Group: "default", ChannelId: 1, RequestId: "req-4", CreatedAt: 103},
		{UserId: 3, Username: "bob", Type: LogTypeError, ModelName: "gpt-4o", Quota: 0, Group: "default", ChannelId: 1, RequestId: "req-5", CreatedAt: 104},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}
}

func TestGetAllLogs_FuzzyUsernameSearch(t *testing.T) {
	db := setupLogQueryTestDB(t)
	seedLogQueryFixtures(t, db)

	logs, total, err := GetAllLogs(LogTypeUnknown, 0, 0, "", "alice", "", 0, 20, 0, "", "")
	if err != nil {
		t.Fatalf("GetAllLogs: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 matching logs, got %d", total)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}
	for _, item := range logs {
		if item.Username != "alice" && item.Username != "alice-admin" {
			t.Fatalf("unexpected username in fuzzy search: %s", item.Username)
		}
	}
}

func TestSumUsedQuota_FuzzyUsernameSearch(t *testing.T) {
	db := setupLogQueryTestDB(t)
	seedLogQueryFixtures(t, db)

	stat, err := SumUsedQuota(LogTypeConsume, 0, 0, "", "alice", "", 0, "", "", true)
	if err != nil {
		t.Fatalf("SumUsedQuota: %v", err)
	}
	if stat.Quota != 440 {
		t.Fatalf("expected quota 440, got %d", stat.Quota)
	}
}

func TestGetTopUsers_SortsIndependently(t *testing.T) {
	db := setupLogQueryTestDB(t)
	seedLogQueryFixtures(t, db)

	filters := AdminLogQueryFilters{
		LogType:   LogTypeConsume,
		ModelName: "gpt-4o",
	}
	byQuota, byRequests, err := GetTopUsers(filters, 10, "desc", "asc")
	if err != nil {
		t.Fatalf("GetTopUsers: %v", err)
	}
	if len(byQuota) != 3 || len(byRequests) != 3 {
		t.Fatalf("expected 3 ranked users, got quota=%d request=%d", len(byQuota), len(byRequests))
	}
	if byQuota[0].Username != "alice-admin" {
		t.Fatalf("expected quota ranking to start with alice-admin, got %s", byQuota[0].Username)
	}
	if byRequests[0].Username != "alice-admin" && byRequests[0].Username != "bob" {
		t.Fatalf("unexpected first user in request ranking: %s", byRequests[0].Username)
	}
	if byRequests[0].RequestCount != 1 {
		t.Fatalf("expected ascending request ranking to start at 1 request, got %d", byRequests[0].RequestCount)
	}
}
