package claude

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

const maxClaudeCacheControlBlocks = 4

func applyClaudeIncrementalCache(request *dto.ClaudeRequest) bool {
	if request == nil {
		return false
	}

	for messageIndex := len(request.Messages) - 1; messageIndex >= 0; messageIndex-- {
		message := &request.Messages[messageIndex]
		if message.Role != "user" {
			continue
		}

		blocks := claudeMessageContentBlocks(message)
		for blockIndex := len(blocks) - 1; blockIndex >= 0; blockIndex-- {
			if !isClaudeIncrementalCacheBlock(blocks[blockIndex]) {
				continue
			}
			if !hasClaudeCacheControl(blocks[blockIndex].CacheControl) {
				blocks[blockIndex].CacheControl = json.RawMessage(`{"type":"ephemeral"}`)
				message.Content = blocks
			}
			trimClaudeCacheControls(request)
			return true
		}
	}

	return false
}

func claudeMessageContentBlocks(message *dto.ClaudeMessage) []dto.ClaudeMediaMessage {
	if message == nil {
		return nil
	}

	switch content := message.Content.(type) {
	case string:
		if strings.TrimSpace(content) == "" {
			return nil
		}
		text := content
		blocks := []dto.ClaudeMediaMessage{{
			Type: "text",
			Text: &text,
		}}
		message.Content = blocks
		return blocks
	case []dto.ClaudeMediaMessage:
		return content
	default:
		blocks, err := message.ParseContent()
		if err != nil {
			return nil
		}
		message.Content = blocks
		return blocks
	}
}

func isClaudeIncrementalCacheBlock(block dto.ClaudeMediaMessage) bool {
	switch block.Type {
	case "text":
		return block.Text != nil && strings.TrimSpace(*block.Text) != ""
	case "tool_result":
		return block.Content != nil
	default:
		return false
	}
}

func hasClaudeCacheControl(cacheControl json.RawMessage) bool {
	cacheControl = bytes.TrimSpace(cacheControl)
	return len(cacheControl) > 0 && !bytes.Equal(cacheControl, []byte("null"))
}

func trimClaudeCacheControls(request *dto.ClaudeRequest) {
	cacheControls := collectClaudeCacheControls(request)
	for len(cacheControls) > maxClaudeCacheControlBlocks {
		*cacheControls[0] = nil
		cacheControls = cacheControls[1:]
	}
}

func collectClaudeCacheControls(request *dto.ClaudeRequest) []*json.RawMessage {
	cacheControls := make([]*json.RawMessage, 0, maxClaudeCacheControlBlocks+1)
	if system, ok := request.System.([]dto.ClaudeMediaMessage); ok {
		for index := range system {
			if hasClaudeCacheControl(system[index].CacheControl) {
				cacheControls = append(cacheControls, &system[index].CacheControl)
			}
		}
		request.System = system
	}
	for messageIndex := range request.Messages {
		blocks, ok := request.Messages[messageIndex].Content.([]dto.ClaudeMediaMessage)
		if !ok {
			continue
		}
		for blockIndex := range blocks {
			if hasClaudeCacheControl(blocks[blockIndex].CacheControl) {
				cacheControls = append(cacheControls, &blocks[blockIndex].CacheControl)
			}
		}
	}
	return cacheControls
}
