package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerLogAPI(parent *gin.RouterGroup, logHandler *handler.LogHandler) {
	g := parent.Group("/log")
	g.GET("/list", logHandler.List)
	g.GET("/file/:fileName", logHandler.File)
}
