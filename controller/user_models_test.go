package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func getUserModelsForGroup(t *testing.T, group string) []string {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group="+group, nil)
	ctx.Set("id", 1)

	GetUserModels(ctx)

	var response struct {
		Success bool     `json:"success"`
		Data    []string `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success, got body: %s", recorder.Body.String())
	}
	return response.Data
}

func TestGetUserModelsFiltersByRequestedGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)

	models := getUserModelsForGroup(t, "default")
	if len(models) != 2 || models[0] != "gpt-5.2" || models[1] != "gpt-5.2-codex" {
		t.Fatalf("unexpected default models: %#v", models)
	}

	models = getUserModelsForGroup(t, "team")
	if len(models) != 1 || models[0] != "gemini-2.5-pro" {
		t.Fatalf("unexpected team models: %#v", models)
	}
}

func TestGetUserModelsRejectsUnavailableGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)

	if models := getUserModelsForGroup(t, "forbidden"); len(models) != 0 {
		t.Fatalf("unexpected forbidden group models: %#v", models)
	}
}

func TestGetUserModelsReturnsArrayForEmptyGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		if err := setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups); err != nil {
			t.Fatalf("restore user usable groups: %v", err)
		}
	})
	if err := setting.UpdateUserUsableGroupsByJSONString(`{"default":"默认分组","vip":"VIP","team":"团队","empty":"空分组"}`); err != nil {
		t.Fatalf("enable empty group: %v", err)
	}

	models := getUserModelsForGroup(t, "empty")
	if models == nil || len(models) != 0 {
		t.Fatalf("expected empty array, got %#v", models)
	}
}

func TestGetUserModelsExpandsAutoGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTokenModelHelperDB(t)
	seedTokenModelHelperData(t)
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		if err := setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups); err != nil {
			t.Fatalf("restore user usable groups: %v", err)
		}
	})
	if err := setting.UpdateUserUsableGroupsByJSONString(`{"auto":"自动分组","default":"默认分组","vip":"VIP","team":"团队"}`); err != nil {
		t.Fatalf("enable auto group: %v", err)
	}

	models := getUserModelsForGroup(t, "auto")
	want := map[string]bool{
		"claude-sonnet-4-6": true,
		"gpt-5.2":           true,
		"gpt-5.2-codex":     true,
	}
	if len(models) != len(want) {
		t.Fatalf("unexpected auto models: %#v", models)
	}
	for _, modelName := range models {
		if !want[modelName] {
			t.Fatalf("unexpected auto model %q in %#v", modelName, models)
		}
	}
}
