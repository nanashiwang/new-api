package openai

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestResponsesPartialUsagePreservesTerminalCountersAndEstimatesDeltas(t *testing.T) {
	setResponsesStreamTestTimeout(t)
	for _, kind := range []string{"response.output_text.delta", "response.function_call_arguments.delta", "response.reasoning_text.delta", "response.refusal.delta"} {
		c, _ := newResponsesStreamTestContext()
		info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o"}}
		info.SetEstimatePromptTokens(10)
		body := `data: {"type":"` + kind + `","delta":"some output"}` + "\n"
		usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponseWithReadError(body, context.Canceled))
		require.NotNil(t, err)
		require.NotNil(t, usage)
		require.True(t, usage.InterruptedOutput)
		require.True(t, usage.InputTokensEstimated)
		require.Equal(t, 10, usage.PromptTokens)
		require.Positive(t, usage.CompletionTokens)
	}
	c, _ := newResponsesStreamTestContext()
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o"}}
	info.SetEstimatePromptTokens(999)
	body := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n" +
		`data: {"type":"response.incomplete","response":{"status":"incomplete","usage":{"input_tokens":5,"output_tokens":8,"total_tokens":13,"input_tokens_details":{"cached_tokens":2}}}}` + "\n"
	usage, err := OaiResponsesStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.NotNil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 13, usage.TotalTokens)
	require.Equal(t, 2, usage.PromptTokensDetails.CachedTokens)
	require.False(t, usage.InputTokensEstimated)
}

func TestResponsesFailedContinuationKeepsBothUsageRecords(t *testing.T) {
	setResponsesStreamTestTimeout(t)
	c, _ := newResponsesStreamTestContext()
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o"}}
	info.SetEstimatePromptTokens(10)
	body := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n"
	var first *dto.Usage
	usage, err := OaiResponsesStreamHandlerWithOptions(c, info, newResponsesStreamHTTPResponse(body), &ResponsesStreamHandlerOptions{
		AutoContinue: func(ctx ResponsesStreamAutoContinueContext) (*dto.Usage, bool) {
			copy := *ctx.Usage
			first = &copy
			return &dto.Usage{PromptTokens: 4, CompletionTokens: 6, TotalTokens: 10, InterruptedOutput: true}, false
		},
	})
	require.NotNil(t, err)
	require.NotNil(t, usage)
	require.NotNil(t, first)
	require.Equal(t, first.TotalTokens+10, usage.TotalTokens)
	require.True(t, usage.InterruptedOutput)
}
