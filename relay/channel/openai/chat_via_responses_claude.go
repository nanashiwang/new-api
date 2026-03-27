package openai

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func OaiResponsesToClaudeStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	var (
		completedResp  *dto.OpenAIResponsesResponse
		streamErr      *types.NewAPIError
		upstreamRespID string
		responseID     = helper.GetResponseID(c)
		model          = info.UpstreamModelName
	)

	helper.StreamScannerHandler(c, resp, info, func(data string) bool {
		if streamErr != nil {
			return false
		}

		var streamResp dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResp); err != nil {
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
		case "response.completed":
			completedResp = streamResp.Response
			if completedResp != nil {
				if strings.EqualFold(strings.TrimSpace(completedResp.Status), "incomplete") {
					streamErr = newResponsesIncompleteError(completedResp)
					return false
				}
				if completedResp.ID != "" {
					upstreamRespID = completedResp.ID
				}
				if completedResp.Model != "" {
					model = completedResp.Model
				}
			}
		case "error", "response.error", "response.failed", "response.incomplete":
			streamErr = newResponsesStreamEventError(streamResp)
			return false
		}
		return true
	})

	if streamErr != nil {
		return nil, streamErr
	}
	if completedResp == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("missing response.completed payload"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	chatResp, usage, err := service.ResponsesResponseToChatCompletionsResponse(completedResp, responseID)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if usage == nil || usage.TotalTokens == 0 {
		text := service.ExtractOutputTextFromResponses(completedResp)
		usage = service.ResponseText2Usage(c, text, model, info.GetEstimatePromptTokens())
		if chatResp != nil {
			chatResp.Usage = *usage
		}
	}
	if chatResp != nil && len(chatResp.Choices) > 0 {
		service.SetResponsesBridgeResult(c, completedResp.ID, chatResp.Choices[0].Message)
	}

	claudeResp := service.ResponseOpenAIResponses2Claude(completedResp, responseID)
	if claudeResp == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("failed to build claude response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	if claudeResp.Model == "" {
		claudeResp.Model = model
	}
	if claudeResp.Id == "" {
		if upstreamRespID != "" {
			claudeResp.Id = upstreamRespID
		} else {
			claudeResp.Id = responseID
		}
	}

	for _, event := range service.StreamClaudeResponse(claudeResp) {
		if err := helper.ClaudeData(c, *event); err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
		}
	}
	return usage, nil
}
