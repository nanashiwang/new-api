package common

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
)

var (
	// OpenAIResponseOnlyModels is a list of models that are only available for OpenAI responses.
	OpenAIResponseOnlyModels = []string{
		"o3-pro",
		"o3-deep-research",
		"o4-mini-deep-research",
	}
	ImageGenerationModels = []string{
		"dall-e-3",
		"dall-e-2",
		"gpt-image-1",
		"gpt-image-2",
		"grok-imagine-image",
		"prefix:imagen-",
		"flux-",
		"flux.1-",
	}
	VideoGenerationModels = []string{
		"grok-imagine-video",
		"sora-2",
	}
	OpenAITextModels = []string{
		"gpt-",
		"o1",
		"o3",
		"o4",
		"chatgpt",
	}
)

func IsOpenAIResponseOnlyModel(modelName string) bool {
	for _, m := range OpenAIResponseOnlyModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

func IsImageGenerationModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range ImageGenerationModels {
		if strings.Contains(modelName, m) {
			return true
		}
		if strings.HasPrefix(m, "prefix:") && strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
			return true
		}
	}
	return false
}

func IsOpenAITextModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range OpenAITextModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

// IsVideoGenerationModel reports whether a model uses the asynchronous video
// generation API rather than a text or image endpoint.
func IsVideoGenerationModel(modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	for _, model := range VideoGenerationModels {
		if modelName == model || strings.HasPrefix(modelName, model+"-") {
			return true
		}
	}
	return false
}

// IsVideoGenerationModelForChannel narrows automatic video endpoint
// selection to models that are actually supported by the channel. This keeps
// a model name such as sora-2 on a normal OpenAI-compatible channel from
// unexpectedly changing that channel's existing text path.
func IsVideoGenerationModelForChannel(channelType int, modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	switch channelType {
	case constant.ChannelTypeXai:
		return modelName == "grok-imagine-video" || strings.HasPrefix(modelName, "grok-imagine-video-")
	case constant.ChannelTypeSora, constant.ChannelTypeOpenAI:
		return modelName == "sora-2" || strings.HasPrefix(modelName, "sora-2-")
	default:
		// A nil/unknown channel is used while building the token-model
		// capability map. Preserve the model-only fallback there; concrete
		// channel routing remains narrowed by the cases above.
		return IsVideoGenerationModel(modelName)
	}
}
