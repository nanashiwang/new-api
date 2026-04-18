package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestOpenAIResponsesResponse_UnmarshalInstructionsAsRawJSON(t *testing.T) {
	t.Parallel()

	payloads := []string{
		`{"id":"resp_1","instructions":{"type":"text","text":"hello"}}`,
		`{"id":"resp_2","instructions":["alpha","beta"]}`,
		`{"id":"resp_3","instructions":"plain text"}`,
	}

	for _, payload := range payloads {
		payload := payload
		t.Run(payload, func(t *testing.T) {
			t.Parallel()

			var resp OpenAIResponsesResponse
			if err := common.Unmarshal([]byte(payload), &resp); err != nil {
				t.Fatalf("Unmarshal returned error: %v", err)
			}
			if len(resp.Instructions) == 0 {
				t.Fatalf("Instructions should preserve raw JSON for %s", payload)
			}
		})
	}
}
