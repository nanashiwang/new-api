package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

func TestUpdateOptionMapResponsesRequestBodyLimitMB(t *testing.T) {
	originLimit := constant.ResponsesRequestBodyLimitMB
	common.OptionMapRWMutex.Lock()
	originOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		constant.ResponsesRequestBodyLimitMB = originLimit
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	if err := updateOptionMap("ResponsesRequestBodyLimitMB", "50"); err != nil {
		t.Fatalf("update option map: %v", err)
	}

	if constant.ResponsesRequestBodyLimitMB != 50 {
		t.Fatalf("ResponsesRequestBodyLimitMB = %d, want 50", constant.ResponsesRequestBodyLimitMB)
	}
	common.OptionMapRWMutex.RLock()
	got := common.OptionMap["ResponsesRequestBodyLimitMB"]
	common.OptionMapRWMutex.RUnlock()
	if got != "50" {
		t.Fatalf("OptionMap ResponsesRequestBodyLimitMB = %q, want 50", got)
	}
}
