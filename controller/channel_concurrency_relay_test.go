package controller

import (
	"testing"

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
