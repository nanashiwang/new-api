package dto

import (
	"reflect"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestOpenAIResponsesResponse_UnmarshalToolChoiceUnion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		expected string
	}{
		{name: "string", payload: `{"tool_choice":"auto"}`, expected: `"auto"`},
		{name: "object", payload: `{"tool_choice":{"type":"function","name":"lookup","future_field":{"enabled":true}}}`, expected: `{"type":"function","name":"lookup","future_field":{"enabled":true}}`},
		{name: "null", payload: `{"tool_choice":null}`, expected: `null`},
		{name: "forward compatible scalar", payload: `{"tool_choice":17}`, expected: `17`},
		{name: "missing", payload: `{}`, expected: ``},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var resp OpenAIResponsesResponse
			if err := common.Unmarshal([]byte(test.payload), &resp); err != nil {
				t.Fatalf("Unmarshal returned error: %v", err)
			}
			if actual := string(resp.ToolChoice); actual != test.expected {
				t.Fatalf("ToolChoice = %q, want %q", actual, test.expected)
			}
		})
	}
}

func TestOpenAIResponsesResponse_UnmarshalRejectsMalformedToolChoiceJSON(t *testing.T) {
	t.Parallel()

	var resp OpenAIResponsesResponse
	if err := common.Unmarshal([]byte(`{"tool_choice":{"type":"function"}`), &resp); err == nil {
		t.Fatal("Unmarshal should reject malformed JSON")
	}
}

func TestOpenAIResponsesResponse_WireShapeRemainsResponsesNative(t *testing.T) {
	t.Parallel()

	payload := `{
		"id":"resp_native",
		"tool_choice":{"type":"function","name":"lookup"},
		"usage":{
			"input_tokens":101,
			"output_tokens":11,
			"total_tokens":112,
			"input_tokens_details":{"cached_tokens":90}
		}
	}`
	var resp OpenAIResponsesResponse
	if err := common.Unmarshal([]byte(payload), &resp); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	data, err := common.Marshal(&resp)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	var decoded map[string]any
	if err := common.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal marshaled response returned error: %v", err)
	}
	usage, ok := decoded["usage"].(map[string]any)
	if !ok {
		t.Fatalf("usage is not an object: %s", data)
	}
	if usage["input_tokens"] != float64(101) || usage["output_tokens"] != float64(11) || usage["total_tokens"] != float64(112) {
		t.Fatalf("Responses usage fields were not preserved: %#v", usage)
	}
	toolChoice, ok := decoded["tool_choice"].(map[string]any)
	if !ok || toolChoice["type"] != "function" || toolChoice["name"] != "lookup" {
		t.Fatalf("object tool_choice was not preserved: %#v", decoded["tool_choice"])
	}
}

func TestOpenAIChatCompletionsWireUsageWhitelist(t *testing.T) {
	t.Parallel()

	internal := Usage{
		PromptTokens:         101,
		CompletionTokens:     11,
		TotalTokens:          112,
		PromptCacheHitTokens: 90,
		UsageSemantic:        "responses",
		UsageSource:          "claude",
		WebSearchRequests:    3,
		PromptTokensDetails: InputTokenDetails{
			CachedTokens:         90,
			CachedCreationTokens: 7,
			TextTokens:           4,
			AudioTokens:          2,
			ImageTokens:          5,
		},
		CompletionTokenDetails: OutputTokenDetails{
			TextTokens:      8,
			AudioTokens:     1,
			ImageTokens:     2,
			ReasoningTokens: 6,
		},
		InputTokens:                 101,
		OutputTokens:                11,
		InputTokensDetails:          &InputTokenDetails{CachedTokens: 90},
		ClaudeCacheCreation5mTokens: 7,
		ClaudeCacheCreation1hTokens: 8,
		Cost:                        1.23,
	}
	before := internal

	nonStream := NewOpenAITextResponseWire(&OpenAITextResponse{
		Id:      "chatcmpl-test",
		Model:   "gpt-test",
		Object:  "chat.completion",
		Created: int64(1700000000),
		Usage:   internal,
	})
	stream := NewChatCompletionsStreamResponseWire(&ChatCompletionsStreamResponse{
		Id:      "chatcmpl-test",
		Model:   "gpt-test",
		Object:  "chat.completion.chunk",
		Created: 1700000000,
		Usage:   &internal,
	})

	assertWireUsageWhitelist(t, nonStream)
	assertWireUsageWhitelist(t, stream)
	if !reflect.DeepEqual(internal, before) {
		t.Fatalf("wire projection mutated internal usage: got %#v want %#v", internal, before)
	}
}

func assertWireUsageWhitelist(t *testing.T, wire any) {
	t.Helper()

	data, err := common.Marshal(wire)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	var decoded map[string]any
	if err := common.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal wire response returned error: %v", err)
	}
	usage, ok := decoded["usage"].(map[string]any)
	if !ok {
		t.Fatalf("usage is not an object: %s", data)
	}
	allowed := map[string]bool{
		"prompt_tokens":             true,
		"completion_tokens":         true,
		"total_tokens":              true,
		"prompt_tokens_details":     true,
		"completion_tokens_details": true,
	}
	if len(usage) != len(allowed) {
		t.Fatalf("usage keys = %#v, want exactly %#v", usage, allowed)
	}
	for key := range usage {
		if !allowed[key] {
			t.Fatalf("unexpected provider-specific usage key %q in %s", key, data)
		}
	}
	if usage["prompt_tokens"] != float64(101) || usage["completion_tokens"] != float64(11) || usage["total_tokens"] != float64(112) {
		t.Fatalf("core token counts were not preserved: %#v", usage)
	}
}

func TestOpenAIResponsesResponse_UnmarshalInstructionsAsRawJSON(t *testing.T) {
	t.Parallel()

	payloads := []string{
		`{"id":"resp_1","instructions":{"type":"text","text":"hello"}}`,
		`{"id":"resp_2","instructions":["alpha","beta"]}`,
		`{"id":"resp_3","instructions":"plain text"}`,
	}

	for _, payload := range payloads {
		payload := payload
		t.Run(payload, func(t *testing.T) {
			t.Parallel()

			var resp OpenAIResponsesResponse
			if err := common.Unmarshal([]byte(payload), &resp); err != nil {
				t.Fatalf("Unmarshal returned error: %v", err)
			}
			if len(resp.Instructions) == 0 {
				t.Fatalf("Instructions should preserve raw JSON for %s", payload)
			}
		})
	}
}
