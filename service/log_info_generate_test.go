package service

import (
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestGenerateTextOtherInfoIncludesParamOverrideAuditAndStreamStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)

	start := time.Unix(100, 0)
	info := &relaycommon.RelayInfo{
		StartTime:         start,
		FirstResponseTime: start.Add(250 * time.Millisecond),
		IsStream:          true,
		ChannelMeta:       &relaycommon.ChannelMeta{},
		ParamOverrideAudit: []string{
			"copy metadata.target_model -> model",
		},
	}
	info.StreamStatus = relaycommon.NewStreamStatus()
	info.StreamStatus.RecordError("soft failure")
	info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonTimeout, nil)

	other := GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 1, 1, 1)

	lines, ok := other["po"].([]string)
	if !ok {
		t.Fatalf("expected po to be []string, got %T", other["po"])
	}
	if len(lines) != 1 || lines[0] != "copy metadata.target_model -> model" {
		t.Fatalf("unexpected po: %#v", lines)
	}

	streamStatus, ok := other["stream_status"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected stream_status map, got %T", other["stream_status"])
	}
	if streamStatus["status"] != "error" {
		t.Fatalf("expected stream status error, got %#v", streamStatus["status"])
	}
	if streamStatus["end_reason"] != string(relaycommon.StreamEndReasonTimeout) {
		t.Fatalf("unexpected end_reason: %#v", streamStatus["end_reason"])
	}
	if streamStatus["error_count"] != 1 {
		t.Fatalf("unexpected error_count: %#v", streamStatus["error_count"])
	}
}
