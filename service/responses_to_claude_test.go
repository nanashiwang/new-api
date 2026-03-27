package service

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestResponseOpenAIResponses2Claude_BuildsWebSearchBlocksAndCitations(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		ID:        "resp_1",
		Model:     "gpt-4.1",
		CreatedAt: 1700000000,
		Usage: &dto.Usage{
			InputTokens:  12,
			OutputTokens: 8,
			TotalTokens:  20,
		},
		Output: []dto.ResponsesOutput{
			{
				Type: dto.BuildInCallWebSearchCall,
				ID:   "ws_1",
				Action: json.RawMessage(`{
					"query":"latest OpenAI news",
					"sources":[{"url":"https://example.com/openai","title":"OpenAI source","snippet":"Alpha summary"}]
				}`),
			},
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{
						Type: "output_text",
						Text: "Alpha summary with context",
						Annotations: []dto.ResponsesOutputAnnotation{
							{
								Type:       "url_citation",
								URL:        "https://example.com/openai",
								Title:      "OpenAI source",
								StartIndex: 0,
								EndIndex:   5,
							},
						},
					},
				},
			},
		},
	}

	claudeResp := ResponseOpenAIResponses2Claude(resp, "msg_1")
	require.NotNil(t, claudeResp)
	require.Equal(t, "msg_1", claudeResp.Id)
	require.NotNil(t, claudeResp.Usage)
	require.NotNil(t, claudeResp.Usage.ServerToolUse)
	require.Equal(t, 1, claudeResp.Usage.ServerToolUse.WebSearchRequests)

	require.GreaterOrEqual(t, len(claudeResp.Content), 4)
	require.Equal(t, "server_tool_use", claudeResp.Content[0].Type)
	require.Equal(t, "web_search", claudeResp.Content[0].Name)

	searchResult, ok := claudeResp.Content[1].Content.([]dto.ClaudeWebSearchResult)
	require.True(t, ok)
	require.Len(t, searchResult, 1)
	require.Equal(t, "https://example.com/openai", searchResult[0].URL)

	var citedBlock *dto.ClaudeMediaMessage
	for i := range claudeResp.Content {
		if claudeResp.Content[i].Type == "text" && claudeResp.Content[i].Citations != nil {
			citedBlock = &claudeResp.Content[i]
			break
		}
	}
	require.NotNil(t, citedBlock)
	citations, ok := citedBlock.Citations.([]dto.ClaudeTextCitation)
	require.True(t, ok)
	require.Len(t, citations, 1)
	require.Equal(t, "web_search_result_location", citations[0].Type)
	require.Equal(t, "https://example.com/openai", citations[0].URL)
	require.Equal(t, "Alpha", citedBlock.GetText())
}

func TestResponseOpenAIResponses2Claude_FallsBackToAnnotationsWhenSourcesMissing(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		ID:    "resp_2",
		Model: "gpt-4.1",
		Output: []dto.ResponsesOutput{
			{
				Type:   dto.BuildInCallWebSearchCall,
				ID:     "ws_1",
				Action: json.RawMessage(`{"query":"latest OpenAI news"}`),
			},
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{
						Type: "output_text",
						Text: "Alpha summary",
						Annotations: []dto.ResponsesOutputAnnotation{
							{
								Type:       "url_citation",
								URL:        "https://example.com/fallback",
								Title:      "Fallback source",
								StartIndex: 0,
								EndIndex:   5,
							},
						},
					},
				},
			},
		},
	}

	claudeResp := ResponseOpenAIResponses2Claude(resp, "msg_2")
	require.NotNil(t, claudeResp)

	results, ok := claudeResp.Content[1].Content.([]dto.ClaudeWebSearchResult)
	require.True(t, ok)
	require.Len(t, results, 1)
	require.Equal(t, "https://example.com/fallback", results[0].URL)
	require.Empty(t, claudeResp.Content[1].ErrorCode)
}

func TestResponseOpenAIResponses2Claude_MarksUnavailableWhenNoSourcesExist(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		ID:    "resp_3",
		Model: "gpt-4.1",
		Output: []dto.ResponsesOutput{
			{
				Type:   dto.BuildInCallWebSearchCall,
				ID:     "ws_1",
				Action: json.RawMessage(`{"query":"latest OpenAI news"}`),
			},
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: "No citation metadata here"},
				},
			},
		},
	}

	claudeResp := ResponseOpenAIResponses2Claude(resp, "msg_3")
	require.NotNil(t, claudeResp)
	require.Equal(t, "web_search_tool_result", claudeResp.Content[1].Type)
	require.Equal(t, "unavailable", claudeResp.Content[1].ErrorCode)
}
