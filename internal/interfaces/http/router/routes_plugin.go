package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerPluginPublic 插件注册/心跳（无 JWT，凭 X-Plugin-Token）。
func registerPluginPublic(g *gin.RouterGroup, h *handler.PluginHandler) {
	g.POST("/plugins/hello", h.Hello)
	g.POST("/plugins/heartbeat", h.Heartbeat)
}

// registerPlugin 插件与插件设备管理。
func registerPlugin(g *gin.RouterGroup, h *handler.PluginHandler) {
	g.GET("/plugins", h.List)
	g.GET("/ipc/devices", h.ListDevices)
	g.POST("/ipc/devices", h.CreateDevice)
	g.DELETE("/ipc/devices", h.DeleteDevice)
	g.GET("/ipc/channels", h.Channels)
	g.POST("/ipc/devices/sync", h.SyncDevice)
	g.GET("/ipc/play/start", h.StartLive)
	g.GET("/ipc/play/stop", h.StopLive)
	g.POST("/ipc/ptz", h.PTZ)
}
