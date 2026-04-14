package common

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamStatus_SetEndReason_FirstWins(t *testing.T) {
	t.Parallel()
	s := NewStreamStatus()

	s.SetEndReason(StreamEndReasonDone, nil)
	s.SetEndReason(StreamEndReasonTimeout, nil)
	s.SetEndReason(StreamEndReasonClientGone, fmt.Errorf("context canceled"))

	assert.Equal(t, StreamEndReasonDone, s.EndReason)
	assert.Nil(t, s.EndError)
}

func TestStreamStatus_RecordError_Basic(t *testing.T) {
	t.Parallel()
	s := NewStreamStatus()

	s.RecordError("bad json")
	s.RecordError("another bad json")

	assert.True(t, s.HasErrors())
	assert.Equal(t, 2, s.TotalErrorCount())
	assert.Len(t, s.Errors, 2)
}

func TestStreamStatus_IsNormalEnd(t *testing.T) {
	t.Parallel()

	s := NewStreamStatus()
	s.SetEndReason(StreamEndReasonDone, nil)
	assert.True(t, s.IsNormalEnd())

	s2 := NewStreamStatus()
	s2.SetEndReason(StreamEndReasonTimeout, nil)
	assert.False(t, s2.IsNormalEnd())
}

func TestStreamStatus_RecordError_Concurrent(t *testing.T) {
	t.Parallel()
	s := NewStreamStatus()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			s.RecordError(fmt.Sprintf("error_%d", idx))
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 50, s.TotalErrorCount())
	assert.LessOrEqual(t, len(s.Errors), maxStreamErrorEntries)
}
