package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestValidUsageAcceptsBillableWebSearchWithoutTokenCounts(t *testing.T) {
	if !ValidUsage(&dto.Usage{WebSearchRequests: 1}) {
		t.Fatal("billable web search usage should be retained even when token counts are zero")
	}
}
