package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerRegion(parent *gin.RouterGroup, regionHandler *handler.RegionHandler) {
	g := parent.Group("/region")
	g.GET("/tree/list", regionHandler.TreeList)
	g.GET("/tree/query", regionHandler.TreeQuery)
	g.POST("/add", regionHandler.Add)
	g.POST("/update", regionHandler.Update)
	g.DELETE("/delete", regionHandler.Delete)
	g.GET("/path", regionHandler.Path)
	g.GET("/description", regionHandler.Description)
	g.GET("/addByCivilCode", regionHandler.AddByCivilCode)
	g.GET("/base/child/list", regionHandler.BaseChildList)
	g.GET("/code_mapping/list", regionHandler.CodeMappingList)
	g.POST("/code_mapping/save", regionHandler.CodeMappingSave)
	g.DELETE("/code_mapping/delete", regionHandler.CodeMappingDelete)
}
