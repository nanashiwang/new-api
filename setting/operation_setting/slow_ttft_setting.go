package operation_setting

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

const (
	defaultSlowTTFTThresholdMS            = 8000
	defaultSlowTTFTBaselineMultiplier     = 3.0
	defaultSlowTTFTMaxSampleMS            = 120000
	defaultSlowTTFTBaselineRefreshSeconds = 3600
	defaultSlowTTFTBaselineMinSamples     = 6
	defaultSlowTTFTBaselineMinPeerTags    = 1
	defaultSlowTTFTEvidenceWindowSeconds  = 900
	defaultSlowTTFTGlobalMinSamples       = 12
	defaultSlowTTFTGlobalSlowRate         = 0.6
	defaultSlowTTFTGlobalMinUsers         = 3
	defaultSlowTTFTGlobalMinTraces        = 5
	defaultSlowTTFTGlobalCircuitSeconds   = 300
	defaultSlowTTFTTraceConsecutiveSlow   = 3
	defaultSlowTTFTTraceCircuitSeconds    = 1800
	defaultSlowTTFTMaxEntries             = 10000
)

var defaultSlowTTFTContextBuckets = []int{50000, 100000, 150000, 200000}

type SlowTTFTSetting struct {
	Enabled                 bool    `json:"enabled"`
	ObserveOnly             bool    `json:"observe_only"`
	ThresholdMS             int     `json:"threshold_ms"`
	BaselineMultiplier      float64 `json:"baseline_multiplier"`
	MaxSampleMS             int     `json:"max_sample_ms"`
	BaselineRefreshSeconds  int     `json:"baseline_refresh_seconds"`
	BaselineMinSamples      int     `json:"baseline_min_samples"`
	BaselineMinPeerTags     int     `json:"baseline_min_peer_tags"`
	EvidenceWindowSeconds   int     `json:"evidence_window_seconds"`
	GlobalMinSamples        int     `json:"global_min_samples"`
	GlobalSlowRate          float64 `json:"global_slow_rate"`
	GlobalMinUsers          int     `json:"global_min_users"`
	GlobalMinTraces         int     `json:"global_min_traces"`
	GlobalCircuitSeconds    int     `json:"global_circuit_seconds"`
	TraceConsecutiveSlow    int     `json:"trace_consecutive_slow"`
	TraceCircuitSeconds     int     `json:"trace_circuit_seconds"`
	MaxEntries              int     `json:"max_entries"`
	ContextBucketBoundaries []int   `json:"context_bucket_boundaries"`
}

var slowTTFTSetting = SlowTTFTSetting{
	Enabled:                 true,
	ObserveOnly:             true,
	ThresholdMS:             defaultSlowTTFTThresholdMS,
	BaselineMultiplier:      defaultSlowTTFTBaselineMultiplier,
	MaxSampleMS:             defaultSlowTTFTMaxSampleMS,
	BaselineRefreshSeconds:  defaultSlowTTFTBaselineRefreshSeconds,
	BaselineMinSamples:      defaultSlowTTFTBaselineMinSamples,
	BaselineMinPeerTags:     defaultSlowTTFTBaselineMinPeerTags,
	EvidenceWindowSeconds:   defaultSlowTTFTEvidenceWindowSeconds,
	GlobalMinSamples:        defaultSlowTTFTGlobalMinSamples,
	GlobalSlowRate:          defaultSlowTTFTGlobalSlowRate,
	GlobalMinUsers:          defaultSlowTTFTGlobalMinUsers,
	GlobalMinTraces:         defaultSlowTTFTGlobalMinTraces,
	GlobalCircuitSeconds:    defaultSlowTTFTGlobalCircuitSeconds,
	TraceConsecutiveSlow:    defaultSlowTTFTTraceConsecutiveSlow,
	TraceCircuitSeconds:     defaultSlowTTFTTraceCircuitSeconds,
	MaxEntries:              defaultSlowTTFTMaxEntries,
	ContextBucketBoundaries: append([]int(nil), defaultSlowTTFTContextBuckets...),
}

func init() {
	config.GlobalConfig.Register("slow_ttft_setting", &slowTTFTSetting)
}

func GetSlowTTFTSetting() *SlowTTFTSetting {
	return &slowTTFTSetting
}

func GetNormalizedSlowTTFTSetting() SlowTTFTSetting {
	return NormalizeSlowTTFTSetting(slowTTFTSetting)
}

func NormalizeSlowTTFTSetting(value SlowTTFTSetting) SlowTTFTSetting {
	value.ObserveOnly = true
	if value.ThresholdMS < 1000 || value.ThresholdMS > 120000 {
		value.ThresholdMS = defaultSlowTTFTThresholdMS
	}
	if math.IsNaN(value.BaselineMultiplier) || math.IsInf(value.BaselineMultiplier, 0) || value.BaselineMultiplier < 1 || value.BaselineMultiplier > 20 {
		value.BaselineMultiplier = defaultSlowTTFTBaselineMultiplier
	}
	if value.MaxSampleMS < 1000 || value.MaxSampleMS > 600000 {
		value.MaxSampleMS = defaultSlowTTFTMaxSampleMS
	}
	if value.BaselineRefreshSeconds < 3600 || value.BaselineRefreshSeconds > 86400 {
		value.BaselineRefreshSeconds = defaultSlowTTFTBaselineRefreshSeconds
	}
	if value.BaselineMinSamples < 1 || value.BaselineMinSamples > 10000 {
		value.BaselineMinSamples = defaultSlowTTFTBaselineMinSamples
	}
	if value.BaselineMinPeerTags < 1 || value.BaselineMinPeerTags > 50 {
		value.BaselineMinPeerTags = defaultSlowTTFTBaselineMinPeerTags
	}
	if value.EvidenceWindowSeconds < 60 || value.EvidenceWindowSeconds > 3600 {
		value.EvidenceWindowSeconds = defaultSlowTTFTEvidenceWindowSeconds
	}
	if value.GlobalMinSamples < 1 || value.GlobalMinSamples > 10000 {
		value.GlobalMinSamples = defaultSlowTTFTGlobalMinSamples
	}
	if math.IsNaN(value.GlobalSlowRate) || math.IsInf(value.GlobalSlowRate, 0) || value.GlobalSlowRate < 0.01 || value.GlobalSlowRate > 1 {
		value.GlobalSlowRate = defaultSlowTTFTGlobalSlowRate
	}
	if value.GlobalMinUsers < 2 || value.GlobalMinUsers > 20 {
		value.GlobalMinUsers = defaultSlowTTFTGlobalMinUsers
	}
	if value.GlobalMinTraces < 1 || value.GlobalMinTraces > 100 {
		value.GlobalMinTraces = defaultSlowTTFTGlobalMinTraces
	}
	if value.GlobalCircuitSeconds < 30 || value.GlobalCircuitSeconds > 86400 {
		value.GlobalCircuitSeconds = defaultSlowTTFTGlobalCircuitSeconds
	}
	if value.TraceConsecutiveSlow < 1 || value.TraceConsecutiveSlow > 20 {
		value.TraceConsecutiveSlow = defaultSlowTTFTTraceConsecutiveSlow
	}
	if value.TraceCircuitSeconds < 30 || value.TraceCircuitSeconds > 86400 {
		value.TraceCircuitSeconds = defaultSlowTTFTTraceCircuitSeconds
	}
	if value.MaxEntries < 100 || value.MaxEntries > 100000 {
		value.MaxEntries = defaultSlowTTFTMaxEntries
	}
	value.ContextBucketBoundaries = normalizeSlowTTFTBuckets(value.ContextBucketBoundaries)
	return value
}

func ValidateSlowTTFTOption(key string, raw string) error {
	if !strings.HasPrefix(key, "slow_ttft_setting.") {
		return nil
	}
	field := strings.TrimPrefix(key, "slow_ttft_setting.")
	if field == "enabled" || field == "observe_only" {
		if _, err := strconv.ParseBool(raw); err != nil {
			return fmt.Errorf("慢首字保护开关必须是 true 或 false")
		}
		return nil
	}
	if field == "baseline_multiplier" || field == "global_slow_rate" {
		value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return fmt.Errorf("慢首字保护配置必须是有效数字")
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("慢首字保护配置必须是有限数字")
		}
		if field == "baseline_multiplier" && (value < 1 || value > 20) {
			return fmt.Errorf("同行基线倍数必须在 1-20 之间")
		}
		if field == "global_slow_rate" && (value < 0.01 || value > 1) {
			return fmt.Errorf("全局慢请求比例必须在 0.01-1 之间")
		}
		return nil
	}
	if field == "context_bucket_boundaries" {
		var buckets []int
		if err := common.UnmarshalJsonStr(raw, &buckets); err != nil {
			return fmt.Errorf("上下文分桶边界必须是整数 JSON 数组")
		}
		if !validSlowTTFTBuckets(buckets) {
			return fmt.Errorf("上下文分桶边界需包含 1-10 个递增正整数，且不能超过 1000000")
		}
		return nil
	}

	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("慢首字保护配置必须是整数")
	}
	ranges := map[string][2]int{
		"threshold_ms":             {1000, 120000},
		"max_sample_ms":            {1000, 600000},
		"baseline_refresh_seconds": {3600, 86400},
		"baseline_min_samples":     {1, 10000},
		"baseline_min_peer_tags":   {1, 50},
		"evidence_window_seconds":  {60, 3600},
		"global_min_samples":       {1, 10000},
		"global_min_users":         {2, 20},
		"global_min_traces":        {1, 100},
		"global_circuit_seconds":   {30, 86400},
		"trace_consecutive_slow":   {1, 20},
		"trace_circuit_seconds":    {30, 86400},
		"max_entries":              {100, 100000},
	}
	allowed, ok := ranges[field]
	if !ok {
		return fmt.Errorf("未知的慢首字保护配置项")
	}
	if value < allowed[0] || value > allowed[1] {
		return fmt.Errorf("慢首字保护配置 %s 必须在 %d-%d 之间", field, allowed[0], allowed[1])
	}
	return nil
}

func normalizeSlowTTFTBuckets(values []int) []int {
	if !validSlowTTFTBuckets(values) {
		return defaultSlowTTFTContextBuckets
	}
	return values
}

func validSlowTTFTBuckets(values []int) bool {
	if len(values) < 1 || len(values) > 10 {
		return false
	}
	previous := 0
	for _, value := range values {
		if value <= previous || value > 1000000 {
			return false
		}
		previous = value
	}
	return true
}
