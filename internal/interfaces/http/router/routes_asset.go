package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerAssets 统一设备资产与资源 API。
func registerAssets(g *gin.RouterGroup, h *handler.AssetHandler) {
	g.GET("/assets", h.List)
	g.GET("/assets/tree", h.Tree)
	g.GET("/channels", h.Channels)
	g.GET("/assets/:id", h.Detail)
	g.PUT("/assets/:id", h.UpdateDevice)
	g.GET("/assets/:id/resources", h.Resources)
	g.POST("/assets/:id/sync", h.Sync)
	g.PUT("/channels/:code", h.UpdateChannel)

	g.GET("/resources/:id", h.ResourceDetail)
	g.GET("/resources/:id/profiles", h.Profiles)
	g.GET("/resources/:id/capabilities", h.Capabilities)
	g.POST("/resources/:id/live/start", h.StartLive)
	g.POST("/resources/:id/live/stop", h.StopLive)
	g.GET("/resources/:id/snapshot", h.Snapshot)
	// 云台控制（按资源 id，速度沿用前端既有取值）。
	g.GET("/resources/:id/ptz", h.PTZ)
	// 按时间回放（统一资源口径）：检索录像、起流、停流。
	g.POST("/resources/:id/playback/records", h.QueryRecords)
	g.POST("/resources/:id/playback/start", h.StartPlayback)
	g.POST("/resources/:id/playback/stop", h.StopPlayback)
	g.POST("/resources/:id/command/:cmd", h.Command)
}
