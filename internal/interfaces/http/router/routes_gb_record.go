package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerGbRecord(parent *gin.RouterGroup, gbRecordHandler *handler.GBRecordHandler) {
	g := parent.Group("/gb_record")
	g.GET("/query/:deviceId/:channelId", gbRecordHandler.Query)
	g.GET("/download/start/:deviceId/:channelId", gbRecordHandler.DownloadStart)
	g.GET("/download/stop/:deviceId/:channelId/:stream", gbRecordHandler.DownloadStop)
	g.GET("/download/progress/:deviceId/:channelId/:stream", gbRecordHandler.DownloadProgress)
}
