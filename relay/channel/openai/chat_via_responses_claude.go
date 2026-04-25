package openai

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// OaiResponsesToClaudeStreamHandler 将上游 OpenAI Responses SSE 流逐事件翻译成
// Claude Messages SSE 下发给客户端。
//
// 关键不变式：
//  1. 任何成功路径必须发出 message_start ... message_stop 的合法序列。
//  2. 任何错误/异常路径在返回 error 之前，必须先把已开启的 content_block 关闭，
//     再补发 message_delta + message_stop，否则 Claude Code 会把流当成被截断的 socket。
//  3. content_block index 单调递增；同一时刻只允许一个 block 处于 open 状态。
func OaiResponsesToClaudeStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	responseID := helper.GetResponseID(c)
	model := info.UpstreamModelName

	var (
		upstreamRespID string

		// SSE 状态机
		messageStarted    bool
		currentBlockOpen  bool
		currentBlockIndex int
		currentBlockType  string // "text" / "thinking" / "tool_use"
		nextBlockIndex    int
		sawToolUse        bool
		terminated        bool

		streamErr        *types.NewAPIError
		finalUsage       *dto.Usage
		finalClaudeUsage *dto.ClaudeUsage
		usageText        strings.Builder

		toolCallCanonicalIDByItemID = make(map[string]string)
		toolCallNameByCallID        = make(map[string]string)
		toolCallArgsByCallID        = make(map[string]string) // 已经下发给客户端的 arguments 累计串，用于 prefix-diff 补差
	)

	sendEvent := func(event *dto.ClaudeResponse) bool {
		if err := helper.ClaudeData(c, *event); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		return true
	}

	sendStartIfNeeded := func() bool {
		if messageStarted {
			return true
		}
		msg := &dto.ClaudeMediaMessage{
			Id:    responseID,
			Type:  "message",
			Role:  "assistant",
			Model: model,
			Usage: &dto.ClaudeUsage{InputTokens: info.GetEstimatePromptTokens()},
		}
		msg.SetContent(make([]any, 0))
		if !sendEvent(&dto.ClaudeResponse{Type: "message_start", Message: msg}) {
			return false
		}
		messageStarted = true
		return true
	}

	closeOpenBlock := func() bool {
		if !currentBlockOpen {
			return true
		}
		idx := currentBlockIndex
		ok := sendEvent(&dto.ClaudeResponse{
			Type:  "content_block_stop",
			Index: &idx,
		})
		currentBlockOpen = false
		currentBlockType = ""
		return ok
	}

	openBlock := func(blockType string, contentBlock *dto.ClaudeMediaMessage) bool {
		if !sendStartIfNeeded() {
			return false
		}
		if !closeOpenBlock() {
			return false
		}
		idx := nextBlockIndex
		nextBlockIndex++
		currentBlockIndex = idx
		currentBlockType = blockType
		currentBlockOpen = true
		if blockType == "tool_use" {
			sawToolUse = true
		}
		return sendEvent(&dto.ClaudeResponse{
			Type:         "content_block_start",
			Index:        &idx,
			ContentBlock: contentBlock,
		})
	}

	sendBlockDelta := func(deltaPayload *dto.ClaudeMediaMessage) bool {
		if !currentBlockOpen {
			return true
		}
		idx := currentBlockIndex
		return sendEvent(&dto.ClaudeResponse{
			Type:  "content_block_delta",
			Index: &idx,
			Delta: deltaPayload,
		})
	}

	sendTextDelta := func(delta string) bool {
		if delta == "" {
			return true
		}
		if currentBlockType != "text" {
			empty := ""
			if !openBlock("text", &dto.ClaudeMediaMessage{
				Type: "text",
				Text: &empty,
			}) {
				return false
			}
		}
		usageText.WriteString(delta)
		deltaCopy := delta
		return sendBlockDelta(&dto.ClaudeMediaMessage{
			Type: "text_delta",
			Text: &deltaCopy,
		})
	}

	sendThinkingDelta := func(delta string) bool {
		if delta == "" {
			return true
		}
		if currentBlockType != "thinking" {
			empty := ""
			if !openBlock("thinking", &dto.ClaudeMediaMessage{
				Type:     "thinking",
				Thinking: &empty,
			}) {
				return false
			}
		}
		usageText.WriteString(delta)
		deltaCopy := delta
		return sendBlockDelta(&dto.ClaudeMediaMessage{
			Type:     "thinking_delta",
			Thinking: &deltaCopy,
		})
	}

	sendToolArgsDelta := func(callID, delta string) bool {
		if delta == "" || currentBlockType != "tool_use" {
			return true
		}
		if callID != "" {
			toolCallArgsByCallID[callID] += delta
		}
		// 工具调用参数是协议结构，不算用户可见输出，不写入 usageText
		// （否则 fallback ResponseText2Usage 会高估 completion tokens）
		deltaCopy := delta
		return sendBlockDelta(&dto.ClaudeMediaMessage{
			Type:        "input_json_delta",
			PartialJson: &deltaCopy,
		})
	}

	sendStop := func(stopReason string, claudeUsage *dto.ClaudeUsage) bool {
		if terminated {
			return true
		}
		if !sendStartIfNeeded() {
			return false
		}
		if !closeOpenBlock() {
			return false
		}
		if stopReason == "" {
			stopReason = "end_turn"
		}
		deltaEvent := &dto.ClaudeResponse{
			Type: "message_delta",
			Delta: &dto.ClaudeMediaMessage{
				StopReason: &stopReason,
			},
		}
		if claudeUsage != nil {
			deltaEvent.Usage = claudeUsage
		}
		if !sendEvent(deltaEvent) {
			return false
		}
		if !sendEvent(&dto.ClaudeResponse{Type: "message_stop"}) {
			return false
		}
		terminated = true
		return true
	}

	helper.StreamScannerHandler(c, resp, info, func(data string) bool {
		if streamErr != nil {
			return false
		}

		var streamResp dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResp); err != nil {
			logger.LogError(c, "failed to unmarshal responses stream event: "+err.Error())
			return true
		}

		switch streamResp.Type {
		case "response.created":
			if streamResp.Response != nil {
				if streamResp.Response.ID != "" {
					upstreamRespID = streamResp.Response.ID
				}
				if streamResp.Response.Model != "" {
					model = streamResp.Response.Model
				}
			}
			if !sendStartIfNeeded() {
				return false
			}

		case "response.in_progress":
			// no-op

		case "response.output_item.added":
			if streamResp.Item == nil {
				break
			}
			switch streamResp.Item.Type {
			case "function_call":
				callID := strings.TrimSpace(streamResp.Item.CallId)
				itemID := strings.TrimSpace(streamResp.Item.ID)
				if callID == "" {
					callID = itemID
				}
				name := strings.TrimSpace(streamResp.Item.Name)
				if itemID != "" && callID != "" {
					toolCallCanonicalIDByItemID[itemID] = callID
				}
				if name != "" {
					toolCallNameByCallID[callID] = name
				}
				if !openBlock("tool_use", &dto.ClaudeMediaMessage{
					Type:  "tool_use",
					Id:    callID,
					Name:  name,
					Input: map[string]any{},
				}) {
					return false
				}
				if streamResp.Item.Arguments != "" {
					if !sendToolArgsDelta(callID, streamResp.Item.Arguments) {
						return false
					}
				}
			case "reasoning", "message":
				// 等 content_part / reasoning_summary_part 到来再开 block
			case dto.BuildInCallWebSearchCall:
				// 暂不在流式里发 server_tool_use；usage 在 response.completed 一并补
			}

		case "response.content_part.added":
			if streamResp.Part == nil {
				break
			}
			partType := streamResp.Part.Type
			if partType == "output_text" || partType == "" || partType == "text" {
				empty := ""
				if !openBlock("text", &dto.ClaudeMediaMessage{
					Type: "text",
					Text: &empty,
				}) {
					return false
				}
			}

		case "response.output_text.delta":
			if !sendTextDelta(streamResp.Delta) {
				return false
			}

		case "response.output_text.done", "response.content_part.done":
			if currentBlockType == "text" {
				if !closeOpenBlock() {
					return false
				}
			}

		case "response.reasoning_summary_part.added":
			if currentBlockType != "thinking" {
				empty := ""
				if !openBlock("thinking", &dto.ClaudeMediaMessage{
					Type:     "thinking",
					Thinking: &empty,
				}) {
					return false
				}
			}

		case "response.reasoning_summary_text.delta":
			if !sendThinkingDelta(streamResp.Delta) {
				return false
			}

		case "response.reasoning_summary_text.done", "response.reasoning_summary_part.done":
			// 不在这里关 thinking block：一个 reasoning output_item 内可能有多个 summary part，
			// 提前关会切碎成多个 thinking block。统一交给 output_item.done 关；
			// 若 reasoning 是最后一个内容（无 output_item.done），sendStop 兜底关。

		case "response.function_call_arguments.delta":
			itemID := strings.TrimSpace(streamResp.ItemID)
			callID := toolCallCanonicalIDByItemID[itemID]
			if callID == "" {
				callID = itemID
			}
			if itemID != "" && currentBlockType != "tool_use" {
				// 防御：未收到 output_item.added 就来 delta，临时补开 tool_use 块
				if !openBlock("tool_use", &dto.ClaudeMediaMessage{
					Type:  "tool_use",
					Id:    callID,
					Name:  toolCallNameByCallID[callID],
					Input: map[string]any{},
				}) {
					return false
				}
			}
			if !sendToolArgsDelta(callID, streamResp.Delta) {
				return false
			}

		case "response.function_call_arguments.done":
			// 等 output_item.done 关 block；上游若把完整参数只放在 done 里，由 output_item.done 分支补差

		case "response.output_item.done":
			// Some upstreams only include the complete tool arguments on item.done.
			if streamResp.Item != nil && streamResp.Item.Type == "function_call" {
				callID := strings.TrimSpace(streamResp.Item.CallId)
				itemID := strings.TrimSpace(streamResp.Item.ID)
				if callID == "" {
					callID = itemID
				}
				if itemID != "" && callID != "" {
					toolCallCanonicalIDByItemID[itemID] = callID
				}
				name := strings.TrimSpace(streamResp.Item.Name)
				if name != "" {
					toolCallNameByCallID[callID] = name
				}
				if currentBlockType != "tool_use" {
					if !openBlock("tool_use", &dto.ClaudeMediaMessage{
						Type:  "tool_use",
						Id:    callID,
						Name:  name,
						Input: map[string]any{},
					}) {
						return false
					}
				}
				newArgs := streamResp.Item.Arguments
				if newArgs != "" {
					prevArgs := toolCallArgsByCallID[callID]
					argsDelta := newArgs
					if strings.HasPrefix(newArgs, prevArgs) {
						argsDelta = newArgs[len(prevArgs):]
					}
					if !sendToolArgsDelta(callID, argsDelta) {
						return false
					}
				}
			}
			if !closeOpenBlock() {
				return false
			}

		case "response.completed":
			if streamResp.Response != nil {
				if streamResp.Response.ID != "" {
					upstreamRespID = streamResp.Response.ID
				}
				if streamResp.Response.Model != "" {
					model = streamResp.Response.Model
				}
				if strings.EqualFold(strings.TrimSpace(streamResp.Response.Status), "incomplete") {
					streamErr = newResponsesIncompleteError(streamResp.Response)
					_ = sendStop("error", finalClaudeUsage)
					return false
				}

				// 借用已有的 ResponseOpenAIResponses2Claude 把 Claude 风格 usage 算出来
				if claudeResp := service.ResponseOpenAIResponses2Claude(streamResp.Response, responseID); claudeResp != nil {
					finalClaudeUsage = claudeResp.Usage
				}

				if streamResp.Response.Usage != nil {
					finalUsage = &dto.Usage{
						PromptTokens:     streamResp.Response.Usage.InputTokens,
						CompletionTokens: streamResp.Response.Usage.OutputTokens,
						TotalTokens:      streamResp.Response.Usage.TotalTokens,
						InputTokens:      streamResp.Response.Usage.InputTokens,
						OutputTokens:     streamResp.Response.Usage.OutputTokens,
					}
					if finalUsage.TotalTokens == 0 {
						finalUsage.TotalTokens = finalUsage.PromptTokens + finalUsage.CompletionTokens
					}
					if streamResp.Response.Usage.InputTokensDetails != nil {
						finalUsage.PromptTokensDetails.CachedTokens = streamResp.Response.Usage.InputTokensDetails.CachedTokens
						finalUsage.PromptTokensDetails.ImageTokens = streamResp.Response.Usage.InputTokensDetails.ImageTokens
						finalUsage.PromptTokensDetails.AudioTokens = streamResp.Response.Usage.InputTokensDetails.AudioTokens
					}
					if streamResp.Response.Usage.CompletionTokenDetails.ReasoningTokens != 0 {
						finalUsage.CompletionTokenDetails.ReasoningTokens = streamResp.Response.Usage.CompletionTokenDetails.ReasoningTokens
					}
				}
				for _, output := range streamResp.Response.Output {
					if output.Type == dto.BuildInCallWebSearchCall {
						if finalUsage == nil {
							finalUsage = &dto.Usage{}
						}
						finalUsage.WebSearchRequests++
					}
				}

				// bridge result（保留原逻辑用于下游账务/日志）
				if chatResp, _, err := service.ResponsesResponseToChatCompletionsResponse(streamResp.Response, responseID); err == nil &&
					chatResp != nil && len(chatResp.Choices) > 0 {
					service.SetResponsesBridgeResult(c, streamResp.Response.ID, chatResp.Choices[0].Message)
				}
			}

			stopReason := "end_turn"
			if sawToolUse {
				stopReason = "tool_use"
			}
			if !sendStop(stopReason, finalClaudeUsage) {
				return false
			}

		case "error", "response.error", "response.failed", "response.incomplete":
			streamErr = newResponsesStreamEventError(streamResp)
			_ = sendStop("error", finalClaudeUsage)
			return false

		default:
			if common.DebugEnabled {
				logger.LogInfo(c, fmt.Sprintf("ignoring unknown responses stream event: %s", streamResp.Type))
			}
		}
		return true
	})

	if streamErr != nil {
		if !terminated {
			_ = sendStop("error", finalClaudeUsage)
		}
		return finalUsage, streamErr
	}

	// 上游结束但没收到 response.completed（EOF / scanner 异常）—— 仍要给客户端一个合法收尾
	if !terminated {
		stopReason := "end_turn"
		if sawToolUse {
			stopReason = "tool_use"
		}
		_ = sendStop(stopReason, finalClaudeUsage)
	}

	if finalUsage == nil || finalUsage.TotalTokens == 0 {
		finalUsage = service.ResponseText2Usage(c, usageText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
	}
	_ = upstreamRespID
	return finalUsage, nil
}
