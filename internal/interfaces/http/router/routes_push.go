package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerPush(parent *gin.RouterGroup, streamPushHandler *handler.StreamPushHandler) {
	g := parent.Group("/push")
	g.GET("/list", streamPushHandler.List)
	g.POST("/add", streamPushHandler.Add)
	g.POST("/update", streamPushHandler.Update)
	g.POST("/remove", streamPushHandler.Remove)
	g.DELETE("/batchRemove", streamPushHandler.BatchRemove)
	g.POST("/save_to_gb", streamPushHandler.SaveToGB)
	g.DELETE("/remove_form_gb", streamPushHandler.RemoveFromGB)
	g.GET("/start", streamPushHandler.Start)
}
