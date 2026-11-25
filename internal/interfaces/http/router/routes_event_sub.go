package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerEventSub(g *gin.RouterGroup, h *handler.EventSubHandler) {
	g.POST("/devices/plugin/devices/arm", h.Arm)
	g.POST("/devices/plugin/devices/disarm", h.Disarm)
	g.GET("/devices/plugin/devices/arm/status", h.Status)
}
