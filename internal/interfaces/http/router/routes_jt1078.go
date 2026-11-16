package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerJt1078(parent *gin.RouterGroup, jt1078Handler *handler.JT1078Handler) {
	g := parent.Group("/jt1078")
	terminal := g.Group("/terminal")
	{
		terminal.GET("/list", jt1078Handler.TerminalList)
		terminal.GET("/query", jt1078Handler.TerminalQuery)
		terminal.POST("/update", jt1078Handler.TerminalUpdate)
		terminal.POST("/add", jt1078Handler.TerminalAdd)
		terminal.DELETE("/delete", jt1078Handler.TerminalDelete)
		terminal.GET("/channel/list", jt1078Handler.ChannelList)
		terminal.POST("/channel/update", jt1078Handler.ChannelUpdate)
		terminal.POST("/channel/add", jt1078Handler.ChannelAdd)
	}
	g.GET("/live/start", jt1078Handler.LiveStart)
	g.GET("/live/stop", jt1078Handler.LiveStop)
	g.GET("/talk/start", jt1078Handler.TalkStart)
	g.GET("/talk/stop", jt1078Handler.TalkStop)
	g.GET("/ptz", jt1078Handler.PTZ)
	g.GET("/wiper", jt1078Handler.Wiper)
	g.GET("/fill-light", jt1078Handler.FillLight)
	g.GET("/record/list", jt1078Handler.RecordList)
	g.GET("/playback/start", jt1078Handler.PlaybackStart)
	g.GET("/playback/downloadUrl", jt1078Handler.PlaybackDownloadURL)
	g.GET("/playback/control", jt1078Handler.PlaybackControl)
	g.GET("/playback/stop", jt1078Handler.PlaybackStop)
	g.GET("/playback/download", jt1078Handler.PlaybackDownload)
	g.GET("/config/get", jt1078Handler.ConfigGet)
	g.POST("/config/set", jt1078Handler.ConfigSet)
	g.GET("/attribute", jt1078Handler.Attribute)
	g.GET("/link-detection", jt1078Handler.LinkDetection)
	g.GET("/position-info", jt1078Handler.PositionInfo)
	g.POST("/text-msg", jt1078Handler.TextMsg)
	g.GET("/telephone-callback", jt1078Handler.TelephoneCallback)
	g.GET("/driver-information", jt1078Handler.DriverInformation)
	g.POST("/control/factory-reset", jt1078Handler.FactoryReset)
	g.POST("/control/reset", jt1078Handler.Reset)
	g.POST("/control/connection", jt1078Handler.Connection)
	g.GET("/control/door", jt1078Handler.ControlDoor)
	g.GET("/media/attribute", jt1078Handler.MediaAttribute)
	g.POST("/media/list", jt1078Handler.MediaList)
	g.POST("/set-phone-book", jt1078Handler.SetPhoneBook)
	g.POST("/shooting", jt1078Handler.Shooting)
	g.GET("/snap", jt1078Handler.Snap)
	g.GET("/media/upload/one/upload", jt1078Handler.MediaUpload)
}
