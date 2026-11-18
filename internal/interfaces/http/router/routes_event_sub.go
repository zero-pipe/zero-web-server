package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerEventSub(g *gin.RouterGroup, h *handler.EventSubHandler) {
	g.POST("/ipc/devices/arm", h.Arm)
	g.POST("/ipc/devices/disarm", h.Disarm)
	g.GET("/ipc/devices/arm/status", h.Status)
}
