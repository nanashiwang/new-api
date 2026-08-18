package openai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIDebugResponseBodyForLogOmitsLargePayloads(t *testing.T) {
	small := []byte(`{"ok":true}`)
	require.Equal(t, string(small), openAIDebugResponseBodyForLog(small))

	large := []byte(strings.Repeat("audio-base64", 1024))
	logged := openAIDebugResponseBodyForLog(large)
	require.NotContains(t, logged, "audio-base64")
	require.Contains(t, logged, "response omitted")
	require.Contains(t, logged, "bytes")
}
