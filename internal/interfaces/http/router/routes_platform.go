package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerPlatform(parent *gin.RouterGroup, platformHandler *handler.PlatformHandler) {
	g := parent.Group("/platform")
	g.GET("/server_config", platformHandler.ServerConfig)
	g.GET("/query", platformHandler.Query)
	g.POST("/add", platformHandler.Add)
	g.POST("/update", platformHandler.Update)
	g.DELETE("/delete", platformHandler.Delete)
	g.GET("/exit/:deviceGbId", platformHandler.Exit)
	g.GET("/channel/push", platformHandler.PushChannel)
	g.GET("/channel/list", platformHandler.ChannelList)
	g.POST("/channel/add", platformHandler.ChannelAdd)
	g.POST("/channel/device/add", platformHandler.ChannelDeviceAdd)
	g.POST("/channel/device/remove", platformHandler.ChannelDeviceRemove)
	g.DELETE("/channel/remove", platformHandler.ChannelRemove)
	g.POST("/channel/custom/update", platformHandler.ChannelCustomUpdate)
	g.GET("/group/list", platformHandler.GroupShareList)
	g.POST("/group/add", platformHandler.GroupShareAdd)
	g.POST("/group/remove", platformHandler.GroupShareRemove)
}
