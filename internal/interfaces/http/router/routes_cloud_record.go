package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerCloudRecord(parent *gin.RouterGroup, cloudRecordHandler *handler.CloudRecordHandler) {
	g := parent.Group("/cloud/record")
	g.GET("/list", cloudRecordHandler.List)
	g.GET("/date/list", cloudRecordHandler.DateList)
	g.DELETE("/delete", cloudRecordHandler.Delete)
	g.GET("/play/path", cloudRecordHandler.PlayPath)
	g.GET("/loadRecord", cloudRecordHandler.LoadRecord)
	g.GET("/seek", cloudRecordHandler.Seek)
	g.GET("/speed", cloudRecordHandler.Speed)
	g.GET("/task/add", cloudRecordHandler.AddTask)
	g.GET("/task/list", cloudRecordHandler.TaskList)
}

// registerPlaybackCloud 播放器云端回放（按内码 + 绝对时间）。
func registerPlaybackCloud(parent *gin.RouterGroup, h *handler.CloudRecordHandler) {
	g := parent.Group("/playback/cloud")
	g.GET("/segments/:deviceCode/:channelCode", h.CloudSegments)
	g.GET("/start", h.CloudStart)
	g.GET("/seek", h.CloudSeek)
	g.GET("/stop", h.CloudStop)
}
