package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeRequestPreservesMessageOutputConfig(t *testing.T) {
	body := `{"model":"claude-test","max_tokens":64,"output_config":{"effort":"medium"},"messages":[
		{"role":"user","content":"summary"},
		{"role":"system","content":[],"output_config":{"effort":"high","future_option":{"keep":true}}},
		{"role":"assistant","content":"done"}]}`
	var request dto.ClaudeRequest
	require.NoError(t, common.UnmarshalJsonStr(body, &request))
	copy, err := common.DeepCopy(&request)
	require.NoError(t, err)
	converted, err := (&Adaptor{}).ConvertClaudeRequest(nil, &relaycommon.RelayInfo{}, copy)
	require.NoError(t, err)
	payload, err := common.Marshal(converted)
	require.NoError(t, err)
	require.JSONEq(t, body, string(payload))
	// Each retry owns its JSON bytes; changing a copy must not affect the input.
	copy.Messages[1].OutputConfig[0] = ' '
	require.Equal(t, byte('{'), request.Messages[1].OutputConfig[0])
}
