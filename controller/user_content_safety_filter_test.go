package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestParseContentSafetyStatusUsesClosedAllowlist(t *testing.T) {
	status, err := parseContentSafetyStatus(" FINAL_WARNING ")
	require.NoError(t, err)
	require.Equal(t, model.ContentSafetyLevelFinalWarning, status)

	_, err = parseContentSafetyStatus("disabled OR 1=1")
	require.Error(t, err)
}

func TestParseContentSafetyCodesNormalizesDeduplicatesAndRejectsUnknown(t *testing.T) {
	codes, err := parseContentSafetyCodes(" CYBER_POLICY,content_filter,cyber_policy ")
	require.NoError(t, err)
	require.Equal(t, []string{"cyber_policy", "content_filter"}, codes)

	_, err = parseContentSafetyCodes("context_length_exceeded")
	require.Error(t, err)
	_, err = parseContentSafetyCodes("rate_limit_exceeded")
	require.Error(t, err)
}
