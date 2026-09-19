package dto

// Optional modality counts are subsets of cached_tokens, not extra tokens.
// Pointers distinguish missing data from an authoritative explicit zero.
type CachedTokenDetails struct {
	TextTokens  *int `json:"text_tokens,omitempty"`
	ImageTokens *int `json:"image_tokens,omitempty"`
	AudioTokens *int `json:"audio_tokens,omitempty"`
}

func cloneTokenCount(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func (d *CachedTokenDetails) Clone() *CachedTokenDetails {
	if d == nil {
		return nil
	}
	return &CachedTokenDetails{TextTokens: cloneTokenCount(d.TextTokens), ImageTokens: cloneTokenCount(d.ImageTokens), AudioTokens: cloneTokenCount(d.AudioTokens)}
}

func (d InputTokenDetails) Clone() InputTokenDetails {
	d.CachedTokensDetails = d.CachedTokensDetails.Clone()
	return d
}

// Add aggregates distinct requests (e.g. automatic continuation). It must not
// be used for cumulative usage snapshots from successive events of one request.
func (d *InputTokenDetails) Add(incoming InputTokenDetails) {
	d.CachedTokens += incoming.CachedTokens
	d.CachedCreationTokens += incoming.CachedCreationTokens
	d.TextTokens += incoming.TextTokens
	d.ImageTokens += incoming.ImageTokens
	d.AudioTokens += incoming.AudioTokens
	d.CachedTokensDetails = d.CachedTokensDetails.Clone()
	if incoming.CachedTokensDetails == nil {
		return
	}
	if d.CachedTokensDetails == nil {
		d.CachedTokensDetails = &CachedTokenDetails{}
	}
	add := func(dst **int, src *int) {
		if src == nil {
			return
		}
		value := *src
		if *dst != nil {
			value += **dst
		}
		*dst = &value
	}
	add(&d.CachedTokensDetails.TextTokens, incoming.CachedTokensDetails.TextTokens)
	add(&d.CachedTokensDetails.ImageTokens, incoming.CachedTokensDetails.ImageTokens)
	add(&d.CachedTokensDetails.AudioTokens, incoming.CachedTokensDetails.AudioTokens)
}
