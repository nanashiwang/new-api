package grouphealth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/stretchr/testify/require"
)

func TestRequestFoldsRetriesByActualGroup(t *testing.T) {
	start := time.Unix(1800000000, 0)
	now := start.Add(3 * time.Second)
	r := NewRequest("client-model", true)
	r.Failure("a")
	info := &relaycommon.RelayInfo{UsingGroup: "a", StartTime: start, GroupHealthFirstOutputTime: start.Add(time.Second)}
	r.Observe(info, false, now)
	info.UsingGroup = "b"
	r.Observe(info, false, now)
	info.GroupHealthFirstOutputTime = start.Add(2 * time.Second)
	r.Observe(info, true, now)
	samples := r.Samples(now)
	require.Len(t, samples, 2)
	byGroup := map[string]Sample{}
	for _, sample := range samples {
		byGroup[sample.Group] = sample
	}
	require.False(t, byGroup["a"].Success)
	require.Nil(t, byGroup["a"].TTFTMs)
	require.True(t, byGroup["b"].Success)
	require.EqualValues(t, 2000, *byGroup["b"].TTFTMs)
	require.Equal(t, "client-model", byGroup["b"].Model)
	r.Failure("auto")
	require.Len(t, r.Samples(now), 2)
}

func TestFailedOrMissingEffectiveStreamNeverContributesLatency(t *testing.T) {
	start := time.Now().Add(-time.Second)
	for _, reason := range []relaycommon.StreamEndReason{relaycommon.StreamEndReasonScannerErr, relaycommon.StreamEndReasonTimeout, relaycommon.StreamEndReasonClientGone} {
		t.Run(string(reason), func(t *testing.T) {
			r := NewRequest("m", true)
			info := &relaycommon.RelayInfo{UsingGroup: "g", StartTime: start, GroupHealthFirstOutputTime: start.Add(time.Millisecond), StreamStatus: relaycommon.NewStreamStatus()}
			info.StreamStatus.SetEndReason(reason, nil)
			r.Observe(info, true, time.Now())
			sample := r.Samples(time.Now())[0]
			require.False(t, sample.Success)
			require.Nil(t, sample.TTFTMs)
		})
	}
	r := NewRequest("m", true)
	info := &relaycommon.RelayInfo{UsingGroup: "g", StartTime: start, FirstResponseTime: start.Add(time.Millisecond), StreamStatus: relaycommon.NewStreamStatus()}
	info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonDone, nil)
	r.Observe(info, true, time.Now())
	require.True(t, r.Samples(time.Now())[0].Success)
	require.Nil(t, r.Samples(time.Now())[0].TTFTMs, "legacy first packet must not become TTFT")
	info.StreamStatus.RecordError("in-band error despite HTTP 200")
	r.Observe(info, true, time.Now())
	require.False(t, r.Samples(time.Now())[0].Success)
	info.FirstEffectiveOutputTime = start.Add(10 * time.Millisecond)
	info.FirstEffectiveOutputChannelId = 12
	ResetAttempt(info)
	require.Nil(t, info.StreamStatus)
	require.True(t, info.GroupHealthFirstOutputTime.IsZero())
	require.Equal(t, start.Add(10*time.Millisecond), info.FirstEffectiveOutputTime, "existing routing guard timestamps must stay unchanged")
	require.Equal(t, 12, info.FirstEffectiveOutputChannelId)
}

func TestImageCompletionAndUnsupportedAsync(t *testing.T) {
	start := time.Now().Add(-2 * time.Second)
	r := NewRequest("image", false)
	info := &relaycommon.RelayInfo{UsingGroup: "image", StartTime: start, RelayMode: relayconstant.RelayModeImagesGenerations}
	r.Observe(info, true, start.Add(2*time.Second))
	sample := r.Samples(time.Now())[0]
	require.EqualValues(t, 2000, *sample.CompletionMs)
	require.Nil(t, sample.TTFTMs)
	info.IsStream = true
	r.Observe(info, true, start.Add(2*time.Second))
	require.Nil(t, r.Samples(time.Now())[0].CompletionMs)
	for _, path := range []string{"/v1/videos", "/mj/submit/imagine", "/v1/realtime", "/v1/videos/id"} {
		require.False(t, SupportsRequest(http.MethodPost, path))
	}
	require.True(t, SupportsRequest(http.MethodPost, "/v1beta/models/gemini-test:streamGenerateContent"))
	require.False(t, SupportsRequest(http.MethodGet, "/v1/chat/completions"))
}

func TestCollectorRetryIsIdempotentAndRetainsConcurrentSamples(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1800000000, 0)
	stored := map[string]model.GroupHealthMetric{}
	fail := true
	var c *collector
	concurrent := false
	c = newCollector(func(_ context.Context, rows []model.GroupHealthMetric) error {
		for _, row := range rows {
			stored[row.ID] = row
		}
		if concurrent {
			concurrent = false
			c.record(Sample{Group: "g", Model: "m", Success: true, At: now})
		}
		if fail {
			return errors.New("ambiguous commit response")
		}
		return nil
	})
	c.record(Sample{Group: "g", Model: "m", Success: true, TTFTMs: common.GetPointer[int64](120), At: now})
	require.Error(t, c.flush(ctx, now.Add(time.Hour)))
	require.True(t, c.persistFailed.Load())
	require.Len(t, c.buckets, 1)
	fail = false
	concurrent = true
	require.NoError(t, c.flush(ctx, now.Add(time.Hour)))
	require.Len(t, c.buckets, 1, "sample received during save must stay pending")
	require.NoError(t, c.flush(ctx, now.Add(time.Hour)))
	require.Empty(t, c.buckets)
	require.False(t, c.persistFailed.Load())
	require.Len(t, stored, 1)
	for _, row := range stored {
		require.EqualValues(t, 2, row.RequestCount)
		require.EqualValues(t, 2, row.SuccessCount)
		require.EqualValues(t, 1, row.TTFTCount)
	}
}

func TestCollectorCapacityAndWeightedHourlySummary(t *testing.T) {
	c := newCollector(func(context.Context, []model.GroupHealthMetric) error { return nil })
	c.record(Sample{Group: "g", Model: strings.Repeat("x", 256), Success: true})
	require.Empty(t, c.buckets, "oversized names must not poison an entire batch")
	require.EqualValues(t, 1, c.dropped.Load())
	for i := 0; i < maxHotBuckets; i++ {
		c.buckets[bucketKey{ts: int64(i)}] = model.GroupHealthMetric{}
	}
	c.record(Sample{Group: "g", Model: "new", Success: true})
	require.EqualValues(t, 2, c.dropped.Load())
	require.Len(t, c.buckets, maxHotBuckets)
	start := int64(1800000000) / BucketSeconds * BucketSeconds
	result := buildResult([]string{"g", "empty"}, []model.GroupHealthMetric{
		{GroupName: "g", BucketTs: start, RequestCount: 1, SuccessCount: 0},
		{GroupName: "g", BucketTs: start + BucketSeconds, RequestCount: 99, SuccessCount: 99, TTFTSumMs: 19800, TTFTCount: 99},
		{GroupName: "private", BucketTs: start, RequestCount: 10000, SuccessCount: 10000},
	}, start, start+23*BucketSeconds+60)
	require.Len(t, result.Groups, 2)
	require.Len(t, result.Groups[0].Series, 24)
	require.Nil(t, result.Groups[0].SuccessRate)
	require.Equal(t, "g", result.Groups[1].Group)
	require.Equal(t, 99.0, *result.Groups[1].SuccessRate)
	require.Equal(t, 200.0, *result.Groups[1].AvgTTFTMs)
	require.EqualValues(t, 1, result.Groups[1].FailureCount)
}
