package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerServerAPI(parent *gin.RouterGroup, mediaServerHandler *handler.MediaServerHandler, gbSipConfigHandler *handler.GbSipConfigHandler, objectStoreHandler *handler.ObjectStoreHandler) {
	g := parent.Group("/server")
	g.GET("/media_server/list", mediaServerHandler.List)
	g.GET("/media_server/online/list", mediaServerHandler.OnlineList)
	g.GET("/media_server/one/:id", mediaServerHandler.One)
	g.GET("/media_server/check", mediaServerHandler.Check)
	g.GET("/media_server/record/check", mediaServerHandler.RecordCheck)
	g.POST("/media_server/save", mediaServerHandler.Save)
	g.DELETE("/media_server/delete", mediaServerHandler.Delete)
	g.GET("/media_server/media_info", mediaServerHandler.MediaInfo)
	g.GET("/media_server/load", mediaServerHandler.Load)
	g.GET("/system/configInfo", mediaServerHandler.SystemConfigInfo)
	g.GET("/gb_sip_config", gbSipConfigHandler.Get)
	g.POST("/gb_sip_config/save", gbSipConfigHandler.Save)
	g.GET("/object_store_config", objectStoreHandler.Get)
	g.POST("/object_store_config/save", objectStoreHandler.Save)
	g.GET("/object_store_config/health", objectStoreHandler.Health)
	g.GET("/resource/info", mediaServerHandler.ResourceInfo)
	g.GET("/info", mediaServerHandler.Info)
	g.GET("/map/config", mediaServerHandler.MapConfig)
	g.GET("/map/model-icon/list", mediaServerHandler.MapModelIconList)
}
