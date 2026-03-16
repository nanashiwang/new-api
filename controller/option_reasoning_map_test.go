package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateClaudeToOpenAIReasoningMap_RejectsMissingLevel(t *testing.T) {
	err := validateClaudeToOpenAIReasoningMap(`{"low":"minimal"}`)

	require.Error(t, err)
	require.Contains(t, err.Error(), "缺少 medium 档位")
}

func TestValidateClaudeToOpenAIReasoningMap_AcceptsValidMapping(t *testing.T) {
	err := validateClaudeToOpenAIReasoningMap(`{"low":"minimal","medium":"low","high":"high","max":"xhigh"}`)

	require.NoError(t, err)
}
