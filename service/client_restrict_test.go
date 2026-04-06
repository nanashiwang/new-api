package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestIsClientAllowed(t *testing.T) {
	tests := []struct {
		name     string
		settings dto.ChannelSettings
		clientID string
		want     bool
	}{
		{
			name: "no restriction allows all",
			settings: dto.ChannelSettings{
				ClientRestrictionMode: dto.ClientRestrictionModeNone,
			},
			clientID: "cursor",
			want:     true,
		},
		{
			name: "allowlist allows matching client",
			settings: dto.ChannelSettings{
				ClientRestrictionMode:    dto.ClientRestrictionModeAllowlist,
				ClientRestrictionClients: []string{"codex-cli"},
			},
			clientID: "codex-cli",
			want:     true,
		},
		{
			name: "allowlist rejects non matching client",
			settings: dto.ChannelSettings{
				ClientRestrictionMode:    dto.ClientRestrictionModeAllowlist,
				ClientRestrictionClients: []string{"codex-cli"},
			},
			clientID: "cursor",
			want:     false,
		},
		{
			name: "allowlist rejects empty client list",
			settings: dto.ChannelSettings{
				ClientRestrictionMode:    dto.ClientRestrictionModeAllowlist,
				ClientRestrictionClients: []string{},
			},
			clientID: "codex-cli",
			want:     false,
		},
		{
			name: "blocklist with empty client list stays unrestricted",
			settings: dto.ChannelSettings{
				ClientRestrictionMode:    dto.ClientRestrictionModeBlocklist,
				ClientRestrictionClients: []string{},
			},
			clientID: "codex-cli",
			want:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isClientAllowed(tc.settings, tc.clientID); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
