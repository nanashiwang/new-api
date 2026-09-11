package dto

import (
	"fmt"
	"regexp"
)

var cpaInstanceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// ValidateCPAIdentity keeps the gateway namespace stable and safe for HTTP headers.
func (s ChannelSettings) ValidateCPAIdentity() error {
	if s.CPAInstanceID != "" && !cpaInstanceIDPattern.MatchString(s.CPAInstanceID) {
		return fmt.Errorf("cpa_instance_id must contain 1-64 ASCII letters, digits, dots, underscores or hyphens and start with a letter or digit")
	}
	if s.CPAUserIdentityEnabled && s.CPAInstanceID == "" {
		return fmt.Errorf("cpa_instance_id is required when CPA user identity is enabled")
	}
	return nil
}
