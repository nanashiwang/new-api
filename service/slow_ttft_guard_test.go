package service

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func slowTTFTTestSetting() operation_setting.SlowTTFTSetting {
	return operation_setting.NormalizeSlowTTFTSetting(operation_setting.SlowTTFTSetting{
		Enabled:                 true,
		ThresholdMS:             1000,
		BaselineMultiplier:      2,
		MaxSampleMS:             120000,
		BaselineRefreshSeconds:  3600,
		BaselineMinSamples:      2,
		BaselineMinPeerTags:     1,
		EvidenceWindowSeconds:   900,
		GlobalMinSamples:        12,
		GlobalSlowRate:          0.6,
		GlobalMinUsers:          3,
		GlobalMinTraces:         5,
		GlobalCircuitSeconds:    300,
		TraceConsecutiveSlow:    3,
		TraceCircuitSeconds:     1800,
		MaxEntries:              10000,
		ContextBucketBoundaries: []int{50000, 100000, 150000, 200000},
	})
}

func seedSlowTTFTBaseline(guard *slowTTFTGuardState, setting operation_setting.SlowTTFTSetting, now time.Time) {
	for index := 0; index < setting.BaselineMinSamples; index++ {
		guard.observe(slowTTFTSample{
			Model: "gpt-5.5", Group: "Pro", Tag: "fast", Bucket: ">=200000", LatencyMS: 200,
		}, now.Add(time.Duration(index)*time.Millisecond), setting)
		guard.observe(slowTTFTSample{
			Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000", LatencyMS: 3000,
		}, now.Add(time.Duration(index)*time.Millisecond), setting)
	}
	guard.refreshBaselines(now.Add(time.Second), setting, true)
}

func slowTTFTTestContext(traceKey string, previous bool) *gin.Context {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(ginKeySlowTTFTModel, "gpt-5.5")
	ctx.Set(ginKeySlowTTFTGroup, "Pro")
	ctx.Set(ginKeySlowTTFTPrevious, previous)
	setChannelAffinityContext(ctx, channelAffinityMeta{
		CacheKey:       "test:" + traceKey,
		TTLSeconds:     3600,
		RuleName:       "codex trace",
		KeyFingerprint: traceKey,
	})
	return ctx
}

func slowTTFTTestChannel(tag string) *model.Channel {
	channel := &model.Channel{}
	channel.SetTag(tag)
	return channel
}

func useSlowTTFTGuardForTest(t *testing.T, guard *slowTTFTGuardState) {
	t.Helper()
	old := slowTTFTGuard
	slowTTFTGuard = guard
	t.Cleanup(func() { slowTTFTGuard = old })
}

func TestSlowTTFTTraceBlockAndPreviousResponseProtection(t *testing.T) {
	setting := slowTTFTTestSetting()
	setting.GlobalMinSamples = 100
	now := time.Unix(1000, 0)
	guard := newSlowTTFTGuardState()
	useSlowTTFTGuardForTest(t, guard)
	seedSlowTTFTBaseline(guard, setting, now)

	trace := "trace-a"
	traceHash := common.GenerateHMAC("test:" + trace)
	for index := 0; index < setting.TraceConsecutiveSlow; index++ {
		decision := guard.observe(slowTTFTSample{
			Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
			TraceKey: traceHash, UserID: 1, LatencyMS: 3000,
		}, now.Add(time.Duration(index+2)*time.Second), setting)
		if index == setting.TraceConsecutiveSlow-1 {
			require.True(t, decision.TraceBlocked)
		}
	}

	ctx := slowTTFTTestContext(trace, false)
	require.True(t, isSlowTTFTTagUnavailableAt(ctx, slowTTFTTestChannel("slow"), now.Add(10*time.Second), setting))

	previousCtx := slowTTFTTestContext(trace, true)
	require.False(t, isSlowTTFTTagUnavailableAt(previousCtx, slowTTFTTestChannel("slow"), now.Add(10*time.Second), setting))
}

func TestSlowTTFTPreviousResponseSamplesDoNotBlockLaterStatelessRequest(t *testing.T) {
	setting := slowTTFTTestSetting()
	setting.GlobalMinSamples = 100
	now := time.Unix(1500, 0)
	guard := newSlowTTFTGuardState()
	useSlowTTFTGuardForTest(t, guard)
	seedSlowTTFTBaseline(guard, setting, now)
	trace := common.GenerateHMAC("test:trace-a")

	for index := 0; index < setting.TraceConsecutiveSlow+2; index++ {
		guard.observe(slowTTFTSample{
			Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
			TraceKey: trace, UserID: 1, LatencyMS: 3000, PreviousResponseID: true,
		}, now.Add(time.Duration(index+2)*time.Second), setting)
	}

	ctx := slowTTFTTestContext("trace-a", false)
	require.False(t, isSlowTTFTTagUnavailableAt(ctx, slowTTFTTestChannel("slow"), now.Add(20*time.Second), setting))
}

func TestSlowTTFTFastSampleResetsTraceSequence(t *testing.T) {
	setting := slowTTFTTestSetting()
	setting.GlobalMinSamples = 100
	now := time.Unix(1700, 0)
	guard := newSlowTTFTGuardState()
	seedSlowTTFTBaseline(guard, setting, now)
	trace := "trace-a"

	for _, latency := range []int64{3000, 3000, 500, 3000, 3000} {
		decision := guard.observe(slowTTFTSample{
			Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
			TraceKey: trace, UserID: 1, LatencyMS: latency,
		}, now.Add(2*time.Second), setting)
		require.False(t, decision.TraceBlocked)
	}
}

func TestSlowTTFTGlobalCircuitRequiresDistinctUsersAndTraces(t *testing.T) {
	setting := slowTTFTTestSetting()
	setting.TraceConsecutiveSlow = 20
	now := time.Unix(2000, 0)
	guard := newSlowTTFTGuardState()
	useSlowTTFTGuardForTest(t, guard)
	seedSlowTTFTBaseline(guard, setting, now)

	for index := 0; index < setting.GlobalMinSamples; index++ {
		guard.observe(slowTTFTSample{
			Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
			TraceKey:  fmt.Sprintf("trace-%d", index%setting.GlobalMinTraces),
			UserID:    index%setting.GlobalMinUsers + 1,
			LatencyMS: 3000,
		}, now.Add(time.Duration(index+2)*time.Second), setting)
	}

	ctx := slowTTFTTestContext("unrelated-trace", false)
	require.True(t, isSlowTTFTTagUnavailableAt(ctx, slowTTFTTestChannel("slow"), now.Add(30*time.Second), setting))

	guard = newSlowTTFTGuardState()
	useSlowTTFTGuardForTest(t, guard)
	seedSlowTTFTBaseline(guard, setting, now)
	for index := 0; index < setting.GlobalMinSamples; index++ {
		guard.observe(slowTTFTSample{
			Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
			TraceKey: fmt.Sprintf("single-user-trace-%d", index), UserID: 1, LatencyMS: 3000,
		}, now.Add(time.Duration(index+2)*time.Second), setting)
	}
	require.False(t, isSlowTTFTTagUnavailableAt(ctx, slowTTFTTestChannel("slow"), now.Add(30*time.Second), setting))
}

func TestSlowTTFTGlobalCircuitRequiresConfiguredSlowRate(t *testing.T) {
	setting := slowTTFTTestSetting()
	setting.TraceConsecutiveSlow = 20
	now := time.Unix(2500, 0)
	guard := newSlowTTFTGuardState()
	useSlowTTFTGuardForTest(t, guard)
	seedSlowTTFTBaseline(guard, setting, now)

	for index := 0; index < 12; index++ {
		latency := int64(500)
		if index < 7 {
			latency = 3000
		}
		guard.observe(slowTTFTSample{
			Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
			TraceKey: fmt.Sprintf("trace-%d", index), UserID: index%3 + 1, LatencyMS: latency,
		}, now.Add(time.Duration(index+2)*time.Second), setting)
	}

	ctx := slowTTFTTestContext("unrelated-trace", false)
	require.False(t, isSlowTTFTTagUnavailableAt(ctx, slowTTFTTestChannel("slow"), now.Add(20*time.Second), setting))
	guard.observe(slowTTFTSample{
		Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
		TraceKey: "trace-extra", UserID: 1, LatencyMS: 3000,
	}, now.Add(21*time.Second), setting)
	require.True(t, isSlowTTFTTagUnavailableAt(ctx, slowTTFTTestChannel("slow"), now.Add(22*time.Second), setting))
}

func TestSlowTTFTSoftFallbackNeverRemovesLastAvailableTag(t *testing.T) {
	setting := slowTTFTTestSetting()
	now := time.Unix(3000, 0)
	guard := newSlowTTFTGuardState()
	useSlowTTFTGuardForTest(t, guard)
	ctx := slowTTFTTestContext("trace-a", false)
	scope := slowTTFTScope{Model: "gpt-5.5", Group: "Pro", Tag: "only-tag"}
	guard.evidence[scope] = &slowTTFTEvidenceState{OpenUntil: now.Add(time.Minute), LastSeen: now}

	channel := slowTTFTTestChannel("only-tag")
	require.True(t, isSlowTTFTTagUnavailableAt(ctx, channel, now, setting))
	require.True(t, EnableSlowTTFTSoftFallback(ctx))
	require.False(t, isSlowTTFTTagUnavailableAt(ctx, channel, now, setting))
}

func TestSlowTTFTObserveOnlyAndExpiryDoNotExcludeTag(t *testing.T) {
	setting := slowTTFTTestSetting()
	now := time.Unix(4000, 0)
	guard := newSlowTTFTGuardState()
	useSlowTTFTGuardForTest(t, guard)
	ctx := slowTTFTTestContext("trace-a", false)
	scope := slowTTFTScope{Model: "gpt-5.5", Group: "Pro", Tag: "slow"}
	guard.evidence[scope] = &slowTTFTEvidenceState{OpenUntil: now.Add(time.Minute), LastSeen: now}

	setting.ObserveOnly = true
	require.False(t, isSlowTTFTTagUnavailableAt(ctx, slowTTFTTestChannel("slow"), now, setting))
	setting.ObserveOnly = false
	require.False(t, isSlowTTFTTagUnavailableAt(ctx, slowTTFTTestChannel("slow"), now.Add(2*time.Minute), setting))
	setting.Enabled = false
	require.False(t, isSlowTTFTTagUnavailableAt(ctx, slowTTFTTestChannel("slow"), now, setting))
}

func TestSlowTTFTMissingPeerBaselineOnlyObserves(t *testing.T) {
	setting := slowTTFTTestSetting()
	now := time.Unix(5000, 0)
	guard := newSlowTTFTGuardState()
	for index := 0; index < setting.BaselineMinSamples; index++ {
		guard.observe(slowTTFTSample{
			Model: "gpt-5.5", Group: "Pro", Tag: "only", Bucket: ">=200000", LatencyMS: 3000,
		}, now, setting)
	}
	guard.refreshBaselines(now.Add(time.Second), setting, true)
	decision := guard.observe(slowTTFTSample{
		Model: "gpt-5.5", Group: "Pro", Tag: "only", Bucket: ">=200000",
		TraceKey: "trace-a", UserID: 1, LatencyMS: 3000,
	}, now.Add(2*time.Second), setting)

	require.False(t, decision.BaselineAvailable)
	require.Empty(t, guard.evidence)
	require.Empty(t, guard.traces)
}

func TestSlowTTFTNewTagWaitsForNextBaselineRefresh(t *testing.T) {
	setting := slowTTFTTestSetting()
	now := time.Unix(5500, 0)
	guard := newSlowTTFTGuardState()
	seedSlowTTFTBaseline(guard, setting, now)

	decision := guard.observe(slowTTFTSample{
		Model: "gpt-5.5", Group: "Pro", Tag: "new-tag", Bucket: ">=200000",
		TraceKey: "trace-a", UserID: 1, LatencyMS: 3000,
	}, now.Add(2*time.Second), setting)

	require.False(t, decision.BaselineAvailable)
	require.NotContains(t, guard.evidence, slowTTFTScope{Model: "gpt-5.5", Group: "Pro", Tag: "new-tag"})
}

func TestSlowTTFTDropsStaleBaselineAfterLongIdlePeriod(t *testing.T) {
	setting := slowTTFTTestSetting()
	now := time.Unix(5800, 0)
	guard := newSlowTTFTGuardState()
	seedSlowTTFTBaseline(guard, setting, now)

	decision := guard.observe(slowTTFTSample{
		Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
		TraceKey: "trace-a", UserID: 1, LatencyMS: 3000,
	}, now.Add(3*time.Hour), setting)

	require.False(t, decision.BaselineAvailable)
	require.Empty(t, guard.baselines)
}

func TestSlowTTFTStateCapacityIsBounded(t *testing.T) {
	setting := slowTTFTTestSetting()
	setting.MaxEntries = 100
	guard := newSlowTTFTGuardState()
	now := time.Unix(6000, 0)
	for index := 0; index < 200; index++ {
		guard.observe(slowTTFTSample{
			Model: fmt.Sprintf("model-%d", index), Group: "Pro", Tag: "tag", Bucket: "<50000", LatencyMS: 100,
		}, now, setting)
	}

	require.LessOrEqual(t, len(guard.pending), slowTTFTPendingCapacity(setting.MaxEntries))
	require.Greater(t, guard.droppedEntries, int64(0))
}

func TestSlowTTFTConcurrentObservation(t *testing.T) {
	setting := slowTTFTTestSetting()
	setting.GlobalMinSamples = 10000
	now := time.Unix(7000, 0)
	guard := newSlowTTFTGuardState()
	seedSlowTTFTBaseline(guard, setting, now)

	var waitGroup sync.WaitGroup
	for index := 0; index < 100; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			guard.observe(slowTTFTSample{
				Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
				TraceKey: fmt.Sprintf("trace-%d", index), UserID: index + 1, LatencyMS: 3000,
			}, now.Add(2*time.Second), setting)
		}(index)
	}
	waitGroup.Wait()
	require.NotEmpty(t, guard.evidence)
}

func TestBuildSlowTTFTSampleRejectsInvalidMeasurements(t *testing.T) {
	setting := slowTTFTTestSetting()
	start := time.Now().Add(-time.Second)
	normalStatus := relaycommon.NewStreamStatus()
	normalStatus.SetEndReason(relaycommon.StreamEndReasonEOF, nil)
	base := &relaycommon.RelayInfo{
		UserId:                         1,
		UsingGroup:                     "Pro",
		StartTime:                      start,
		FirstEffectiveOutputTime:       start.Add(500 * time.Millisecond),
		FirstEffectiveOutputChannelTag: "tag-a",
		IsStream:                       true,
		OriginModelName:                "gpt-5.5",
		StreamStatus:                   normalStatus,
	}
	usage := &dto.Usage{PromptTokens: 200001}

	_, ok := buildSlowTTFTSample(nil, base, usage, setting)
	require.True(t, ok)

	base.IsStream = false
	_, ok = buildSlowTTFTSample(nil, base, usage, setting)
	require.False(t, ok)
	base.IsStream = true

	base.FirstEffectiveOutputTime = time.Time{}
	_, ok = buildSlowTTFTSample(nil, base, usage, setting)
	require.False(t, ok)
	base.FirstEffectiveOutputTime = start.Add(500 * time.Millisecond)

	base.FirstEffectiveOutputChannelTag = ""
	_, ok = buildSlowTTFTSample(nil, base, usage, setting)
	require.False(t, ok)

	base.FirstEffectiveOutputChannelTag = "tag-a"
	base.Request = &dto.OpenAIResponsesRequest{Tools: []byte(`[{"type":"image_generation"}]`)}
	_, ok = buildSlowTTFTSample(nil, base, usage, setting)
	require.False(t, ok)

	base.Request = nil
	ctx := slowTTFTTestContext("trace-a", false)
	common.SetContextKey(ctx, constant.ContextKeyResponsesAutoContinue, true)
	_, ok = buildSlowTTFTSample(ctx, base, usage, setting)
	require.False(t, ok)
}

func BenchmarkSlowTTFTObserve(b *testing.B) {
	setting := slowTTFTTestSetting()
	setting.GlobalMinSamples = 1 << 30
	setting.TraceConsecutiveSlow = 1 << 30
	guard := newSlowTTFTGuardState()
	now := time.Unix(8000, 0)
	seedSlowTTFTBaseline(guard, setting, now)
	sample := slowTTFTSample{
		Model: "gpt-5.5", Group: "Pro", Tag: "slow", Bucket: ">=200000",
		TraceKey: "trace-a", UserID: 1, LatencyMS: 3000,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		guard.observe(sample, now.Add(2*time.Second), setting)
	}
}
