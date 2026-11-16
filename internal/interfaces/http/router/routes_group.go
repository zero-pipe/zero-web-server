package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerGroup(parent *gin.RouterGroup, groupHandler *handler.GroupHandler) {
	g := parent.Group("/group")
	g.GET("/tree/list", groupHandler.TreeList)
	g.GET("/tree/query", groupHandler.TreeQuery)
	g.POST("/add", groupHandler.Add)
	g.POST("/update", groupHandler.Update)
	g.DELETE("/delete", groupHandler.Delete)
	g.GET("/path", groupHandler.Path)
}
