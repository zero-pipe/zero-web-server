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
func registerPluginDevices(parent *gin.RouterGroup, h *handler.PluginHandler) {
	g := parent.Group("/plugin")
	g.GET("/devices", h.ListDevices)
	g.POST("/devices", h.CreateDevice)
	g.DELETE("/devices", h.DeleteDevice)
	g.GET("/channels", h.Channels)
	g.POST("/devices/sync", h.SyncDevice)
	g.GET("/play/start", h.StartLive)
	g.GET("/play/stop", h.StopLive)
	g.POST("/ptz", h.PTZ)
}
