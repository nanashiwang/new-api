package relay

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	openaichannel "github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func applySystemPromptIfNeeded(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) {
	if info == nil || request == nil {
		return
	}
	if info.ChannelSetting.SystemPrompt == "" {
		return
	}

	systemRole := request.GetSystemRoleName()

	containSystemPrompt := false
	for _, message := range request.Messages {
		if message.Role == systemRole {
			containSystemPrompt = true
			break
		}
	}
	if !containSystemPrompt {
		systemMessage := dto.Message{
			Role:    systemRole,
			Content: info.ChannelSetting.SystemPrompt,
		}
		request.Messages = append([]dto.Message{systemMessage}, request.Messages...)
		return
	}

	if !info.ChannelSetting.SystemPromptOverride {
		return
	}

	common.SetContextKey(c, constant.ContextKeySystemPromptOverride, true)
	for i, message := range request.Messages {
		if message.Role != systemRole {
			continue
		}
		if message.IsStringContent() {
			request.Messages[i].SetStringContent(info.ChannelSetting.SystemPrompt + "\n" + message.StringContent())
			return
		}
		contents := message.ParseContent()
		contents = append([]dto.MediaContent{
			{
				Type: dto.ContentTypeText,
				Text: info.ChannelSetting.SystemPrompt,
			},
		}, contents...)
		request.Messages[i].Content = contents
		return
	}
}

func shouldRemoveDisabledFieldsInResponsesBridge(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		return false
	}
	settings := info.ChannelOtherSettings
	return !settings.AllowServiceTier || !settings.AllowInferenceGeo || settings.DisableStore || !settings.AllowSafetyIdentifier || !settings.AllowIncludeObfuscation
}

func restoreResponsesInstructionsFromOriginalChat(responsesReq *dto.OpenAIResponsesRequest, originalChatReq *dto.GeneralOpenAIRequest) error {
	if responsesReq == nil || originalChatReq == nil || len(responsesReq.Instructions) > 0 {
		return nil
	}

	originalResponsesReq, err := service.ChatCompletionsRequestToResponsesRequest(originalChatReq)
	if err != nil {
		return err
	}
	if len(originalResponsesReq.Instructions) > 0 {
		responsesReq.Instructions = originalResponsesReq.Instructions
	}
	return nil
}

func chatCompletionsViaResponses(c *gin.Context, info *relaycommon.RelayInfo, adaptor channel.Adaptor, request *dto.GeneralOpenAIRequest) (*dto.Usage, *types.NewAPIError) {
	shouldRemoveDisabledFields := shouldRemoveDisabledFieldsInResponsesBridge(info)

	overriddenChatReq, err := common.DeepCopy(request)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	if shouldRemoveDisabledFields || len(info.ParamOverride) > 0 {
		chatJSON, err := common.Marshal(request)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		if shouldRemoveDisabledFields {
			chatJSON, err = relaycommon.RemoveDisabledFields(chatJSON, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
			if err != nil {
				return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
		}

		if len(info.ParamOverride) > 0 {
			chatJSON, err = relaycommon.ApplyParamOverrideWithRelayInfo(chatJSON, info)
			if err != nil {
				return nil, types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
			}
		}

		if err := common.Unmarshal(chatJSON, overriddenChatReq); err != nil {
			return nil, types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
		}
	}

	requestForResponses := overriddenChatReq
	sessionMatch, err := service.ApplyResponsesSessionBridge(info, overriddenChatReq)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	if sessionMatch != nil && sessionMatch.Trimmed != nil {
		requestForResponses = sessionMatch.Trimmed
	}

	responsesReq, err := service.ChatCompletionsRequestToResponsesRequest(requestForResponses)
	if err != nil {
		return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if sessionMatch != nil {
		if err := restoreResponsesInstructionsFromOriginalChat(responsesReq, overriddenChatReq); err != nil {
			return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
	}
	relaycommon.NormalizeResponsesStreamOptions(responsesReq, info.SupportsResponsesStreamOptions)
	if sessionMatch != nil {
		responsesReq.PreviousResponseID = sessionMatch.ResponseID
	}
	var fallbackResponsesReq *dto.OpenAIResponsesRequest
	if sessionMatch != nil {
		fallbackResponsesReq, err = service.ChatCompletionsRequestToResponsesRequest(overriddenChatReq)
		if err != nil {
			return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		relaycommon.NormalizeResponsesStreamOptions(fallbackResponsesReq, info.SupportsResponsesStreamOptions)
	}
	info.AppendRequestConversion(types.RelayFormatOpenAIResponses)

	savedRelayMode := info.RelayMode
	savedRequestURLPath := info.RequestURLPath
	defer func() {
		info.RelayMode = savedRelayMode
		info.RequestURLPath = savedRequestURLPath
	}()

	info.RelayMode = relayconstant.RelayModeResponses
	info.RequestURLPath = "/v1/responses"

	statusCodeMappingStr := c.GetString("status_code_mapping")
	strippedStreamOptions := false
	retriedWithoutPreviousResponseID := false
	var httpResp *http.Response
	for {
		requestTemplate := responsesReq
		if retriedWithoutPreviousResponseID && fallbackResponsesReq != nil {
			requestTemplate = fallbackResponsesReq
		}

		responsesReqForUpstream, err := common.DeepCopy(requestTemplate)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		if strippedStreamOptions {
			responsesReqForUpstream.StreamOptions = nil
		}

		convertedRequest, err := adaptor.ConvertOpenAIResponsesRequest(c, info, *responsesReqForUpstream)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		if shouldRemoveDisabledFields {
			jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
			if err != nil {
				return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
		}
		jsonData, err = relaycommon.NormalizeJSONStreamOptions(jsonData)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		if strippedStreamOptions {
			jsonData, err = relaycommon.RemoveJSONStreamOptions(jsonData)
			if err != nil {
				return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
		}
		service.SyncRelayReasoningEffortFromResponsesPayload(info, jsonData)

		resp, err := adaptor.DoRequest(c, info, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
		}
		if resp == nil {
			return nil, types.NewOpenAIError(nil, types.ErrorCodeBadResponse, http.StatusInternalServerError)
		}

		httpResp = resp.(*http.Response)
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
		if httpResp.StatusCode == http.StatusOK {
			break
		}

		issue := classifyUpstreamCompatibilityIssue(httpResp, info.RelayMode)
		switch {
		case issue == upstreamCompatIssueStreamOptions && !strippedStreamOptions:
			strippedStreamOptions = true
			logCompatFallback(c, info, string(issue))
			_ = httpResp.Body.Close()
			continue
		case issue == upstreamCompatIssuePreviousResponseID && !retriedWithoutPreviousResponseID && fallbackResponsesReq != nil:
			retriedWithoutPreviousResponseID = true
			service.MarkResponsesPreviousResponseIDUnsupported(info)
			logCompatFallback(c, info, string(issue))
			_ = httpResp.Body.Close()
			continue
		case issue == upstreamCompatIssueResponsesAPI:
			logCompatFallback(c, info, string(issue))
			markFallbackToNativeChat(c)
		}

		newApiErr := service.RelayErrorHandler(c.Request.Context(), httpResp, false)
		service.ResetStatusCode(newApiErr, statusCodeMappingStr)
		return nil, newApiErr
	}

	if info.IsStream {
		usage, newApiErr := openaichannel.OaiResponsesToChatStreamHandler(c, info, httpResp)
		if newApiErr != nil {
			service.ResetStatusCode(newApiErr, statusCodeMappingStr)
			return nil, newApiErr
		}
		if bridgeResult, ok := service.GetResponsesBridgeResult(c); ok {
			_ = service.StoreResponsesSessionBridge(info, overriddenChatReq, bridgeResult.AssistantMessage, bridgeResult.ResponseID)
		}
		return usage, nil
	}

	usage, newApiErr := openaichannel.OaiResponsesToChatHandler(c, info, httpResp)
	if newApiErr != nil {
		service.ResetStatusCode(newApiErr, statusCodeMappingStr)
		return nil, newApiErr
	}
	if bridgeResult, ok := service.GetResponsesBridgeResult(c); ok {
		_ = service.StoreResponsesSessionBridge(info, overriddenChatReq, bridgeResult.AssistantMessage, bridgeResult.ResponseID)
	}
	return usage, nil
}
