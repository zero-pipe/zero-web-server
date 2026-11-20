package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerPluginPackagePublic 挂载 Agent 专用下载（凭 X-Agent-Token，无 JWT）。
func registerPluginPackagePublic(g *gin.RouterGroup, h *handler.PluginPackageHandler) {
	g.GET("/plugin-packages/:id/download", h.Download)
}

// registerPluginPackages 挂载插件包管理与安装（需登录）。
func registerPluginPackages(g *gin.RouterGroup, h *handler.PluginPackageHandler) {
	g.GET("/plugin-packages", h.List)
	g.POST("/plugin-packages", h.Upload)
	g.GET("/plugin-packages/:id", h.Get)
	g.DELETE("/plugin-packages/:id", h.Delete)
	g.POST("/plugin-packages/:id/install", h.Install)
	g.GET("/plugin-installs", h.ListInstalls)
	g.DELETE("/plugin-installs/:id", h.Uninstall)
}
