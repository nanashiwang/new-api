package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

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

// TestBuildChannelOverload503 验证过载 503 错误的状态码与跳过重试属性
// （已在 Layer C 等过，不应被 controller 再次重试）。
func TestBuildChannelOverload503(t *testing.T) {
	setting := &operation_setting.ChannelConcurrencySetting{RetryAfterSeconds: 5}
	apiErr := buildChannelOverload503(nil, setting, "all channels reached concurrency limit")
	if assert.NotNil(t, apiErr) {
		assert.Equal(t, 503, apiErr.StatusCode)
		assert.True(t, types.IsSkipRetryError(apiErr), "过载 503 已等待过，不应再被重试")
	}
}

// TestBuildClientCanceledError 验证客户端断开错误跳过重试且隐藏内部细节。
func TestBuildClientCanceledError(t *testing.T) {
	apiErr := buildClientCanceledError()
	if assert.NotNil(t, apiErr) {
		assert.True(t, types.IsSkipRetryError(apiErr), "客户端断开不应重试")
		assert.Equal(t, "client canceled while waiting for channel capacity", apiErr.Error())
	}
}

func TestConcurrencyPressureLessUsesNormalizedLoad(t *testing.T) {
	left := channelConcurrencyCandidate{
		channel:        &model.Channel{Id: 1},
		inflight:       1,
		maxConcurrency: 2,
	}
	right := channelConcurrencyCandidate{
		channel:        &model.Channel{Id: 2},
		inflight:       2,
		maxConcurrency: 10,
	}

	assert.False(t, concurrencyPressureLess(left, right), "50% 负载不应排在 20% 负载之前")
	assert.True(t, concurrencyPressureLess(right, left), "20% 负载应优先于 50% 负载")
}

func TestRankChannelsByConcurrencyPressurePrefersLowerNormalizedLoad(t *testing.T) {
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

	ranked := rankChannelsByConcurrencyPressure([]*model.Channel{channelA, channelB})
	if assert.Len(t, ranked, 2) {
		assert.Equal(t, channelB.Id, ranked[0].Id, "20% 负载渠道应优先于 50% 负载渠道")
	}
}
