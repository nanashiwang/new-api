package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func SetDesktopUpdateRouter(router *gin.Engine) {
	public := router.Group("/desktop/update")
	public.Use(middleware.RouteTag("desktop_update"))
	{
		public.GET("/latest.json", controller.ServeDesktopUpdateManifest)
		public.HEAD("/latest.json", controller.ServeDesktopUpdateManifest)
		public.GET("/releases/:version/:filename", controller.ServeDesktopUpdateArtifact)
		public.HEAD("/releases/:version/:filename", controller.ServeDesktopUpdateArtifact)
		public.PUT("/publish/:version/:filename", controller.PublishDesktopUpdateArtifact)
		public.PUT("/publish/latest.json", controller.PublishDesktopUpdateManifestWithToken)
	}

	admin := router.Group("/api/desktop-update")
	admin.Use(middleware.RouteTag("api"))
	admin.Use(middleware.BodyStorageCleanup())
	admin.Use(middleware.GlobalAPIRateLimit())
	admin.Use(middleware.RootAuth())
	{
		admin.GET("/status", controller.GetDesktopUpdateStatus)
		admin.PUT("/settings", controller.UpdateDesktopUpdateSettings)
		admin.POST("/token/rotate", controller.RotateDesktopUpdatePublishToken)
		admin.GET("/releases", controller.ListDesktopUpdateReleases)
		admin.POST("/releases/:version/files", controller.UploadDesktopUpdateArtifact)
		admin.POST("/manifest", controller.PublishDesktopUpdateManifest)
		admin.DELETE("/releases/:version", controller.DeleteDesktopUpdateRelease)
	}
}
