package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerProxy(parent *gin.RouterGroup, streamProxyHandler *handler.StreamProxyHandler) {
	g := parent.Group("/proxy")
	g.GET("/list", streamProxyHandler.List)
	g.POST("/add", streamProxyHandler.Add)
	g.POST("/save", streamProxyHandler.Save)
	g.POST("/update", streamProxyHandler.Update)
	g.DELETE("/delete", streamProxyHandler.Delete)
	g.GET("/start", streamProxyHandler.Start)
	g.GET("/stop", streamProxyHandler.Stop)
	g.GET("/ffmpeg_cmd/list", streamProxyHandler.FFmpegCmdList)
}
