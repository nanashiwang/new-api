package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

func TestModelMappedHelperKeepsCodexAutoReviewWithoutMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	request := &dto.OpenAIResponsesRequest{Model: constant.CodexAutoReviewModel}
	info := &relaycommon.RelayInfo{
		OriginModelName: constant.CodexAutoReviewModel,
		RelayMode:       relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: constant.CodexAutoReviewModel,
		},
	}

	if err := ModelMappedHelper(c, info, request); err != nil {
		t.Fatalf("model mapped helper: %v", err)
	}
	if info.IsModelMapped {
		t.Fatal("codex-auto-review should not be marked as model-mapped without channel mapping")
	}
	if info.UpstreamModelName != constant.CodexAutoReviewModel {
		t.Fatalf("upstream model = %q, want %q", info.UpstreamModelName, constant.CodexAutoReviewModel)
	}
	if request.Model != constant.CodexAutoReviewModel {
		t.Fatalf("request model = %q, want %q", request.Model, constant.CodexAutoReviewModel)
	}
}

func TestModelMappedHelperKeepsCodexAutoReviewForCompact(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	originModel := ratio_setting.WithCompactModelSuffix(constant.CodexAutoReviewModel)
	request := &dto.OpenAIResponsesRequest{Model: originModel}
	info := &relaycommon.RelayInfo{
		OriginModelName: originModel,
		RelayMode:       relayconstant.RelayModeResponsesCompact,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: originModel,
		},
	}

	if err := ModelMappedHelper(c, info, request); err != nil {
		t.Fatalf("model mapped helper: %v", err)
	}
	if info.UpstreamModelName != constant.CodexAutoReviewModel {
		t.Fatalf("compact upstream model = %q, want %q", info.UpstreamModelName, constant.CodexAutoReviewModel)
	}
	if info.OriginModelName != originModel {
		t.Fatalf("compact origin model = %q, want %q", info.OriginModelName, originModel)
	}
	if request.Model != constant.CodexAutoReviewModel {
		t.Fatalf("compact request model = %q, want %q", request.Model, constant.CodexAutoReviewModel)
	}
}
