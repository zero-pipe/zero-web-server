package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerAssets 统一设备资产与资源 API。
func registerAssets(g *gin.RouterGroup, h *handler.AssetHandler) {
	g.GET("/assets", h.List)
	g.GET("/assets/:id", h.Detail)
	g.GET("/assets/:id/resources", h.Resources)
	g.POST("/assets/:id/sync", h.Sync)

	g.GET("/resources/:id", h.ResourceDetail)
	g.GET("/resources/:id/profiles", h.Profiles)
	g.GET("/resources/:id/capabilities", h.Capabilities)
	g.POST("/resources/:id/live/start", h.StartLive)
	g.POST("/resources/:id/live/stop", h.StopLive)
}
