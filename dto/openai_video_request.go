package dto

import (
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// OpenAIVideoRequest is the minimal request used by the OpenAI-compatible
// asynchronous video endpoint and channel tests. Provider-specific adapters
// may normalize aliases such as size/seconds before forwarding upstream.
type OpenAIVideoRequest struct {
	BaseRequest
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	Duration    int    `json:"duration,omitempty"`
	Seconds     string `json:"seconds,omitempty"`
	Size        string `json:"size,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Image       string `json:"image,omitempty"`
	InputRef    string `json:"input_reference,omitempty"`
}

func (r *OpenAIVideoRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{CombineText: r.Prompt}
}

func (r *OpenAIVideoRequest) IsStream(_ *gin.Context) bool { return false }

func (r *OpenAIVideoRequest) SetModelName(modelName string) { r.Model = modelName }
