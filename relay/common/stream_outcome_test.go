package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamOutcomeSeparatesTransportAndProtocol(t *testing.T) {
	s := NewStreamStatus()
	s.RequireTerminal()
	s.SetEndReason(StreamEndReasonEOF, nil)
	require.False(t, s.IsSuccessful(), "HTTP 200/EOF alone is not completion")
	s.MarkOutcome(ResponseOutcomeCompleted)
	require.True(t, s.IsSuccessful())
	s.MarkOutcome(ResponseOutcomeFailed)
	s.MarkOutcome(ResponseOutcomeCompleted)
	require.False(t, s.IsSuccessful(), "a later completion cannot erase an error")
	for _, outcome := range []ResponseOutcome{ResponseOutcomeIncomplete, ResponseOutcomeCancelled} {
		s = NewStreamStatus()
		s.SetEndReason(StreamEndReasonDone, nil)
		s.MarkOutcome(outcome)
		require.False(t, s.IsSuccessful())
	}
	s = NewStreamStatus()
	s.SetEndReason(StreamEndReasonEOF, nil)
	require.True(t, s.IsSuccessful(), "legacy protocols without explicit terminals remain compatible")
	s.RecordError("bad event")
	require.False(t, s.IsSuccessful())
}
