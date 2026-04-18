package ratio_setting

import "testing"

func TestClaudeOpus47DefaultRatios(t *testing.T) {
	t.Parallel()
	InitRatioSettings()

	for _, model := range []string{"claude-opus-4-7", "claude-opus-4-7-xhigh"} {
		ratio, ok, _ := GetModelRatio(model)
		if !ok {
			t.Fatalf("%s missing from model ratios", model)
		}
		if ratio <= 0 {
			t.Fatalf("%s ratio must be positive, got %v", model, ratio)
		}
	}

	for _, model := range []string{"claude-opus-4-7", "claude-opus-4-7-thinking", "claude-opus-4-7-xhigh"} {
		cacheRatio, ok := GetCacheRatio(model)
		if !ok {
			t.Fatalf("%s missing from cache ratios", model)
		}
		if cacheRatio <= 0 {
			t.Fatalf("%s cache ratio must be positive, got %v", model, cacheRatio)
		}

		createCacheRatio, ok := GetCreateCacheRatio(model)
		if !ok {
			t.Fatalf("%s missing from create-cache ratios", model)
		}
		if createCacheRatio <= 0 {
			t.Fatalf("%s create-cache ratio must be positive, got %v", model, createCacheRatio)
		}
	}
}
