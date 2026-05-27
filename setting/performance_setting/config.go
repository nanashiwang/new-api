package performance_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

// PerformanceSetting 性能设置配置
type PerformanceSetting struct {
	// DiskCacheEnabled 是否启用磁盘缓存（磁盘换内存）
	DiskCacheEnabled bool `json:"disk_cache_enabled"`
	// DiskCacheThresholdMB 触发磁盘缓存的请求体大小阈值（MB）
	DiskCacheThresholdMB int `json:"disk_cache_threshold_mb"`
	// DiskCacheMaxSizeMB 磁盘缓存最大总大小（MB）
	DiskCacheMaxSizeMB int `json:"disk_cache_max_size_mb"`
	// DiskCachePath 磁盘缓存目录
	DiskCachePath string `json:"disk_cache_path"`

	// MonitorEnabled 是否启用性能监控
	MonitorEnabled bool `json:"monitor_enabled"`
	// MonitorCPUThreshold CPU 使用率阈值（%）
	MonitorCPUThreshold int `json:"monitor_cpu_threshold"`
	// MonitorMemoryThreshold 内存使用率阈值（%）
	MonitorMemoryThreshold int `json:"monitor_memory_threshold"`
	// MonitorDiskThreshold 磁盘使用率阈值（%）
	MonitorDiskThreshold int `json:"monitor_disk_threshold"`

	// RelayResponseHeaderTimeoutSec 普通 relay HTTP 客户端等待上游响应头的超时（秒）
	RelayResponseHeaderTimeoutSec int `json:"relay_response_header_timeout_sec"`
	// RelayImageResponseHeaderTimeoutSec image relay HTTP 客户端等待上游响应头的超时（秒），生图多张耗时较长时应调大
	RelayImageResponseHeaderTimeoutSec int `json:"relay_image_response_header_timeout_sec"`
	// ImagePlaygroundPreferredOrigin 用于覆盖 image-playground 启动 URL 与 BaseURL 的 origin，便于绕开 Cloudflare 等带短超时的前置 CDN
	ImagePlaygroundPreferredOrigin string `json:"image_playground_preferred_origin"`
}

// 默认配置
var performanceSetting = PerformanceSetting{
	DiskCacheEnabled:     false,
	DiskCacheThresholdMB: 10,   // 超过 10MB 使用磁盘缓存
	DiskCacheMaxSizeMB:   1024, // 最大 1GB 磁盘缓存
	DiskCachePath:        "",   // 空表示使用系统临时目录

	MonitorEnabled:         true,
	MonitorCPUThreshold:    90,
	MonitorMemoryThreshold: 90,
	MonitorDiskThreshold:   90,

	RelayResponseHeaderTimeoutSec:      60,
	RelayImageResponseHeaderTimeoutSec: 300,
	ImagePlaygroundPreferredOrigin:     "",
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("performance_setting", &performanceSetting)
	// 同步初始配置到 common 包
	syncToCommon()
}

// syncToCommon 将配置同步到 common 包
func syncToCommon() {
	common.SetDiskCacheConfig(common.DiskCacheConfig{
		Enabled:     performanceSetting.DiskCacheEnabled,
		ThresholdMB: performanceSetting.DiskCacheThresholdMB,
		MaxSizeMB:   performanceSetting.DiskCacheMaxSizeMB,
		Path:        performanceSetting.DiskCachePath,
	})

	common.SetPerformanceMonitorConfig(common.PerformanceMonitorConfig{
		Enabled:         performanceSetting.MonitorEnabled,
		CPUThreshold:    performanceSetting.MonitorCPUThreshold,
		MemoryThreshold: performanceSetting.MonitorMemoryThreshold,
		DiskThreshold:   performanceSetting.MonitorDiskThreshold,
	})

	// 中转网关超时与 image-playground 首选 origin
	common.RelayResponseHeaderTimeout = performanceSetting.RelayResponseHeaderTimeoutSec
	common.RelayImageResponseHeaderTimeout = performanceSetting.RelayImageResponseHeaderTimeoutSec
	common.ImagePlaygroundPreferredOrigin = strings.TrimSpace(performanceSetting.ImagePlaygroundPreferredOrigin)

	// 通过 hook 触发 service 包重建 relay HTTP 客户端，使新超时立即对新请求生效（旧请求继续使用旧 client）。
	// 启动期 service 尚未注册时 hook 为 nil，由 service.InitHttpClient 自行完成首次初始化。
	if common.ReloadRelayHTTPClients != nil {
		common.ReloadRelayHTTPClients()
	}
}

// GetPerformanceSetting 获取性能设置
func GetPerformanceSetting() *PerformanceSetting {
	return &performanceSetting
}

// UpdateAndSync 更新配置并同步到 common 包
// 当配置从数据库加载后，需要调用此函数同步
func UpdateAndSync() {
	syncToCommon()
}

// GetCacheStats 获取缓存统计信息（代理到 common 包）
func GetCacheStats() common.DiskCacheStats {
	return common.GetDiskCacheStats()
}

// ResetStats 重置统计信息
func ResetStats() {
	common.ResetDiskCacheStats()
}
