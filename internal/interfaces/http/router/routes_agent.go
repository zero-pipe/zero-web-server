package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerAgentPublic zero-agent 注册/心跳（无 JWT，凭 X-Agent-Token）。
func registerAgentPublic(g *gin.RouterGroup, h *handler.AgentHandler) {
	g.POST("/agents/register", h.Register)
	g.POST("/agents/heartbeat", h.Heartbeat)
}

func registerAgents(g *gin.RouterGroup, h *handler.AgentHandler) {
	g.GET("/agents", h.List)
	g.GET("/agents/:id/plugins", h.Plugins)
	g.POST("/agents/:id/plugins/:pluginId/:action", h.SetPluginState)
}
