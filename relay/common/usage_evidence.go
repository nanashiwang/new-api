package common

// UsageCountEvidence contains only bounded accounting metadata. It must never
// include response text, prompts, credentials or arbitrary provider field names.
type UsageCountEvidence struct {
	Status                 string `json:"status"`
	Source                 string `json:"source,omitempty"`
	Value                  *int   `json:"value,omitempty"`
	RestoredFromPriorChunk bool   `json:"restored_from_prior_chunk,omitempty"`
}

type KimiUsageEvidence struct {
	Cache     UsageCountEvidence `json:"cache"`
	Reasoning UsageCountEvidence `json:"reasoning"`
}
