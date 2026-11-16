package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerDeviceControl(parent *gin.RouterGroup, deviceHandler *handler.DeviceHandler) {
	g := parent.Group("/device/control")
	g.GET("/guard", deviceHandler.Guard)
	g.GET("/record", deviceHandler.Record)
	g.GET("/home_position", deviceHandler.HomePosition)
	g.GET("/drag_zoom/zoom_in", deviceHandler.DragZoomIn)
	g.GET("/drag_zoom/zoom_out", deviceHandler.DragZoomOut)
}
