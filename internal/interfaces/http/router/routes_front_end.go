package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerFrontEnd(parent *gin.RouterGroup, ptzHandler *handler.PTZHandler) {
	g := parent.Group("/front-end")
	g.GET("/ptz/:deviceId/:channelId", ptzHandler.PTZ)
	g.GET("/preset/query/:deviceId/:channelId", ptzHandler.QueryPreset)
	g.GET("/ptz_position/:deviceId/:channelId", ptzHandler.PTZPosition)
	g.GET("/ptz_position/query/:deviceId/:channelId", ptzHandler.PTZPositionQuery)
	g.GET("/ptz_position/subscribe/:deviceId/:channelId", ptzHandler.PTZPositionSubscribe)
	g.GET("/preset/add/:deviceId/:channelId", ptzHandler.AddPreset)
	g.GET("/preset/call/:deviceId/:channelId", ptzHandler.CallPreset)
	g.GET("/preset/delete/:deviceId/:channelId", ptzHandler.DeletePreset)
	g.GET("/fi/focus/:deviceId/:channelId", ptzHandler.Focus)
	g.GET("/fi/iris/:deviceId/:channelId", ptzHandler.Iris)
	g.GET("/wiper/:deviceId/:channelId", ptzHandler.Wiper)
	g.GET("/auxiliary/:deviceId/:channelId", ptzHandler.Auxiliary)
	g.GET("/cruise/point/add/:deviceId/:channelId", ptzHandler.AddCruisePoint)
	g.GET("/cruise/point/delete/:deviceId/:channelId", ptzHandler.DeleteCruisePoint)
	g.GET("/cruise/speed/:deviceId/:channelId", ptzHandler.SetCruiseSpeed)
	g.GET("/cruise/time/:deviceId/:channelId", ptzHandler.SetCruiseTime)
	g.GET("/cruise/start/:deviceId/:channelId", ptzHandler.StartCruise)
	g.GET("/cruise/stop/:deviceId/:channelId", ptzHandler.StopCruise)
	g.GET("/scan/start/:deviceId/:channelId", ptzHandler.StartScan)
	g.GET("/scan/stop/:deviceId/:channelId", ptzHandler.StopScan)
	g.GET("/scan/set/left/:deviceId/:channelId", ptzHandler.SetScanLeft)
	g.GET("/scan/set/right/:deviceId/:channelId", ptzHandler.SetScanRight)
	g.GET("/scan/set/speed/:deviceId/:channelId", ptzHandler.SetScanSpeed)
}
