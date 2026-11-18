package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerInfra 基础设施配置与事件管道状态。
func registerInfra(g *gin.RouterGroup, h *handler.InfraHandler) {
	g.GET("/infra/overview", h.Overview)
	g.POST("/infra/mysql/save", h.SaveMySQL)
	g.POST("/infra/redis/save", h.SaveRedis)
	g.GET("/infra/mqtt", h.GetMQTT)
	g.POST("/infra/mqtt/save", h.SaveMQTT)
	g.GET("/infra/mqtt/health", h.MQTTHealth)
	g.GET("/event_pipeline/status", h.PipelineStatus)
}
