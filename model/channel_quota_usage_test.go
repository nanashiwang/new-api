package model

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupChannelQuotaUsageTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	originDB := DB
	originSQLite, originMySQL, originPG := common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL
	DB = db
	common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = true, false, false
	initCol()
	t.Cleanup(func() {
		DB = originDB
		common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = originSQLite, originMySQL, originPG
		initCol()
	})
	if err := db.AutoMigrate(&ChannelQuotaUsage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func TestIncrChannelQuotaUsageUpsert(t *testing.T) {
	setupChannelQuotaUsageTestDB(t)
	used, count, err := IncrChannelQuotaUsage("channel", "1", "day", 10, 20, 5, 1)
	if err != nil || used != 5 || count != 1 {
		t.Fatalf("first incr used=%d count=%d err=%v", used, count, err)
	}
	used, count, err = IncrChannelQuotaUsage("channel", "1", "day", 10, 20, 7, 2)
	if err != nil || used != 12 || count != 3 {
		t.Fatalf("second incr used=%d count=%d err=%v", used, count, err)
	}
}

func TestMarkUsageTriggeredCAS(t *testing.T) {
	setupChannelQuotaUsageTestDB(t)
	_, _, err := IncrChannelQuotaUsage("tag", "pool", "day", 10, 20, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := MarkUsageTriggered("tag", "pool", 10)
			if err != nil {
				t.Errorf("mark: %v", err)
			}
			results <- ok
		}()
	}
	wg.Wait()
	close(results)
	seen := 0
	for ok := range results {
		if ok {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("want exactly one first trigger, got %d", seen)
	}
}

func TestListExpiredTriggeredUsages(t *testing.T) {
	setupChannelQuotaUsageTestDB(t)
	_, _, _ = IncrChannelQuotaUsage("channel", "1", "day", 10, 20, 1, 1)
	_, _, _ = IncrChannelQuotaUsage("channel", "2", "day", 30, 40, 1, 1)
	_, _ = MarkUsageTriggered("channel", "1", 10)
	_, _ = MarkUsageTriggered("channel", "2", 30)
	usages, err := ListExpiredTriggeredUsages(25, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(usages) != 1 || usages[0].ScopeKey != "1" {
		t.Fatalf("unexpected usages: %+v", usages)
	}
	if err := ClearTriggeredFlag(usages[0].Id); err != nil {
		t.Fatal(err)
	}
	usages, _ = ListExpiredTriggeredUsages(25, 10)
	if len(usages) != 0 {
		t.Fatalf("expected cleared usage, got %+v", usages)
	}
}
