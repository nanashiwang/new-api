package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestChannelNonStreamHeaderTimeoutValidation(t *testing.T) {
	for _, seconds := range []int{0, 1, 120, 600} {
		value := ChannelSettings{NonStreamResponseHeaderTimeoutSec: seconds}
		require.NoError(t, value.Validate())
		raw, err := common.Marshal(value)
		require.NoError(t, err)
		var decoded ChannelSettings
		require.NoError(t, common.Unmarshal(raw, &decoded))
		require.Equal(t, seconds, decoded.NonStreamResponseHeaderTimeoutSec)
	}
	for _, seconds := range []int{-1, 601} {
		require.Error(t, (ChannelSettings{NonStreamResponseHeaderTimeoutSec: seconds}).Validate())
	}
	for _, raw := range []string{`{"nonstream_response_header_timeout_sec":1.5}`, `{"nonstream_response_header_timeout_sec":"120"}`, `{"nonstream_response_header_timeout_sec":999999999999999999999}`} {
		var decoded ChannelSettings
		require.Error(t, common.Unmarshal([]byte(raw), &decoded))
	}
}
