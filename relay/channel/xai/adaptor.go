package xai

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	plan, err := ensureXAITextProtocolPlan(info)
	if err != nil {
		return nil, err
	}
	if plan.Converter != relaycommon.TextProtocolConverterClaudeToOpenAIChat {
		return nil, fmt.Errorf("unsupported xAI text protocol converter: %s", plan.Converter)
	}

	openAIRequest, err := service.ClaudeToOpenAIRequest(c, *request, info)
	if err != nil {
		return nil, err
	}
	if info.SupportsChatStreamOptions && info.IsStream {
		openAIRequest.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
	}
	convertedRequest, err := a.ConvertOpenAIRequest(c, info, openAIRequest)
	if err != nil {
		return nil, err
	}
	info.CommitTextProtocolPlan()
	return convertedRequest, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//not available
	return nil, errors.New("not available")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	imageN := uint(1)
	if request.N != nil {
		imageN = *request.N
	}
	xaiRequest := ImageRequest{
		Model:          request.Model,
		Prompt:         request.Prompt,
		N:              int(imageN),
		ResponseFormat: request.ResponseFormat,
	}
	return xaiRequest, nil
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info != nil && info.RelayFormat == types.RelayFormatClaude &&
		info.RelayMode != constant.RelayModeResponses &&
		info.RelayMode != constant.RelayModeResponsesCompact {
		plan, err := ensureXAITextProtocolPlan(info)
		if err != nil {
			return "", err
		}
		if plan.UpstreamFormat != types.RelayFormatOpenAI {
			return "", fmt.Errorf("unsupported xAI upstream text protocol: %s", plan.UpstreamFormat)
		}
		return xAIChatCompletionsURL(info.ChannelBaseUrl)
	}
	return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, info.RequestURLPath, info.ChannelType), nil
}

func ensureXAITextProtocolPlan(info *relaycommon.RelayInfo) (*relaycommon.TextProtocolPlan, error) {
	if info == nil || info.ChannelMeta == nil {
		return nil, errors.New("xAI channel metadata is required")
	}
	if info.NativeTextFormats == 0 {
		capabilities := relaycommon.ResolveChannelCapabilities(info.ChannelType, info.ChannelBaseUrl, info.ChannelOtherSettings)
		info.NativeTextFormats = capabilities.NativeTextFormats
	}
	if info.TextProtocolPlan == nil {
		if err := info.PrepareTextProtocolPlan(); err != nil {
			return nil, err
		}
	}
	if info.TextProtocolPlan == nil {
		return nil, errors.New("xAI text protocol plan is unavailable")
	}
	return info.TextProtocolPlan, nil
}

func xAIChatCompletionsURL(baseURL string) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", fmt.Errorf("invalid xAI channel base URL: %s", baseURL)
	}
	path := strings.TrimRight(parsedURL.Path, "/")
	switch {
	case strings.HasSuffix(path, "/v1/chat/completions"):
	case strings.HasSuffix(path, "/v1"):
		path += "/chat/completions"
	default:
		path += "/v1/chat/completions"
	}
	parsedURL.Path = path
	parsedURL.RawPath = ""
	return parsedURL.String(), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	if strings.HasSuffix(info.UpstreamModelName, "-search") {
		info.UpstreamModelName = strings.TrimSuffix(info.UpstreamModelName, "-search")
		request.Model = info.UpstreamModelName
		toMap := request.ToMap()
		toMap["search_parameters"] = map[string]any{
			"mode": "on",
		}
		return toMap, nil
	}
	if strings.HasPrefix(request.Model, "grok-3-mini") {
		if request.MaxCompletionTokens == 0 && request.MaxTokens != 0 {
			request.MaxCompletionTokens = request.MaxTokens
			request.MaxTokens = 0
		}
		if strings.HasSuffix(request.Model, "-high") {
			request.ReasoningEffort = "high"
			request.Model = strings.TrimSuffix(request.Model, "-high")
		} else if strings.HasSuffix(request.Model, "-low") {
			request.ReasoningEffort = "low"
			request.Model = strings.TrimSuffix(request.Model, "-low")
		}
		info.ReasoningEffort = request.ReasoningEffort
		info.UpstreamModelName = request.Model
	}
	return request, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	//not available
	return nil, errors.New("not available")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	if request.Model == "" && info != nil {
		request.Model = info.UpstreamModelName
	}
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	switch info.RelayMode {
	case constant.RelayModeImagesGenerations, constant.RelayModeImagesEdits:
		usage, err = openai.OpenaiHandlerWithUsage(c, info, resp)
	case constant.RelayModeResponses:
		if info.IsStream {
			usage, err = openai.OaiResponsesStreamHandler(c, info, resp)
		} else {
			usage, err = openai.OaiResponsesHandler(c, info, resp)
		}
	default:
		if info.IsStream {
			usage, err = xAIStreamHandler(c, info, resp)
		} else {
			usage, err = xAIHandler(c, info, resp)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
