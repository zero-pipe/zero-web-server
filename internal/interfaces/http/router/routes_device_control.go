package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerDeviceControl 国标专属控制动作，挂在 /devices/gb/control。
// 仅保留前端实际使用的接口（布防/撤防、拖动变倍）。
func registerDeviceControl(parent *gin.RouterGroup, deviceHandler *handler.DeviceHandler) {
	g := parent.Group("/gb/control")
	g.GET("/guard", deviceHandler.Guard)
	g.GET("/drag_zoom/zoom_in", deviceHandler.DragZoomIn)
	g.GET("/drag_zoom/zoom_out", deviceHandler.DragZoomOut)
}
