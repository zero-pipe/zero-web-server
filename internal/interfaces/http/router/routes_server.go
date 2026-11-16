package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerServer(parent *gin.RouterGroup, healthHandler *handler.HealthHandler, serverHandler *handler.ServerHandler) {
	g := parent.Group("/server")
	g.GET("/version", healthHandler.Version)
	g.GET("/config", serverHandler.Config)
	g.GET("/system/info", serverHandler.SystemInfo)
}
