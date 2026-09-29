package dto

// NormalizeKimiOutputLimit maps the Chat alias to K3's max_tokens field. It
// follows GetMaxTokens precedence, never raises a completion limit, and leaves
// unknown models untouched. It must only run on structured Chat requests.
func (r *GeneralOpenAIRequest) NormalizeKimiOutputLimit() {
	if r == nil || (r.Model != "kimi-k3" && r.Model != "k3") || r.MaxCompletionTokens == 0 {
		return
	}
	r.MaxTokens = r.GetMaxTokens()
	r.MaxCompletionTokens = 0
}
