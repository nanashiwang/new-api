package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// Claude's any means required, not auto. Reject unknown choices rather than
// silently weakening an explicit request. The optional bool preserves false.
func claudeToolChoiceToChat(value any) (any, *bool, error) {
	if value == nil {
		return nil, nil, nil
	}
	raw, err := common.Marshal(value)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid tool_choice: %w", err)
	}
	var choice struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Parallel *bool  `json:"disable_parallel_tool_use"`
	}
	if err := common.Unmarshal(raw, &choice); err != nil {
		return nil, nil, fmt.Errorf("invalid tool_choice: expected a typed object")
	}
	if choice.Type != "tool" && choice.Name != "" {
		return nil, nil, fmt.Errorf("tool_choice.name requires type=tool")
	}
	var parallel *bool
	if choice.Parallel != nil {
		parallel = common.GetPointer(!*choice.Parallel)
	}
	switch choice.Type {
	case "auto", "none":
		return choice.Type, parallel, nil
	case "any":
		return "required", parallel, nil
	case "tool":
		if strings.TrimSpace(choice.Name) == "" {
			return nil, nil, fmt.Errorf("tool_choice.name is required for a named tool")
		}
		return map[string]any{"type": "function", "function": map[string]any{"name": choice.Name}}, parallel, nil
	default:
		return nil, nil, fmt.Errorf("unsupported tool_choice.type")
	}
}
