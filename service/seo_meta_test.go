package service

import (
	"bytes"
	"testing"
)

const sampleIndex = `<!doctype html>
<html><head>
<title>New API</title>
<meta name="description" content="old desc" />
<meta property="og:title" content="old og title" />
<meta property="og:description" content="old og desc" />
<meta property="og:url" content="https://old.example.com/" />
<meta name="twitter:title" content="old tw" />
<meta name="twitter:description" content="old tw desc" />
<link rel="canonical" href="https://old.example.com/" />
</head><body></body></html>`

func TestRenderIndexWithMetaUnknownPathReturnsOriginal(t *testing.T) {
	out := RenderIndexWithMeta([]byte(sampleIndex), "/some/random")
	if !bytes.Equal(out, []byte(sampleIndex)) {
		t.Fatalf("unknown path should return template unchanged")
	}
}

func TestRenderIndexWithMetaRootInjectsTitleAndCanonical(t *testing.T) {
	out := RenderIndexWithMeta([]byte(sampleIndex), "/")
	if !bytes.Contains(out, []byte("AI 多渠道聚合中转网关")) {
		t.Fatalf("title not injected: %s", out)
	}
	if !bytes.Contains(out, []byte(`href="https://cn.meta-api.vip/"`)) {
		t.Fatalf("canonical not injected: %s", out)
	}
	if !bytes.Contains(out, []byte(`property="og:url" content="https://cn.meta-api.vip/"`)) {
		t.Fatalf("og:url not injected: %s", out)
	}
	if bytes.Contains(out, []byte("old og title")) {
		t.Fatalf("og:title was not replaced")
	}
}

func TestRenderIndexWithMetaImagePlaygroundUsesProductMeta(t *testing.T) {
	out := RenderIndexWithMeta([]byte(sampleIndex), "/image-playground/")
	if !bytes.Contains(out, []byte("GPT Image Playground")) {
		t.Fatalf("image-playground title not injected: %s", out)
	}
	if !bytes.Contains(out, []byte("href=\"https://cn.meta-api.vip/image-playground/\"")) {
		t.Fatalf("image-playground canonical not injected: %s", out)
	}
}

func TestRenderIndexWithMetaTitleAndCanonicalAlsoWorkForNoTrailingSlashPath(t *testing.T) {
	out := RenderIndexWithMeta([]byte(sampleIndex), "/image-playground")
	if !bytes.Contains(out, []byte("GPT Image Playground")) {
		t.Fatalf("title not injected for non-trailing slash path: %s", out)
	}
}

func TestRenderIndexWithMetaEscapesHTMLSpecialChars(t *testing.T) {
	const tmpl = `<title>x</title><meta name="description" content="x" />`
	// 临时注入一个带特殊字符的 meta（模拟 description 含 < > & "）。
	old := routeMetaMap["/login"]
	routeMetaMap["/login"] = routeMeta{
		Title:       `T<&">`,
		Description: `D<&">`,
		Canonical:   `https://x.example.com/`,
		OGURL:       `https://x.example.com/`,
	}
	defer func() { routeMetaMap["/login"] = old }()

	out := RenderIndexWithMeta([]byte(tmpl), "/login")
	if !bytes.Contains(out, []byte("T&lt;&amp;&quot;&gt;")) {
		t.Fatalf("title not escaped: %s", out)
	}
	if !bytes.Contains(out, []byte("D&lt;&amp;&quot;&gt;")) {
		t.Fatalf("description not escaped: %s", out)
	}
}

func TestGetPrerenderedHTMLEmptyByDefault(t *testing.T) {
	// 测试环境未调 LoadPrerendered，应返回 nil。
	if got := GetPrerenderedHTML("/"); got != nil {
		t.Fatalf("default should return nil, got %d bytes", len(got))
	}
	if got := GetPrerenderedHTML("/login"); got != nil {
		t.Fatalf("default should return nil for /login, got %d bytes", len(got))
	}
}
