package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerPlugin 插件注册表（非设备维度，挂在 /plugins）。
func registerPlugin(g *gin.RouterGroup, h *handler.PluginHandler) {
	g.GET("/plugins", h.List)
}

// registerPluginDevices 插件设备专属动作，挂在 /devices/plugin。
// 仅保留前端实际使用的接口：新增（登录插件→入库）、同步。布防/撤防/状态见 routes_event_sub。
func registerPluginDevices(parent *gin.RouterGroup, h *handler.PluginHandler) {
	g := parent.Group("/plugin")
	g.POST("/devices", h.CreateDevice)
	g.POST("/devices/sync", h.SyncDevice)
}
