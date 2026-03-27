package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func SyncRelayReasoningEffortFromResponsesRequest(info *relaycommon.RelayInfo, req *dto.OpenAIResponsesRequest) {
	if info == nil || req == nil || req.Reasoning == nil {
		return
	}
	if effort := strings.TrimSpace(req.Reasoning.Effort); effort != "" {
		info.ReasoningEffort = effort
	}
}

func SyncRelayReasoningEffortFromResponsesPayload(info *relaycommon.RelayInfo, payload []byte) {
	if info == nil || len(payload) == 0 {
		return
	}

	var req dto.OpenAIResponsesRequest
	if err := common.Unmarshal(payload, &req); err != nil {
		return
	}
	SyncRelayReasoningEffortFromResponsesRequest(info, &req)
}
