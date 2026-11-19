package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerAlarm(parent *gin.RouterGroup, alarmHandler *handler.AlarmHandler) {
	g := parent.Group("/alarm")
	g.GET("/list", alarmHandler.List)
	g.PUT("/:id/status", alarmHandler.SetStatus)
	g.DELETE("/delete", alarmHandler.Delete)
	g.DELETE("/clear", alarmHandler.Clear)
}
