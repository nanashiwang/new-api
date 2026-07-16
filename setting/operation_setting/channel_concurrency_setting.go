package operation_setting

import (
	"fmt"
	"os"
	"strconv"

	"github.com/QuantumNous/new-api/setting/config"
)

// ChannelConcurrencySetting 渠道并发控制（含 Layer C 过载处理）的全局配置。
//
// 语义：给每个上游渠道设一个在途并发上限。请求发往某渠道前占用一个名额、结束后归还。
// 当被选中的渠道已满时会自动换到其它未满的渠道；当所有候选渠道都满时，进入有界等待
// （Layer C）——短暂等待容量释放，客户端断开立即取消，超时仍无容量则返回 503 + Retry-After。
//
// 默认 Enabled=false：不改变任何现状，必须显式开启才生效。
type ChannelConcurrencySetting struct {
	// 是否启用渠道并发控制。false 时所有逻辑短路，行为与未引入本功能完全一致。
	Enabled bool `json:"enabled"`
	// 单渠道默认在途并发上限。渠道可通过 channel.setting.max_concurrency 覆盖（>0 时）。
	DefaultMaxConcurrency int `json:"default_max_concurrency"`
	// Layer C 有界等待的总超时（毫秒）。所有渠道满时，最多等这么久，超时返回 503。
	WaitTimeoutMs int `json:"wait_timeout_ms"`
	// 全局同时处于 Layer C 等待状态的请求数上限。超过则新请求快速失败（503），
	// 防止过载时等待请求无界堆积。
	MaxQueueLength int `json:"max_queue_length"`
	// 等待期间重新探测容量的轮询间隔（毫秒）。多实例部署下靠短轮询重试 acquire。
	PollIntervalMs int `json:"poll_interval_ms"`
	// 返回 503 时 Retry-After 响应头的秒数，指导客户端退避重试。
	RetryAfterSeconds int `json:"retry_after_seconds"`
}

// 默认配置：关闭状态 + 一组安全默认值。
var channelConcurrencySetting = ChannelConcurrencySetting{
	Enabled:               false,
	DefaultMaxConcurrency: 100,
	WaitTimeoutMs:         3000,
	MaxQueueLength:        200,
	PollIntervalMs:        50,
	RetryAfterSeconds:     3,
}

func init() {
	// 注册到全局配置管理器：OptionMap 自动同步、前端点号 key 自动读写，无需改 model/option.go。
	config.GlobalConfig.Register("channel_concurrency_setting", &channelConcurrencySetting)
}

// GetChannelConcurrencySetting 返回当前配置（含 env 兜底覆盖）。
// env（若设置）在读取时覆写内存值，实现 “env 提供初始值 + 后台可热调” 双通道。
func GetChannelConcurrencySetting() *ChannelConcurrencySetting {
	if v := os.Getenv("CHANNEL_CONCURRENCY_ENABLED"); v != "" {
		channelConcurrencySetting.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("CHANNEL_CONCURRENCY_DEFAULT_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			channelConcurrencySetting.DefaultMaxConcurrency = n
		}
	}
	return &channelConcurrencySetting
}

// Normalized* 系列：对可能被后台改成非法值的字段做兜底，保证运行时永远拿到合理值。

func (s *ChannelConcurrencySetting) NormalizedDefaultMaxConcurrency() int {
	if s.DefaultMaxConcurrency <= 0 {
		return 100
	}
	return s.DefaultMaxConcurrency
}

func (s *ChannelConcurrencySetting) NormalizedWaitTimeoutMs() int {
	if s.WaitTimeoutMs <= 0 {
		return 3000
	}
	return s.WaitTimeoutMs
}

func (s *ChannelConcurrencySetting) NormalizedMaxQueueLength() int {
	if s.MaxQueueLength <= 0 {
		return 200
	}
	return s.MaxQueueLength
}

func (s *ChannelConcurrencySetting) NormalizedPollIntervalMs() int {
	// 下限 10ms，避免过载时轮询过密打爆 Redis / CPU；上限不超过总等待时长。
	if s.PollIntervalMs < 10 {
		return 50
	}
	return s.PollIntervalMs
}

func (s *ChannelConcurrencySetting) NormalizedRetryAfterSeconds() int {
	if s.RetryAfterSeconds <= 0 {
		return 3
	}
	return s.RetryAfterSeconds
}

// ValidateChannelConcurrencyOption 校验后台写入的 channel_concurrency_setting.* 配置值，
// 供 controller/option.go 在保存前调用。数值字段必须为非负整数；enabled 为 bool 不校验。
func ValidateChannelConcurrencyOption(key, value string) error {
	switch key {
	case "channel_concurrency_setting.enabled":
		return nil
	case "channel_concurrency_setting.default_max_concurrency",
		"channel_concurrency_setting.wait_timeout_ms",
		"channel_concurrency_setting.max_queue_length",
		"channel_concurrency_setting.poll_interval_ms",
		"channel_concurrency_setting.retry_after_seconds":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("%s 必须是非负整数", key)
		}
		return nil
	}
	return nil
}
