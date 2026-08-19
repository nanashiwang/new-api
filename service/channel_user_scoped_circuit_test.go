package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func withUserScopedCircuitTestSettings(t *testing.T) {
	t.Helper()
	originRedisEnabled := common.RedisEnabled
	originEnabled := operation_setting.UserScopedCircuitBreakerEnabled
	originRanges := operation_setting.UserScopedCircuitBreakerStatusCodeRanges
	originTTL := operation_setting.UserScopedCircuitBreakerTTLSeconds
	originThreshold := operation_setting.UserScopedCircuitBreakerFailureThreshold
	common.RedisEnabled = false
	operation_setting.UserScopedCircuitBreakerEnabled = true
	requireNoError(t, operation_setting.UserScopedCircuitBreakerStatusCodesFromString("503"))
	operation_setting.UserScopedCircuitBreakerTTLSeconds = 60
	operation_setting.UserScopedCircuitBreakerFailureThreshold = 2
	resetUserScopedCircuitForTest()
	t.Cleanup(func() {
		common.RedisEnabled = originRedisEnabled
		operation_setting.UserScopedCircuitBreakerEnabled = originEnabled
		operation_setting.UserScopedCircuitBreakerStatusCodeRanges = originRanges
		operation_setting.UserScopedCircuitBreakerTTLSeconds = originTTL
		operation_setting.UserScopedCircuitBreakerFailureThreshold = originThreshold
		resetUserScopedCircuitForTest()
	})
}

func newUserScopedCircuitTestContext(userID int) *gin.Context {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	common.SetContextKey(ctx, constant.ContextKeyUserId, userID)
	return ctx
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserScopedCircuitBreakerOpensAfterThresholdForSameUserTag(t *testing.T) {
	withUserScopedCircuitTestSettings(t)
	tag := "shared-upstream"
	channel := &model.Channel{Id: 1, Status: common.ChannelStatusEnabled, Tag: &tag}
	sibling := &model.Channel{Id: 2, Status: common.ChannelStatusEnabled, Tag: &tag}
	otherTag := "other-upstream"
	other := &model.Channel{Id: 3, Status: common.ChannelStatusEnabled, Tag: &otherTag}
	ctx := newUserScopedCircuitTestContext(1001)
	err := types.NewOpenAIError(errors.New("unexpected status 503 Service Unavailable"), types.ErrorCodeBadResponseStatusCode, http.StatusServiceUnavailable)

	if RecordUserScopedCircuitFailure(ctx, channel, err) {
		t.Fatalf("first failure should not open circuit")
	}
	if IsUserScopedCircuitOpen(ctx, sibling) {
		t.Fatalf("circuit opened before reaching threshold")
	}
	if !RecordUserScopedCircuitFailure(ctx, channel, err) {
		t.Fatalf("second failure should open circuit")
	}
	if !IsUserScopedCircuitOpen(ctx, sibling) {
		t.Fatalf("same user and same tag should be short-circuited")
	}
	if IsUserScopedCircuitOpen(newUserScopedCircuitTestContext(1002), sibling) {
		t.Fatalf("other users should not be affected")
	}
	if IsUserScopedCircuitOpen(ctx, other) {
		t.Fatalf("other tags should not be affected")
	}
}

func TestUserScopedCircuitBreakerIgnoresUnconfiguredStatusAndClearsOnSuccess(t *testing.T) {
	withUserScopedCircuitTestSettings(t)
	tag := "shared-upstream"
	channel := &model.Channel{Id: 1, Status: common.ChannelStatusEnabled, Tag: &tag}
	ctx := newUserScopedCircuitTestContext(1001)
	err502 := types.NewOpenAIError(errors.New("bad gateway"), types.ErrorCodeBadResponseStatusCode, http.StatusBadGateway)

	if RecordUserScopedCircuitFailure(ctx, channel, err502) {
		t.Fatalf("unconfigured 502 should not open circuit")
	}
	if IsUserScopedCircuitOpen(ctx, channel) {
		t.Fatalf("unconfigured status should not be short-circuited")
	}

	err503 := types.NewOpenAIError(errors.New("service unavailable"), types.ErrorCodeBadResponseStatusCode, http.StatusServiceUnavailable)
	RecordUserScopedCircuitFailure(ctx, channel, err503)
	RecordUserScopedCircuitFailure(ctx, channel, err503)
	if !IsChannelUnavailableForRequestContext(ctx, channel) {
		t.Fatalf("opened circuit should make channel unavailable for this request context")
	}
	ClearUserScopedCircuit(ctx, channel)
	if IsUserScopedCircuitOpen(ctx, channel) {
		t.Fatalf("success clear should close user scoped circuit")
	}
}

func TestUserScopedCircuitBreakerIgnoresProviderOverload(t *testing.T) {
	withUserScopedCircuitTestSettings(t)
	tag := "shared-upstream"
	channel := &model.Channel{Id: 1, Status: common.ChannelStatusEnabled, Tag: &tag}
	ctx := newUserScopedCircuitTestContext(1001)
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "Our servers are currently overloaded. Please try again later.",
		Type:    "service_unavailable_error",
		Code:    "server_is_overloaded",
	}, http.StatusServiceUnavailable)

	RecordUserScopedCircuitFailure(ctx, channel, err)
	RecordUserScopedCircuitFailure(ctx, channel, err)
	if IsUserScopedCircuitOpen(ctx, channel) {
		t.Fatal("provider overload should reroute instead of opening a user-scoped circuit")
	}
}

func TestUserScopedCircuitBreakerStatusSkipsAutomaticDisable(t *testing.T) {
	withUserScopedCircuitTestSettings(t)
	originAutoDisable := common.AutomaticDisableChannelEnabled
	originDisableRanges := operation_setting.AutomaticDisableStatusCodeRanges
	common.AutomaticDisableChannelEnabled = true
	requireNoError(t, operation_setting.AutomaticDisableStatusCodesFromString("503"))
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled = originAutoDisable
		operation_setting.AutomaticDisableStatusCodeRanges = originDisableRanges
	})

	err503 := types.NewOpenAIError(errors.New("service unavailable"), types.ErrorCodeBadResponseStatusCode, http.StatusServiceUnavailable)
	if ShouldDisableChannel(0, err503) {
		t.Fatalf("user scoped circuit status should not trigger channel auto-disable")
	}
}
