package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerDeviceControl(parent *gin.RouterGroup, deviceHandler *handler.DeviceHandler) {
	g := parent.Group("/gb/control")
	g.GET("/guard", deviceHandler.Guard)
	g.GET("/record", deviceHandler.Record)
	g.GET("/home_position", deviceHandler.HomePosition)
	g.GET("/drag_zoom/zoom_in", deviceHandler.DragZoomIn)
	g.GET("/drag_zoom/zoom_out", deviceHandler.DragZoomOut)
	g.GET("/alarm_reset", deviceHandler.AlarmReset)
	g.GET("/teleboot", deviceHandler.TeleBoot)
	g.GET("/iframe", deviceHandler.IFrame)
	g.GET("/ptz_precise", deviceHandler.PTZPrecise)
	g.GET("/sdcard_format", deviceHandler.FormatSDCard)
	g.GET("/target_track", deviceHandler.TargetTrack)
}
