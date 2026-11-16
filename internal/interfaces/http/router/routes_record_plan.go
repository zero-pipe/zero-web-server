package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerRecordPlan(parent *gin.RouterGroup, recordPlanHandler *handler.RecordPlanHandler) {
	g := parent.Group("/record/plan")
	g.POST("/add", recordPlanHandler.Add)
	g.POST("/update", recordPlanHandler.Update)
	g.GET("/get", recordPlanHandler.Get)
	g.GET("/query", recordPlanHandler.Query)
	g.DELETE("/delete", recordPlanHandler.Delete)
	g.POST("/link", recordPlanHandler.Link)
	g.GET("/channel/list", recordPlanHandler.ChannelList)
}
