package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func TestShouldSkipRetryAfterChannelAffinityFailure_IgnoresRuleMiss(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	setChannelAffinityContext(ctx, channelAffinityMeta{
		CacheKey:   "test:rule:default:key",
		TTLSeconds: 300,
		RuleName:   "claude code trace",
		SkipRetry:  true,
	})

	if ShouldSkipRetryAfterChannelAffinityFailure(ctx) {
		t.Fatal("expected retry to remain enabled when affinity rule matched but no affinity channel was used")
	}
}

func TestShouldSkipRetryAfterChannelAffinityFailure_UsesAffinitySelection(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	setChannelAffinityContext(ctx, channelAffinityMeta{
		CacheKey:   "test:rule:default:key",
		TTLSeconds: 300,
		RuleName:   "claude code trace",
		SkipRetry:  true,
	})

	MarkChannelAffinityUsed(ctx, "default", 123)

	if !ShouldSkipRetryAfterChannelAffinityFailure(ctx) {
		t.Fatal("expected retry to be skipped after an affinity-selected channel failed")
	}
}

func TestShouldSkipRetryAfterChannelAffinityFailure_TemporaryUpstreamErrorAllowsRetry(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	setChannelAffinityContext(ctx, channelAffinityMeta{
		CacheKey:   "test:rule:default:key",
		TTLSeconds: 300,
		RuleName:   "claude code trace",
		SkipRetry:  true,
	})
	MarkChannelAffinityUsed(ctx, "default", 123)

	err := types.WithOpenAIError(types.OpenAIError{
		Message: "rate limited",
		Type:    "rate_limit_error",
		Code:    "rate_limit_error",
	}, 429)

	if ShouldSkipRetryAfterChannelAffinityFailure(ctx, err) {
		t.Fatal("expected temporary upstream error to bypass affinity skip-retry")
	}
}

func TestShouldSkipRetryAfterChannelAffinityFailure_InvalidRequestStillSkipsRetry(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	setChannelAffinityContext(ctx, channelAffinityMeta{
		CacheKey:   "test:rule:default:key",
		TTLSeconds: 300,
		RuleName:   "claude code trace",
		SkipRetry:  true,
	})
	MarkChannelAffinityUsed(ctx, "default", 123)

	err := types.WithOpenAIError(types.OpenAIError{
		Message: "bad input",
		Type:    "invalid_request_error",
		Code:    "invalid_request_error",
	}, 400)

	if !ShouldSkipRetryAfterChannelAffinityFailure(ctx, err) {
		t.Fatal("expected invalid request error to keep affinity skip-retry")
	}
}

func TestShouldSkipRetryAfterChannelAffinityFailure_ExplicitSkipRetryStillSkips(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	setChannelAffinityContext(ctx, channelAffinityMeta{
		CacheKey:   "test:rule:default:key",
		TTLSeconds: 300,
		RuleName:   "claude code trace",
		SkipRetry:  true,
	})
	MarkChannelAffinityUsed(ctx, "default", 123)

	err := types.WithOpenAIError(types.OpenAIError{
		Message: "rate limited",
		Type:    "rate_limit_error",
		Code:    "rate_limit_error",
	}, 429, types.ErrOptionWithSkipRetry())

	if !ShouldSkipRetryAfterChannelAffinityFailure(ctx, err) {
		t.Fatal("expected explicit skip-retry error to keep affinity skip-retry")
	}
}
