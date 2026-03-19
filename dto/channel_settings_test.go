package dto

import "testing"

func TestChannelSettingsValidateChatCompletionsToResponsesMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    ChatCompletionsToResponsesMode
		wantErr bool
		want    ChatCompletionsToResponsesMode
	}{
		{name: "empty defaults to inherit", mode: "", want: ChatCompletionsToResponsesModeInherit},
		{name: "inherit", mode: ChatCompletionsToResponsesModeInherit, want: ChatCompletionsToResponsesModeInherit},
		{name: "enabled", mode: ChatCompletionsToResponsesModeEnabled, want: ChatCompletionsToResponsesModeEnabled},
		{name: "disabled", mode: ChatCompletionsToResponsesModeDisabled, want: ChatCompletionsToResponsesModeDisabled},
		{name: "invalid", mode: "bad", wantErr: true, want: ChatCompletionsToResponsesModeInherit},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setting := ChannelSettings{ChatCompletionsToResponsesMode: tc.mode}
			err := setting.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for mode %q", tc.mode)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for mode %q: %v", tc.mode, err)
			}
			if got := setting.GetChatCompletionsToResponsesMode(); got != tc.want {
				t.Fatalf("mode=%q got=%q want=%q", tc.mode, got, tc.want)
			}
		})
	}
}

func TestChannelSettingsValidateClaudeImageTransportMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    ClaudeImageTransportMode
		wantErr bool
		want    ClaudeImageTransportMode
	}{
		{name: "empty defaults to inherit", mode: "", want: ClaudeImageTransportModeInherit},
		{name: "inherit", mode: ClaudeImageTransportModeInherit, want: ClaudeImageTransportModeInherit},
		{name: "auto", mode: ClaudeImageTransportModeAuto, want: ClaudeImageTransportModeAuto},
		{name: "data", mode: ClaudeImageTransportModeData, want: ClaudeImageTransportModeData},
		{name: "bridge", mode: ClaudeImageTransportModeBridge, want: ClaudeImageTransportModeBridge},
		{name: "invalid", mode: "bad", wantErr: true, want: ClaudeImageTransportModeInherit},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setting := ChannelSettings{ClaudeImageTransportMode: tc.mode}
			err := setting.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for mode %q", tc.mode)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for mode %q: %v", tc.mode, err)
			}
			if got := setting.GetClaudeImageTransportMode(); got != tc.want {
				t.Fatalf("mode=%q got=%q want=%q", tc.mode, got, tc.want)
			}
		})
	}
}
