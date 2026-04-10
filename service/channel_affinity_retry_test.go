package service

import (
	"net/http/httptest"
	"testing"

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
