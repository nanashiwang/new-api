package controller

import (
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestParseAdminLogScope(t *testing.T) {
	for _, tc := range []struct {
		raw string
		ids []int
		bad bool
	}{
		{"", nil, false}, {"1,2,1", []int{1, 2}, false},
		{"0", nil, true}, {"-1", nil, true}, {"1,", nil, true},
		{"bad", nil, true}, {"1 OR 1=1", nil, true},
		{strings.Repeat("1,", 200) + "1", nil, true},
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/?channel_ids="+url.QueryEscape(tc.raw)+"&group_vendor=OpenAI", nil)
		scope, err := parseAdminLogScope(c)
		if (err != nil) != tc.bad {
			t.Fatalf("raw=%q err=%v", tc.raw, err)
		}
		if err == nil && (!reflect.DeepEqual(scope.ChannelIDs, tc.ids) || scope.Vendor != "OpenAI") {
			t.Fatalf("raw=%q scope=%+v", tc.raw, scope)
		}
	}
}

func TestLogScopeEndpointsUseSameSelection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	oldDB, oldLogDB, oldCache, oldLogType := model.DB, model.LOG_DB, common.MemoryCacheEnabled, common.LogSqlType
	model.DB, model.LOG_DB, common.MemoryCacheEnabled, common.LogSqlType = db, db, false, common.DatabaseTypeSQLite
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.MemoryCacheEnabled, common.LogSqlType = oldDB, oldLogDB, oldCache, oldLogType
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	if err := db.AutoMigrate(&model.Channel{}, &model.Log{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	for _, channel := range []model.Channel{
		{Id: 1, Name: "muze-a", Status: 1}, {Id: 2, Name: "muze-b", Status: 2},
	} {
		if err := db.Create(&channel).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []model.Log{
		{Type: model.LogTypeConsume, ChannelId: 1, Group: "OpenAI · 优质", Quota: 10},
		{Type: model.LogTypeConsume, ChannelId: 2, Group: "OpenAI · 独享账号", Quota: 20},
		{Type: model.LogTypeConsume, ChannelId: 2, Group: "Claude · 优质", Quota: 100},
		{Type: model.LogTypeConsume, ChannelId: 3, Group: "OpenAI · 优质", Quota: 1000},
		{Type: model.LogTypeError, ChannelId: 1, Group: "OpenAI · 优质"},
	} {
		row.CreatedAt, row.Username, row.UserId = now-2, "local-test", 1
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	query := url.Values{
		"group_vendor": {"OpenAI"}, "channel_ids": {"1,2"}, "type": {"2"},
		"username":        {" 001 "},
		"start_timestamp": {strconv.FormatInt(now-20, 10)}, "end_timestamp": {strconv.FormatInt(now, 10)},
		"page_size": {"1"}, "p": {"1"},
	}.Encode()
	// A numeric username must not leak into an exact user-ID query.
	if err := db.Create(&model.Log{
		UserId: 10, Username: "1", CreatedAt: now - 2, Type: model.LogTypeConsume,
		ChannelId: 1, Group: "OpenAI · 优质", Quota: 999,
	}).Error; err != nil {
		t.Fatal(err)
	}
	call := func(handler gin.HandlerFunc) map[string]interface{} {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/?"+query, nil)
		handler(c)
		var body map[string]interface{}
		if err := common.Unmarshal(w.Body.Bytes(), &body); err != nil || body["success"] != true {
			t.Fatalf("request failed: %s %v", w.Body.String(), err)
		}
		return body
	}
	list := call(GetAllLogs)["data"].(map[string]interface{})
	if list["total"] != float64(2) || len(list["items"].([]interface{})) != 1 {
		t.Fatalf("list mismatch: %+v", list)
	}
	stat := call(GetLogsStat)["data"].(map[string]interface{})
	if stat["quota"] != float64(30) || stat["rpm"] != float64(2) {
		t.Fatalf("stat mismatch: %+v", stat)
	}
	summary := call(GetLogGroupSummary)["data"].([]interface{})
	if len(summary) != 2 {
		t.Fatalf("summary mismatch: %+v", summary)
	}
	var count, quota float64
	for _, item := range summary {
		row := item.(map[string]interface{})
		count += row["request_count"].(float64)
		quota += row["quota"].(float64)
	}
	if count != 2 || quota != 30 {
		t.Fatalf("summary mismatch: %+v", summary)
	}
	top := call(GetTopUsers)["data"].(map[string]interface{})["by_quota"].([]interface{})
	quota = 0
	for _, item := range top {
		quota += item.(map[string]interface{})["quota"].(float64)
	}
	if quota != 30 {
		t.Fatalf("top-users scope mismatch: %+v", top)
	}
	// Plain text input resolves all matching names, including disabled channels.
	for _, tc := range []struct {
		input string
		count float64
		quota float64
	}{
		{"MUZE", 2, 30},
		{"1，2", 2, 30},
		{"9999", 0, 0},
		{"no-such-upstream", 0, 0},
		{"%_", 0, 0},
	} {
		t.Run(tc.input, func(t *testing.T) {
			values, err := url.ParseQuery(query)
			if err != nil {
				t.Fatal(err)
			}
			values.Del("channel_ids")
			values.Set("channel_keyword", tc.input)
			query = values.Encode()
			list := call(GetAllLogs)["data"].(map[string]interface{})
			stat := call(GetLogsStat)["data"].(map[string]interface{})
			if list["total"] != tc.count || stat["quota"] != tc.quota {
				t.Fatalf("name/ID scope mismatch: list=%+v stat=%+v", list, stat)
			}
			rows := call(GetLogGroupSummary)["data"].([]interface{})
			sum := float64(0)
			for _, row := range rows {
				sum += row.(map[string]interface{})["quota"].(float64)
			}
			if sum != tc.quota {
				t.Fatalf("summary scope mismatch: %+v", rows)
			}
			users := call(GetTopUsers)["data"].(map[string]interface{})["by_quota"].([]interface{})
			sum = 0
			for _, row := range users {
				sum += row.(map[string]interface{})["quota"].(float64)
			}
			if sum != tc.quota {
				t.Fatalf("top-users scope mismatch: %+v", users)
			}
		})
	}
}
