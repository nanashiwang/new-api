package model

import (
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestLogUserFilterExactIDAcrossQueries(t *testing.T) {
	db := setupLogQueryTestDB(t)
	now := time.Now().Unix()
	for _, row := range []Log{
		{UserId: 1481, Username: "old-name", Quota: 10},
		{UserId: 1481, Username: "豆豆", Quota: 20},
		{UserId: 14810, Username: "1481", Quota: 300},
		{UserId: 148, Username: "contains1481", Quota: 400},
	} {
		row.Type, row.CreatedAt, row.Group = LogTypeConsume, now-2, "default"
		row.PromptTokens, row.CompletionTokens, row.Other = 100, 10, `{"cache_tokens":50}`
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		input string
		id    int
		count int64
		quota int
	}{
		{" 001481 ", 1481, 2, 30},
		{"999999", 999999, 0, 0},
		{"name:1481", 14810, 1, 300},
		{"豆豆", 1481, 1, 20},
		{"%", 0, 0, 0},
		{"1 OR 1=1", 0, 0, 0},
	} {
		t.Run(tc.input, func(t *testing.T) {
			rows, total, err := GetAllLogs(0, now-20, now, "", tc.input, "", 0, 1, 0, "", "")
			if err != nil || total != tc.count {
				t.Fatalf("list total=%d err=%v", total, err)
			}
			for _, row := range rows {
				if row.UserId != tc.id {
					t.Fatalf("wrong user: %+v", row)
				}
			}
			stat, err := SumUsedQuota(0, now-20, now, "", tc.input, "", 0, "", "", true)
			if err != nil || stat.Quota != tc.quota || int64(stat.Rpm) != tc.count {
				t.Fatalf("stat=%+v err=%v", stat, err)
			}
			if tc.count > 0 && (stat.CacheHitRate != 0.5 || stat.CacheGlobalRate != 0.5) {
				t.Fatalf("cache stats must follow the same user: %+v", stat)
			}
			f := AdminLogQueryFilters{Username: tc.input, StartTimestamp: now - 20, EndTimestamp: now}
			groups, err := GetLogGroupSummary(f)
			if err != nil {
				t.Fatal(err)
			}
			var sum int64
			for _, group := range groups {
				sum += group.Quota
			}
			if sum != int64(tc.quota) {
				t.Fatalf("group quota=%d", sum)
			}
			top, _, err := GetTopUsers(f, 10, "desc", "desc")
			if err != nil {
				t.Fatal(err)
			}
			for _, user := range top {
				if user.UserID != tc.id {
					t.Fatalf("wrong ranked user: %+v", user)
				}
			}
		})
	}
	for _, input := range []string{"0", strings.Repeat("9", 100), "name: "} {
		if _, _, err := GetAllLogs(0, 0, 0, "", input, "", 0, 20, 0, "", ""); err == nil {
			t.Fatalf("invalid input accepted: %q", input)
		}
		if _, err := SumUsedQuota(0, 0, 0, "", input, "", 0, "", "", true); err == nil {
			t.Fatalf("invalid stat input accepted: %q", input)
		}
	}
	// A self-service numeric username remains a username, never an arbitrary ID.
	stat, err := SumUsedQuota(0, now-20, now, "", "1481", "", 0, "", "", false)
	if err != nil || stat.Quota != 300 {
		t.Fatalf("self numeric username changed: %+v %v", stat, err)
	}
	rows, total, err := GetUserLogs(14810, 0, now-20, now, "", "", 0, 20, "", "")
	if err != nil || total != 1 || len(rows) != 1 || rows[0].UserId != 14810 {
		t.Fatalf("self logs changed: %v %d %v", rows, total, err)
	}
}

func TestLogUserFilterParameterizedAcrossDialects(t *testing.T) {
	base := setupLogQueryTestDB(t)
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		db := base.Session(&gorm.Session{DryRun: true, NewDB: true})
		config := *db.Config
		config.Dialector = logScopeTestDialect{Dialector: db.Dialector, name: dialect}
		db.Config = &config
		tx, err := applyLogUserFilter(db.Model(&Log{}), "1481", true)
		if err != nil {
			t.Fatal(err)
		}
		var rows []Log
		tx = tx.Find(&rows)
		sql := tx.Statement.SQL.String()
		if !strings.Contains(sql, "logs.user_id = ?") || strings.Contains(sql, "1481") ||
			len(tx.Statement.Vars) != 1 || tx.Statement.Vars[0] != 1481 {
			t.Fatalf("%s: query is not exact and parameterized: %s %v", dialect, sql, tx.Statement.Vars)
		}
	}
}
