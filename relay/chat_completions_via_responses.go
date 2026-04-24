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

func chatCompletionsViaResponses(c *gin.Context, info *relaycommon.RelayInfo, adaptor channel.Adaptor, request *dto.GeneralOpenAIRequest) (*dto.Usage, *types.NewAPIError) {
	chatJSON, err := common.Marshal(request)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	chatJSON, err = relaycommon.RemoveDisabledFields(chatJSON, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	if len(info.ParamOverride) > 0 {
		chatJSON, err = relaycommon.ApplyParamOverrideWithRelayInfo(chatJSON, info)
		if err != nil {
			return nil, newAPIErrorFromParamOverride(err)
		}
	}

	var overriddenChatReq dto.GeneralOpenAIRequest
	if err := common.Unmarshal(chatJSON, &overriddenChatReq); err != nil {
		return nil, types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
	}

	requestForResponses := &overriddenChatReq
	var fallbackChatReq *dto.GeneralOpenAIRequest
	var previousResponseID string
	if sessionMatch, err := service.ApplyResponsesSessionBridge(info, &overriddenChatReq); err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	} else if sessionMatch != nil {
		requestForResponses = sessionMatch.Trimmed
		fallbackChatReq = &overriddenChatReq
		previousResponseID = sessionMatch.ResponseID
	}

	responsesReq, err := service.ChatCompletionsRequestToResponsesRequest(requestForResponses)
	if err != nil {
		return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if previousResponseID != "" {
		responsesReq.PreviousResponseID = previousResponseID
	}
	if fallbackChatReq != nil {
		if err := restoreResponsesInstructionsFromOriginalChat(responsesReq, fallbackChatReq); err != nil {
			return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
	}
	var fallbackResponsesReq *dto.OpenAIResponsesRequest
	if fallbackChatReq != nil {
		fallbackResponsesReq, err = service.ChatCompletionsRequestToResponsesRequest(fallbackChatReq)
		if err != nil {
			return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
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

	convertedRequest, err := adaptor.ConvertOpenAIResponsesRequest(c, info, *responsesReq)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

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
		if retriedWithoutPreviousResponseID {
			responsesReqForUpstream.PreviousResponseID = ""
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

		jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

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
		if issue == upstreamCompatIssueStreamOptions && !strippedStreamOptions {
			strippedStreamOptions = true
			logCompatFallback(c, info, string(issue))
			_ = httpResp.Body.Close()
			continue
		}
		if issue == upstreamCompatIssuePreviousResponseID && !retriedWithoutPreviousResponseID {
			retriedWithoutPreviousResponseID = true
			service.MarkResponsesPreviousResponseIDUnsupported(info)
			logCompatFallback(c, info, string(issue))
			_ = httpResp.Body.Close()
			continue
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
		return usage, nil
	}

	usage, newApiErr := openaichannel.OaiResponsesToChatHandler(c, info, httpResp)
	if newApiErr != nil {
		service.ResetStatusCode(newApiErr, statusCodeMappingStr)
		return nil, newApiErr
	}
	return usage, nil
}

func restoreResponsesInstructionsFromOriginalChat(responsesReq *dto.OpenAIResponsesRequest, originalChatReq *dto.GeneralOpenAIRequest) error {
	if responsesReq == nil || originalChatReq == nil || len(responsesReq.Instructions) > 0 {
		return nil
	}
	var parts []string
	for _, msg := range originalChatReq.Messages {
		if msg.Role != "system" && msg.Role != "developer" {
			continue
		}
		if msg.IsStringContent() {
			if text := strings.TrimSpace(msg.StringContent()); text != "" {
				parts = append(parts, text)
			}
			continue
		}
		for _, item := range msg.ParseContent() {
			if item.Type == dto.ContentTypeText && strings.TrimSpace(item.Text) != "" {
				parts = append(parts, strings.TrimSpace(item.Text))
			}
		}
	}
	if len(parts) == 0 {
		return nil
	}
	instructions, err := common.Marshal(strings.Join(parts, "\n\n"))
	if err != nil {
		return err
	}
	responsesReq.Instructions = instructions
	return nil
}
