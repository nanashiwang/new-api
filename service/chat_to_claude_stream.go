package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// StreamResponseOpenAI2Claude handles first and subsequent chunks identically.
// Upstream tool indexes are keys, never allocation sizes.
func StreamResponseOpenAI2Claude(event *dto.ChatCompletionsStreamResponse, info *relaycommon.RelayInfo) ([]*dto.ClaudeResponse, error) {
	if info.ClaudeConvertInfo == nil {
		info.ClaudeConvertInfo = &relaycommon.ClaudeConvertInfo{}
	}
	state := info.ClaudeConvertInfo
	if state.Done {
		return nil, nil
	}
	var out []*dto.ClaudeResponse
	if !state.Started {
		state.Started = true
		out = append(out, &dto.ClaudeResponse{Type: "message_start", Message: &dto.ClaudeMediaMessage{
			Id: event.Id, Model: event.Model, Type: "message", Role: "assistant", Content: []any{},
			Usage: &dto.ClaudeUsage{InputTokens: info.GetEstimatePromptTokens()},
		}})
	}
	closeBlocks := func() error {
		switch state.LastMessagesType {
		case relaycommon.LastMessageTypeText, relaycommon.LastMessageTypeThinking:
			out = append(out, generateStopBlock(state.Index))
		case relaycommon.LastMessageTypeTools:
			for _, key := range state.ToolOrder {
				tool := state.ToolStates[key]
				if tool.Closed {
					continue
				}
				if !tool.Started {
					return fmt.Errorf("upstream tool call ended without id or name")
				}
				out = append(out, generateStopBlock(tool.BlockIndex))
				tool.Closed = true
			}
		}
		state.LastMessagesType = relaycommon.LastMessageTypeNone
		return nil
	}
	emitText := func(kind, text string) error {
		if text == "" {
			return nil
		}
		if state.LastMessagesType != kind {
			if err := closeBlocks(); err != nil {
				return err
			}
			state.Index = state.NextBlockIndex
			state.NextBlockIndex++
			block := &dto.ClaudeMediaMessage{Type: kind}
			if kind == relaycommon.LastMessageTypeThinking {
				block.Thinking = common.GetPointer("")
			} else {
				block.Text = common.GetPointer("")
			}
			out = append(out, &dto.ClaudeResponse{Type: "content_block_start", Index: common.GetPointer(state.Index), ContentBlock: block})
			state.LastMessagesType = kind
		}
		delta := &dto.ClaudeMediaMessage{Type: kind + "_delta"}
		if kind == relaycommon.LastMessageTypeThinking {
			delta.Thinking = common.GetPointer(text)
		} else {
			delta.Text = common.GetPointer(text)
		}
		out = append(out, &dto.ClaudeResponse{Type: "content_block_delta", Index: common.GetPointer(state.Index), Delta: delta})
		return nil
	}
	if len(event.Choices) > 1 {
		return nil, fmt.Errorf("multiple choices cannot be converted to one Claude message")
	}
	if len(event.Choices) == 0 {
		return out, nil
	}
	choice := event.Choices[0]
	if err := emitText(relaycommon.LastMessageTypeThinking, choice.Delta.GetReasoningContent()); err != nil {
		return nil, err
	}
	if err := emitText(relaycommon.LastMessageTypeText, choice.Delta.GetContentString()); err != nil {
		return nil, err
	}
	if len(choice.Delta.ToolCalls) > 0 {
		if state.LastMessagesType != relaycommon.LastMessageTypeTools {
			if err := closeBlocks(); err != nil {
				return nil, err
			}
			state.LastMessagesType = relaycommon.LastMessageTypeTools
		}
		if state.ToolStates == nil {
			state.ToolStates = make(map[int]*relaycommon.ClaudeStreamToolState)
		}
		for i, delta := range choice.Delta.ToolCalls {
			key := i
			if delta.Index != nil {
				key = *delta.Index
			}
			if key < 0 {
				return nil, fmt.Errorf("negative upstream tool call index")
			}
			tool := state.ToolStates[key]
			if tool == nil {
				if len(state.ToolStates) >= 1024 {
					return nil, fmt.Errorf("too many upstream tool calls in one message")
				}
				tool = &relaycommon.ClaudeStreamToolState{BlockIndex: state.NextBlockIndex}
				state.NextBlockIndex++
				state.ToolStates[key] = tool
				state.ToolOrder = append(state.ToolOrder, key)
			}
			if tool.Closed {
				return nil, fmt.Errorf("upstream tool call resumed after its block closed")
			}
			if delta.ID != "" {
				if tool.ID != "" && tool.ID != delta.ID {
					return nil, fmt.Errorf("upstream tool call changed identity")
				}
				tool.ID = delta.ID
			}
			if delta.Function.Name != "" {
				if tool.Name != "" && tool.Name != delta.Function.Name {
					return nil, fmt.Errorf("upstream tool call changed name")
				}
				tool.Name = delta.Function.Name
			}
			if !tool.Started {
				if len(tool.PendingArguments)+len(delta.Function.Arguments) > 64*1024 {
					return nil, fmt.Errorf("upstream tool arguments exceed buffer before identity")
				}
				tool.PendingArguments += delta.Function.Arguments
				if tool.ID == "" || tool.Name == "" {
					continue
				}
				out = append(out, &dto.ClaudeResponse{Type: "content_block_start", Index: common.GetPointer(tool.BlockIndex),
					ContentBlock: &dto.ClaudeMediaMessage{Type: "tool_use", Id: tool.ID, Name: tool.Name, Input: map[string]any{}},
				})
				tool.Started = true
				delta.Function.Arguments = tool.PendingArguments
				tool.PendingArguments = ""
			}
			if delta.Function.Arguments != "" {
				out = append(out, &dto.ClaudeResponse{Type: "content_block_delta", Index: common.GetPointer(tool.BlockIndex),
					Delta: &dto.ClaudeMediaMessage{Type: "input_json_delta", PartialJson: common.GetPointer(delta.Function.Arguments)},
				})
			}
		}
	}
	if choice.FinishReason != nil && *choice.FinishReason != "" {
		if err := closeBlocks(); err != nil {
			return nil, err
		}
		info.FinishReason = *choice.FinishReason
		usage := event.Usage
		if usage == nil {
			usage = state.Usage
		}
		out = append(out, &dto.ClaudeResponse{Type: "message_delta", Usage: buildClaudeUsageFromOpenAIUsage(usage),
			Delta: &dto.ClaudeMediaMessage{StopReason: common.GetPointer(stopReasonOpenAI2Claude(info.FinishReason))},
		}, &dto.ClaudeResponse{Type: "message_stop"})
		state.Done = true
	}
	return out, nil
}
