package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerSubordinate(parent *gin.RouterGroup, subordinateHandler *handler.SubordinateHandler) {
	g := parent.Group("/subordinate")
	g.GET("/query", subordinateHandler.List)
	g.GET("/one/:id", subordinateHandler.Get)
	g.POST("/add", subordinateHandler.Add)
	g.POST("/update/:id", subordinateHandler.Update)
	g.DELETE("/delete/:id", subordinateHandler.Delete)
}
