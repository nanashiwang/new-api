/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestValidateChannelRequiresExplicitGroupWhenAdding(t *testing.T) {
	channel := &model.Channel{
		Type:   1,
		Key:    "test-key",
		Models: "gpt-test",
	}

	if err := validateChannel(channel, true); err == nil || err.Error() != "请选择分组" {
		t.Fatalf("validateChannel() error = %v, want 请选择分组", err)
	}
}

func TestValidateChannelAcceptsExplicitGroupWhenAdding(t *testing.T) {
	channel := &model.Channel{
		Type:   1,
		Key:    "test-key",
		Models: "gpt-test",
		Group:  "OpenAI · 优质",
	}

	if err := validateChannel(channel, true); err != nil {
		t.Fatalf("validateChannel() error = %v, want nil", err)
	}
}

func TestValidateChannelPreservesLegacyEmptyGroupWhenEditing(t *testing.T) {
	channel := &model.Channel{
		Type:   1,
		Models: "gpt-test",
	}

	if err := validateChannel(channel, false); err != nil {
		t.Fatalf("validateChannel() error = %v, want nil", err)
	}
}

func TestPreserveChannelGroupWhenNewClientDidNotTouchIt(t *testing.T) {
	touched := false
	channel := &PatchChannel{
		Channel:      model.Channel{Group: "default"},
		GroupTouched: &touched,
	}
	origin := &model.Channel{Group: "OpenAI · 企业专属,OpenAI · 优质"}

	preserveChannelGroupIfUntouched(channel, origin)

	if channel.Group != origin.Group {
		t.Fatalf("group = %q, want %q", channel.Group, origin.Group)
	}
}

func TestPreserveChannelGroupAllowsExplicitChange(t *testing.T) {
	touched := true
	channel := &PatchChannel{
		Channel:      model.Channel{Group: "default"},
		GroupTouched: &touched,
	}
	origin := &model.Channel{Group: "OpenAI · 企业专属,OpenAI · 优质"}

	preserveChannelGroupIfUntouched(channel, origin)

	if channel.Group != "default" {
		t.Fatalf("group = %q, want default", channel.Group)
	}
}

func TestPreserveChannelGroupProtectsLegacyStaleForm(t *testing.T) {
	channel := &PatchChannel{Channel: model.Channel{Group: "default"}}
	origin := &model.Channel{Group: "OpenAI · 企业专属,OpenAI · 优质"}

	preserveChannelGroupIfUntouched(channel, origin)

	if channel.Group != origin.Group {
		t.Fatalf("group = %q, want %q", channel.Group, origin.Group)
	}
}

func TestPreserveChannelGroupKeepsCompatibleLegacyUpdates(t *testing.T) {
	channel := &PatchChannel{Channel: model.Channel{Group: "Claude · 优质"}}
	origin := &model.Channel{Group: "OpenAI · 优质"}

	preserveChannelGroupIfUntouched(channel, origin)

	if channel.Group != "Claude · 优质" {
		t.Fatalf("group = %q, want Claude · 优质", channel.Group)
	}
}
