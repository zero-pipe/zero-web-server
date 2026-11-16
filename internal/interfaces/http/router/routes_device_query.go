package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerDeviceQuery(parent *gin.RouterGroup, deviceHandler *handler.DeviceHandler, snapHandler *handler.SnapHandler) {
	g := parent.Group("/device/query")
	g.GET("/devices/:deviceId", deviceHandler.GetDevice)
	g.GET("/devices", deviceHandler.ListDevices)
	g.GET("/devices/:deviceId/channels", deviceHandler.ListChannels)
	g.GET("/devices/:deviceId/sync", deviceHandler.SyncDevice)
	g.DELETE("/devices/:deviceId/delete", deviceHandler.DeleteDevice)
	g.GET("/sync_status", deviceHandler.SyncStatus)
	g.POST("/device/add", deviceHandler.AddDevice)
	g.POST("/device/update", deviceHandler.UpdateDevice)
	g.GET("/channel/one", deviceHandler.GetChannel)
	g.POST("/transport/:deviceId/:streamMode", deviceHandler.SetTransport)
	g.GET("/subscribe/catalog", deviceHandler.SubscribeCatalog)
	g.GET("/subscribe/mobile-position", deviceHandler.SubscribeMobilePosition)
	g.GET("/subscribe/alarm", deviceHandler.SubscribeAlarm)
	g.GET("/statistics/keepalive", deviceHandler.KeepaliveStatistics)
	g.GET("/statistics/register", deviceHandler.RegisterStatistics)
	g.POST("/channel/audio", deviceHandler.ChangeChannelAudio)
	g.GET("/snap/:deviceId/:channelId", snapHandler.GetChannelSnap)
	g.POST("/snap/:deviceId/:channelId", snapHandler.UploadChannelSnap)
}
