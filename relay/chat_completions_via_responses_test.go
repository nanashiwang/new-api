package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestRestoreResponsesInstructionsFromOriginalChatBackfillsTrimmedSessionRequest(t *testing.T) {
	t.Parallel()

	trimmedResponsesReq := &dto.OpenAIResponsesRequest{}
	originalChatReq := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "be helpful"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "what's next?"},
		},
	}

	err := restoreResponsesInstructionsFromOriginalChat(trimmedResponsesReq, originalChatReq)
	require.NoError(t, err)

	var instructions string
	err = common.Unmarshal(trimmedResponsesReq.Instructions, &instructions)
	require.NoError(t, err)
	require.Equal(t, "be helpful", instructions)
}

func TestRestoreResponsesInstructionsFromOriginalChatKeepsExistingInstructions(t *testing.T) {
	t.Parallel()

	trimmedResponsesReq := &dto.OpenAIResponsesRequest{
		Instructions: []byte(`"keep me"`),
	}
	originalChatReq := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "system", Content: "be helpful"},
			{Role: "user", Content: "hello"},
		},
	}

	err := restoreResponsesInstructionsFromOriginalChat(trimmedResponsesReq, originalChatReq)
	require.NoError(t, err)

	var instructions string
	err = common.Unmarshal(trimmedResponsesReq.Instructions, &instructions)
	require.NoError(t, err)
	require.Equal(t, "keep me", instructions)
}
