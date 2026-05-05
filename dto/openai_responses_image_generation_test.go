package dto

import (
	"encoding/json"
	"testing"
)

func TestOpenAIResponsesRequestHasImageGenerationTool(t *testing.T) {
	tests := []struct {
		name string
		req  OpenAIResponsesRequest
		want bool
	}{
		{
			name: "tools array",
			req: OpenAIResponsesRequest{
				Tools: json.RawMessage(`[{"type":"image_generation","size":"1536x1024"}]`),
			},
			want: true,
		},
		{
			name: "tool choice object",
			req: OpenAIResponsesRequest{
				ToolChoice: json.RawMessage(`{"type":"image_generation"}`),
			},
			want: true,
		},
		{
			name: "other tool",
			req: OpenAIResponsesRequest{
				Tools:      json.RawMessage(`[{"type":"web_search_preview"}]`),
				ToolChoice: json.RawMessage(`"auto"`),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.HasImageGenerationTool(); got != tt.want {
				t.Fatalf("HasImageGenerationTool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenAIResponsesResponseCountImageGenerationCalls(t *testing.T) {
	resp := OpenAIResponsesResponse{
		Output: []ResponsesOutput{
			{Type: ResponsesOutputTypeImageGenerationCall},
			{Type: "message"},
			{Type: ResponsesOutputTypeImageGenerationCall},
		},
	}

	if got := resp.CountImageGenerationCalls(); got != 2 {
		t.Fatalf("CountImageGenerationCalls() = %d, want 2", got)
	}
	if !resp.HasImageGenerationCall() {
		t.Fatalf("HasImageGenerationCall() should be true")
	}
}
