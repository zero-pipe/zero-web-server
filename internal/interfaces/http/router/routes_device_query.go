package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerDeviceQuery 国标专属查询/订阅/统计动作，挂在 /devices/gb/query。
// 仅保留前端实际使用的接口（设备列表、刷新同步、同步状态、目录/位置/报警订阅、
// 心跳/注册统计、抓拍）。
func registerDeviceQuery(parent *gin.RouterGroup, deviceHandler *handler.DeviceHandler, snapHandler *handler.SnapHandler) {
	g := parent.Group("/gb/query")
	g.GET("/devices", deviceHandler.ListDevices)
	g.GET("/devices/:deviceId/sync", deviceHandler.SyncDevice)
	g.GET("/sync_status", deviceHandler.SyncStatus)
	g.GET("/subscribe/catalog", deviceHandler.SubscribeCatalog)
	g.GET("/subscribe/mobile-position", deviceHandler.SubscribeMobilePosition)
	g.GET("/subscribe/alarm", deviceHandler.SubscribeAlarm)
	g.GET("/statistics/keepalive", deviceHandler.KeepaliveStatistics)
	g.GET("/statistics/register", deviceHandler.RegisterStatistics)
	// 抓拍：通道缩略图按 URL 直接引用 / 设备上传
	g.GET("/snap/:deviceId/:channelId", snapHandler.GetChannelSnap)
	g.POST("/snap/:deviceId/:channelId", snapHandler.UploadChannelSnap)
	g.POST("/snap/config", deviceHandler.SnapShotConfig)
	g.GET("/snap/finished/:sessionId", snapHandler.Finished)
}
