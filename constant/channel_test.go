package constant

import "testing"

func TestResolveChannelSpecialBaseKimiAliases(t *testing.T) {
	want := ChannelSpecialBases["kimi-coding-plan"]
	for _, baseURL := range []string{
		"kimi-coding-plan",
		"kimi-coding-plan/",
		"https://api.kimi.com/coding",
		"https://api.kimi.com/coding/",
		"https://api.kimi.com/coding/v1",
		"https://api.kimi.com/coding/v1/",
	} {
		t.Run(baseURL, func(t *testing.T) {
			got, ok := ResolveChannelSpecialBase(baseURL)
			if !ok {
				t.Fatalf("expected %q to resolve", baseURL)
			}
			if got != want {
				t.Fatalf("unexpected plan for %q: got %+v, want %+v", baseURL, got, want)
			}
		})
	}
}

func TestResolveChannelSpecialBaseDoesNotMatchUnknownURL(t *testing.T) {
	if _, ok := ResolveChannelSpecialBase("https://api.moonshot.cn"); ok {
		t.Fatal("expected regular Moonshot URL to keep its existing routing")
	}
}
