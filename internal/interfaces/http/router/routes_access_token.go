package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
	"zero-web-server/internal/interfaces/http/middleware"
)

// registerAccessToken 对外接入密钥（可视化配置）。
func registerAccessToken(parent *gin.RouterGroup, h *handler.AccessTokenHandler) {
	g := parent.Group("/accessToken")
	g.Use(middleware.RequireMenu("media"))
	g.GET("/list", h.List)
	g.GET("/first", h.First)
	g.POST("/add", h.Add)
	g.POST("/update", h.Update)
	g.POST("/enable", h.Enable)
	g.POST("/disable", h.Disable)
	g.POST("/reset", h.Reset)
	g.DELETE("/delete", h.Delete)
}
