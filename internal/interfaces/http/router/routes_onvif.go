package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerOnvif ONVIF 专属动作，挂在 /devices/onvif。
// 仅保留前端实际使用的接口：局域网发现、事件订阅开关。
func registerOnvif(parent *gin.RouterGroup, onvifHandler *handler.ONVIFHandler) {
	g := parent.Group("/onvif")
	g.POST("/device/discover", onvifHandler.Discover)
	g.GET("/device/discover", onvifHandler.Discover)
	g.POST("/device/event-subscribe", onvifHandler.SetEventSubscribe)
}
