package constant

import "strings"

const (
	CodexAutoReviewModel        = "codex-auto-review"
	CodexAutoReviewRoutingModel = "gpt-5.3-codex"
)

func IsHiddenModel(modelName string) bool {
	return strings.TrimSpace(modelName) == CodexAutoReviewModel
}

func IsCodexAutoReviewCompatibleChannelType(channelType int) bool {
	return channelType == ChannelTypeCodex || channelType == ChannelTypeOpenAI
}
