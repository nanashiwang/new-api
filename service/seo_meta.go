package service

import (
	"bytes"
	"strings"
)

// seo_meta.go 负责给 SPA 的 indexPage 按当前请求路径注入差异化的 SEO meta（title/description/canonical/og）。
// 这是 prerender（阶段 2）落地前的过渡方案：即便没有真预渲染产物，搜索引擎抓 HTML 时也能拿到差异化标签。
// 一旦 prerender 上线，GetPrerenderedHTML 优先返回静态 HTML，本文件的模板替换链路自动降级为兜底。

// routeMeta 描述单个公开路由的 SEO 元数据。
// 字段语义：
//   - Title       完整 <title>
//   - Description <meta name="description"> 与 og:description
//   - Canonical   <link rel="canonical"> 的绝对 URL
//   - OGURL       og:url（通常与 Canonical 相同）
type routeMeta struct {
	Title       string
	Description string
	Canonical   string
	OGURL       string
}

const seoSiteOrigin = "https://cn.meta-api.vip"

var imagePlaygroundMeta = routeMeta{
	Title:       "GPT Image Playground · AI 电商套图与图像编辑工作台 | gpt-image-2 在线生成",
	Description: "GPT Image Playground 是基于 gpt-image-2 的在线 AI 图像生成与编辑工作台，支持电商套图一键生成、行业起步包、个人模板库、批量出图与多平台尺寸预设。",
	Canonical:   seoSiteOrigin + "/image-playground/",
	OGURL:       seoSiteOrigin + "/image-playground/",
}

// routeMetaMap 公开路由 → 元数据映射。
// key 必须是请求的 URL.Path 精确字符串（注意结尾斜杠是否存在）。
var routeMetaMap = map[string]routeMeta{
	"/": {
		Title:       "New API · AI 多渠道聚合中转网关 | OpenAI / Claude / Gemini 一站式代理",
		Description: "New API 是开源的 AI 大模型聚合中转网关，统一对接 OpenAI、Claude、Gemini、DeepSeek、Qwen、Azure、Midjourney 等 30+ 渠道，提供多账户调度、计费统计、Web 管理台。Docker 一键部署。",
		Canonical:   seoSiteOrigin + "/",
		OGURL:       seoSiteOrigin + "/",
	},
	"/login": {
		Title:       "登录 · New API AI 中转网关",
		Description: "登录 New API 控制台，管理你的 AI API Key、查看用量与计费、配置多账户调度策略。",
		Canonical:   seoSiteOrigin + "/login",
		OGURL:       seoSiteOrigin + "/login",
	},
	"/register": {
		Title:       "注册 · New API AI 中转网关",
		Description: "注册 New API 账户，立即获取统一 AI API Key，一套密钥调用 OpenAI / Claude / Gemini 等 30+ 渠道。",
		Canonical:   seoSiteOrigin + "/register",
		OGURL:       seoSiteOrigin + "/register",
	},
	"/pricing": {
		Title:       "模型价格 · OpenAI / Claude / Gemini 中转报价 | New API",
		Description: "New API 提供 OpenAI、Claude、Gemini、DeepSeek、Qwen、gpt-image 等模型的中转价格与计费倍率，所有模型按 Token / 调用次数透明计费。",
		Canonical:   seoSiteOrigin + "/pricing",
		OGURL:       seoSiteOrigin + "/pricing",
	},
	"/about": {
		Title:       "关于 · New API AI 中转网关",
		Description: "了解 New API 项目背景、技术架构、运维状态与服务条款。",
		Canonical:   seoSiteOrigin + "/about",
		OGURL:       seoSiteOrigin + "/about",
	},
	"/image-playground":  imagePlaygroundMeta,
	"/image-playground/": imagePlaygroundMeta,
}

// GetRouteMeta 暴露给测试与未来需要预渲染时使用。
func GetRouteMeta(path string) (routeMeta, bool) {
	m, ok := routeMetaMap[path]
	return m, ok
}

// RenderIndexWithMeta 在不破坏原 indexPage 字节的前提下，按路径注入差异化的 SEO meta。
// 未识别的路径返回 template 原始内容，保证向后兼容（旧路径行为完全不变）。
//
// 注入策略：
//  1. 替换第一个 <title>...</title> 为新 title
//  2. 替换第一个 name="description" content="..." 为新 description
//  3. 替换第一个 rel="canonical" href="..." 为新 canonical
//  4. 替换 property="og:url" content="..." 为新 OGURL
//  5. 替换 property="og:title" content="..." 为新 Title
//  6. 替换 property="og:description" content="..." 为新 Description
//
// 如果模板里没有对应的字段（旧的 image-playground/index.html 没 og:url），则跳过，不报错。
func RenderIndexWithMeta(template []byte, path string) []byte {
	meta, ok := routeMetaMap[path]
	if !ok {
		return template
	}
	out := template
	out = replaceTitle(out, meta.Title)
	out = replaceMetaName(out, "description", meta.Description)
	out = replaceCanonical(out, meta.Canonical)
	out = replaceMetaProperty(out, "og:url", meta.OGURL)
	out = replaceMetaProperty(out, "og:title", meta.Title)
	out = replaceMetaProperty(out, "og:description", meta.Description)
	out = replaceMetaName(out, "twitter:title", meta.Title)
	out = replaceMetaName(out, "twitter:description", meta.Description)
	return out
}

// GetPrerenderedHTML 阶段 1 永远返回 nil（无预渲染产物）。
// 阶段 2 上线 vite-prerender-plugin 后，由 LoadPrerendered 在启动时从 embed.FS 加载产物到内存 map，
// 此函数改为查 map 返回静态 HTML，从而旁路掉模板替换链路。
func GetPrerenderedHTML(path string) []byte {
	return nil
}

// replaceTitle 替换首个 <title>...</title>。
func replaceTitle(html []byte, newTitle string) []byte {
	start := bytes.Index(html, []byte("<title>"))
	if start < 0 {
		return html
	}
	end := bytes.Index(html[start:], []byte("</title>"))
	if end < 0 {
		return html
	}
	end += start
	buf := make([]byte, 0, len(html)+len(newTitle))
	buf = append(buf, html[:start+len("<title>")]...)
	buf = append(buf, []byte(htmlEscape(newTitle))...)
	buf = append(buf, html[end:]...)
	return buf
}

// replaceMetaName 替换 <meta name="<key>" content="..." />。匹配 key 完全一致。
func replaceMetaName(html []byte, key string, newContent string) []byte {
	return replaceMetaTag(html, `name="`+key+`"`, newContent)
}

// replaceMetaProperty 替换 <meta property="<key>" content="..." />。
func replaceMetaProperty(html []byte, key string, newContent string) []byte {
	return replaceMetaTag(html, `property="`+key+`"`, newContent)
}

// replaceMetaTag 在含 marker（如 name="description"）的 <meta> 标签里替换 content 属性值。
// 同一文档里只替换第一处出现。未找到 marker / content 属性时返回原 html，不报错（向后兼容）。
func replaceMetaTag(html []byte, marker string, newContent string) []byte {
	idx := bytes.Index(html, []byte(marker))
	if idx < 0 {
		return html
	}
	// 在 marker 周围定位 <meta> 标签起止
	tagStart := bytes.LastIndex(html[:idx], []byte("<meta"))
	if tagStart < 0 {
		return html
	}
	tagEnd := bytes.Index(html[idx:], []byte(">"))
	if tagEnd < 0 {
		return html
	}
	tagEnd += idx
	contentIdx := bytes.Index(html[tagStart:tagEnd], []byte(`content="`))
	if contentIdx < 0 {
		return html
	}
	contentIdx += tagStart + len(`content="`)
	contentEnd := bytes.Index(html[contentIdx:tagEnd], []byte(`"`))
	if contentEnd < 0 {
		return html
	}
	contentEnd += contentIdx
	buf := make([]byte, 0, len(html)+len(newContent))
	buf = append(buf, html[:contentIdx]...)
	buf = append(buf, []byte(htmlEscape(newContent))...)
	buf = append(buf, html[contentEnd:]...)
	return buf
}

// replaceCanonical 替换 <link rel="canonical" href="..." />。
func replaceCanonical(html []byte, newHref string) []byte {
	idx := bytes.Index(html, []byte(`rel="canonical"`))
	if idx < 0 {
		return html
	}
	tagStart := bytes.LastIndex(html[:idx], []byte("<link"))
	if tagStart < 0 {
		return html
	}
	tagEnd := bytes.Index(html[idx:], []byte(">"))
	if tagEnd < 0 {
		return html
	}
	tagEnd += idx
	hrefIdx := bytes.Index(html[tagStart:tagEnd], []byte(`href="`))
	if hrefIdx < 0 {
		return html
	}
	hrefIdx += tagStart + len(`href="`)
	hrefEnd := bytes.Index(html[hrefIdx:tagEnd], []byte(`"`))
	if hrefEnd < 0 {
		return html
	}
	hrefEnd += hrefIdx
	buf := make([]byte, 0, len(html)+len(newHref))
	buf = append(buf, html[:hrefIdx]...)
	buf = append(buf, []byte(htmlEscape(newHref))...)
	buf = append(buf, html[hrefEnd:]...)
	return buf
}

// htmlEscape 用最小转义防止注入：仅处理 SEO 字段里可能出现的 " < > &。
// 不使用 html/template 是为了避免引入额外渲染开销，且这些字段都是受控的内置字符串。
func htmlEscape(s string) string {
	r := strings.NewReplacer(
		`&`, `&amp;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
		`"`, `&quot;`,
	)
	return r.Replace(s)
}
