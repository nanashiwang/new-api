package logger

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLogCounterConcurrentRequestFailures(t *testing.T) {
	oldCount := logCount.Swap(0)
	t.Cleanup(func() { logCount.Store(oldCount) })
	oldDir, oldWriter, oldErrorWriter := *common.LogDir, gin.DefaultWriter, gin.DefaultErrorWriter
	*common.LogDir, gin.DefaultWriter, gin.DefaultErrorWriter = "", io.Discard, io.Discard
	t.Cleanup(func() {
		*common.LogDir, gin.DefaultWriter, gin.DefaultErrorWriter = oldDir, oldWriter, oldErrorWriter
	})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 256; n++ {
				LogInfo(context.Background(), "stream canceled fixture")
				LogError(context.Background(), "refund fixture")
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 16*256*2, logCount.Load())
	require.NotNil(t, gin.DefaultWriter)
}

func TestLogRotationTriggerReleasesOnlyItsOwnState(t *testing.T) {
	oldDir, oldWriter, oldErrorWriter := *common.LogDir, gin.DefaultWriter, gin.DefaultErrorWriter
	oldCount := logCount.Swap(maxLogCount)
	*common.LogDir, gin.DefaultWriter, gin.DefaultErrorWriter = "", io.Discard, io.Discard
	t.Cleanup(func() {
		*common.LogDir, gin.DefaultWriter, gin.DefaultErrorWriter = oldDir, oldWriter, oldErrorWriter
		logCount.Store(oldCount)
		setupLogWorking.Store(false)
	})
	// Direct initialization must not release an async rotation's ownership.
	setupLogWorking.Store(true)
	SetupLogger()
	require.True(t, setupLogWorking.Load())
	for i := 0; i < 10; i++ {
		LogInfo(context.Background(), "rotation already scheduled")
	}
	require.EqualValues(t, maxLogCount+10, logCount.Load())
	setupLogWorking.Store(false)
	LogInfo(context.Background(), "schedule one rotation")
	require.Eventually(t, func() bool { return !setupLogWorking.Load() }, time.Second, time.Millisecond)
	require.Zero(t, logCount.Load())
	LogError(context.Background(), "after rotation")
	require.EqualValues(t, 1, logCount.Load())
}
