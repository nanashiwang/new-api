package common

import (
	"fmt"
	"sync/atomic"
)

const (
	InviteBindingRateMin = 0
	InviteBindingRateMax = 100
)

type InviteBindingSettings struct {
	Threshold          int `json:"threshold"`
	RateAfterThreshold int `json:"rate_after_threshold"`
}

type inviteBindingSettingsPayload struct {
	Threshold          *int `json:"threshold"`
	RateAfterThreshold *int `json:"rate_after_threshold"`
}

var inviteBindingSettings atomic.Value

func init() {
	inviteBindingSettings.Store(InviteBindingSettings{
		Threshold:          0,
		RateAfterThreshold: InviteBindingRateMax,
	})
}

func GetInviteBindingSettings() InviteBindingSettings {
	return inviteBindingSettings.Load().(InviteBindingSettings)
}

func ValidateInviteBindingSettings(settings InviteBindingSettings) error {
	if settings.Threshold < 0 {
		return fmt.Errorf("invite binding threshold must be a non-negative integer")
	}
	if settings.RateAfterThreshold < InviteBindingRateMin || settings.RateAfterThreshold > InviteBindingRateMax {
		return fmt.Errorf("invite binding rate must be between %d and %d", InviteBindingRateMin, InviteBindingRateMax)
	}
	return nil
}

func ParseInviteBindingSettings(raw string) (InviteBindingSettings, error) {
	payload := inviteBindingSettingsPayload{}
	if err := UnmarshalJsonStr(raw, &payload); err != nil {
		return InviteBindingSettings{}, fmt.Errorf("invalid invite binding settings: %w", err)
	}
	if payload.Threshold == nil || payload.RateAfterThreshold == nil {
		return InviteBindingSettings{}, fmt.Errorf("invite binding threshold and rate are both required")
	}
	settings := InviteBindingSettings{
		Threshold:          *payload.Threshold,
		RateAfterThreshold: *payload.RateAfterThreshold,
	}
	if err := ValidateInviteBindingSettings(settings); err != nil {
		return InviteBindingSettings{}, err
	}
	return settings, nil
}

func SetInviteBindingSettings(settings InviteBindingSettings) error {
	if err := ValidateInviteBindingSettings(settings); err != nil {
		return err
	}
	inviteBindingSettings.Store(settings)
	return nil
}

func UpdateInviteBindingSettingsByJSONString(raw string) error {
	settings, err := ParseInviteBindingSettings(raw)
	if err != nil {
		return err
	}
	return SetInviteBindingSettings(settings)
}

func InviteBindingSettings2JSONString() string {
	raw, err := Marshal(GetInviteBindingSettings())
	if err != nil {
		return "{\"threshold\":0,\"rate_after_threshold\":100}"
	}
	return string(raw)
}
