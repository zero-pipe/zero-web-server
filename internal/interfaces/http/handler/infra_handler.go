package handler

import (
	"context"
	"time"

	infraapp "zero-web-server/internal/application/infraconfig"
	domaininfra "zero-web-server/internal/domain/infraconfig"
	"zero-web-server/internal/infrastructure/config"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type InfraHandler struct {
	svc *infraapp.Service
}

func NewInfraHandler(svc *infraapp.Service) *InfraHandler {
	return &InfraHandler{svc: svc}
}

func (h *InfraHandler) Overview(c *gin.Context) {
	response.OK(c, h.svc.Overview(c.Request.Context()))
}

func (h *InfraHandler) SaveMySQL(c *gin.Context) {
	var body config.MySQLConfig
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	restart, warning, err := h.svc.SaveMySQL(body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"restartRequired": restart, "message": "已保存，重启服务后生效", "warning": warning})
}

func (h *InfraHandler) SaveRedis(c *gin.Context) {
	var body config.RedisConfig
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	restart, warning, err := h.svc.SaveRedis(body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"restartRequired": restart, "message": "已保存，重启服务后生效", "warning": warning})
}

func (h *InfraHandler) GetMQTT(c *gin.Context) {
	row, err := h.svc.GetMQTT()
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, row)
}

func (h *InfraHandler) SaveMQTT(c *gin.Context) {
	var body domaininfra.MQTTConfig
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SaveMQTT(&body); err != nil {
		response.FromError(c, err)
		return
	}
	connected, msg := h.svc.MQTTStatus()
	response.OK(c, gin.H{"connected": connected, "message": "已保存并热重载", "error": msg})
}

func (h *InfraHandler) MQTTHealth(c *gin.Context) {
	connected, msg := h.svc.MQTTStatus()
	response.OK(c, gin.H{"ok": connected, "error": msg})
}

func (h *InfraHandler) PipelineStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	response.OK(c, h.svc.PipelineStatus(ctx))
}
