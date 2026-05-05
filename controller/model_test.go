package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

func TestListModelsHidesCodexAutoReview(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originSelfUseModeEnabled := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = true
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = originSelfUseModeEnabled
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(c, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(c, constant.ContextKeyTokenModelLimit, map[string]bool{
		constant.CodexAutoReviewModel: true,
		"gpt-5":                       true,
	})

	ListModels(c, constant.ChannelTypeOpenAI)

	var resp struct {
		Data []dto.OpenAIModels `json:"data"`
	}
	if err := common.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	for _, model := range resp.Data {
		if model.Id == constant.CodexAutoReviewModel {
			t.Fatalf("codex-auto-review should be hidden from /v1/models: %#v", resp.Data)
		}
	}
	if len(resp.Data) == 0 {
		t.Fatal("expected at least one public model")
	}
}

func TestRetrieveModelAllowsCodexAutoReview(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models/"+constant.CodexAutoReviewModel, nil)
	c.Params = gin.Params{{Key: "model", Value: constant.CodexAutoReviewModel}}

	RetrieveModel(c, constant.ChannelTypeOpenAI)

	var resp struct {
		ID    string `json:"id"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := common.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("expected model response, got error: %#v", resp.Error)
	}
	if resp.ID != constant.CodexAutoReviewModel {
		t.Fatalf("model id = %q, want %q", resp.ID, constant.CodexAutoReviewModel)
	}
}
