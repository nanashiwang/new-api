package claude

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatClaudeResponseInfo_MessageStart(t *testing.T) {
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{},
	}
	claudeResponse := &dto.ClaudeResponse{
		Type: "message_start",
		Message: &dto.ClaudeMediaMessage{
			Id:    "msg_123",
			Model: "claude-3-5-sonnet",
			Usage: &dto.ClaudeUsage{
				InputTokens:              100,
				OutputTokens:             1,
				CacheCreationInputTokens: 50,
				CacheReadInputTokens:     30,
			},
		},
	}

	ok := FormatClaudeResponseInfo(claudeResponse, nil, claudeInfo)
	if !ok {
		t.Fatal("expected true")
	}
	if claudeInfo.Usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", claudeInfo.Usage.PromptTokens)
	}
	if claudeInfo.Usage.PromptTokensDetails.CachedTokens != 30 {
		t.Errorf("CachedTokens = %d, want 30", claudeInfo.Usage.PromptTokensDetails.CachedTokens)
	}
	if claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens != 50 {
		t.Errorf("CachedCreationTokens = %d, want 50", claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens)
	}
	if claudeInfo.ResponseId != "msg_123" {
		t.Errorf("ResponseId = %s, want msg_123", claudeInfo.ResponseId)
	}
	if claudeInfo.Model != "claude-3-5-sonnet" {
		t.Errorf("Model = %s, want claude-3-5-sonnet", claudeInfo.Model)
	}
}

func TestFormatClaudeResponseInfo_MessageDelta_FullUsage(t *testing.T) {
	// message_start 先积累 usage
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{
			PromptTokens: 100,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:         30,
				CachedCreationTokens: 50,
			},
			CompletionTokens: 1,
		},
	}

	// message_delta 带完整 usage（原生 Anthropic 场景）
	claudeResponse := &dto.ClaudeResponse{
		Type: "message_delta",
		Usage: &dto.ClaudeUsage{
			InputTokens:              100,
			OutputTokens:             200,
			CacheCreationInputTokens: 50,
			CacheReadInputTokens:     30,
		},
	}

	ok := FormatClaudeResponseInfo(claudeResponse, nil, claudeInfo)
	if !ok {
		t.Fatal("expected true")
	}
	if claudeInfo.Usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", claudeInfo.Usage.PromptTokens)
	}
	if claudeInfo.Usage.CompletionTokens != 200 {
		t.Errorf("CompletionTokens = %d, want 200", claudeInfo.Usage.CompletionTokens)
	}
	if claudeInfo.Usage.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", claudeInfo.Usage.TotalTokens)
	}
	if !claudeInfo.Done {
		t.Error("expected Done = true")
	}
}

func TestFormatClaudeResponseInfo_MessageDelta_OnlyOutputTokens(t *testing.T) {
	// 模拟 Bedrock: message_start 已积累 usage
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{
			PromptTokens: 100,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:         30,
				CachedCreationTokens: 50,
			},
			CompletionTokens:            1,
			ClaudeCacheCreation5mTokens: 10,
			ClaudeCacheCreation1hTokens: 20,
		},
	}

	// Bedrock 的 message_delta 只有 output_tokens，缺少 input_tokens 和 cache 字段
	claudeResponse := &dto.ClaudeResponse{
		Type: "message_delta",
		Usage: &dto.ClaudeUsage{
			OutputTokens: 200,
			// InputTokens, CacheCreationInputTokens, CacheReadInputTokens 都是 0
		},
	}

	ok := FormatClaudeResponseInfo(claudeResponse, nil, claudeInfo)
	if !ok {
		t.Fatal("expected true")
	}
	// PromptTokens 应保持 message_start 的值（因为 message_delta 的 InputTokens=0，不更新）
	if claudeInfo.Usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", claudeInfo.Usage.PromptTokens)
	}
	if claudeInfo.Usage.CompletionTokens != 200 {
		t.Errorf("CompletionTokens = %d, want 200", claudeInfo.Usage.CompletionTokens)
	}
	if claudeInfo.Usage.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", claudeInfo.Usage.TotalTokens)
	}
	// cache 字段应保持 message_start 的值
	if claudeInfo.Usage.PromptTokensDetails.CachedTokens != 30 {
		t.Errorf("CachedTokens = %d, want 30", claudeInfo.Usage.PromptTokensDetails.CachedTokens)
	}
	if claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens != 50 {
		t.Errorf("CachedCreationTokens = %d, want 50", claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens)
	}
	if claudeInfo.Usage.ClaudeCacheCreation5mTokens != 10 {
		t.Errorf("ClaudeCacheCreation5mTokens = %d, want 10", claudeInfo.Usage.ClaudeCacheCreation5mTokens)
	}
	if claudeInfo.Usage.ClaudeCacheCreation1hTokens != 20 {
		t.Errorf("ClaudeCacheCreation1hTokens = %d, want 20", claudeInfo.Usage.ClaudeCacheCreation1hTokens)
	}
	if !claudeInfo.Done {
		t.Error("expected Done = true")
	}
}

func TestFormatClaudeResponseInfo_NilClaudeInfo(t *testing.T) {
	claudeResponse := &dto.ClaudeResponse{Type: "message_start"}
	ok := FormatClaudeResponseInfo(claudeResponse, nil, nil)
	if ok {
		t.Error("expected false for nil claudeInfo")
	}
}

func TestFormatClaudeResponseInfo_ContentBlockDelta(t *testing.T) {
	text := "hello"
	claudeInfo := &ClaudeResponseInfo{
		Usage:        &dto.Usage{},
		ResponseText: strings.Builder{},
	}
	claudeResponse := &dto.ClaudeResponse{
		Type: "content_block_delta",
		Delta: &dto.ClaudeMediaMessage{
			Text: &text,
		},
	}

	ok := FormatClaudeResponseInfo(claudeResponse, nil, claudeInfo)
	if !ok {
		t.Fatal("expected true")
	}
	if claudeInfo.ResponseText.String() != "hello" {
		t.Errorf("ResponseText = %q, want %q", claudeInfo.ResponseText.String(), "hello")
	}
}

func TestBuildOpenAIStyleUsageFromClaudeUsage(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 20,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         30,
			CachedCreationTokens: 50,
		},
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 20,
	}

	openAIUsage := buildOpenAIStyleUsageFromClaudeUsage(usage)

	require.Equal(t, 180, openAIUsage.PromptTokens)
	require.Equal(t, 180, openAIUsage.InputTokens)
	require.Equal(t, 200, openAIUsage.TotalTokens)
	require.Equal(t, 30, openAIUsage.ClaudeCacheCreation5mTokens)
	require.Equal(t, 20, openAIUsage.ClaudeCacheCreation1hTokens)
}

func TestBuildOpenAIStyleUsageFromClaudeUsagePreservesCacheCreationRemainder(t *testing.T) {
	tests := []struct {
		name                    string
		cachedCreationTokens    int
		cacheCreationTokens5m   int
		cacheCreationTokens1h   int
		expectedTotalInputToken int
	}{
		{
			name:                    "prefers aggregate when it includes remainder",
			cachedCreationTokens:    50,
			cacheCreationTokens5m:   10,
			cacheCreationTokens1h:   20,
			expectedTotalInputToken: 180,
		},
		{
			name:                    "falls back to split tokens when aggregate missing",
			cachedCreationTokens:    0,
			cacheCreationTokens5m:   10,
			cacheCreationTokens1h:   20,
			expectedTotalInputToken: 160,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := &dto.Usage{
				PromptTokens:     100,
				CompletionTokens: 20,
				PromptTokensDetails: dto.InputTokenDetails{
					CachedTokens:         30,
					CachedCreationTokens: tt.cachedCreationTokens,
				},
				ClaudeCacheCreation5mTokens: tt.cacheCreationTokens5m,
				ClaudeCacheCreation1hTokens: tt.cacheCreationTokens1h,
			}

			openAIUsage := buildOpenAIStyleUsageFromClaudeUsage(usage)
			require.Equal(t, tt.expectedTotalInputToken, openAIUsage.PromptTokens)
			require.Equal(t, tt.expectedTotalInputToken, openAIUsage.InputTokens)
		})
	}
}

func TestBuildOpenAIStyleUsageFromClaudeUsageDefaultsAggregateCacheCreationTo5m(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 20,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         30,
			CachedCreationTokens: 50,
		},
	}

	openAIUsage := buildOpenAIStyleUsageFromClaudeUsage(usage)

	require.Equal(t, 50, openAIUsage.ClaudeCacheCreation5mTokens)
	require.Equal(t, 0, openAIUsage.ClaudeCacheCreation1hTokens)
}

func TestHandleStreamFinalResponsePreservesCacheFieldsOnIncompleteStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-3-5-sonnet",
		},
		RelayFormat: types.RelayFormatClaude,
	}
	info.SetEstimatePromptTokens(120)

	claudeInfo := &ClaudeResponseInfo{
		ResponseText: strings.Builder{},
		Usage: &dto.Usage{
			PromptTokens: 100,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:         30,
				CachedCreationTokens: 50,
			},
			ClaudeCacheCreation5mTokens: 10,
			ClaudeCacheCreation1hTokens: 20,
		},
		Done: false,
	}
	claudeInfo.ResponseText.WriteString("hello world from claude stream")

	HandleStreamFinalResponse(ctx, info, claudeInfo)

	require.Equal(t, 100, claudeInfo.Usage.PromptTokens)
	require.Equal(t, 30, claudeInfo.Usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 50, claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens)
	require.Equal(t, 10, claudeInfo.Usage.ClaudeCacheCreation5mTokens)
	require.Equal(t, 20, claudeInfo.Usage.ClaudeCacheCreation1hTokens)
	require.Greater(t, claudeInfo.Usage.CompletionTokens, 0)
	require.Equal(t, claudeInfo.Usage.PromptTokens+claudeInfo.Usage.CompletionTokens, claudeInfo.Usage.TotalTokens)
}

func TestRequestOpenAI2ClaudeMessage_AssistantToolCallWithMalformedArguments(t *testing.T) {
	assistantMessage := dto.Message{
		Role:    "assistant",
		Content: "calling tool",
	}
	assistantMessage.SetToolCalls([]dto.ToolCallRequest{
		{
			ID:   "call_1",
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      "search_notes",
				Arguments: `{"query":`,
			},
		},
	})

	claudeReq, err := RequestOpenAI2ClaudeMessage(nil, dto.GeneralOpenAIRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "find it",
			},
			assistantMessage,
		},
	})
	require.NoError(t, err)
	require.Len(t, claudeReq.Messages, 2)

	content, err := claudeReq.Messages[1].ParseContent()
	require.NoError(t, err)
	require.Len(t, content, 2)
	assert.Equal(t, "tool_use", content[1].Type)
	assert.Equal(t, "call_1", content[1].Id)
	assert.Equal(t, "search_notes", content[1].Name)

	inputObj, ok := content[1].Input.(map[string]any)
	require.True(t, ok)
	assert.Empty(t, inputObj)
}

func TestStreamResponseClaude2OpenAI_EmptyInputJSONDeltaIgnored(t *testing.T) {
	empty := ""
	resp := &dto.ClaudeResponse{
		Type:  "content_block_delta",
		Index: func() *int { v := 1; return &v }(),
		Delta: &dto.ClaudeMediaMessage{
			Type:        "input_json_delta",
			PartialJson: &empty,
		},
	}

	chunk := StreamResponseClaude2OpenAI(resp, &ClaudeResponseInfo{})
	require.Nil(t, chunk)
}

func TestStreamResponseClaude2OpenAI_NonEmptyInputJSONDeltaPreserved(t *testing.T) {
	partial := `{"query":"today"}`
	resp := &dto.ClaudeResponse{
		Type:  "content_block_delta",
		Index: func() *int { v := 1; return &v }(),
		Delta: &dto.ClaudeMediaMessage{
			Type:        "input_json_delta",
			PartialJson: &partial,
		},
	}

	chunk := StreamResponseClaude2OpenAI(resp, &ClaudeResponseInfo{})
	require.NotNil(t, chunk)
	require.Len(t, chunk.Choices, 1)
	require.Len(t, chunk.Choices[0].Delta.ToolCalls, 1)
	assert.Equal(t, partial, chunk.Choices[0].Delta.ToolCalls[0].Function.Arguments)
}

func TestStreamResponseClaude2OpenAI_NoArgToolEmitsObjectAtStop(t *testing.T) {
	claudeInfo := &ClaudeResponseInfo{}
	start := &dto.ClaudeResponse{
		Type:  "content_block_start",
		Index: func() *int { v := 1; return &v }(),
		ContentBlock: &dto.ClaudeMediaMessage{
			Type: "tool_use",
			Id:   "toolu_1",
			Name: "get_current_time",
		},
	}
	stop := &dto.ClaudeResponse{
		Type:  "content_block_stop",
		Index: func() *int { v := 1; return &v }(),
	}

	startChunk := StreamResponseClaude2OpenAI(start, claudeInfo)
	require.Nil(t, startChunk)

	stopChunk := StreamResponseClaude2OpenAI(stop, claudeInfo)
	require.NotNil(t, stopChunk)
	require.Len(t, stopChunk.Choices, 1)
	require.Len(t, stopChunk.Choices[0].Delta.ToolCalls, 1)
	assert.Equal(t, "toolu_1", stopChunk.Choices[0].Delta.ToolCalls[0].ID)
	assert.Equal(t, "get_current_time", stopChunk.Choices[0].Delta.ToolCalls[0].Function.Name)
	assert.Equal(t, "{}", stopChunk.Choices[0].Delta.ToolCalls[0].Function.Arguments)
}

func TestStreamResponseClaude2OpenAI_ArgToolKeepsIDNameOnDelta(t *testing.T) {
	claudeInfo := &ClaudeResponseInfo{}
	start := &dto.ClaudeResponse{
		Type:  "content_block_start",
		Index: func() *int { v := 1; return &v }(),
		ContentBlock: &dto.ClaudeMediaMessage{
			Type: "tool_use",
			Id:   "toolu_2",
			Name: "search_notes",
		},
	}
	partial := `{"query":"today"}`
	delta := &dto.ClaudeResponse{
		Type:  "content_block_delta",
		Index: func() *int { v := 1; return &v }(),
		Delta: &dto.ClaudeMediaMessage{
			Type:        "input_json_delta",
			PartialJson: &partial,
		},
	}

	startChunk := StreamResponseClaude2OpenAI(start, claudeInfo)
	require.Nil(t, startChunk)

	deltaChunk := StreamResponseClaude2OpenAI(delta, claudeInfo)
	require.NotNil(t, deltaChunk)
	require.Len(t, deltaChunk.Choices, 1)
	require.Len(t, deltaChunk.Choices[0].Delta.ToolCalls, 1)
	assert.Equal(t, "toolu_2", deltaChunk.Choices[0].Delta.ToolCalls[0].ID)
	assert.Equal(t, "search_notes", deltaChunk.Choices[0].Delta.ToolCalls[0].Function.Name)
	assert.Equal(t, partial, deltaChunk.Choices[0].Delta.ToolCalls[0].Function.Arguments)
}
