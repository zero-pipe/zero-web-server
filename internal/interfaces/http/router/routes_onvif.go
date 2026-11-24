package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerOnvif(parent *gin.RouterGroup, onvifHandler *handler.ONVIFHandler) {
	g := parent.Group("/onvif")
	g.POST("/device/discover", onvifHandler.Discover)
	g.GET("/device/discover", onvifHandler.Discover)
	g.GET("/device/query", onvifHandler.QueryDevices)
	g.POST("/device/add", onvifHandler.AddDevice)
	g.DELETE("/device/delete/:id", onvifHandler.DeleteDevice)
	g.POST("/device/sync/:id", onvifHandler.SyncDevice)
	g.POST("/device/probe", onvifHandler.Probe)
	g.GET("/device/event-subscribe", onvifHandler.GetEventSubscribe)
	g.POST("/device/event-subscribe", onvifHandler.SetEventSubscribe)
	g.GET("/channel/query", onvifHandler.QueryChannels)
	g.POST("/channel/update", onvifHandler.UpdateChannel)
	g.GET("/play/start", onvifHandler.StartPlay)
	g.GET("/play/stop", onvifHandler.StopPlay)
	g.POST("/ptz/control", onvifHandler.PTZControl)
	g.GET("/ptz/preset/query", onvifHandler.QueryPresets)
	g.GET("/ptz/preset/call", onvifHandler.GotoPreset)
	g.GET("/ptz/preset/add", onvifHandler.SetPreset)
	g.GET("/ptz/preset/delete", onvifHandler.RemovePreset)
}
