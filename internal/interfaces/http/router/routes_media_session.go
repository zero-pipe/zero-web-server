package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerMediaSessions(parent *gin.RouterGroup, h *handler.MediaSessionHandler) {
	g := parent.Group("/media-sessions")
	g.GET("", h.List)
	g.POST("/stop", h.Stop)
	g.POST("/proxy", h.AddProxy)
}
