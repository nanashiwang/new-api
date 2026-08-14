package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func TestSanitizeClaudeRequestEmptyThinkingPreservesValidBlocks(t *testing.T) {
	empty := "  "
	valid := "real reasoning"
	text := "answer"
	request := &dto.ClaudeRequest{Messages: []dto.ClaudeMessage{
		{
			Role: "assistant",
			Content: []dto.ClaudeMediaMessage{
				{Type: "thinking", Thinking: &empty, Signature: "invalid-empty-signature"},
				{Type: "redacted_thinking", Signature: "redacted-signature"},
				{Type: "thinking", Thinking: &valid, Signature: "valid-signature"},
				{Type: "text", Text: &text},
			},
		},
	}}

	result, err := sanitizeClaudeRequestEmptyThinking(request)
	if err != nil {
		t.Fatalf("sanitizeClaudeRequestEmptyThinking() error = %v", err)
	}
	if result.RemovedBlocks != 1 || result.RemovedMessages != 0 || result.MergedMessages != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	blocks, err := request.Messages[0].ParseContent()
	if err != nil {
		t.Fatalf("ParseContent() error = %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("len(blocks) = %d, want 3", len(blocks))
	}
	if blocks[0].Type != "redacted_thinking" || blocks[0].Signature != "redacted-signature" {
		t.Fatalf("redacted thinking changed: %+v", blocks[0])
	}
	if blocks[1].Type != "thinking" || blocks[1].Thinking == nil || *blocks[1].Thinking != valid || blocks[1].Signature != "valid-signature" {
		t.Fatalf("valid thinking changed: %+v", blocks[1])
	}
	if blocks[2].GetText() != text {
		t.Fatalf("text block changed: %+v", blocks[2])
	}
}

func TestSanitizeClaudeRequestEmptyThinkingRemovesEmptyMessageAndMergesRole(t *testing.T) {
	empty := ""
	request := &dto.ClaudeRequest{Messages: []dto.ClaudeMessage{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: []dto.ClaudeMediaMessage{{Type: "thinking", Thinking: &empty}}},
		{Role: "user", Content: "second"},
	}}

	result, err := sanitizeClaudeRequestEmptyThinking(request)
	if err != nil {
		t.Fatalf("sanitizeClaudeRequestEmptyThinking() error = %v", err)
	}
	if result.RemovedBlocks != 1 || result.RemovedMessages != 1 || result.MergedMessages != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(request.Messages) != 1 || request.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v, want one merged user message", request.Messages)
	}
	blocks, err := request.Messages[0].ParseContent()
	if err != nil {
		t.Fatalf("ParseContent() error = %v", err)
	}
	if len(blocks) != 2 || blocks[0].GetText() != "first" || blocks[1].GetText() != "second" {
		t.Fatalf("merged blocks = %+v", blocks)
	}
}

func TestSanitizeClaudeRequestEmptyThinkingIsIdempotent(t *testing.T) {
	valid := "reasoning"
	request := &dto.ClaudeRequest{Messages: []dto.ClaudeMessage{
		{Role: "assistant", Content: []dto.ClaudeMediaMessage{{Type: "thinking", Thinking: &valid, Signature: "sig"}}},
	}}

	result, err := sanitizeClaudeRequestEmptyThinking(request)
	if err != nil {
		t.Fatalf("sanitizeClaudeRequestEmptyThinking() error = %v", err)
	}
	if result.Changed() {
		t.Fatalf("valid request unexpectedly changed: %+v", result)
	}
}

func TestSanitizeEmptyClaudeThinkingJSONPreservesUnknownFields(t *testing.T) {
	requestJSON := []byte(`{
		"model":"claude-test",
		"custom_top_level":{"keep":true},
		"messages":[
			{"role":"assistant","custom_message":"keep","content":[
				{"type":"thinking","thinking":"","signature":"remove-with-empty-block"},
				{"type":"thinking","thinking":"valid","signature":"keep-signature","custom_block":42},
				{"type":"redacted_thinking","data":"keep-redacted"},
				{"type":"text","text":"answer"}
			]}
		]
	}`)

	cleanedJSON, result, err := sanitizeEmptyClaudeThinkingJSON(requestJSON)
	if err != nil {
		t.Fatalf("sanitizeEmptyClaudeThinkingJSON() error = %v", err)
	}
	if result.RemovedBlocks != 1 || result.RemovedMessages != 0 || result.MergedMessages != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}

	var payload map[string]any
	if err := common.Unmarshal(cleanedJSON, &payload); err != nil {
		t.Fatalf("unmarshal cleaned JSON: %v", err)
	}
	if custom, ok := payload["custom_top_level"].(map[string]any); !ok || custom["keep"] != true {
		t.Fatalf("top-level unknown field changed: %#v", payload["custom_top_level"])
	}
	messages := payload["messages"].([]any)
	message := messages[0].(map[string]any)
	if message["custom_message"] != "keep" {
		t.Fatalf("message unknown field changed: %#v", message["custom_message"])
	}
	blocks := message["content"].([]any)
	if len(blocks) != 3 {
		t.Fatalf("len(blocks) = %d, want 3", len(blocks))
	}
	valid := blocks[0].(map[string]any)
	if valid["thinking"] != "valid" || valid["signature"] != "keep-signature" || valid["custom_block"] != float64(42) {
		t.Fatalf("valid thinking changed: %#v", valid)
	}
	if blocks[1].(map[string]any)["type"] != "redacted_thinking" {
		t.Fatalf("redacted thinking changed: %#v", blocks[1])
	}
}

func TestSanitizeEmptyClaudeThinkingJSONRemovesPureEmptyMessage(t *testing.T) {
	requestJSON := []byte(`{"messages":[
		{"role":"user","content":"first"},
		{"role":"assistant","content":[{"type":"thinking","thinking":null},{"type":"thinking"}]},
		{"role":"user","content":[{"type":"text","text":"second"}]}
	]}`)

	cleanedJSON, result, err := sanitizeEmptyClaudeThinkingJSON(requestJSON)
	if err != nil {
		t.Fatalf("sanitizeEmptyClaudeThinkingJSON() error = %v", err)
	}
	if result.RemovedBlocks != 2 || result.RemovedMessages != 1 || result.MergedMessages != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}

	var payload map[string]any
	if err := common.Unmarshal(cleanedJSON, &payload); err != nil {
		t.Fatalf("unmarshal cleaned JSON: %v", err)
	}
	messages := payload["messages"].([]any)
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	content := messages[0].(map[string]any)["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("len(content) = %d, want 2", len(content))
	}
}
