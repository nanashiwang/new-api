package zhipu_4v

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestReasoningEffortForwarded(t *testing.T) {
	for _, effort := range []string{"", "none", "low", "high"} {
		request := requestOpenAI2Zhipu(dto.GeneralOpenAIRequest{Model: "glm-test", ReasoningEffort: effort})
		require.Equal(t, effort, request.ReasoningEffort)
		payload, err := common.Marshal(request)
		require.NoError(t, err)
		if effort == "" {
			require.NotContains(t, string(payload), "reasoning_effort")
		} else {
			require.Contains(t, string(payload), `"reasoning_effort":"`+effort+`"`)
		}
	}
}
