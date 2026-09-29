package openai

import (
	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

type kimiUsageCountSnapshot struct {
	evidence relaycommon.UsageCountEvidence
	total    int
	hasTotal bool
}

// Counts are snapshots, not deltas. Retain only two counts plus their matching
// totals; never retain prompts, reasoning text or the whole stream here.
type kimiUsageStreamEvidence struct {
	responseID string
	model      string
	counts     [2]kimiUsageCountSnapshot
}

func (state *kimiUsageStreamEvidence) normalize(info *relaycommon.RelayInfo, body []byte) []byte {
	if !usesKimiChatUsage(info) {
		return body
	}
	if info.KimiUsageEvidence == nil {
		info.KimiUsageEvidence = &relaycommon.KimiUsageEvidence{
			Cache:     relaycommon.UsageCountEvidence{Status: "missing"},
			Reasoning: relaycommon.UsageCountEvidence{Status: "missing"},
		}
	}
	root := usageObject(body)
	for _, identity := range []struct {
		key  string
		last *string
	}{
		{"id", &state.responseID}, {"model", &state.model},
	} {
		var value string
		if common.Unmarshal(root[identity.key], &value) == nil && value != "" {
			if *identity.last != "" && *identity.last != value {
				state.counts = [2]kimiUsageCountSnapshot{}
			}
			*identity.last = value
		}
	}
	usage := usageObject(root["usage"])
	if usage == nil {
		return body
	}
	choice := singleChoiceUsage(info, root)
	changed := false
	var observed [2]relaycommon.UsageCountEvidence
	for index, field := range kimiUsageFields {
		evidence := kimiCountEvidence(usage, choice, field)
		total, hasTotal := kimiUsageTotal(usage, field)
		previous := state.counts[index]
		if evidence.Status == "missing" && hasTotal && previous.hasTotal &&
			total == previous.total && previous.evidence.Status == "reported" &&
			previous.evidence.Value != nil {
			// Only omission can be filled. An explicit zero, null or invalid
			// current value is authoritative and cannot inherit an older value.
			details := usageObject(usage[field.details])
			if details == nil {
				details = rawUsageObject{}
			}
			value, err := common.Marshal(*previous.evidence.Value)
			if err != nil {
				return body
			}
			details[field.count] = value
			raw, err := common.Marshal(details)
			if err != nil {
				return body
			}
			usage[field.details] = raw
			evidence = previous.evidence
			evidence.RestoredFromPriorChunk = true
			changed = true
		}
		observed[index] = evidence
		state.counts[index] = kimiUsageCountSnapshot{evidence: evidence, total: total, hasTotal: hasTotal}
	}
	if changed {
		raw, err := common.Marshal(usage)
		if err != nil {
			return body
		}
		root["usage"] = raw
		normalized, err := common.Marshal(root)
		if err != nil {
			return body
		}
		body = normalized
	}
	info.KimiUsageEvidence = &relaycommon.KimiUsageEvidence{
		Cache: observed[0], Reasoning: observed[1],
	}
	return normalizeKimiChatUsage(info, body)
}
