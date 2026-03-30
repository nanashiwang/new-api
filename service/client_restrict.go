package service

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

// IsChannelAllowedForClient checks whether the given channel allows the given client.
// Returns true if the channel has no restriction or the client passes the restriction check.
func IsChannelAllowedForClient(channel *model.Channel, clientID string) bool {
	if channel == nil {
		return true
	}
	settings := channel.GetSetting()
	return isClientAllowed(settings, clientID)
}

func isClientAllowed(settings dto.ChannelSettings, clientID string) bool {
	mode := settings.ClientRestrictionMode
	if mode == "" || mode == dto.ClientRestrictionModeNone {
		return true
	}

	clients := settings.ClientRestrictionClients
	if len(clients) == 0 {
		return true // empty list means no restriction regardless of mode
	}

	found := false
	for _, c := range clients {
		if c == clientID {
			found = true
			break
		}
	}

	switch mode {
	case dto.ClientRestrictionModeAllowlist:
		return found
	case dto.ClientRestrictionModeBlocklist:
		return !found
	default:
		return true
	}
}
