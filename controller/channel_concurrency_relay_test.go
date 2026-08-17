package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestRemoveChannelsFromExclusion 验证从排除列表中精确移除“并发满”渠道，保留其它排除项。
func TestRemoveChannelsFromExclusion(t *testing.T) {
	// 空移除集：原样返回。
	assert.Equal(t, []int{1, 2, 3}, removeChannelsFromExclusion([]int{1, 2, 3}, map[int]bool{}))
	// 空排除列表：原样返回。
	assert.Empty(t, removeChannelsFromExclusion([]int{}, map[int]bool{1: true}))
	// 精确移除并发满渠道，保留其它（如失败重试排除的渠道）。
	result := removeChannelsFromExclusion([]int{1, 2, 3, 4}, map[int]bool{2: true, 4: true})
	assert.Equal(t, []int{1, 3}, result)
	// 全部移除。
	assert.Empty(t, removeChannelsFromExclusion([]int{5, 6}, map[int]bool{5: true, 6: true}))
}

func TestBuildChannelCapacityErrorRoundsRetryAfterUp(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	apiErr := buildChannelCapacityError(ctx, http.StatusTooManyRequests, "rpm limited", 1500*time.Millisecond)
	if assert.NotNil(t, apiErr) {
		assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
		assert.Equal(t, "2", recorder.Header().Get("Retry-After"))
		assert.Equal(t, 2*time.Second, apiErr.RetryAfter)
		assert.True(t, types.IsSkipRetryError(apiErr), "容量终态错误不应再次重试")
	}
}

func TestBuildBlockedChannelCapacityErrorOnlyRpmReturns429(t *testing.T) {
	setting := &operation_setting.ChannelConcurrencySetting{RetryAfterSeconds: 5}
	summary := channelCapacityBlockSummary{}
	summary.Record(middleware.ChannelAdmissionResult{Reason: middleware.ChannelAdmissionRpmLimited, RetryAfter: 2200 * time.Millisecond})
	summary.Record(middleware.ChannelAdmissionResult{Reason: middleware.ChannelAdmissionRpmLimited, RetryAfter: 1200 * time.Millisecond})

	apiErr := buildBlockedChannelCapacityError(nil, setting, summary, "all channels reached rpm limit")
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	assert.Equal(t, 2*time.Second, apiErr.RetryAfter)
}

func TestBuildBlockedChannelCapacityErrorMixedReturns503(t *testing.T) {
	setting := &operation_setting.ChannelConcurrencySetting{RetryAfterSeconds: 5}
	summary := channelCapacityBlockSummary{}
	summary.Record(middleware.ChannelAdmissionResult{Reason: middleware.ChannelAdmissionRpmLimited, RetryAfter: time.Second})
	summary.Record(middleware.ChannelAdmissionResult{Reason: middleware.ChannelAdmissionConcurrencyLimited})

	apiErr := buildBlockedChannelCapacityError(nil, setting, summary, "all channels reached capacity limit")
	assert.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
	assert.Equal(t, 5*time.Second, apiErr.RetryAfter)
}

// TestBuildClientCanceledError 验证客户端断开错误跳过重试且隐藏内部细节。
func TestBuildClientCanceledError(t *testing.T) {
	apiErr := buildClientCanceledError()
	if assert.NotNil(t, apiErr) {
		assert.True(t, types.IsSkipRetryError(apiErr), "客户端断开不应重试")
		assert.Equal(t, "client canceled while waiting for channel capacity", apiErr.Error())
	}
}

func TestCapacityPressureLessUsesHighestDimension(t *testing.T) {
	left := channelCapacityCandidate{
		channel:        &model.Channel{Id: 1},
		inflight:       1,
		maxConcurrency: 10,
		rpmUsed:        9,
		rpmLimit:       10,
	}
	right := channelCapacityCandidate{
		channel:        &model.Channel{Id: 2},
		inflight:       2,
		maxConcurrency: 10,
		rpmUsed:        3,
		rpmLimit:       10,
	}

	assert.False(t, capacityPressureLess(left, right), "90% RPM 压力不应排在 30% 压力之前")
	assert.True(t, capacityPressureLess(right, left), "30% 压力应优先于 90% RPM 压力")
}

func TestCapacityPressureIgnoresUnlimitedRpm(t *testing.T) {
	left := channelCapacityCandidate{inflight: 1, maxConcurrency: 10, rpmUsed: 1000, rpmLimit: 0}
	right := channelCapacityCandidate{inflight: 2, maxConcurrency: 10, rpmUsed: 0, rpmLimit: 10}
	assert.True(t, capacityPressureLess(left, right), "RPM 不限时只比较并发压力")
}

func TestRankChannelsByCapacityPressurePrefersLowerNormalizedLoad(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = originalRedisEnabled })

	settingA := `{"max_concurrency":2}`
	settingB := `{"max_concurrency":10}`
	channelA := &model.Channel{Id: 991001, Setting: &settingA}
	channelB := &model.Channel{Id: 991002, Setting: &settingB}

	releaseA, ok := middleware.TryAcquireChannelSlot(channelA.Id, 2)
	assert.True(t, ok)
	releaseB1, ok := middleware.TryAcquireChannelSlot(channelB.Id, 10)
	assert.True(t, ok)
	releaseB2, ok := middleware.TryAcquireChannelSlot(channelB.Id, 10)
	assert.True(t, ok)
	t.Cleanup(func() {
		releaseA()
		releaseB1()
		releaseB2()
	})

	ranked := rankChannelsByCapacityPressure([]*model.Channel{channelA, channelB})
	if assert.Len(t, ranked, 2) {
		assert.Equal(t, channelB.Id, ranked[0].Id, "20% 负载渠道应优先于 50% 负载渠道")
	}
}

func TestRankChannelsByCapacityPressureIncludesRpm(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = originalRedisEnabled })
	t.Setenv("CHANNEL_RPM_WINDOW_SECONDS", "")

	setting := operation_setting.GetChannelConcurrencySetting()
	originalWindow := setting.RpmWindowSeconds
	setting.RpmWindowSeconds = 60
	t.Cleanup(func() { setting.RpmWindowSeconds = originalWindow })

	settingA := `{"max_concurrency":10,"rpm_limit":10}`
	settingB := `{"max_concurrency":10,"rpm_limit":10}`
	channelA := &model.Channel{Id: 991011, Setting: &settingA}
	channelB := &model.Channel{Id: 991012, Setting: &settingB}

	for i := 0; i < 8; i++ {
		admission := middleware.TryAcquireChannelCapacity(channelA.Id, 10, 10, time.Minute, "rpm-a")
		assert.True(t, admission.Acquired)
		admission.Release()
	}
	for i := 0; i < 2; i++ {
		admission := middleware.TryAcquireChannelCapacity(channelB.Id, 10, 10, time.Minute, "rpm-b")
		assert.True(t, admission.Acquired)
		admission.Release()
	}

	ranked := rankChannelsByCapacityPressure([]*model.Channel{channelA, channelB})
	if assert.Len(t, ranked, 2) {
		assert.Equal(t, channelB.Id, ranked[0].Id, "20% RPM 压力渠道应优先于 80% RPM 压力渠道")
	}
}
