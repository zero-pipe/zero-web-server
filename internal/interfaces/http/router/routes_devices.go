package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerDevices(parent *gin.RouterGroup, deviceAccessHandler *handler.DeviceAccessHandler) {
	g := parent.Group("/devices")
	g.GET("", deviceAccessHandler.List)
	g.GET("/detail", deviceAccessHandler.Get)
	g.POST("", deviceAccessHandler.Create)
	g.PUT("", deviceAccessHandler.Update)
	g.DELETE("", deviceAccessHandler.Delete)
	g.POST("/delete", deviceAccessHandler.Delete)
	g.POST("/sync", deviceAccessHandler.Sync)
}
