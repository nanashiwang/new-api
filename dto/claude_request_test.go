package dto

import (
	"reflect"
	"testing"
)

func TestClaudePassthroughFieldsMatchOfficialBranch(t *testing.T) {
	t.Parallel()

	claudeRequestType := reflect.TypeOf(ClaudeRequest{})

	cacheControlField, ok := claudeRequestType.FieldByName("CacheControl")
	if !ok {
		t.Fatalf("ClaudeRequest should expose CacheControl passthrough field")
	}
	if got := cacheControlField.Tag.Get("json"); got != "cache_control,omitempty" {
		t.Fatalf("ClaudeRequest.CacheControl json tag = %q, want %q", got, "cache_control,omitempty")
	}

	speedField, ok := claudeRequestType.FieldByName("Speed")
	if !ok {
		t.Fatalf("ClaudeRequest should expose Speed passthrough field")
	}
	if got := speedField.Tag.Get("json"); got != "speed,omitempty" {
		t.Fatalf("ClaudeRequest.Speed json tag = %q, want %q", got, "speed,omitempty")
	}

	channelOtherSettingsType := reflect.TypeOf(ChannelOtherSettings{})
	allowSpeedField, ok := channelOtherSettingsType.FieldByName("AllowSpeed")
	if !ok {
		t.Fatalf("ChannelOtherSettings should expose AllowSpeed switch")
	}
	if got := allowSpeedField.Tag.Get("json"); got != "allow_speed,omitempty" {
		t.Fatalf("ChannelOtherSettings.AllowSpeed json tag = %q, want %q", got, "allow_speed,omitempty")
	}
}
