package openai

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func kimiUsageEvents(t *testing.T, output string) []rawUsageObject {
	t.Helper()
	var events []rawUsageObject
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "data: {") {
			continue
		}
		root := usageObject([]byte(strings.TrimPrefix(line, "data: ")))
		if usage := usageObject(root["usage"]); usage != nil {
			events = append(events, usage)
		}
	}
	return events
}

func TestKimiSameTotalUsageTailKeepsReportedCounts(t *testing.T) {
	c, recorder := newResponsesStreamTestContext()
	info := kimiUsageInfo(true)
	body := "data: " + `{"id":"same","choices":[{"delta":{"content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120,"cached_tokens":80,"reasoning_tokens":18}}` + "\n\n" +
		"data: " + `{"id":"same","choices":[],"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120}}` + "\n\ndata: [DONE]\n\n"
	usage, apiErr := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.Nil(t, apiErr)
	require.Equal(t, 80, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 18, usage.CompletionTokenDetails.ReasoningTokens)
	events := kimiUsageEvents(t, recorder.Body.String())
	require.Len(t, events, 2)
	require.JSONEq(t, "80", string(events[1]["cached_tokens"]))
	require.JSONEq(t, "18", string(events[1]["reasoning_tokens"]))
	require.Equal(t, 120, usage.TotalTokens, "snapshots must not be summed")
	require.Equal(t, "usage.cached_tokens", info.KimiUsageEvidence.Cache.Source)
	require.True(t, info.KimiUsageEvidence.Cache.RestoredFromPriorChunk)
	require.True(t, info.KimiUsageEvidence.Reasoning.RestoredFromPriorChunk)
}

func TestKimiSingleChoiceReasoningEvidenceIsPreserved(t *testing.T) {
	for _, choiceUsage := range []string{
		`{"reasoning_tokens":18}`,
		`{"completion_tokens_details":{"reasoning_tokens":18}}`,
	} {
		raw := []byte(`{"choices":[{"usage":` + choiceUsage + `}],"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120}}`)
		result := normalizeKimiChatUsage(kimiUsageInfo(false), raw)
		var decoded struct {
			Usage struct {
				Reasoning int `json:"reasoning_tokens"`
			} `json:"usage"`
		}
		require.NoError(t, common.Unmarshal(result, &decoded))
		require.Equal(t, 18, decoded.Usage.Reasoning)
	}
	multi := []byte(`{"choices":[{"usage":{"reasoning_tokens":8}},{"usage":{"reasoning_tokens":10}}],"usage":{"prompt_tokens":100,"completion_tokens":20}}`)
	require.Equal(t, multi, normalizeKimiChatUsage(kimiUsageInfo(false), multi))
	canonical := []byte(`{"choices":[{"usage":{"reasoning_tokens":18}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"completion_tokens_details":{"reasoning_tokens":0}}}`)
	result := usageObject(usageObject(normalizeKimiChatUsage(kimiUsageInfo(false), canonical))["usage"])
	require.JSONEq(t, "0", string(result["reasoning_tokens"]))
	for _, raw := range []string{
		`{"choices":[{"index":0,"usage":{"reasoning_tokens":8}}],"usage":{"prompt_tokens":100,"completion_tokens":20}}`,
		`{"choices":[{"index":1,"usage":{"reasoning_tokens":10}}],"usage":{"prompt_tokens":100,"completion_tokens":20}}`,
	} {
		info := kimiUsageInfo(false)
		info.Request = &dto.GeneralOpenAIRequest{N: 2}
		require.Equal(t, raw, string(normalizeKimiChatUsage(info, []byte(raw))), "partial multi-choice chunks must not become response totals")
	}
}

func TestKimiUsageSnapshotRestorationIsConservative(t *testing.T) {
	for _, tc := range []struct {
		name, tail, cacheStatus, reasoningStatus string
		cacheValue, reasoningValue               *int
	}{
		{"same_totals", `{"prompt_tokens":100,"completion_tokens":20}`, "reported", "reported", common.GetPointer(80), common.GetPointer(18)},
		{"changed_input", `{"prompt_tokens":101,"completion_tokens":20}`, "missing", "reported", nil, common.GetPointer(18)},
		{"changed_output", `{"prompt_tokens":100,"completion_tokens":21}`, "reported", "missing", common.GetPointer(80), nil},
		{"no_totals", `{}`, "missing", "missing", nil, nil},
		{"explicit_zero", `{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":0,"reasoning_tokens":0}`, "reported", "reported", common.GetPointer(0), common.GetPointer(0)},
		{"explicit_null", `{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":{"cached_tokens":null},"completion_tokens_details":{"reasoning_tokens":null}}`, "unknown", "unknown", nil, nil},
		{"null_containers", `{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":null,"completion_tokens_details":null}`, "unknown", "unknown", nil, nil},
		{"null_alias_containers", `{"prompt_tokens":100,"completion_tokens":20,"input_tokens_details":null,"output_tokens_details":null}`, "unknown", "unknown", nil, nil},
		{"fresh_count_with_null_container", `{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":null,"completion_tokens_details":null,"cached_tokens":0,"reasoning_tokens":0}`, "reported", "reported", common.GetPointer(0), common.GetPointer(0)},
		{"bad_container", `{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":"bad","completion_tokens_details":[]}`, "invalid", "invalid", nil, nil},
		{"negative_counts", `{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":-1,"reasoning_tokens":-2}`, "invalid", "invalid", nil, nil},
		{"oversized_counts", `{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":101,"reasoning_tokens":21}`, "invalid", "invalid", nil, nil},
		{"string_counts", `{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":"80","reasoning_tokens":"18"}`, "invalid", "invalid", nil, nil},
		{"new_counts", `{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":40,"reasoning_tokens":9}`, "reported", "reported", common.GetPointer(40), common.GetPointer(9)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := kimiUsageInfo(false)
			var state kimiUsageStreamEvidence
			state.normalize(info, []byte(`{"id":"same","usage":{"prompt_tokens":100,"completion_tokens":20,"cache_read_tokens":80,"reasoning_tokens":18}}`))
			output := state.normalize(info, []byte(`{"id":"same","usage":`+tc.tail+`}`))
			require.Equal(t, tc.cacheStatus, info.KimiUsageEvidence.Cache.Status)
			require.Equal(t, tc.reasoningStatus, info.KimiUsageEvidence.Reasoning.Status)
			require.Equal(t, tc.cacheValue, info.KimiUsageEvidence.Cache.Value)
			require.Equal(t, tc.reasoningValue, info.KimiUsageEvidence.Reasoning.Value)
			if tc.cacheStatus != "reported" {
				require.False(t, info.KimiUsageEvidence.Cache.RestoredFromPriorChunk)
			}
			if tc.reasoningStatus != "reported" {
				require.False(t, info.KimiUsageEvidence.Reasoning.RestoredFromPriorChunk)
			}
			if tc.name == "no_totals" {
				require.NotContains(t, string(output), `"cached_tokens"`)
				require.NotContains(t, string(output), `"reasoning_tokens"`)
			}
		})
	}
}

func TestKimiUsageSnapshotsDoNotCrossResponsesOrUnknownTotals(t *testing.T) {
	for _, tc := range []struct {
		name, first, between, last string
	}{
		{"different_id", `{"id":"a","usage":{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":80,"reasoning_tokens":18}}`, `{}`, `{"id":"b","usage":{"prompt_tokens":100,"completion_tokens":20}}`},
		{"id_changed_in_metadata", `{"id":"a","usage":{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":80,"reasoning_tokens":18}}`, `{"id":"b","choices":[]}`, `{"usage":{"prompt_tokens":100,"completion_tokens":20}}`},
		{"different_model", `{"model":"k3-a","usage":{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":80,"reasoning_tokens":18}}`, `{"model":"k3-b"}`, `{"usage":{"prompt_tokens":100,"completion_tokens":20}}`},
		{"unknown_initial_totals", `{"usage":{"cached_tokens":80,"reasoning_tokens":18}}`, `{}`, `{"usage":{"prompt_tokens":100,"completion_tokens":20}}`},
		{"invalidates_old_counts", `{"usage":{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":80,"reasoning_tokens":18}}`, `{"usage":{"prompt_tokens":101,"completion_tokens":21}}`, `{"usage":{"prompt_tokens":100,"completion_tokens":20}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := kimiUsageInfo(false)
			var state kimiUsageStreamEvidence
			state.normalize(info, []byte(tc.first))
			state.normalize(info, []byte(tc.between))
			output := state.normalize(info, []byte(tc.last))
			require.NotContains(t, string(output), `"cached_tokens"`)
			require.NotContains(t, string(output), `"reasoning_tokens"`)
			require.Equal(t, "missing", info.KimiUsageEvidence.Cache.Status)
			require.Equal(t, "missing", info.KimiUsageEvidence.Reasoning.Status)
		})
	}
}

func TestKimiUsageEvidenceOptOutAndModelIsolation(t *testing.T) {
	for _, channelType := range []int{constant.ChannelTypeOpenAI, constant.ChannelTypeMoonshot} {
		for _, include := range []bool{false, true} {
			t.Run(fmt.Sprintf("channel=%d/include=%t", channelType, include), func(t *testing.T) {
				c, recorder := newResponsesStreamTestContext()
				info := kimiUsageInfo(true)
				info.ChannelType = channelType
				info.ShouldIncludeUsage = include
				body := "data: " + `{"choices":[{"delta":{"content":"private-fixture-content"},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":20,"cache_read_tokens":80,"reasoning_tokens":18}}` + "\n\n" +
					"data: " + `{"choices":[],"usage":{"prompt_tokens":100,"completion_tokens":20}}` + "\n\ndata: [DONE]\n\n"
				usage, apiErr := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
				require.Nil(t, apiErr)
				require.Equal(t, 80, usage.PromptTokensDetails.CachedTokens)
				require.Equal(t, include, strings.Contains(recorder.Body.String(), `"usage"`))
				raw, err := common.Marshal(info.KimiUsageEvidence)
				require.NoError(t, err)
				require.NotContains(t, string(raw), "private-fixture-content")
				require.Contains(t, string(raw), `"source":"usage.cache_read_tokens"`)
			})
		}
	}
	info := kimiUsageInfo(false)
	info.UpstreamModelName = "kimi-k2.5"
	raw := []byte(`{"usage":{"prompt_tokens":100,"cache_read_tokens":80}}`)
	var state kimiUsageStreamEvidence
	require.Equal(t, raw, state.normalize(info, raw))
	require.Nil(t, info.KimiUsageEvidence)
}
