package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestGetUserLogsStripsParamOverrideAuditForUserView(t *testing.T) {
	db := setupLogQueryTestDB(t)

	entry := Log{
		UserId:           1,
		Username:         "alice",
		Type:             LogTypeConsume,
		ModelName:        "gpt-4o",
		TokenName:        "token-a",
		PromptTokens:     10,
		CompletionTokens: 5,
		CreatedAt:        100,
		Other:            `{"po":["copy metadata.target_model -> model"],"admin_info":{"use_channel":[1]},"request_path":"/v1/chat/completions"}`,
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	logs, total, err := GetUserLogs(1, LogTypeConsume, 0, 0, "", "", 0, 10, "", "")
	if err != nil {
		t.Fatalf("GetUserLogs: %v", err)
	}
	if total != 1 || len(logs) != 1 {
		t.Fatalf("expected 1 log, got total=%d len=%d", total, len(logs))
	}

	otherMap, err := common.StrToMap(logs[0].Other)
	if err != nil {
		t.Fatalf("parse other: %v", err)
	}
	if _, ok := otherMap["po"]; ok {
		t.Fatalf("expected po to be stripped, got %#v", otherMap["po"])
	}
	if _, ok := otherMap["admin_info"]; ok {
		t.Fatalf("expected admin_info to be stripped")
	}
	if otherMap["request_path"] != "/v1/chat/completions" {
		t.Fatalf("expected request_path preserved, got %#v", otherMap["request_path"])
	}
}
