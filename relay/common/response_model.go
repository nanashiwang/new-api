package common

import (
	"strings"
	"time"
)

// ResponseModel records upstream declarations before response conversion. It is
// diagnostic only: it must never change routing, pricing, or downstream output.
// Store names, not a verdict: historical rows must use the current comparison.
type ResponseModel struct {
	RequestedModel string `json:"requested_model"`
	UpstreamModel  string `json:"upstream_model"`
	ReturnedModel  string `json:"returned_model"`
}

// matches permits case/whitespace differences, a provider path on an unqualified
// name, and a real YYYY-MM-DD or YYYYMMDD snapshot date. Arbitrary variants
// (mini/nano/pro) and substring matches must remain visible for investigation.
// Keep this rule in sync with web/src/helpers/responseModel.js and shared tests.
func (r *ResponseModel) matches(model string) bool {
	returned := strings.ToLower(strings.TrimSpace(model))
	for _, name := range []string{r.RequestedModel, r.UpstreamModel} {
		expected := strings.ToLower(strings.TrimSpace(name))
		if expected == "" {
			continue
		}
		candidate := returned
		if !strings.Contains(expected, "/") && strings.Contains(candidate, "/") {
			parts := strings.Split(candidate, "/")
			validPath := true
			for _, part := range parts {
				if strings.TrimSpace(part) == "" || part == "." || part == ".." {
					validPath = false
					break
				}
			}
			if !validPath {
				continue
			}
			candidate = parts[len(parts)-1]
		}
		if candidate == expected {
			return true
		}
		if suffix, ok := strings.CutPrefix(candidate, expected+"-"); ok {
			for _, layout := range []string{"2006-01-02", "20060102"} {
				if parsed, err := time.Parse(layout, suffix); err == nil && parsed.Format(layout) == suffix {
					return true
				}
			}
		}
	}
	return false
}

// Mismatch is recomputed even for old JSON carrying an obsolete mismatch flag.
func (r *ResponseModel) Mismatch() bool {
	return r != nil && strings.TrimSpace(r.ReturnedModel) != "" && !r.matches(r.ReturnedModel)
}

// ObserveResponseModel retains the first differing model for inspection, with
// mismatches taking priority over compatible naming differences. A later
// matching or empty event cannot erase it. Only observe upstream declarations,
// never models synthesized by a response converter.
func (info *RelayInfo) ObserveResponseModel(model string) {
	if info == nil || strings.TrimSpace(model) == "" {
		return
	}
	if info.ResponseModel == nil {
		info.ResponseModel = &ResponseModel{
			RequestedModel: info.OriginModelName,
			UpstreamModel:  info.OriginModelName,
		}
	}
	if info.ChannelMeta != nil && info.ResponseModel.ReturnedModel == "" {
		info.ResponseModel.UpstreamModel = info.UpstreamModelName
	}
	observation := info.ResponseModel
	if observation.Mismatch() {
		return
	}
	if observation.matches(model) && observation.ReturnedModel != "" &&
		observation.ReturnedModel != observation.RequestedModel && observation.ReturnedModel != observation.UpstreamModel {
		return
	}
	observation.ReturnedModel = model
}
