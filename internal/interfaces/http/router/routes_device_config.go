package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerDeviceConfig(parent *gin.RouterGroup, deviceHandler *handler.DeviceHandler) {
	g := parent.Group("/device/config")
	g.GET("/query/basicParam", deviceHandler.QueryBasicParam)
	g.GET("/set/basicParam", deviceHandler.SetBasicParam)
}
