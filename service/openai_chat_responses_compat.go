package service

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service/openaicompat"
)

func ChatCompletionsRequestToResponsesRequest(req *dto.GeneralOpenAIRequest) (*dto.OpenAIResponsesRequest, error) {
	return openaicompat.ChatCompletionsRequestToResponsesRequest(req)
}

func NormalizeChatToolProtocol(req *dto.GeneralOpenAIRequest) (dto.ChatToolProtocol, error) {
	return openaicompat.NormalizeChatToolProtocol(req)
}

func ResponsesResponseToChatCompletionsResponse(resp *dto.OpenAIResponsesResponse, id string) (*dto.OpenAITextResponse, *dto.Usage, error) {
	return openaicompat.ResponsesResponseToChatCompletionsResponse(resp, id)
}

func ResponsesResponseToChatCompletionsResponseWithToolProtocol(resp *dto.OpenAIResponsesResponse, id string, protocol dto.ChatToolProtocol) (*dto.OpenAITextResponse, *dto.Usage, error) {
	return openaicompat.ResponsesResponseToChatCompletionsResponseWithToolProtocol(resp, id, protocol)
}

func ExtractOutputTextFromResponses(resp *dto.OpenAIResponsesResponse) string {
	return openaicompat.ExtractOutputTextFromResponses(resp)
}

func ExtractReasoningSummaryTextFromResponses(resp *dto.OpenAIResponsesResponse) string {
	return openaicompat.ExtractReasoningSummaryTextFromResponses(resp)
}
