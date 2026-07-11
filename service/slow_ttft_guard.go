package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const (
	ginKeySlowTTFTModel          = "slow_ttft_model"
	ginKeySlowTTFTGroup          = "slow_ttft_group"
	ginKeySlowTTFTSetting        = "slow_ttft_setting_snapshot"
	ginKeySlowTTFTPrevious       = "slow_ttft_previous_response_id"
	ginKeySlowTTFTImageTool      = "slow_ttft_image_generation_tool"
	ginKeySlowTTFTTraceKey       = "slow_ttft_trace_key"
	ginKeySlowTTFTExcluded       = "slow_ttft_tag_excluded"
	ginKeySlowTTFTBypass         = "slow_ttft_soft_fallback"
	ginKeySlowTTFTLogInfo        = "slow_ttft_log_info"
	slowTTFTEvidenceBucketCount  = 8
	slowTTFTStatusCircuitListMax = 50
)

type slowTTFTDimension struct {
	Model  string
	Group  string
	Bucket string
}

type slowTTFTBaselineKey struct {
	slowTTFTDimension
	Tag string
}

type slowTTFTScope struct {
	Model string
	Group string
	Tag   string
}

type slowTTFTTraceScope struct {
	slowTTFTScope
	TraceKey string
}

type slowTTFTBaselineAggregate struct {
	Count int64
	SumMS float64
}

type slowTTFTBaseline struct {
	PeerMedianMS float64
	PeerTagCount int
}

type slowTTFTEvidenceBucket struct {
	SlotID int64
	Total  int64
	Slow   int64
	Users  map[int]struct{}
	Traces map[string]struct{}
}

type slowTTFTEvidenceState struct {
	Buckets           [slowTTFTEvidenceBucketCount]slowTTFTEvidenceBucket
	OpenUntil         time.Time
	LastSeen          time.Time
	LastTriggerTotal  int64
	LastTriggerSlow   int64
	LastTriggerUsers  int
	LastTriggerTraces int
}

type slowTTFTTraceState struct {
	ConsecutiveSlow int
	BlockedUntil    time.Time
	LastSeen        time.Time
}

type slowTTFTGuardState struct {
	mu sync.RWMutex

	windowStartedAt      time.Time
	lastRefreshAt        time.Time
	nextPruneAt          time.Time
	emptyBaselineWindows int

	pending   map[slowTTFTBaselineKey]slowTTFTBaselineAggregate
	baselines map[slowTTFTBaselineKey]slowTTFTBaseline
	evidence  map[slowTTFTScope]*slowTTFTEvidenceState
	traces    map[slowTTFTTraceScope]*slowTTFTTraceState

	droppedEntries int64
}

type slowTTFTSample struct {
	Model              string
	Group              string
	Tag                string
	Bucket             string
	TraceKey           string
	UserID             int
	LatencyMS          int64
	PreviousResponseID bool
}

type slowTTFTDecision struct {
	BaselineAvailable bool
	BaselineMS        float64
	BaselinePeerTags  int
	Slow              bool
	TraceBlocked      bool
	TraceWouldBlock   bool
	GlobalOpened      bool
	GlobalWouldOpen   bool
	GlobalOpenUntil   time.Time
}

type SlowTTFTGuardCircuitStatus struct {
	Model          string  `json:"model"`
	Group          string  `json:"group"`
	Tag            string  `json:"tag"`
	OpenUntil      int64   `json:"open_until"`
	TriggerTotal   int64   `json:"trigger_total"`
	TriggerSlow    int64   `json:"trigger_slow"`
	TriggerRate    float64 `json:"trigger_rate"`
	DistinctUsers  int     `json:"distinct_users"`
	DistinctTraces int     `json:"distinct_traces"`
}

type SlowTTFTGuardStats struct {
	Enabled                bool                         `json:"enabled"`
	ObserveOnly            bool                         `json:"observe_only"`
	LastBaselineRefreshAt  int64                        `json:"last_baseline_refresh_at"`
	NextBaselineRefreshAt  int64                        `json:"next_baseline_refresh_at"`
	PendingBaselineEntries int                          `json:"pending_baseline_entries"`
	BaselineEntries        int                          `json:"baseline_entries"`
	EvidenceEntries        int                          `json:"evidence_entries"`
	TraceEntries           int                          `json:"trace_entries"`
	OpenGlobalCircuits     int                          `json:"open_global_circuits"`
	ActiveTraceBlocks      int                          `json:"active_trace_blocks"`
	DroppedEntries         int64                        `json:"dropped_entries"`
	MaxEntries             int                          `json:"max_entries"`
	Circuits               []SlowTTFTGuardCircuitStatus `json:"circuits"`
}

var (
	slowTTFTGuard    = newSlowTTFTGuardState()
	slowTTFTTaskOnce sync.Once
)

func newSlowTTFTGuardState() *slowTTFTGuardState {
	return &slowTTFTGuardState{
		pending:   make(map[slowTTFTBaselineKey]slowTTFTBaselineAggregate),
		baselines: make(map[slowTTFTBaselineKey]slowTTFTBaseline),
		evidence:  make(map[slowTTFTScope]*slowTTFTEvidenceState),
		traces:    make(map[slowTTFTTraceScope]*slowTTFTTraceState),
	}
}

func StartSlowTTFTBaselineTask() {
	slowTTFTTaskOnce.Do(func() {
		go func() {
			for {
				setting := operation_setting.GetNormalizedSlowTTFTSetting()
				timer := time.NewTimer(time.Duration(setting.BaselineRefreshSeconds) * time.Second)
				<-timer.C
				if setting.Enabled {
					slowTTFTGuard.refreshBaselines(time.Now(), setting, false)
				}
			}
		}()
	})
}

func PrepareSlowTTFTRequest(c *gin.Context, modelName string, usingGroup string, previousResponseID bool, imageGenerationTool bool) {
	if c == nil {
		return
	}
	setting := operation_setting.GetNormalizedSlowTTFTSetting()
	c.Set(ginKeySlowTTFTSetting, setting)
	if !setting.Enabled {
		return
	}
	c.Set(ginKeySlowTTFTModel, strings.TrimSpace(modelName))
	c.Set(ginKeySlowTTFTGroup, strings.TrimSpace(usingGroup))
	c.Set(ginKeySlowTTFTPrevious, previousResponseID)
	c.Set(ginKeySlowTTFTImageTool, imageGenerationTool)
}

func IsSlowTTFTTagUnavailable(c *gin.Context, channel *model.Channel) bool {
	return isSlowTTFTTagUnavailableAt(c, channel, time.Now(), slowTTFTSettingForContext(c))
}

func isSlowTTFTTagUnavailableAt(c *gin.Context, channel *model.Channel, now time.Time, setting operation_setting.SlowTTFTSetting) bool {
	if c == nil || channel == nil || !setting.Enabled || setting.ObserveOnly || c.GetBool(ginKeySlowTTFTBypass) || c.GetBool(ginKeySlowTTFTPrevious) {
		return false
	}
	modelName := strings.TrimSpace(c.GetString(ginKeySlowTTFTModel))
	usingGroup := strings.TrimSpace(c.GetString(ginKeySlowTTFTGroup))
	tag := strings.TrimSpace(channel.GetTag())
	if modelName == "" || tag == "" {
		return false
	}

	scope := slowTTFTScope{Model: modelName, Group: usingGroup, Tag: tag}
	traceKey := slowTTFTTraceKey(c)
	reason := ""
	slowTTFTGuard.mu.RLock()
	if state := slowTTFTGuard.evidence[scope]; state != nil {
		if state.OpenUntil.After(now) {
			reason = "global"
		}
	}
	if reason == "" && traceKey != "" {
		traceScope := slowTTFTTraceScope{slowTTFTScope: scope, TraceKey: traceKey}
		if state := slowTTFTGuard.traces[traceScope]; state != nil {
			if state.BlockedUntil.After(now) {
				reason = "trace"
			}
		}
	}
	slowTTFTGuard.mu.RUnlock()
	if reason == "" {
		return false
	}
	recordSlowTTFTRouteExclusion(c, tag, reason)
	return true
}

func EnableSlowTTFTSoftFallback(c *gin.Context) bool {
	if c == nil || !c.GetBool(ginKeySlowTTFTExcluded) {
		return false
	}
	c.Set(ginKeySlowTTFTBypass, true)
	mergeSlowTTFTLogInfo(c, map[string]interface{}{"soft_fallback": true})
	return true
}

func ObserveSlowTTFT(c *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) {
	setting := slowTTFTSettingForContext(c)
	if !setting.Enabled {
		return
	}
	sample, ok := buildSlowTTFTSample(c, relayInfo, usage, setting)
	if !ok {
		return
	}
	decision := slowTTFTGuard.observe(sample, time.Now(), setting)
	info := map[string]interface{}{
		"tag":                       sample.Tag,
		"context_bucket":            sample.Bucket,
		"first_effective_output_ms": sample.LatencyMS,
		"previous_response_id":      sample.PreviousResponseID,
		"baseline_available":        decision.BaselineAvailable,
	}
	if decision.BaselineAvailable {
		info["peer_baseline_ms"] = decision.BaselineMS
		info["peer_tag_count"] = decision.BaselinePeerTags
		info["slow"] = decision.Slow
	}
	if decision.TraceBlocked || decision.TraceWouldBlock {
		info["trace_blocked"] = decision.TraceBlocked
		info["trace_would_block"] = decision.TraceWouldBlock
	}
	if decision.GlobalOpened || decision.GlobalWouldOpen {
		info["global_opened"] = decision.GlobalOpened
		info["global_would_open"] = decision.GlobalWouldOpen
	}
	if !decision.GlobalOpenUntil.IsZero() {
		info["global_open_until"] = decision.GlobalOpenUntil.Unix()
	}
	mergeSlowTTFTLogInfo(c, info)
}

func AppendSlowTTFTAdminInfo(c *gin.Context, adminInfo map[string]interface{}) {
	if c == nil || adminInfo == nil {
		return
	}
	if value, ok := c.Get(ginKeySlowTTFTLogInfo); ok && value != nil {
		adminInfo["slow_ttft_guard"] = value
	}
}

func GetSlowTTFTGuardStats() SlowTTFTGuardStats {
	setting := operation_setting.GetNormalizedSlowTTFTSetting()
	now := time.Now()
	slowTTFTGuard.mu.Lock()
	defer slowTTFTGuard.mu.Unlock()
	slowTTFTGuard.pruneLocked(now, setting)

	stats := SlowTTFTGuardStats{
		Enabled:                setting.Enabled,
		ObserveOnly:            setting.ObserveOnly,
		PendingBaselineEntries: len(slowTTFTGuard.pending),
		EvidenceEntries:        len(slowTTFTGuard.evidence),
		TraceEntries:           len(slowTTFTGuard.traces),
		DroppedEntries:         slowTTFTGuard.droppedEntries,
		MaxEntries:             setting.MaxEntries,
		Circuits:               make([]SlowTTFTGuardCircuitStatus, 0),
	}
	for _, baseline := range slowTTFTGuard.baselines {
		if baseline.PeerTagCount > 0 && baseline.PeerMedianMS > 0 {
			stats.BaselineEntries++
		}
	}
	if !slowTTFTGuard.lastRefreshAt.IsZero() {
		stats.LastBaselineRefreshAt = slowTTFTGuard.lastRefreshAt.Unix()
	}
	if !slowTTFTGuard.windowStartedAt.IsZero() {
		stats.NextBaselineRefreshAt = slowTTFTGuard.windowStartedAt.Add(time.Duration(setting.BaselineRefreshSeconds) * time.Second).Unix()
	}
	for scope, state := range slowTTFTGuard.evidence {
		if !state.OpenUntil.After(now) {
			continue
		}
		stats.OpenGlobalCircuits++
		if len(stats.Circuits) >= slowTTFTStatusCircuitListMax {
			continue
		}
		rate := 0.0
		if state.LastTriggerTotal > 0 {
			rate = float64(state.LastTriggerSlow) / float64(state.LastTriggerTotal)
		}
		stats.Circuits = append(stats.Circuits, SlowTTFTGuardCircuitStatus{
			Model:          scope.Model,
			Group:          scope.Group,
			Tag:            scope.Tag,
			OpenUntil:      state.OpenUntil.Unix(),
			TriggerTotal:   state.LastTriggerTotal,
			TriggerSlow:    state.LastTriggerSlow,
			TriggerRate:    rate,
			DistinctUsers:  state.LastTriggerUsers,
			DistinctTraces: state.LastTriggerTraces,
		})
	}
	for _, state := range slowTTFTGuard.traces {
		if state.BlockedUntil.After(now) {
			stats.ActiveTraceBlocks++
		}
	}
	sort.Slice(stats.Circuits, func(i, j int) bool {
		return stats.Circuits[i].OpenUntil < stats.Circuits[j].OpenUntil
	})
	return stats
}

func ClearSlowTTFTGuardState() {
	slowTTFTGuard.mu.Lock()
	slowTTFTGuard.pending = make(map[slowTTFTBaselineKey]slowTTFTBaselineAggregate)
	slowTTFTGuard.baselines = make(map[slowTTFTBaselineKey]slowTTFTBaseline)
	slowTTFTGuard.evidence = make(map[slowTTFTScope]*slowTTFTEvidenceState)
	slowTTFTGuard.traces = make(map[slowTTFTTraceScope]*slowTTFTTraceState)
	slowTTFTGuard.windowStartedAt = time.Time{}
	slowTTFTGuard.lastRefreshAt = time.Time{}
	slowTTFTGuard.nextPruneAt = time.Time{}
	slowTTFTGuard.emptyBaselineWindows = 0
	slowTTFTGuard.droppedEntries = 0
	slowTTFTGuard.mu.Unlock()
}

func RefreshSlowTTFTBaselines() SlowTTFTGuardStats {
	setting := operation_setting.GetNormalizedSlowTTFTSetting()
	slowTTFTGuard.refreshBaselines(time.Now(), setting, true)
	return GetSlowTTFTGuardStats()
}

func (guard *slowTTFTGuardState) observe(sample slowTTFTSample, now time.Time, setting operation_setting.SlowTTFTSetting) slowTTFTDecision {
	guard.mu.Lock()
	defer guard.mu.Unlock()
	guard.refreshBaselinesLocked(now, setting, false)
	guard.pruneLocked(now, setting)
	guard.addPendingBaselineLocked(sample, setting)

	decision := slowTTFTDecision{}
	baselineKey := slowTTFTBaselineKey{
		slowTTFTDimension: slowTTFTDimension{Model: sample.Model, Group: sample.Group, Bucket: sample.Bucket},
		Tag:               sample.Tag,
	}
	baseline, ok := guard.baselines[baselineKey]
	if !ok || baseline.PeerTagCount < setting.BaselineMinPeerTags || baseline.PeerMedianMS <= 0 {
		return decision
	}
	decision.BaselineAvailable = true
	decision.BaselineMS = baseline.PeerMedianMS
	decision.BaselinePeerTags = baseline.PeerTagCount
	decision.Slow = sample.LatencyMS >= int64(setting.ThresholdMS) && float64(sample.LatencyMS) >= baseline.PeerMedianMS*setting.BaselineMultiplier

	scope := slowTTFTScope{Model: sample.Model, Group: sample.Group, Tag: sample.Tag}
	evidence := guard.getOrCreateEvidenceLocked(scope, now, setting)
	if evidence != nil {
		if evidence.OpenUntil.After(now) {
			decision.GlobalOpenUntil = evidence.OpenUntil
		} else {
			guard.addEvidenceLocked(evidence, sample, decision.Slow, now, setting)
			total, slow, users, traces := aggregateSlowTTFTEvidence(evidence, now, setting)
			if total >= int64(setting.GlobalMinSamples) &&
				float64(slow)/float64(total) >= setting.GlobalSlowRate &&
				users >= setting.GlobalMinUsers && traces >= setting.GlobalMinTraces {
				decision.GlobalWouldOpen = true
				if !setting.ObserveOnly {
					evidence.OpenUntil = now.Add(time.Duration(setting.GlobalCircuitSeconds) * time.Second)
					evidence.LastTriggerTotal = total
					evidence.LastTriggerSlow = slow
					evidence.LastTriggerUsers = users
					evidence.LastTriggerTraces = traces
					clearSlowTTFTEvidenceBuckets(evidence)
					decision.GlobalOpened = true
					decision.GlobalOpenUntil = evidence.OpenUntil
					common.SysLog(fmt.Sprintf("slow ttft tag circuit opened: model=%s group=%s tag=%s until=%s", sample.Model, sample.Group, sample.Tag, evidence.OpenUntil.Format(time.RFC3339)))
				}
			}
		}
	}

	if sample.TraceKey == "" || sample.PreviousResponseID {
		return decision
	}
	traceScope := slowTTFTTraceScope{slowTTFTScope: scope, TraceKey: sample.TraceKey}
	trace := guard.getOrCreateTraceLocked(traceScope, now, setting)
	if trace == nil {
		return decision
	}
	trace.LastSeen = now
	if trace.BlockedUntil.After(now) {
		decision.TraceBlocked = true
		return decision
	}
	if !trace.BlockedUntil.IsZero() {
		trace.BlockedUntil = time.Time{}
		trace.ConsecutiveSlow = 0
	}
	if !decision.Slow {
		trace.ConsecutiveSlow = 0
		return decision
	}
	trace.ConsecutiveSlow++
	if trace.ConsecutiveSlow < setting.TraceConsecutiveSlow {
		return decision
	}
	decision.TraceWouldBlock = true
	trace.ConsecutiveSlow = 0
	if !setting.ObserveOnly {
		trace.BlockedUntil = now.Add(time.Duration(setting.TraceCircuitSeconds) * time.Second)
		decision.TraceBlocked = true
	}
	return decision
}

func (guard *slowTTFTGuardState) addPendingBaselineLocked(sample slowTTFTSample, setting operation_setting.SlowTTFTSetting) {
	key := slowTTFTBaselineKey{
		slowTTFTDimension: slowTTFTDimension{Model: sample.Model, Group: sample.Group, Bucket: sample.Bucket},
		Tag:               sample.Tag,
	}
	aggregate, ok := guard.pending[key]
	if !ok && len(guard.pending) >= slowTTFTPendingCapacity(setting.MaxEntries) {
		guard.droppedEntries++
		return
	}
	latency := sample.LatencyMS
	if latency > int64(setting.MaxSampleMS) {
		latency = int64(setting.MaxSampleMS)
	}
	aggregate.Count++
	aggregate.SumMS += float64(latency)
	guard.pending[key] = aggregate
}

func (guard *slowTTFTGuardState) refreshBaselines(now time.Time, setting operation_setting.SlowTTFTSetting, force bool) {
	guard.mu.Lock()
	guard.refreshBaselinesLocked(now, setting, force)
	guard.mu.Unlock()
}

func (guard *slowTTFTGuardState) refreshBaselinesLocked(now time.Time, setting operation_setting.SlowTTFTSetting, force bool) {
	if guard.windowStartedAt.IsZero() {
		guard.windowStartedAt = now
		if !force {
			return
		}
	}
	refreshAfter := time.Duration(setting.BaselineRefreshSeconds) * time.Second
	if !force && now.Sub(guard.windowStartedAt) > 2*refreshAfter {
		guard.pending = make(map[slowTTFTBaselineKey]slowTTFTBaselineAggregate)
		guard.baselines = make(map[slowTTFTBaselineKey]slowTTFTBaseline)
		guard.emptyBaselineWindows = 0
		guard.lastRefreshAt = now
		guard.windowStartedAt = now
		return
	}
	if !force && now.Sub(guard.windowStartedAt) < refreshAfter {
		return
	}
	if len(guard.pending) > 0 {
		guard.baselines = buildSlowTTFTBaselines(guard.pending, setting, slowTTFTBaselineCapacity(setting.MaxEntries))
		guard.emptyBaselineWindows = 0
	} else if !force {
		guard.emptyBaselineWindows++
		if guard.emptyBaselineWindows >= 2 {
			guard.baselines = make(map[slowTTFTBaselineKey]slowTTFTBaseline)
		}
	}
	guard.pending = make(map[slowTTFTBaselineKey]slowTTFTBaselineAggregate)
	guard.lastRefreshAt = now
	guard.windowStartedAt = now
}

type slowTTFTTagMean struct {
	Tag  string
	Mean float64
}

func buildSlowTTFTBaselines(pending map[slowTTFTBaselineKey]slowTTFTBaselineAggregate, setting operation_setting.SlowTTFTSetting, capacity int) map[slowTTFTBaselineKey]slowTTFTBaseline {
	allTags := make(map[slowTTFTDimension]map[string]struct{})
	valid := make(map[slowTTFTDimension][]slowTTFTTagMean)
	for key, aggregate := range pending {
		tags := allTags[key.slowTTFTDimension]
		if tags == nil {
			tags = make(map[string]struct{})
			allTags[key.slowTTFTDimension] = tags
		}
		tags[key.Tag] = struct{}{}
		if aggregate.Count < int64(setting.BaselineMinSamples) {
			continue
		}
		valid[key.slowTTFTDimension] = append(valid[key.slowTTFTDimension], slowTTFTTagMean{
			Tag:  key.Tag,
			Mean: aggregate.SumMS / float64(aggregate.Count),
		})
	}

	result := make(map[slowTTFTBaselineKey]slowTTFTBaseline)
	for dimension, tags := range allTags {
		means := valid[dimension]
		sort.Slice(means, func(i, j int) bool { return means[i].Mean < means[j].Mean })
		indexByTag := make(map[string]int, len(means))
		for index, mean := range means {
			indexByTag[mean.Tag] = index
		}
		for tag := range tags {
			peerCount := len(means)
			excludedIndex, isValidTag := indexByTag[tag]
			if isValidTag {
				peerCount--
			}
			if len(result) >= capacity {
				continue
			}
			if peerCount < setting.BaselineMinPeerTags {
				result[slowTTFTBaselineKey{slowTTFTDimension: dimension, Tag: tag}] = slowTTFTBaseline{}
				continue
			}
			median := medianSlowTTFTMeans(means)
			if isValidTag {
				median = medianSlowTTFTMeansExcluding(means, excludedIndex)
			}
			result[slowTTFTBaselineKey{slowTTFTDimension: dimension, Tag: tag}] = slowTTFTBaseline{
				PeerMedianMS: median,
				PeerTagCount: peerCount,
			}
		}
	}
	return result
}

func medianSlowTTFTMeans(values []slowTTFTTagMean) float64 {
	length := len(values)
	if length == 0 {
		return 0
	}
	middle := length / 2
	if length%2 == 1 {
		return values[middle].Mean
	}
	return (values[middle-1].Mean + values[middle].Mean) / 2
}

func medianSlowTTFTMeansExcluding(values []slowTTFTTagMean, excluded int) float64 {
	length := len(values) - 1
	if length <= 0 {
		return 0
	}
	valueAt := func(index int) float64 {
		if index >= excluded {
			index++
		}
		return values[index].Mean
	}
	middle := length / 2
	if length%2 == 1 {
		return valueAt(middle)
	}
	return (valueAt(middle-1) + valueAt(middle)) / 2
}

func (guard *slowTTFTGuardState) getOrCreateEvidenceLocked(scope slowTTFTScope, now time.Time, setting operation_setting.SlowTTFTSetting) *slowTTFTEvidenceState {
	if state := guard.evidence[scope]; state != nil {
		state.LastSeen = now
		return state
	}
	if len(guard.evidence) >= slowTTFTEvidenceCapacity(setting.MaxEntries) {
		guard.droppedEntries++
		return nil
	}
	state := &slowTTFTEvidenceState{LastSeen: now}
	guard.evidence[scope] = state
	return state
}

func (guard *slowTTFTGuardState) getOrCreateTraceLocked(scope slowTTFTTraceScope, now time.Time, setting operation_setting.SlowTTFTSetting) *slowTTFTTraceState {
	if state := guard.traces[scope]; state != nil {
		return state
	}
	if len(guard.traces) >= slowTTFTTraceCapacity(setting.MaxEntries) {
		guard.droppedEntries++
		return nil
	}
	state := &slowTTFTTraceState{LastSeen: now}
	guard.traces[scope] = state
	return state
}

func (guard *slowTTFTGuardState) addEvidenceLocked(state *slowTTFTEvidenceState, sample slowTTFTSample, slow bool, now time.Time, setting operation_setting.SlowTTFTSetting) {
	width := slowTTFTEvidenceBucketWidth(setting.EvidenceWindowSeconds)
	slotID := now.Unix() / width
	index := int(slotID % slowTTFTEvidenceBucketCount)
	bucket := &state.Buckets[index]
	if bucket.SlotID != slotID {
		*bucket = slowTTFTEvidenceBucket{SlotID: slotID}
	}
	bucket.Total++
	if !slow {
		return
	}
	bucket.Slow++
	if sample.UserID > 0 {
		if bucket.Users == nil {
			bucket.Users = make(map[int]struct{})
		}
		if len(bucket.Users) < setting.GlobalMinUsers {
			bucket.Users[sample.UserID] = struct{}{}
		}
	}
	if sample.TraceKey != "" {
		if bucket.Traces == nil {
			bucket.Traces = make(map[string]struct{})
		}
		if len(bucket.Traces) < setting.GlobalMinTraces {
			bucket.Traces[sample.TraceKey] = struct{}{}
		}
	}
}

func aggregateSlowTTFTEvidence(state *slowTTFTEvidenceState, now time.Time, setting operation_setting.SlowTTFTSetting) (int64, int64, int, int) {
	if state == nil {
		return 0, 0, 0, 0
	}
	width := slowTTFTEvidenceBucketWidth(setting.EvidenceWindowSeconds)
	currentSlot := now.Unix() / width
	oldestSlot := currentSlot - slowTTFTEvidenceBucketCount + 1
	var users [20]int
	var traces [100]string
	userCount := 0
	traceCount := 0
	userLimit := min(max(setting.GlobalMinUsers, 0), len(users))
	traceLimit := min(max(setting.GlobalMinTraces, 0), len(traces))
	var total int64
	var slow int64
	for index := range state.Buckets {
		bucket := &state.Buckets[index]
		if bucket.SlotID < oldestSlot || bucket.SlotID > currentSlot {
			continue
		}
		total += bucket.Total
		slow += bucket.Slow
		for userID := range bucket.Users {
			if userCount >= userLimit {
				break
			}
			if !containsSlowTTFTUser(users[:userCount], userID) {
				users[userCount] = userID
				userCount++
			}
		}
		for traceKey := range bucket.Traces {
			if traceCount >= traceLimit {
				break
			}
			if !containsSlowTTFTTrace(traces[:traceCount], traceKey) {
				traces[traceCount] = traceKey
				traceCount++
			}
		}
	}
	return total, slow, userCount, traceCount
}

func containsSlowTTFTUser(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsSlowTTFTTrace(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func clearSlowTTFTEvidenceBuckets(state *slowTTFTEvidenceState) {
	if state != nil {
		state.Buckets = [slowTTFTEvidenceBucketCount]slowTTFTEvidenceBucket{}
	}
}

func slowTTFTEvidenceBucketWidth(windowSeconds int) int64 {
	width := (windowSeconds + slowTTFTEvidenceBucketCount - 1) / slowTTFTEvidenceBucketCount
	if width < 1 {
		width = 1
	}
	return int64(width)
}

func (guard *slowTTFTGuardState) pruneLocked(now time.Time, setting operation_setting.SlowTTFTSetting) {
	if !guard.nextPruneAt.IsZero() && now.Before(guard.nextPruneAt) {
		return
	}
	staleAfter := time.Duration(max(setting.EvidenceWindowSeconds, setting.TraceCircuitSeconds)+300) * time.Second
	for scope, state := range guard.evidence {
		if !state.OpenUntil.After(now) && now.Sub(state.LastSeen) > staleAfter {
			delete(guard.evidence, scope)
		}
	}
	for scope, state := range guard.traces {
		if !state.BlockedUntil.After(now) && now.Sub(state.LastSeen) > staleAfter {
			delete(guard.traces, scope)
		}
	}
	guard.nextPruneAt = now.Add(5 * time.Minute)
}

func buildSlowTTFTSample(c *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, setting operation_setting.SlowTTFTSetting) (slowTTFTSample, bool) {
	if relayInfo == nil || usage == nil || !relayInfo.IsStream || relayInfo.IsChannelTest || relayInfo.StartTime.IsZero() {
		return slowTTFTSample{}, false
	}
	if c != nil && common.GetContextKeyBool(c, constant.ContextKeyResponsesAutoContinue) {
		return slowTTFTSample{}, false
	}
	if slowTTFTRequestHasImageGenerationTool(c, relayInfo) {
		return slowTTFTSample{}, false
	}
	if relayInfo.StreamStatus != nil && (relayInfo.StreamStatus.HasErrors() || !relayInfo.StreamStatus.IsNormalEnd()) {
		return slowTTFTSample{}, false
	}
	firstEffective := relayInfo.FirstEffectiveOutputTime
	if firstEffective.IsZero() || !firstEffective.After(relayInfo.StartTime) {
		return slowTTFTSample{}, false
	}
	modelName := strings.TrimSpace(relayInfo.OriginModelName)
	tag := strings.TrimSpace(relayInfo.FirstEffectiveOutputChannelTag)
	if tag == "" && relayInfo.ChannelMeta != nil {
		tag = strings.TrimSpace(relayInfo.ChannelTag)
	}
	if modelName == "" || tag == "" {
		return slowTTFTSample{}, false
	}
	promptTokens := usage.PromptTokens
	if promptTokens <= 0 {
		promptTokens = relayInfo.GetEstimatePromptTokens()
	}
	if promptTokens <= 0 {
		return slowTTFTSample{}, false
	}
	group := strings.TrimSpace(relayInfo.UsingGroup)
	if group == "" {
		group = strings.TrimSpace(relayInfo.TokenGroup)
	}
	latency := firstEffective.Sub(relayInfo.StartTime).Milliseconds()
	if latency <= 0 {
		return slowTTFTSample{}, false
	}
	return slowTTFTSample{
		Model:              modelName,
		Group:              group,
		Tag:                tag,
		Bucket:             slowTTFTContextBucket(promptTokens, setting.ContextBucketBoundaries),
		TraceKey:           slowTTFTTraceKey(c),
		UserID:             relayInfo.UserId,
		LatencyMS:          latency,
		PreviousResponseID: slowTTFTPreviousResponseID(c, relayInfo),
	}, true
}

func slowTTFTContextBucket(tokens int, boundaries []int) string {
	previous := 0
	for _, boundary := range boundaries {
		if tokens < boundary {
			if previous == 0 {
				return "<" + strconv.Itoa(boundary)
			}
			return strconv.Itoa(previous) + "-" + strconv.Itoa(boundary)
		}
		previous = boundary
	}
	return ">=" + strconv.Itoa(previous)
}

func slowTTFTTraceKey(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if value := strings.TrimSpace(c.GetString(ginKeySlowTTFTTraceKey)); value != "" {
		return value
	}
	meta, ok := getChannelAffinityMeta(c)
	if !ok || strings.TrimSpace(meta.CacheKey) == "" {
		return ""
	}
	value := common.GenerateHMAC(meta.CacheKey)
	c.Set(ginKeySlowTTFTTraceKey, value)
	return value
}

func slowTTFTSettingForContext(c *gin.Context) operation_setting.SlowTTFTSetting {
	if c != nil {
		if value, ok := c.Get(ginKeySlowTTFTSetting); ok {
			if setting, ok := value.(operation_setting.SlowTTFTSetting); ok {
				return setting
			}
		}
	}
	return operation_setting.GetNormalizedSlowTTFTSetting()
}

func slowTTFTPreviousResponseID(c *gin.Context, relayInfo *relaycommon.RelayInfo) bool {
	if c != nil && c.GetBool(ginKeySlowTTFTPrevious) {
		return true
	}
	if relayInfo == nil || relayInfo.Request == nil {
		return false
	}
	switch request := relayInfo.Request.(type) {
	case *dto.OpenAIResponsesRequest:
		return strings.TrimSpace(request.PreviousResponseID) != ""
	case *dto.OpenAIResponsesCompactionRequest:
		return strings.TrimSpace(request.PreviousResponseID) != ""
	default:
		return false
	}
}

func slowTTFTRequestHasImageGenerationTool(c *gin.Context, relayInfo *relaycommon.RelayInfo) bool {
	if c != nil {
		if value, exists := c.Get(ginKeySlowTTFTImageTool); exists {
			requested, _ := value.(bool)
			return requested
		}
	}
	request, ok := relayInfo.Request.(*dto.OpenAIResponsesRequest)
	return ok && request.HasImageGenerationTool()
}

func recordSlowTTFTRouteExclusion(c *gin.Context, tag string, reason string) {
	if c == nil {
		return
	}
	c.Set(ginKeySlowTTFTExcluded, true)
	info := getSlowTTFTLogInfo(c)
	excluded, _ := info["route_excluded_tags"].([]string)
	found := false
	for _, value := range excluded {
		if value == tag {
			found = true
			break
		}
	}
	if !found && len(excluded) < 8 {
		excluded = append(excluded, tag)
	}
	info["route_excluded_tags"] = excluded
	info["route_exclusion_reason"] = reason
	c.Set(ginKeySlowTTFTLogInfo, info)
}

func mergeSlowTTFTLogInfo(c *gin.Context, values map[string]interface{}) {
	if c == nil || len(values) == 0 {
		return
	}
	info := getSlowTTFTLogInfo(c)
	for key, value := range values {
		info[key] = value
	}
	c.Set(ginKeySlowTTFTLogInfo, info)
}

func getSlowTTFTLogInfo(c *gin.Context) map[string]interface{} {
	result := make(map[string]interface{})
	if c == nil {
		return result
	}
	if current, ok := c.Get(ginKeySlowTTFTLogInfo); ok {
		if values, ok := current.(map[string]interface{}); ok {
			for key, value := range values {
				result[key] = value
			}
		}
	}
	return result
}

func slowTTFTPendingCapacity(maxEntries int) int {
	return max(1, maxEntries/3)
}

func slowTTFTBaselineCapacity(maxEntries int) int {
	return max(1, maxEntries/3)
}

func slowTTFTEvidenceCapacity(maxEntries int) int {
	return max(1, maxEntries/6)
}

func slowTTFTTraceCapacity(maxEntries int) int {
	return max(1, maxEntries-slowTTFTPendingCapacity(maxEntries)-slowTTFTBaselineCapacity(maxEntries)-slowTTFTEvidenceCapacity(maxEntries))
}
