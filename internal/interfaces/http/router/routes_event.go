package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerEvents(g *gin.RouterGroup, h *handler.EventHandler) {
	g.GET("/events", h.List)
	g.GET("/events/:id", h.Get)
}
