package openai

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/stretchr/testify/require"
)

// These are usage-only excerpts from synthetic requests on 2026-09-29.
// The first cases are gateway-boundary observations; direct_* cases are
// independently captured supplier responses. Prompts/output are fixtures.
func TestKimiObservedUsageExcerptsThroughLocalHandlers(t *testing.T) {
	for _, tc := range []struct {
		name, usage  string
		cache, think *int
	}{
		{
			"reported_cache_zero",
			`{"prompt_tokens":5720,"completion_tokens":57,"total_tokens":5777,"completion_tokens_details":{"reasoning_tokens":41},"prompt_tokens_details":{},"cache_write_tokens":0,"cache_read_tokens":0}`,
			common.GetPointer(0), common.GetPointer(41),
		},
		{
			"cache_hit",
			`{"prompt_tokens":5720,"completion_tokens":73,"total_tokens":5793,"cached_tokens":5720,"completion_tokens_details":{"reasoning_tokens":56},"prompt_tokens_details":{"cached_tokens":5720},"cache_write_tokens":0,"cache_read_tokens":5720}`,
			common.GetPointer(5720), common.GetPointer(56),
		},
		{
			"reasoning_count_unreported",
			`{"prompt_tokens":119,"completion_tokens":525,"total_tokens":644,"prompt_tokens_details":{"cached_tokens":64}}`,
			common.GetPointer(64), nil,
		},
		{
			"cache_count_unreported",
			`{"prompt_tokens":5720,"completion_tokens":32,"total_tokens":5752,"completion_tokens_details":{"reasoning_tokens":18},"prompt_tokens_details":{}}`,
			nil, common.GetPointer(18),
		},
		// Direct supplier samples captured without the production gateway on
		// 2026-09-29. Missing counts must stay missing even after local replay.
		{
			"direct_1468_high_missing_reasoning",
			`{"prompt_tokens":119,"completion_tokens":328,"total_tokens":447,"prompt_tokens_details":{"cached_tokens":64}}`,
			common.GetPointer(64), nil,
		},
		{
			"direct_1468_max_missing_reasoning",
			`{"prompt_tokens":119,"completion_tokens":331,"total_tokens":450,"prompt_tokens_details":{"cached_tokens":64}}`,
			common.GetPointer(64), nil,
		},
		{
			"direct_1466_high",
			`{"prompt_tokens":120,"completion_tokens":145,"total_tokens":265,"completion_tokens_details":{"reasoning_tokens":98},"prompt_tokens_details":{},"cache_write_tokens":0,"cache_read_tokens":0}`,
			common.GetPointer(0), common.GetPointer(98),
		},
		{
			"direct_1466_max",
			`{"prompt_tokens":119,"completion_tokens":147,"total_tokens":266,"completion_tokens_details":{"reasoning_tokens":92},"prompt_tokens_details":{},"cache_write_tokens":0,"cache_read_tokens":0}`,
			common.GetPointer(0), common.GetPointer(92),
		},
		{
			"direct_1466_cache_first",
			`{"prompt_tokens":5723,"completion_tokens":61,"total_tokens":5784,"completion_tokens_details":{"reasoning_tokens":45},"prompt_tokens_details":{},"cache_write_tokens":0,"cache_read_tokens":0}`,
			common.GetPointer(0), common.GetPointer(45),
		},
		{
			"direct_1466_cache_repeat",
			`{"prompt_tokens":5723,"completion_tokens":73,"total_tokens":5796,"cached_tokens":5723,"completion_tokens_details":{"reasoning_tokens":56},"prompt_tokens_details":{"cached_tokens":5723},"cache_write_tokens":0,"cache_read_tokens":5723}`,
			common.GetPointer(5723), common.GetPointer(56),
		},
	} {
		for _, stream := range []bool{false, true} {
			for _, force := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/stream=%t/force=%t", tc.name, stream, force), func(t *testing.T) {
					c, recorder := newResponsesStreamTestContext()
					info := kimiUsageInfo(force)
					raw := `{"id":"observed-usage-fixture","model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":` + tc.usage + `}`
					var usage *dto.Usage
					if stream {
						body := "data: " + strings.Replace(raw, `"message":{"role":"assistant","content":"OK"}`, `"delta":{"content":"OK"}`, 1) + "\n\n" +
							"data: " + `{"choices":[],"metadata":"trailer"}` + "\n\ndata: [DONE]\n\n"
						u, apiErr := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
						require.Nil(t, apiErr)
						usage = u
					} else {
						u, apiErr := OpenaiHandler(c, info, &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(raw))})
						require.Nil(t, apiErr)
						usage = u
					}
					output := recorder.Body.String()
					if stream {
						for _, line := range strings.Split(output, "\n") {
							if strings.HasPrefix(line, "data: {") && strings.Contains(line, `"usage"`) {
								output = strings.TrimPrefix(line, "data: ")
								break
							}
						}
					}
					result := usageObject(usageObject([]byte(output))["usage"])
					for _, field := range []struct {
						details, name string
						expected      *int
						internal      int
					}{
						{"prompt_tokens_details", "cached_tokens", tc.cache, usage.PromptTokensDetails.CachedTokens},
						{"completion_tokens_details", "reasoning_tokens", tc.think, usage.CompletionTokenDetails.ReasoningTokens},
					} {
						rawCount, exists := result[field.name]
						if field.expected == nil {
							require.False(t, exists, "unreported alias must remain unknown")
							require.NotContains(t, usageObject(result[field.details]), field.name)
							continue
						}
						require.True(t, exists)
						count, valid := nonnegativeUsageCount(rawCount)
						require.True(t, valid)
						require.Equal(t, *field.expected, count)
						require.Equal(t, *field.expected, field.internal)
					}
					source := usageObject([]byte(tc.usage))
					for _, key := range []string{"prompt_tokens", "completion_tokens", "total_tokens", "cache_write_tokens", "cache_read_tokens"} {
						if value, exists := source[key]; exists {
							require.JSONEq(t, string(value), string(result[key]), "reported field %s must not change", key)
						}
					}
				})
			}
		}
	}
}

func kimiUsageInfo(force bool) *relaycommon.RelayInfo {
	info := chatDeliveryInfo()
	info.ChannelType = constant.ChannelTypeMoonshot
	info.ChannelSetting.ForceFormat = force
	return info
}

func TestKimiUsageEvidenceBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, fields, expected string
	}{
		{"unknown", `"prompt_tokens":100,"completion_tokens":20`, `{"prompt_tokens":100,"completion_tokens":20}`},
		{"zero", `"prompt_tokens":100,"completion_tokens":20,"cached_tokens":0,"reasoning_tokens":0`,
			`{"prompt_tokens":100,"completion_tokens":20,"cached_tokens":0,"reasoning_tokens":0,"prompt_tokens_details":{"cached_tokens":0},"completion_tokens_details":{"reasoning_tokens":0}}`},
		{"canonical_zero", `"prompt_tokens":100,"cached_tokens":80,"prompt_tokens_details":{"cached_tokens":0}`,
			`{"prompt_tokens":100,"cached_tokens":80,"prompt_tokens_details":{"cached_tokens":0}}`},
		{"canonical_null", `"prompt_tokens":100,"cached_tokens":80,"prompt_tokens_details":{"cached_tokens":null}`,
			`{"prompt_tokens":100,"cached_tokens":80,"prompt_tokens_details":{"cached_tokens":null}}`},
		{"details_aliases", `"prompt_tokens":100,"completion_tokens":20,"input_tokens_details":{"cached_tokens":50},"output_tokens_details":{"reasoning_tokens":12}`,
			`{"prompt_tokens":100,"completion_tokens":20,"input_tokens_details":{"cached_tokens":50},"output_tokens_details":{"reasoning_tokens":12},"prompt_tokens_details":{"cached_tokens":50},"completion_tokens_details":{"reasoning_tokens":12},"cached_tokens":50,"reasoning_tokens":12}`},
		{"unknown_sibling", `"prompt_tokens":100,"cached_tokens":80,"prompt_tokens_details":{"vendor_marker":"keep"}`,
			`{"prompt_tokens":100,"cached_tokens":80,"prompt_tokens_details":{"vendor_marker":"keep","cached_tokens":80}}`},
		{"cache_read_alias", `"prompt_tokens":100,"cache_read_tokens":80,"cache_write_tokens":20`,
			`{"prompt_tokens":100,"cache_read_tokens":80,"cache_write_tokens":20,"cached_tokens":80,"prompt_tokens_details":{"cached_tokens":80}}`},
		{"cache_read_zero_conflict", `"prompt_tokens":100,"cache_read_tokens":80,"prompt_tokens_details":{"cached_tokens":0}`,
			`{"prompt_tokens":100,"cache_read_tokens":80,"cached_tokens":0,"prompt_tokens_details":{"cached_tokens":0}}`},
		{"cache_write_is_not_read", `"prompt_tokens":100,"cache_write_tokens":80`,
			`{"prompt_tokens":100,"cache_write_tokens":80}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(`{"choices":[],"usage":{` + tc.fields + `}}`)
			result := normalizeKimiChatUsage(kimiUsageInfo(false), raw)
			require.JSONEq(t, `{"choices":[],"usage":`+tc.expected+`}`, string(result))
			require.Equal(t, result, normalizeKimiChatUsage(kimiUsageInfo(false), result), "normalization must be idempotent")
		})
	}
	for _, value := range []string{`null`, `-1`, `"80"`, `1.5`, `99999999999999999999999999`, `101`} {
		for _, field := range []string{"cached_tokens", "cache_read_tokens"} {
			raw := []byte(`{"usage":{"prompt_tokens":100,"` + field + `":` + value + `}}`)
			require.Equal(t, raw, normalizeKimiChatUsage(kimiUsageInfo(false), raw), value)
		}
	}
	for _, raw := range []string{`{"usage":null}`, `{"usage":[]}`, `{"choices":[]}`, `not-json`} {
		require.Equal(t, raw, string(normalizeKimiChatUsage(kimiUsageInfo(false), []byte(raw))))
	}
}

func TestKimiChoiceCacheUsageSurvivesTrailingMetadata(t *testing.T) {
	for _, count := range []int{0, 80} {
		c, recorder := newResponsesStreamTestContext()
		info := kimiUsageInfo(true)
		raw, err := common.Marshal(map[string]any{
			"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": "OK"}, "finish_reason": "stop", "usage": map[string]any{"cached_tokens": count}}},
			"usage":   map[string]any{"prompt_tokens": 100, "completion_tokens": 20, "total_tokens": 120},
		})
		require.NoError(t, err)
		body := "data: " + string(raw) + "\n\ndata: " + `{"choices":[],"metadata":"trailer"}` + "\n\ndata: [DONE]\n\n"
		u, apiErr := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
		require.Nil(t, apiErr)
		require.Equal(t, count, u.PromptTokensDetails.CachedTokens)
		var output map[string]any
		for _, line := range strings.Split(recorder.Body.String(), "\n") {
			if strings.HasPrefix(line, "data: {") && strings.Contains(line, `"usage"`) {
				require.NoError(t, common.UnmarshalJsonStr(strings.TrimPrefix(line, "data: "), &output))
				break
			}
		}
		require.EqualValues(t, count, output["usage"].(map[string]any)["cached_tokens"])
	}
	multi := []byte(`{"choices":[{"usage":{"cached_tokens":5}},{"usage":{"cached_tokens":8}}],"usage":{"prompt_tokens":100}}`)
	require.Equal(t, multi, normalizeKimiChatUsage(kimiUsageInfo(false), multi), "do not sum per-choice caches")
}

func TestKimiNormalizationDoesNotAffectOtherModelsOrProtocols(t *testing.T) {
	body := []byte(`{"usage":{"prompt_tokens":100,"cached_tokens":80,"reasoning_tokens":10}}`)
	for _, name := range []string{"gpt-4.1", "kimi-k2.5", "other-kimi-k3"} {
		info := kimiUsageInfo(false)
		info.UpstreamModelName = name
		require.Equal(t, body, normalizeKimiChatUsage(info, body))
	}
	info := kimiUsageInfo(false)
	info.RelayMode = relayconstant.RelayModeResponses
	require.Equal(t, body, normalizeKimiChatUsage(info, body))
	info = kimiUsageInfo(false)
	info.ChannelType = constant.ChannelTypeDeepSeek
	require.Equal(t, body, normalizeKimiChatUsage(info, body))
}

func TestKimiMissingUsageDetailsAreNotInventedByFormatting(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, reported := range []bool{false, true} {
			c, recorder := newResponsesStreamTestContext()
			info := kimiUsageInfo(true)
			usagePart := ""
			if reported {
				usagePart = `,"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120}`
			}
			if stream {
				body := "data: " + `{"choices":[{"delta":{"content":"OK"},"finish_reason":"stop"}]` + usagePart + "}\n\ndata: [DONE]\n\n"
				_, err := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
				require.Nil(t, err)
			} else {
				body := `{"choices":[{"message":{"content":"OK"},"finish_reason":"stop"}]` + usagePart + `}`
				_, err := OpenaiHandler(c, info, &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))})
				require.Nil(t, err)
			}
			require.NotContains(t, recorder.Body.String(), "cached_tokens")
			require.NotContains(t, recorder.Body.String(), "reasoning_tokens")
		}
	}
}

func TestKimiStreamUsageOptOutStillKeepsInternalCounts(t *testing.T) {
	c, recorder := newResponsesStreamTestContext()
	info := kimiUsageInfo(true)
	info.ShouldIncludeUsage = false
	body := "data: " + `{"choices":[{"delta":{"content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120,"cached_tokens":80,"reasoning_tokens":18}}` + "\n\ndata: [DONE]\n\n"
	u, err := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
	require.Nil(t, err)
	require.Equal(t, 80, u.PromptTokensDetails.CachedTokens)
	require.Equal(t, 18, u.CompletionTokenDetails.ReasoningTokens)
	require.NotContains(t, recorder.Body.String(), `"usage"`)
	require.Contains(t, recorder.Body.String(), `"content":"OK"`)
}

func TestKimiReportedUsageSurvivesChatBoundaries(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, force := range []bool{false, true} {
			for _, fields := range []string{
				`"cached_tokens":80,"reasoning_tokens":18`,
				`"prompt_tokens_details":{"cached_tokens":80},"completion_tokens_details":{"reasoning_tokens":18}`,
			} {
				c, recorder := newResponsesStreamTestContext()
				info := kimiUsageInfo(force)
				raw := `{"id":"fixture","model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120,` + fields + `}}`
				var usage *dto.Usage
				if stream {
					body := "data: " + strings.Replace(raw, `"message":{"role":"assistant","content":"OK"}`, `"delta":{"content":"OK"}`, 1) + "\n\n" +
						"data: " + `{"choices":[],"metadata":"trailer"}` + "\n\ndata: [DONE]\n\n"
					u, err := OaiStreamHandler(c, info, newResponsesStreamHTTPResponse(body))
					require.Nil(t, err)
					usage = u
				} else {
					u, err := OpenaiHandler(c, info, &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(raw))})
					require.Nil(t, err)
					usage = u
				}
				require.Equal(t, 80, usage.PromptTokensDetails.CachedTokens)
				require.Equal(t, 18, usage.CompletionTokenDetails.ReasoningTokens)
				require.Equal(t, 120, usage.TotalTokens)
				out := recorder.Body.String()
				if stream {
					for _, line := range strings.Split(out, "\n") {
						if strings.HasPrefix(line, "data: {") && strings.Contains(line, `"usage"`) {
							out = strings.TrimPrefix(line, "data: ")
							break
						}
					}
				}
				var decoded struct {
					Usage map[string]any `json:"usage"`
				}
				require.NoError(t, common.UnmarshalJsonStr(out, &decoded))
				require.EqualValues(t, 80, decoded.Usage["cached_tokens"])
				require.EqualValues(t, 18, decoded.Usage["reasoning_tokens"])
				require.EqualValues(t, 80, decoded.Usage["prompt_tokens_details"].(map[string]any)["cached_tokens"])
				require.EqualValues(t, 18, decoded.Usage["completion_tokens_details"].(map[string]any)["reasoning_tokens"])
			}
		}
	}
}
