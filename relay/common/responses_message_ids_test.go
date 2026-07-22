package common

import (
	"encoding/json"
	"strings"
	"testing"

	basecommon "github.com/QuantumNous/new-api/common"
)

func TestNormalizeResponsesMessageIDsRepairsCCSwitchIDs(t *testing.T) {
	input := json.RawMessage(`[
		{"type":"message","id":"resp_chatcmpl-first_msg_0","role":"assistant"},
		{"type":"message","id":"resp_chatcmpl-second_msg","role":"assistant"},
		{"type":"message","id":"msg_valid","role":"assistant"},
		{"type":"function_call","id":"resp_chatcmpl-tool_msg_0","call_id":"call_1"}
	]`)

	output, count, err := NormalizeResponsesMessageIDs(input)
	if err != nil {
		t.Fatalf("NormalizeResponsesMessageIDs returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("normalized count = %d, want 2", count)
	}

	var items []map[string]any
	if err := basecommon.Unmarshal(output, &items); err != nil {
		t.Fatalf("failed to decode output: %v", err)
	}
	firstID, _ := items[0]["id"].(string)
	secondID, _ := items[1]["id"].(string)
	if !strings.HasPrefix(firstID, "msg_") || !strings.HasPrefix(secondID, "msg_") {
		t.Fatalf("normalized IDs must use msg_ prefix: %q, %q", firstID, secondID)
	}
	if firstID == secondID {
		t.Fatalf("different source IDs must remain distinct: %q", firstID)
	}
	if items[2]["id"] != "msg_valid" {
		t.Fatalf("valid message ID changed: %v", items[2]["id"])
	}
	if items[3]["id"] != "resp_chatcmpl-tool_msg_0" {
		t.Fatalf("non-message item ID changed: %v", items[3]["id"])
	}
}

func TestNormalizeResponsesMessageIDsIsDeterministic(t *testing.T) {
	input := json.RawMessage(`[{"type":"message","id":"resp_chatcmpl-same_msg_0"}]`)

	first, firstCount, err := NormalizeResponsesMessageIDs(input)
	if err != nil {
		t.Fatalf("first normalization returned error: %v", err)
	}
	second, secondCount, err := NormalizeResponsesMessageIDs(input)
	if err != nil {
		t.Fatalf("second normalization returned error: %v", err)
	}
	if firstCount != 1 || secondCount != 1 || string(first) != string(second) {
		t.Fatalf("normalization must be deterministic: first=%s second=%s", first, second)
	}
}

func TestNormalizeResponsesMessageIDsInJSONPreservesPayload(t *testing.T) {
	payload := []byte(`{"model":"gpt-5.5","stream":true,"custom":{"enabled":true},"input":[{"type":"message","id":"resp_chatcmpl-1_msg_0"}]}`)

	output, count, err := NormalizeResponsesMessageIDsInJSON(payload)
	if err != nil {
		t.Fatalf("NormalizeResponsesMessageIDsInJSON returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("normalized count = %d, want 1", count)
	}

	var body map[string]any
	if err := basecommon.Unmarshal(output, &body); err != nil {
		t.Fatalf("failed to decode output: %v", err)
	}
	if body["model"] != "gpt-5.5" || body["stream"] != true {
		t.Fatalf("standard fields changed: %v", body)
	}
	custom, _ := body["custom"].(map[string]any)
	if custom["enabled"] != true {
		t.Fatalf("custom fields changed: %v", custom)
	}
}

func TestNormalizeResponsesMessageIDsLeavesOtherInputsUntouched(t *testing.T) {
	inputs := []json.RawMessage{
		json.RawMessage(`"hello"`),
		json.RawMessage(`[{"type":"message","id":"other_invalid_id"}]`),
		json.RawMessage(`[{"type":"message","id":"resp_not_a_ccswitch_item"}]`),
	}
	for _, input := range inputs {
		output, count, err := NormalizeResponsesMessageIDs(input)
		if err != nil {
			t.Fatalf("NormalizeResponsesMessageIDs returned error: %v", err)
		}
		if count != 0 || string(output) != string(input) {
			t.Fatalf("input changed unexpectedly: input=%s output=%s count=%d", input, output, count)
		}
	}
}
