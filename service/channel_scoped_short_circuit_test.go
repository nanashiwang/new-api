package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func TestApplyChannelScopedShortCircuitTripsTagAndBaseURL(t *testing.T) {
	resetCRSShortCircuitForTest()
	originRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = originRedisEnabled
		resetCRSShortCircuitForTest()
	})

	tag := "shared-crs"
	otherTag := "other-crs"
	baseURL := "https://CRS.example.com/openai/"
	normalizedBaseURL := "https://crs.example.com/openai"
	otherBaseURL := "https://other.example.com/openai"
	channel := &model.Channel{Id: 10, Status: common.ChannelStatusEnabled, Tag: &tag, BaseURL: &baseURL}

	trips := ApplyChannelScopedShortCircuit(channel, ResponsesStreamMissingCompletedReason)
	if len(trips) != 2 {
		t.Fatalf("expected tag and base_url trips, got %+v", trips)
	}
	if !IsChannelUnavailableForRequest(channel) {
		t.Fatalf("expected original channel to be short-circuited")
	}
	if !IsChannelUnavailableForRequest(&model.Channel{Id: 11, Status: common.ChannelStatusEnabled, Tag: &tag, BaseURL: &otherBaseURL}) {
		t.Fatalf("expected same tag to be short-circuited")
	}
	if !IsChannelUnavailableForRequest(&model.Channel{Id: 12, Status: common.ChannelStatusEnabled, Tag: &otherTag, BaseURL: &normalizedBaseURL}) {
		t.Fatalf("expected same base_url to be short-circuited")
	}
	if IsChannelUnavailableForRequest(&model.Channel{Id: 13, Status: common.ChannelStatusEnabled, Tag: &otherTag, BaseURL: &otherBaseURL}) {
		t.Fatalf("did not expect unrelated channel to be short-circuited")
	}
}

func TestScheduleCurrentChannelScopedPreDisableWaitUsesContextBaseURLFallback(t *testing.T) {
	resetCRSShortCircuitForTest()
	originRedisEnabled := common.RedisEnabled
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	common.RedisEnabled = false
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.RedisEnabled = originRedisEnabled
		common.MemoryCacheEnabled = originMemoryCacheEnabled
		resetCRSShortCircuitForTest()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	baseURL := "https://CRS.example.com/openai/"
	normalizedBaseURL := "https://crs.example.com/openai"
	common.SetContextKey(ctx, constant.ContextKeyChannelAutoBan, true)
	common.SetContextKey(ctx, constant.ContextKeyChannelId, 77)
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	common.SetContextKey(ctx, constant.ContextKeyChannelBaseUrl, baseURL)

	scheduled, trips, err := ScheduleCurrentChannelScopedPreDisableWait(ctx, ResponsesStreamMissingCompletedReason)
	if err != nil {
		t.Fatalf("schedule scoped pre-disable: %v", err)
	}
	if !scheduled {
		t.Fatalf("expected scoped cooldown to be scheduled")
	}
	if len(trips) != 1 || trips[0].Scope != "base_url" {
		t.Fatalf("expected base_url fallback trip, got %+v", trips)
	}
	if !IsChannelUnavailableForRequest(&model.Channel{Id: 78, Status: common.ChannelStatusEnabled, BaseURL: &normalizedBaseURL}) {
		t.Fatalf("expected same base_url to be short-circuited")
	}
}
