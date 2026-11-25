package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
	"zero-web-server/internal/interfaces/http/middleware"
)

// registerDict 数据字典：读接口对所有登录用户开放（下拉数据源）；
// 写接口（类型/项维护）需要「系统配置」菜单权限。
func registerDict(parent *gin.RouterGroup, h *handler.DictHandler) {
	g := parent.Group("/dict")
	g.GET("/types", h.ListTypes)
	g.GET("/items", h.ListItems)

	w := g.Group("")
	w.Use(middleware.RequireMenu("config"))
	w.POST("/types", h.CreateType)
	w.PUT("/types/:id", h.UpdateType)
	w.DELETE("/types/:id", h.DeleteType)
	w.POST("/items", h.CreateItem)
	w.PUT("/items/:id", h.UpdateItem)
	w.DELETE("/items/:id", h.DeleteItem)
}
