package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerCustomEvents(g *gin.RouterGroup, h *handler.CustomEventHandler) {
	g.GET("/custom-events", h.List)
	g.GET("/custom-events/types", h.Types)
	g.GET("/custom-events/:id", h.Get)
}
