package moonshot

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURLKimiCodingSupportsOpenAIAndClaude(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	tests := []struct {
		name        string
		baseURL     string
		relayFormat types.RelayFormat
		want        string
	}{
		{
			name:        "symbolic plan openai",
			baseURL:     "kimi-coding-plan",
			relayFormat: types.RelayFormatOpenAI,
			want:        "https://api.kimi.com/coding/v1/chat/completions",
		},
		{
			name:        "canonical url openai",
			baseURL:     "https://api.kimi.com/coding/",
			relayFormat: types.RelayFormatOpenAI,
			want:        "https://api.kimi.com/coding/v1/chat/completions",
		},
		{
			name:        "symbolic plan claude",
			baseURL:     "kimi-coding-plan",
			relayFormat: types.RelayFormatClaude,
			want:        "https://api.kimi.com/coding/v1/messages",
		},
		{
			name:        "canonical url claude",
			baseURL:     "https://api.kimi.com/coding/",
			relayFormat: types.RelayFormatClaude,
			want:        "https://api.kimi.com/coding/v1/messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := adaptor.GetRequestURL(&common.RelayInfo{
				ChannelMeta: &common.ChannelMeta{ChannelBaseUrl: tt.baseURL},
				RelayFormat: tt.relayFormat,
				RelayMode:   relayconstant.RelayModeChatCompletions,
			})
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGetRequestURLRegularMoonshotKeepsExistingClaudePath(t *testing.T) {
	t.Parallel()

	got, err := (&Adaptor{}).GetRequestURL(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{ChannelBaseUrl: "https://api.moonshot.cn"},
		RelayFormat: types.RelayFormatClaude,
	})
	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/anthropic/v1/messages", got)
}

func TestSetupRequestHeaderForClaude(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("anthropic-version", "2024-01-01")
	ctx.Request.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")

	header := http.Header{}
	err := (&Adaptor{}).SetupRequestHeader(ctx, &header, &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{ApiKey: "kimi-key"},
		RelayFormat: types.RelayFormatClaude,
	})
	require.NoError(t, err)
	require.Equal(t, "Bearer kimi-key", header.Get("Authorization"))
	require.Empty(t, header.Get("x-api-key"))
	require.Equal(t, "2024-01-01", header.Get("anthropic-version"))
	require.Equal(t, "prompt-caching-2024-07-31", header.Get("anthropic-beta"))
}

func TestSetupRequestHeaderForClaudeDefaultsAnthropicVersion(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	header := http.Header{}
	err := (&Adaptor{}).SetupRequestHeader(ctx, &header, &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{ApiKey: "kimi-key"},
		RelayFormat: types.RelayFormatClaude,
	})
	require.NoError(t, err)
	require.Equal(t, "2023-06-01", header.Get("anthropic-version"))
}
