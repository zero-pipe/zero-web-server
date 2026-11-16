package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerPosition(parent *gin.RouterGroup, positionHandler *handler.PositionHandler) {
	g := parent.Group("/position")
	g.GET("/history/:deviceId", positionHandler.History)
	g.GET("/latest", positionHandler.Latest)
}
