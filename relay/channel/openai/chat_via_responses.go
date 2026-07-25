package openai

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func responsesStreamIndexKey(itemID string, idx *int) string {
	if itemID == "" {
		return ""
	}
	if idx == nil {
		return itemID
	}
	return fmt.Sprintf("%s:%d", itemID, *idx)
}

func stringDeltaFromPrefix(prev string, next string) string {
	if next == "" {
		return ""
	}
	if prev != "" && strings.HasPrefix(next, prev) {
		return next[len(prev):]
	}
	return next
}

func hasResponsesWebSearchOutput(resp *dto.OpenAIResponsesResponse) bool {
	if resp == nil {
		return false
	}
	for _, output := range resp.Output {
		if output.Type == dto.BuildInCallWebSearchCall {
			return true
		}
	}
	return false
}

func getResponsesIncompleteReason(resp *dto.OpenAIResponsesResponse) string {
	if resp == nil || resp.IncompleteDetails == nil {
		return ""
	}
	if reason := strings.TrimSpace(resp.IncompleteDetails.Reason); reason != "" {
		return reason
	}
	return strings.TrimSpace(resp.IncompleteDetails.Reasoning)
}

func newResponsesIncompleteError(resp *dto.OpenAIResponsesResponse) *types.NewAPIError {
	message := "responses stream incomplete"
	if reason := getResponsesIncompleteReason(resp); reason != "" {
		message = fmt.Sprintf("%s: %s", message, reason)
	} else if resp != nil && strings.TrimSpace(resp.Status) != "" {
		message = fmt.Sprintf("%s: status=%s", message, strings.TrimSpace(resp.Status))
	}
	return types.NewOpenAIError(fmt.Errorf("%s", message), types.ErrorCodeBadResponse, http.StatusInternalServerError)
}

func newResponsesStreamEventError(streamResp dto.ResponsesStreamResponse) *types.NewAPIError {
	if streamResp.Response != nil {
		if oaiErr := streamResp.Response.GetOpenAIError(); hasResponsesOpenAIError(oaiErr) {
			return types.WithOpenAIError(*oaiErr, responsesStreamErrorStatus(*oaiErr))
		}
		if strings.EqualFold(strings.TrimSpace(streamResp.Response.Status), "incomplete") {
			return newResponsesIncompleteError(streamResp.Response)
		}
	}
	if oaiErr := dto.GetOpenAIError(streamResp.Error); hasResponsesOpenAIError(oaiErr) {
		return types.WithOpenAIError(*oaiErr, responsesStreamErrorStatus(*oaiErr))
	}
	if streamResp.Type == "response.incomplete" {
		return newResponsesIncompleteError(streamResp.Response)
	}
	return types.NewOpenAIError(fmt.Errorf("responses stream error: %s", streamResp.Type), types.ErrorCodeBadResponse, http.StatusInternalServerError)
}

func hasResponsesOpenAIError(err *types.OpenAIError) bool {
	return err != nil && (strings.TrimSpace(err.Message) != "" || strings.TrimSpace(err.Type) != "" || err.Code != nil)
}

func responsesStreamErrorStatus(err types.OpenAIError) int {
	signal := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(err.Type),
		strings.TrimSpace(fmt.Sprintf("%v", err.Code)),
		strings.TrimSpace(err.Message),
	}, " "))
	for _, requestError := range []string{
		"invalid_request",
		"cyber_policy",
		"context_length_exceeded",
		"maximum context length",
		"context window",
	} {
		if strings.Contains(signal, requestError) {
			return http.StatusBadRequest
		}
	}
	return http.StatusInternalServerError
}

func OaiResponsesToChatHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	defer service.CloseResponseBodyGracefully(resp)

	var responsesResp dto.OpenAIResponsesResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	if err := common.Unmarshal(body, &responsesResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	if oaiError := responsesResp.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}
	if strings.EqualFold(strings.TrimSpace(responsesResp.Status), "incomplete") {
		return nil, newResponsesIncompleteError(&responsesResp)
	}

	return finishResponsesToChatConversion(c, info, resp, &responsesResp)
}

// finishResponsesToChatConversion 把一个完整的 Responses 响应对象转换为客户端期望的
// chat.completion / Claude / Gemini 非流式 JSON 并写出,返回计费 usage。
// 供「上游非流式直连」(OaiResponsesToChatHandler)与「上游强制流式后聚合」
// (OaiResponsesToChatAggregateHandler)两条路径共用,保证转换与计费口径完全一致。
func finishResponsesToChatConversion(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response, responsesResp *dto.OpenAIResponsesResponse) (*dto.Usage, *types.NewAPIError) {
	chatId := helper.GetResponseID(c)
	chatResp, usage, err := service.ResponsesResponseToChatCompletionsResponseWithToolProtocol(responsesResp, chatId, info.ChatToolProtocol)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	if usage == nil || usage.TotalTokens == 0 {
		text := service.ExtractOutputTextFromResponses(responsesResp)
		usage = service.ResponseText2Usage(c, text, info.UpstreamModelName, info.GetEstimatePromptTokens())
		chatResp.Usage = *usage
	}
	if len(chatResp.Choices) > 0 {
		service.SetResponsesBridgeResult(c, responsesResp.ID, chatResp.Choices[0].Message)
	}

	var responseBody []byte
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		claudeResp := service.ResponseOpenAIResponses2Claude(responsesResp, chatId)
		responseBody, err = common.Marshal(claudeResp)
	case types.RelayFormatGemini:
		geminiResp := service.ResponseOpenAI2Gemini(chatResp, info)
		responseBody, err = common.Marshal(geminiResp)
	default:
		responseBody, err = common.Marshal(dto.NewOpenAITextResponseWire(chatResp))
	}
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}

	service.IOCopyBytesGracefully(c, resp, responseBody)
	return usage, nil
}

// OaiResponsesToChatAggregateHandler 与 OaiResponsesToChatHandler 对应,但用于「codex 强制流式」
// 场景:客户端要非流式的 chat/claude/gemini 结果,而上游被 newapi 强制返回 SSE。这里先把上游 SSE
// 聚合成完整 Responses 对象,再走与非流式直连完全相同的转换路径。上游是 SSE,写出前需把响应头改回
// application/json(sanitizeAggregatedResponseHeaders),避免客户端把 JSON 误当作 SSE。
func OaiResponsesToChatAggregateHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	finalResponse, apiErr := aggregateResponsesStream(c, info, resp)
	if apiErr != nil {
		return nil, apiErr
	}
	sanitizeAggregatedResponseHeaders(resp)
	return finishResponsesToChatConversion(c, info, resp, finalResponse)
}

func OaiResponsesToChatStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	responseId := helper.GetResponseID(c)
	createAt := time.Now().Unix()
	model := info.UpstreamModelName

	var (
		usage            = &dto.Usage{}
		outputText       strings.Builder
		usageText        strings.Builder
		sentStart        bool
		sentStop         bool
		sawToolCall      bool
		streamErr        *types.NewAPIError
		upstreamResponse string
		legacyToolCallID string
	)

	toolCallIndexByID := make(map[string]int)
	toolCallNameByID := make(map[string]string)
	toolCallArgsByID := make(map[string]string)
	// SendMessage needs complete JSON before conditional Claude fields can be repaired.
	toolCallBufferedArgsByID := make(map[string]string)
	toolCallNameSent := make(map[string]bool)
	toolCallCanonicalIDByItemID := make(map[string]string)
	hasSentReasoningSummary := false
	needsReasoningSummarySeparator := false
	//reasoningSummaryTextByKey := make(map[string]string)

	if info.RelayFormat == types.RelayFormatClaude && info.ClaudeConvertInfo == nil {
		info.ClaudeConvertInfo = &relaycommon.ClaudeConvertInfo{LastMessagesType: relaycommon.LastMessageTypeNone}
	}

	sendChatChunk := func(chunk *dto.ChatCompletionsStreamResponse) bool {
		if chunk == nil {
			return true
		}
		if info.RelayFormat == types.RelayFormatOpenAI {
			if err := helper.ObjectData(c, dto.NewChatCompletionsStreamResponseWire(chunk)); err != nil {
				streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
				return false
			}
			return true
		}

		chunkData, err := common.Marshal(chunk)
		if err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
			return false
		}
		if err := HandleStreamFormat(c, info, string(chunkData), false, false); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		return true
	}

	sendStartIfNeeded := func() bool {
		if sentStart {
			return true
		}
		if !sendChatChunk(helper.GenerateStartEmptyResponse(responseId, createAt, model, nil)) {
			return false
		}
		sentStart = true
		return true
	}

	//sendReasoningDelta := func(delta string) bool {
	//	if delta == "" {
	//		return true
	//	}
	//	if !sendStartIfNeeded() {
	//		return false
	//	}
	//
	//	usageText.WriteString(delta)
	//	chunk := &dto.ChatCompletionsStreamResponse{
	//		Id:      responseId,
	//		Object:  "chat.completion.chunk",
	//		Created: createAt,
	//		Model:   model,
	//		Choices: []dto.ChatCompletionsStreamResponseChoice{
	//			{
	//				Index: 0,
	//				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
	//					ReasoningContent: &delta,
	//				},
	//			},
	//		},
	//	}
	//	if err := helper.ObjectData(c, chunk); err != nil {
	//		streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
	//		return false
	//	}
	//	return true
	//}

	sendReasoningSummaryDelta := func(delta string) bool {
		if delta == "" {
			return true
		}
		if needsReasoningSummarySeparator {
			if strings.HasPrefix(delta, "\n\n") {
				needsReasoningSummarySeparator = false
			} else if strings.HasPrefix(delta, "\n") {
				delta = "\n" + delta
				needsReasoningSummarySeparator = false
			} else {
				delta = "\n\n" + delta
				needsReasoningSummarySeparator = false
			}
		}
		if !sendStartIfNeeded() {
			return false
		}

		usageText.WriteString(delta)
		chunk := &dto.ChatCompletionsStreamResponse{
			Id:      responseId,
			Object:  "chat.completion.chunk",
			Created: createAt,
			Model:   model,
			Choices: []dto.ChatCompletionsStreamResponseChoice{
				{
					Index: 0,
					Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
						ReasoningContent: &delta,
					},
				},
			},
		}
		if !sendChatChunk(chunk) {
			return false
		}
		hasSentReasoningSummary = true
		return true
	}

	sendToolCallDelta := func(callID string, name string, argsDelta string) bool {
		if callID == "" {
			return true
		}
		if info.ChatToolProtocol == dto.ChatToolProtocolLegacy {
			if legacyToolCallID == "" {
				legacyToolCallID = callID
			} else if legacyToolCallID != callID {
				streamErr = types.NewOpenAIError(
					fmt.Errorf("legacy functions protocol cannot represent multiple function calls"),
					types.ErrorCodeBadResponse,
					http.StatusBadGateway,
				)
				return false
			}
		}
		if !sendStartIfNeeded() {
			return false
		}

		idx, ok := toolCallIndexByID[callID]
		if !ok {
			idx = len(toolCallIndexByID)
			toolCallIndexByID[callID] = idx
		}
		if name != "" {
			toolCallNameByID[callID] = name
		}
		if toolCallNameByID[callID] != "" {
			name = toolCallNameByID[callID]
		}

		function := dto.FunctionResponse{Arguments: argsDelta}
		if name != "" && !toolCallNameSent[callID] {
			function.Name = name
			toolCallNameSent[callID] = true
		}
		delta := dto.ChatCompletionsStreamResponseChoiceDelta{}
		if info.ChatToolProtocol == dto.ChatToolProtocolLegacy {
			delta.FunctionCall = &function
		} else {
			tool := dto.ToolCallResponse{
				ID:       callID,
				Type:     "function",
				Function: function,
			}
			tool.SetIndex(idx)
			delta.ToolCalls = []dto.ToolCallResponse{tool}
		}

		chunk := &dto.ChatCompletionsStreamResponse{
			Id:      responseId,
			Object:  "chat.completion.chunk",
			Created: createAt,
			Model:   model,
			Choices: []dto.ChatCompletionsStreamResponseChoice{
				{
					Index: 0,
					Delta: delta,
				},
			},
		}
		if !sendChatChunk(chunk) {
			return false
		}
		sawToolCall = true

		// Include tool call data in the local builder for fallback token estimation.
		if function.Name != "" {
			usageText.WriteString(function.Name)
		}
		if argsDelta != "" {
			usageText.WriteString(argsDelta)
		}
		return true
	}

	emitCompletedToolCalls := func(response *dto.OpenAIResponsesResponse) bool {
		if response == nil {
			return true
		}
		for _, output := range response.Output {
			if output.Type != "function_call" {
				continue
			}
			callID := strings.TrimSpace(output.CallId)
			if callID == "" {
				callID = strings.TrimSpace(output.ID)
			}
			if callID == "" {
				streamErr = types.NewOpenAIError(
					fmt.Errorf("completed function call is missing call_id"),
					types.ErrorCodeBadResponse,
					http.StatusBadGateway,
				)
				return false
			}

			name := strings.TrimSpace(output.Name)
			fullArgs := output.ArgumentsString()
			previousArgs := toolCallArgsByID[callID]
			argsDelta := ""
			switch {
			case previousArgs == "":
				argsDelta = fullArgs
			case fullArgs == previousArgs:
			case strings.HasPrefix(fullArgs, previousArgs):
				argsDelta = fullArgs[len(previousArgs):]
			default:
				streamErr = types.NewOpenAIError(
					fmt.Errorf("completed function call arguments do not match streamed arguments"),
					types.ErrorCodeBadResponse,
					http.StatusBadGateway,
				)
				return false
			}
			if fullArgs != "" {
				toolCallArgsByID[callID] = fullArgs
			}
			if toolCallNameSent[callID] && argsDelta == "" {
				continue
			}
			if !sendToolCallDelta(callID, name, argsDelta) {
				return false
			}
		}
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
					upstreamResponse = streamResp.Response.ID
				}
				if streamResp.Response.Model != "" {
					model = streamResp.Response.Model
				}
				if streamResp.Response.CreatedAt != 0 {
					createAt = int64(streamResp.Response.CreatedAt)
				}
			}

		//case "response.reasoning_text.delta":
		//if !sendReasoningDelta(streamResp.Delta) {
		//	return false
		//}

		//case "response.reasoning_text.done":

		case "response.reasoning_summary_text.delta":
			if !sendReasoningSummaryDelta(streamResp.Delta) {
				return false
			}

		case "response.reasoning_summary_text.done":
			if hasSentReasoningSummary {
				needsReasoningSummarySeparator = true
			}

		//case "response.reasoning_summary_part.added", "response.reasoning_summary_part.done":
		//	key := responsesStreamIndexKey(strings.TrimSpace(streamResp.ItemID), streamResp.SummaryIndex)
		//	if key == "" || streamResp.Part == nil {
		//		break
		//	}
		//	// Only handle summary text parts, ignore other part types.
		//	if streamResp.Part.Type != "" && streamResp.Part.Type != "summary_text" {
		//		break
		//	}
		//	prev := reasoningSummaryTextByKey[key]
		//	next := streamResp.Part.Text
		//	delta := stringDeltaFromPrefix(prev, next)
		//	reasoningSummaryTextByKey[key] = next
		//	if !sendReasoningSummaryDelta(delta) {
		//		return false
		//	}

		case "response.output_text.delta":
			if !sendStartIfNeeded() {
				return false
			}

			if streamResp.Delta != "" {
				outputText.WriteString(streamResp.Delta)
				usageText.WriteString(streamResp.Delta)
				delta := streamResp.Delta
				chunk := &dto.ChatCompletionsStreamResponse{
					Id:      responseId,
					Object:  "chat.completion.chunk",
					Created: createAt,
					Model:   model,
					Choices: []dto.ChatCompletionsStreamResponseChoice{
						{
							Index: 0,
							Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
								Content: &delta,
							},
						},
					},
				}
				if !sendChatChunk(chunk) {
					return false
				}
			}

		case "response.output_item.added", "response.output_item.done":
			if streamResp.Item == nil {
				break
			}
			if streamResp.Item.Type == dto.BuildInCallWebSearchCall {
				if streamResp.Type == dto.ResponsesOutputTypeItemDone {
					usage.WebSearchRequests++
				}
				break
			}
			if streamResp.Item.Type != "function_call" {
				break
			}

			itemID := strings.TrimSpace(streamResp.Item.ID)
			callID := strings.TrimSpace(streamResp.Item.CallId)
			if callID == "" {
				callID = toolCallCanonicalIDByItemID[itemID]
				if callID == "" {
					callID = itemID
				}
			}
			if itemID != "" && callID != "" {
				toolCallCanonicalIDByItemID[itemID] = callID
			}
			name := strings.TrimSpace(streamResp.Item.Name)
			if name != "" {
				toolCallNameByID[callID] = name
			} else {
				name = toolCallNameByID[callID]
			}

			newArgs := streamResp.Item.ArgumentsString()
			isClaudeSendMessage := info.RelayFormat == types.RelayFormatClaude &&
				strings.EqualFold(name, "SendMessage")
			if isClaudeSendMessage && streamResp.Type == dto.ResponsesOutputTypeItemAdded {
				toolCallBufferedArgsByID[callID] += newArgs
				if !sendToolCallDelta(callID, name, "") {
					return false
				}
				break
			}
			if isClaudeSendMessage {
				if newArgs == "" {
					newArgs = toolCallBufferedArgsByID[callID]
				}
				newArgs = service.SanitizeClaudeToolArguments(name, newArgs)
				delete(toolCallBufferedArgsByID, callID)
			}
			prevArgs := toolCallArgsByID[callID]
			argsDelta := ""
			if newArgs != "" {
				if strings.HasPrefix(newArgs, prevArgs) {
					argsDelta = newArgs[len(prevArgs):]
				} else {
					argsDelta = newArgs
				}
				toolCallArgsByID[callID] = newArgs
			}

			if !sendToolCallDelta(callID, name, argsDelta) {
				return false
			}

		case "response.function_call_arguments.delta":
			itemID := strings.TrimSpace(streamResp.ItemID)
			callID := toolCallCanonicalIDByItemID[itemID]
			if callID == "" {
				callID = itemID
			}
			if callID == "" {
				break
			}
			name := toolCallNameByID[callID]
			if info.RelayFormat == types.RelayFormatClaude && strings.EqualFold(name, "SendMessage") {
				toolCallBufferedArgsByID[callID] += streamResp.Delta
				break
			}
			toolCallArgsByID[callID] += streamResp.Delta
			if !sendToolCallDelta(callID, "", streamResp.Delta) {
				return false
			}

		case "response.function_call_arguments.done":

		case "response.completed":
			if streamResp.Response != nil {
				if strings.EqualFold(strings.TrimSpace(streamResp.Response.Status), "incomplete") {
					streamErr = newResponsesIncompleteError(streamResp.Response)
					return false
				}
				if streamResp.Response.ID != "" {
					upstreamResponse = streamResp.Response.ID
				}
				if streamResp.Response.Model != "" {
					model = streamResp.Response.Model
				}
				if streamResp.Response.CreatedAt != 0 {
					createAt = int64(streamResp.Response.CreatedAt)
				}
				if streamResp.Response.Usage != nil {
					if streamResp.Response.Usage.InputTokens != 0 {
						usage.PromptTokens = streamResp.Response.Usage.InputTokens
						usage.InputTokens = streamResp.Response.Usage.InputTokens
					}
					if streamResp.Response.Usage.OutputTokens != 0 {
						usage.CompletionTokens = streamResp.Response.Usage.OutputTokens
						usage.OutputTokens = streamResp.Response.Usage.OutputTokens
					}
					if streamResp.Response.Usage.TotalTokens != 0 {
						usage.TotalTokens = streamResp.Response.Usage.TotalTokens
					} else {
						usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
					}
					if streamResp.Response.Usage.InputTokensDetails != nil {
						usage.PromptTokensDetails.CachedTokens = streamResp.Response.Usage.InputTokensDetails.CachedTokens
						usage.PromptTokensDetails.ImageTokens = streamResp.Response.Usage.InputTokensDetails.ImageTokens
						usage.PromptTokensDetails.AudioTokens = streamResp.Response.Usage.InputTokensDetails.AudioTokens
					}
					if streamResp.Response.Usage.CompletionTokenDetails.ReasoningTokens != 0 {
						usage.CompletionTokenDetails.ReasoningTokens = streamResp.Response.Usage.CompletionTokenDetails.ReasoningTokens
					}
				}
				if usage.WebSearchRequests == 0 {
					for _, output := range streamResp.Response.Output {
						if output.Type == dto.BuildInCallWebSearchCall {
							usage.WebSearchRequests++
						}
					}
				}
				if !emitCompletedToolCalls(streamResp.Response) {
					return false
				}
			}

			if info.RelayFormat == types.RelayFormatClaude &&
				!sentStart &&
				streamResp.Response != nil &&
				hasResponsesWebSearchOutput(streamResp.Response) {
				claudeResp := service.ResponseOpenAIResponses2Claude(streamResp.Response, responseId)
				for _, event := range service.StreamClaudeResponse(claudeResp) {
					if err := helper.ClaudeData(c, *event); err != nil {
						streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
						return false
					}
				}
				sentStart = true
				sentStop = true
				break
			}

			if !sendStartIfNeeded() {
				return false
			}
			if !sentStop {
				if info.RelayFormat == types.RelayFormatClaude && info.ClaudeConvertInfo != nil {
					info.ClaudeConvertInfo.Usage = usage
				}
				finishReason := "stop"
				if sawToolCall {
					finishReason = "tool_calls"
					if info.ChatToolProtocol == dto.ChatToolProtocolLegacy {
						finishReason = "function_call"
					}
				}
				stop := helper.GenerateStopResponse(responseId, createAt, model, finishReason)
				if !sendChatChunk(stop) {
					return false
				}
				sentStop = true
			}

		case "error", "response.error", "response.failed", "response.incomplete":
			streamErr = newResponsesStreamEventError(streamResp)
			return false

		default:
		}

		return true
	})

	if streamErr != nil {
		return nil, streamErr
	}

	if usage.TotalTokens == 0 {
		usage = service.ResponseText2Usage(c, usageText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
	}

	if !sentStart {
		if !sendChatChunk(helper.GenerateStartEmptyResponse(responseId, createAt, model, nil)) {
			return nil, streamErr
		}
	}
	if !sentStop {
		if info.RelayFormat == types.RelayFormatClaude && info.ClaudeConvertInfo != nil {
			info.ClaudeConvertInfo.Usage = usage
		}
		finishReason := "stop"
		if sawToolCall {
			finishReason = "tool_calls"
			if info.ChatToolProtocol == dto.ChatToolProtocolLegacy {
				finishReason = "function_call"
			}
		}
		stop := helper.GenerateStopResponse(responseId, createAt, model, finishReason)
		if !sendChatChunk(stop) {
			return nil, streamErr
		}
	}
	if info.RelayFormat == types.RelayFormatOpenAI && info.ShouldIncludeUsage && usage != nil {
		if err := helper.ObjectData(c, helper.GenerateFinalUsageResponse(responseId, createAt, model, *usage)); err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
		}
	}

	if info.RelayFormat == types.RelayFormatOpenAI {
		helper.Done(c)
	}
	if upstreamResponse != "" {
		assistantMessage := dto.Message{Role: "assistant"}
		if outputText.Len() > 0 {
			assistantMessage.SetStringContent(outputText.String())
		}
		if sawToolCall {
			type orderedToolCall struct {
				index int
				call  dto.ToolCallRequest
			}
			var toolCalls []orderedToolCall
			for callID, index := range toolCallIndexByID {
				toolCall := dto.ToolCallRequest{
					ID:   callID,
					Type: "function",
					Function: dto.FunctionRequest{
						Name:      toolCallNameByID[callID],
						Arguments: toolCallArgsByID[callID],
					},
				}
				toolCalls = append(toolCalls, orderedToolCall{index: index, call: toolCall})
			}
			sort.Slice(toolCalls, func(i, j int) bool {
				return toolCalls[i].index < toolCalls[j].index
			})
			normalizedToolCalls := make([]dto.ToolCallRequest, 0, len(toolCalls))
			for _, item := range toolCalls {
				normalizedToolCalls = append(normalizedToolCalls, item.call)
			}
			assistantMessage.SetToolCalls(normalizedToolCalls)
		}
		service.SetResponsesBridgeResult(c, upstreamResponse, assistantMessage)
	}
	return usage, nil
}
