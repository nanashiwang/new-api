package model

import (
	"strings"
	"testing"
)

func TestGetUserLogsRedactsLegacyAdminIdentityFromManageContent(t *testing.T) {
	db := setupLogQueryTestDB(t)

	entry := Log{
		UserId:    1,
		Username:  "alice",
		Type:      LogTypeManage,
		Content:   "管理员(ID:7)强制禁用了用户的两步验证",
		CreatedAt: 100,
		Other:     `{"admin_info":{"admin_id":7,"admin_username":"root"}}`,
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	logs, total, err := GetUserLogs(1, LogTypeManage, 0, 0, "", "", 0, 10, "", "")
	if err != nil {
		t.Fatalf("GetUserLogs: %v", err)
	}
	if total != 1 || len(logs) != 1 {
		t.Fatalf("expected 1 log, got total=%d len=%d", total, len(logs))
	}

	if strings.Contains(logs[0].Content, "ID:7") {
		t.Fatalf("expected user-visible log content to hide admin identity, got %q", logs[0].Content)
	}
	if logs[0].Content != "管理员强制禁用了用户的两步验证" {
		t.Fatalf("unexpected redacted content: %q", logs[0].Content)
	}
}
