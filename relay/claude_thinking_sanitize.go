package relay

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

type claudeThinkingSanitizeResult struct {
	RemovedBlocks   int
	RemovedMessages int
	MergedMessages  int
}

func (r claudeThinkingSanitizeResult) Changed() bool {
	return r.RemovedBlocks > 0 || r.RemovedMessages > 0 || r.MergedMessages > 0
}

// sanitizeClaudeRequestEmptyThinking removes only invalid, empty thinking blocks.
// Valid thinking blocks (including their signatures), redacted_thinking blocks,
// and every other content block are preserved unchanged.
func sanitizeClaudeRequestEmptyThinking(request *dto.ClaudeRequest) (claudeThinkingSanitizeResult, error) {
	var result claudeThinkingSanitizeResult
	if request == nil || len(request.Messages) == 0 {
		return result, nil
	}

	type retainedMessage struct {
		message       dto.ClaudeMessage
		originalIndex int
	}
	retained := make([]retainedMessage, 0, len(request.Messages))

	for messageIndex := range request.Messages {
		message := request.Messages[messageIndex]
		if message.IsStringContent() {
			retained = append(retained, retainedMessage{message: message, originalIndex: messageIndex})
			continue
		}

		blocks, err := message.ParseContent()
		if err != nil {
			return result, fmt.Errorf("parse Claude message %d content: %w", messageIndex, err)
		}
		filtered := make([]dto.ClaudeMediaMessage, 0, len(blocks))
		removedFromMessage := 0
		for blockIndex := range blocks {
			block := blocks[blockIndex]
			if isEmptyClaudeThinkingBlock(block) {
				removedFromMessage++
				continue
			}
			filtered = append(filtered, block)
		}
		if removedFromMessage == 0 {
			retained = append(retained, retainedMessage{message: message, originalIndex: messageIndex})
			continue
		}

		result.RemovedBlocks += removedFromMessage
		if len(filtered) == 0 {
			result.RemovedMessages++
			continue
		}
		message.Content = filtered
		retained = append(retained, retainedMessage{message: message, originalIndex: messageIndex})
	}

	cleaned := make([]dto.ClaudeMessage, 0, len(retained))
	lastOriginalIndex := -1
	for _, item := range retained {
		if len(cleaned) > 0 && lastOriginalIndex+1 < item.originalIndex && cleaned[len(cleaned)-1].Role == item.message.Role {
			merged, err := mergeClaudeMessageContent(cleaned[len(cleaned)-1], item.message)
			if err != nil {
				return result, err
			}
			cleaned[len(cleaned)-1] = merged
			result.MergedMessages++
		} else {
			cleaned = append(cleaned, item.message)
		}
		lastOriginalIndex = item.originalIndex
	}

	request.Messages = cleaned
	return result, nil
}

func isEmptyClaudeThinkingBlock(block dto.ClaudeMediaMessage) bool {
	return strings.EqualFold(strings.TrimSpace(block.Type), "thinking") &&
		(block.Thinking == nil || strings.TrimSpace(*block.Thinking) == "")
}

func mergeClaudeMessageContent(left dto.ClaudeMessage, right dto.ClaudeMessage) (dto.ClaudeMessage, error) {
	leftBlocks, err := claudeMessageBlocks(left)
	if err != nil {
		return dto.ClaudeMessage{}, err
	}
	rightBlocks, err := claudeMessageBlocks(right)
	if err != nil {
		return dto.ClaudeMessage{}, err
	}
	left.Content = append(leftBlocks, rightBlocks...)
	return left, nil
}

func claudeMessageBlocks(message dto.ClaudeMessage) ([]dto.ClaudeMediaMessage, error) {
	if message.IsStringContent() {
		text := message.GetStringContent()
		return []dto.ClaudeMediaMessage{{Type: dto.ContentTypeText, Text: &text}}, nil
	}
	blocks, err := message.ParseContent()
	if err != nil {
		return nil, fmt.Errorf("parse Claude %s message content for merge: %w", message.Role, err)
	}
	return blocks, nil
}

// sanitizeEmptyClaudeThinkingJSON is used only after Anthropic explicitly
// rejects a request for containing an empty thinking block. It preserves
// unknown top-level and content-block fields while applying the same repair.
func sanitizeEmptyClaudeThinkingJSON(requestJSON []byte) ([]byte, claudeThinkingSanitizeResult, error) {
	var result claudeThinkingSanitizeResult
	var payload map[string]any
	if err := common.Unmarshal(requestJSON, &payload); err != nil {
		return nil, result, err
	}
	messages, ok := payload["messages"].([]any)
	if !ok || len(messages) == 0 {
		return requestJSON, result, nil
	}

	type retainedJSONMessage struct {
		message       map[string]any
		originalIndex int
	}
	retained := make([]retainedJSONMessage, 0, len(messages))
	for messageIndex, rawMessage := range messages {
		message, ok := rawMessage.(map[string]any)
		if !ok {
			return nil, result, fmt.Errorf("Claude message %d must be an object", messageIndex)
		}
		content, ok := message["content"].([]any)
		if !ok {
			retained = append(retained, retainedJSONMessage{message: message, originalIndex: messageIndex})
			continue
		}

		filtered := make([]any, 0, len(content))
		removedFromMessage := 0
		for _, rawBlock := range content {
			block, ok := rawBlock.(map[string]any)
			if ok && isEmptyClaudeThinkingMap(block) {
				removedFromMessage++
				continue
			}
			filtered = append(filtered, rawBlock)
		}
		if removedFromMessage == 0 {
			retained = append(retained, retainedJSONMessage{message: message, originalIndex: messageIndex})
			continue
		}

		result.RemovedBlocks += removedFromMessage
		if len(filtered) == 0 {
			result.RemovedMessages++
			continue
		}
		message["content"] = filtered
		retained = append(retained, retainedJSONMessage{message: message, originalIndex: messageIndex})
	}

	cleaned := make([]any, 0, len(retained))
	lastOriginalIndex := -1
	for _, item := range retained {
		if len(cleaned) > 0 && lastOriginalIndex+1 < item.originalIndex {
			previous, previousOK := cleaned[len(cleaned)-1].(map[string]any)
			if previousOK && strings.TrimSpace(fmt.Sprint(previous["role"])) == strings.TrimSpace(fmt.Sprint(item.message["role"])) {
				previous["content"] = append(claudeJSONContentBlocks(previous["content"]), claudeJSONContentBlocks(item.message["content"])...)
				result.MergedMessages++
				lastOriginalIndex = item.originalIndex
				continue
			}
		}
		cleaned = append(cleaned, item.message)
		lastOriginalIndex = item.originalIndex
	}

	payload["messages"] = cleaned
	cleanedJSON, err := common.Marshal(payload)
	if err != nil {
		return nil, result, err
	}
	return cleanedJSON, result, nil
}

func isEmptyClaudeThinkingMap(block map[string]any) bool {
	if !strings.EqualFold(strings.TrimSpace(fmt.Sprint(block["type"])), "thinking") {
		return false
	}
	thinking, ok := block["thinking"]
	if !ok || thinking == nil {
		return true
	}
	text, ok := thinking.(string)
	return ok && strings.TrimSpace(text) == ""
}

func claudeJSONContentBlocks(content any) []any {
	switch value := content.(type) {
	case []any:
		return value
	case string:
		return []any{map[string]any{"type": dto.ContentTypeText, "text": value}}
	case nil:
		return nil
	default:
		return []any{value}
	}
}
