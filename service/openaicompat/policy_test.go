package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func TestShouldChatCompletionsUseResponsesByChannelSetting(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		AllChannels:   true,
		ModelPatterns: []string{"^gpt-5.*$"},
	}

	tests := []struct {
		name    string
		setting dto.ChannelSettings
		model   string
		want    bool
	}{
		{
			name:    "enabled overrides global miss",
			setting: dto.ChannelSettings{ChatCompletionsToResponsesMode: dto.ChatCompletionsToResponsesModeEnabled},
			model:   "claude-3-7-sonnet",
			want:    true,
		},
		{
			name:    "disabled overrides global hit",
			setting: dto.ChannelSettings{ChatCompletionsToResponsesMode: dto.ChatCompletionsToResponsesModeDisabled},
			model:   "gpt-5",
			want:    false,
		},
		{
			name:    "inherit follows global hit",
			setting: dto.ChannelSettings{ChatCompletionsToResponsesMode: dto.ChatCompletionsToResponsesModeInherit},
			model:   "gpt-5-mini",
			want:    true,
		},
		{
			name:    "inherit follows global miss",
			setting: dto.ChannelSettings{},
			model:   "claude-3-7-sonnet",
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldChatCompletionsUseResponsesByChannelSetting(tc.setting, policy, 1, 1, tc.model)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
