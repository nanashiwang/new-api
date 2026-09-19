package grouphealth

import (
	"net/http"
	"strings"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

// Request folds retries within one concrete group, not across fallback groups.
// It is owned by the relay request goroutine and flushed exactly once on return.
type Request struct {
	model        string
	clientStream bool
	groups       map[string]Sample
}

func NewRequest(modelName string, clientStream bool) *Request {
	return &Request{model: modelName, clientStream: clientStream, groups: make(map[string]Sample)}
}

func (r *Request) Failure(group string) {
	if r == nil || group == "" || group == "auto" {
		return
	}
	r.groups[group] = Sample{Group: group, Model: r.model}
}

func (r *Request) Observe(info *relaycommon.RelayInfo, success bool, now time.Time) {
	if r == nil || info == nil || info.UsingGroup == "" || info.UsingGroup == "auto" {
		return
	}
	if info.StreamStatus != nil && (info.StreamStatus.HasErrors() || info.StreamStatus.EndError != nil || !info.StreamStatus.IsNormalEnd()) {
		success = false
	}
	sample := Sample{Group: info.UsingGroup, Model: r.model, Success: success}
	if success && info.GroupHealthCacheUsage != nil {
		usage := *info.GroupHealthCacheUsage
		sample.CacheUsage = &usage
	}
	isImage := info.RelayMode == relayconstant.RelayModeImagesGenerations || info.RelayMode == relayconstant.RelayModeImagesEdits
	if success && !info.StartTime.IsZero() {
		if r.clientStream && !isImage && !info.GroupHealthFirstOutputTime.IsZero() &&
			!info.GroupHealthFirstOutputTime.Before(info.StartTime) && !info.GroupHealthFirstOutputTime.After(now) {
			value := info.GroupHealthFirstOutputTime.Sub(info.StartTime).Milliseconds()
			sample.TTFTMs = &value
		}
		if isImage && !info.IsStream && !now.Before(info.StartTime) {
			value := now.Sub(info.StartTime).Milliseconds()
			sample.CompletionMs = &value
		}
	}
	r.groups[info.UsingGroup] = sample
}

func (r *Request) Samples(now time.Time) []Sample {
	if r == nil {
		return nil
	}
	samples := make([]Sample, 0, len(r.groups))
	for _, sample := range r.groups {
		sample.At = now
		samples = append(samples, sample)
	}
	return samples
}

func (r *Request) Finish() {
	for _, sample := range r.Samples(time.Now()) {
		Record(sample)
	}
}

// SupportsRequest excludes asynchronous task submission/polling and realtime
// sessions. Their HTTP acceptance is not evidence of completed generation.
func SupportsRequest(method, path string) bool {
	if method != http.MethodPost {
		return false
	}
	switch path {
	case "/v1/chat/completions", "/v1/completions", "/v1/messages", "/v1/responses", "/v1/responses/compact",
		"/v1/images/generations", "/v1/images/edits", "/v1/edits", "/pg/chat/completions",
		"/v1/embeddings", "/v1/moderations", "/v1/rerank", "/v1/audio/speech", "/v1/audio/transcriptions", "/v1/audio/translations":
		return true
	}
	return (strings.HasPrefix(path, "/v1beta/models/") || strings.HasPrefix(path, "/v1/models/")) &&
		(strings.HasSuffix(path, ":generateContent") || strings.HasSuffix(path, ":streamGenerateContent") || strings.HasSuffix(path, ":embedContent") || strings.HasSuffix(path, ":batchEmbedContents"))
}

// ResetAttempt prevents the previous failed upstream's stream state and first
// output timestamp from contaminating a successful retry. Request start remains
// unchanged, so successful latency includes selection, queueing and retries.
func ResetAttempt(info *relaycommon.RelayInfo) {
	info.GroupHealthFirstOutputTime = time.Time{}
	info.StreamStatus = nil
	info.GroupHealthCacheUsage = nil
}
