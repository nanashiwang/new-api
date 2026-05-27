package router

import (
	"embed"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func SetWebRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	if imagePlaygroundIndex, err := buildFS.ReadFile("web/dist/image-playground/index.html"); err == nil {
		registerImagePlaygroundIndexRoutes(router, imagePlaygroundIndex)
	}
	router.Use(static.Serve("/", common.EmbedFolder(buildFS, "web/dist")))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		// 阶段 2 SSG 上线后，GetPrerenderedHTML 会返回预渲染产物；阶段 1 始终返回 nil。
		if seoHTML := service.GetPrerenderedHTML(c.Request.URL.Path); seoHTML != nil {
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", seoHTML)
			return
		}
		// 按路径注入差异化 SEO meta；未识别的路径返回原 indexPage（行为兼容旧逻辑）。
		html := service.RenderIndexWithMeta(indexPage, c.Request.URL.Path)
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", html)
	})
}

func registerImagePlaygroundIndexRoutes(router *gin.Engine, indexPage []byte) {
	serve := func(c *gin.Context) {
		// image-playground 子站使用独立的 meta 模板（产品词路线），同样支持后续 prerender 接管。
		if seoHTML := service.GetPrerenderedHTML(c.Request.URL.Path); seoHTML != nil {
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", seoHTML)
			return
		}
		html := service.RenderIndexWithMeta(indexPage, c.Request.URL.Path)
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", html)
	}
	router.GET("/image-playground", serve)
	router.GET("/image-playground/", serve)
}
