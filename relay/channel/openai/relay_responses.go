package openai

import (
	"errors"
	"fmt"
	"io"
	"net/http"
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

func OaiResponsesHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	// read response body
	var responsesResponse dto.OpenAIResponsesResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	err = common.Unmarshal(responseBody, &responsesResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := responsesResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	if responsesResponse.HasImageGenerationCall() {
		c.Set("image_generation_call", true)
		c.Set("image_generation_call_count", responsesResponse.CountImageGenerationCalls())
		c.Set("image_generation_call_quality", responsesResponse.GetQuality())
		c.Set("image_generation_call_size", responsesResponse.GetSize())
	}

	// 写入新的 response body
	service.IOCopyBytesGracefully(c, resp, responseBody)

	// compute usage
	usage := dto.Usage{}
	if responsesResponse.Usage != nil {
		usage.PromptTokens = responsesResponse.Usage.InputTokens
		usage.CompletionTokens = responsesResponse.Usage.OutputTokens
		usage.TotalTokens = responsesResponse.Usage.TotalTokens
		if responsesResponse.Usage.InputTokensDetails != nil {
			usage.PromptTokensDetails.CachedTokens = responsesResponse.Usage.InputTokensDetails.CachedTokens
		}
	}
	for _, output := range responsesResponse.Output {
		if output.Type == dto.BuildInCallWebSearchCall {
			usage.WebSearchRequests++
		}
	}
	if info == nil || info.ResponsesUsageInfo == nil || info.ResponsesUsageInfo.BuiltInTools == nil {
		return &usage, nil
	}
	if usage.WebSearchRequests > 0 {
		if webSearchTool, exists := info.ResponsesUsageInfo.BuiltInTools[dto.NormalizeBuiltInToolType(dto.BuildInToolWebSearch)]; exists && webSearchTool != nil {
			webSearchTool.CallCount = usage.WebSearchRequests
		}
	}
	// 解析 Tools 用量
	for _, tool := range responsesResponse.Tools {
		toolType := dto.NormalizeBuiltInToolType(common.Interface2String(tool["type"]))
		buildToolinfo, ok := info.ResponsesUsageInfo.BuiltInTools[toolType]
		if !ok || buildToolinfo == nil {
			logger.LogError(c, fmt.Sprintf("BuiltInTools not found for tool type: %v", tool["type"]))
			continue
		}
		if dto.IsBuiltInWebSearchToolType(toolType) {
			buildToolinfo.CallCount = usage.WebSearchRequests
			continue
		}
		buildToolinfo.CallCount++
	}
	return &usage, nil
}

func OaiResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		logger.LogError(c, "invalid response or response body")
		return nil, types.NewError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse)
	}

	defer service.CloseResponseBodyGracefully(resp)

	var usage = &dto.Usage{}
	var responseTextBuilder strings.Builder
	completed := false
	terminalWithoutCompleted := false
	hasEffectiveOutput := false
	responseID := ""
	responseModel := ""
	responseCreatedAt := 0

	helper.StreamScannerHandler(c, resp, info, func(data string) bool {

		// 检查当前数据是否包含 completed 状态和 usage 信息
		var streamResponse dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResponse); err == nil {
			sendResponsesStreamData(c, streamResponse, data)
			if isEffectiveResponsesStreamOutput(streamResponse) {
				hasEffectiveOutput = true
			}
			if streamResponse.Response != nil {
				if streamResponse.Response.ID != "" {
					responseID = streamResponse.Response.ID
				}
				if streamResponse.Response.Model != "" {
					responseModel = streamResponse.Response.Model
				}
				if streamResponse.Response.CreatedAt != 0 {
					responseCreatedAt = streamResponse.Response.CreatedAt
				}
			}
			if streamResponse.Error != nil {
				terminalWithoutCompleted = true
			}
			switch streamResponse.Type {
			case "response.completed":
				completed = true
				if streamResponse.Response != nil {
					if streamResponse.Response.Usage != nil {
						if streamResponse.Response.Usage.InputTokens != 0 {
							usage.PromptTokens = streamResponse.Response.Usage.InputTokens
						}
						if streamResponse.Response.Usage.OutputTokens != 0 {
							usage.CompletionTokens = streamResponse.Response.Usage.OutputTokens
						}
						if streamResponse.Response.Usage.TotalTokens != 0 {
							usage.TotalTokens = streamResponse.Response.Usage.TotalTokens
						}
						if streamResponse.Response.Usage.InputTokensDetails != nil {
							usage.PromptTokensDetails.CachedTokens = streamResponse.Response.Usage.InputTokensDetails.CachedTokens
						}
					}
					if streamResponse.Response.HasImageGenerationCall() {
						c.Set("image_generation_call", true)
						c.Set("image_generation_call_count", streamResponse.Response.CountImageGenerationCalls())
						c.Set("image_generation_call_quality", streamResponse.Response.GetQuality())
						c.Set("image_generation_call_size", streamResponse.Response.GetSize())
					}
				}
			case "response.output_text.delta":
				// 处理输出文本
				responseTextBuilder.WriteString(streamResponse.Delta)
			case "response.failed", "response.incomplete", "error":
				terminalWithoutCompleted = true
			case dto.ResponsesOutputTypeItemDone:
				// 函数调用处理
				if streamResponse.Item != nil {
					switch streamResponse.Item.Type {
					case dto.BuildInCallWebSearchCall:
						usage.WebSearchRequests++
						if info != nil && info.ResponsesUsageInfo != nil && info.ResponsesUsageInfo.BuiltInTools != nil {
							if webSearchTool, exists := info.ResponsesUsageInfo.BuiltInTools[dto.NormalizeBuiltInToolType(dto.BuildInToolWebSearch)]; exists && webSearchTool != nil {
								webSearchTool.CallCount++
							}
						}
					}
				}
			}
		} else {
			logger.LogError(c, "failed to unmarshal stream response: "+err.Error())
		}
		return true
	})

	if usage.CompletionTokens == 0 {
		// 计算输出文本的 token 数量
		tempStr := responseTextBuilder.String()
		if len(tempStr) > 0 {
			// 非正常结束，使用输出文本的 token 数量
			completionTokens := service.CountTextToken(tempStr, info.UpstreamModelName)
			usage.CompletionTokens = completionTokens
		}
	}

	if usage.PromptTokens == 0 && usage.CompletionTokens != 0 {
		usage.PromptTokens = info.GetEstimatePromptTokens()
	}

	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

	if !completed {
		reason := service.ResponsesStreamMissingCompletedReason
		if info != nil && info.StreamStatus != nil {
			info.StreamStatus.RecordError(reason)
		}
		scheduleResponsesStreamCooldown(c, reason)
		if info != nil && info.IsChannelTest {
			return usage, types.NewOpenAIError(errors.New(reason), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
		if !terminalWithoutCompleted {
			if shouldFailMissingResponsesCompleted(info, hasEffectiveOutput) {
				sendSyntheticResponsesFailed(c, info, usage, reason, responseID, responseModel, responseCreatedAt)
			} else {
				sendSyntheticResponsesCompleted(c, info, usage, responseID, responseModel, responseCreatedAt)
			}
		}
	}

	return usage, nil
}

func isEffectiveResponsesStreamOutput(streamResponse dto.ResponsesStreamResponse) bool {
	if streamResponse.Delta != "" {
		return true
	}
	if streamResponse.Item != nil && streamResponse.Item.Type != "" {
		return true
	}
	return streamResponse.Response != nil && len(streamResponse.Response.Output) > 0
}

func shouldFailMissingResponsesCompleted(info *relaycommon.RelayInfo, hasEffectiveOutput bool) bool {
	if hasEffectiveOutput || info == nil || info.StreamStatus == nil {
		return false
	}
	switch info.StreamStatus.EndReason {
	case relaycommon.StreamEndReasonTimeout,
		relaycommon.StreamEndReasonClientGone,
		relaycommon.StreamEndReasonScannerErr,
		relaycommon.StreamEndReasonPanic,
		relaycommon.StreamEndReasonPingFail:
		return true
	default:
		return false
	}
}

func scheduleResponsesStreamCooldown(c *gin.Context, reason string) {
	scheduled, err := service.ScheduleCurrentChannelPreDisableWait(c, reason)
	if err != nil {
		logger.LogError(c, fmt.Sprintf("schedule responses stream cooldown failed: %v", err))
		return
	}
	if scheduled {
		logger.LogInfo(c, fmt.Sprintf("responses stream cooldown scheduled: %s", reason))
	}
}

func sendSyntheticResponsesCompleted(c *gin.Context, info *relaycommon.RelayInfo, usage *dto.Usage, responseID, model string, createdAt int) {
	if c == nil {
		return
	}
	if responseID == "" {
		responseID = "resp_" + c.GetString(common.RequestIdKey)
	}
	if model == "" && info != nil {
		model = info.UpstreamModelName
		if model == "" {
			model = info.OriginModelName
		}
	}
	if createdAt == 0 {
		createdAt = int(time.Now().Unix())
	}
	responseUsage := normalizeResponsesUsage(usage)
	streamResponse := dto.ResponsesStreamResponse{
		Type: "response.completed",
		Response: &dto.OpenAIResponsesResponse{
			ID:        responseID,
			Object:    "response",
			CreatedAt: createdAt,
			Status:    "completed",
			Model:     model,
			Output:    []dto.ResponsesOutput{},
			Usage:     &responseUsage,
		},
	}
	jsonData, err := common.Marshal(streamResponse)
	if err != nil {
		logger.LogError(c, "failed to marshal synthetic responses completed event: "+err.Error())
		return
	}
	sendResponsesStreamData(c, streamResponse, string(jsonData))
}

func sendSyntheticResponsesFailed(c *gin.Context, info *relaycommon.RelayInfo, usage *dto.Usage, reason, responseID, model string, createdAt int) {
	if c == nil {
		return
	}
	if responseID == "" {
		responseID = "resp_" + c.GetString(common.RequestIdKey)
	}
	if model == "" && info != nil {
		model = info.UpstreamModelName
		if model == "" {
			model = info.OriginModelName
		}
	}
	if createdAt == 0 {
		createdAt = int(time.Now().Unix())
	}
	message := reason
	if info != nil && info.StreamStatus != nil && info.StreamStatus.EndReason != "" {
		message = fmt.Sprintf("%s (stream end: %s)", reason, info.StreamStatus.EndReason)
	}
	logger.LogError(c, "responses stream failed without output: "+message)
	openaiError := types.OpenAIError{
		Message: message,
		Type:    string(types.ErrorCodeBadResponseBody),
		Code:    types.ErrorCodeBadResponseBody,
	}
	responseUsage := normalizeResponsesUsage(usage)
	streamResponse := dto.ResponsesStreamResponse{
		Type: "response.failed",
		Response: &dto.OpenAIResponsesResponse{
			ID:        responseID,
			Object:    "response",
			CreatedAt: createdAt,
			Status:    "failed",
			Model:     model,
			Error:     openaiError,
			Output:    []dto.ResponsesOutput{},
			Usage:     &responseUsage,
		},
	}
	jsonData, err := common.Marshal(streamResponse)
	if err != nil {
		logger.LogError(c, "failed to marshal synthetic responses failed event: "+err.Error())
		return
	}
	sendResponsesStreamData(c, streamResponse, string(jsonData))
}

func normalizeResponsesUsage(usage *dto.Usage) dto.Usage {
	if usage == nil {
		usage = &dto.Usage{}
	}
	responseUsage := *usage
	if responseUsage.InputTokens == 0 {
		responseUsage.InputTokens = responseUsage.PromptTokens
	}
	if responseUsage.OutputTokens == 0 {
		responseUsage.OutputTokens = responseUsage.CompletionTokens
	}
	if responseUsage.TotalTokens == 0 {
		responseUsage.TotalTokens = responseUsage.InputTokens + responseUsage.OutputTokens
	}
	if responseUsage.InputTokensDetails == nil && usage.PromptTokensDetails.CachedTokens != 0 {
		responseUsage.InputTokensDetails = &dto.InputTokenDetails{
			CachedTokens: usage.PromptTokensDetails.CachedTokens,
		}
	}
	return responseUsage
}
