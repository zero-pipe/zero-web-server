package handler

import (
	"strings"

	agentapp "zero-web-server/internal/application/agent"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type AgentHandler struct {
	svc   *agentapp.Service
	token string
}

func NewAgentHandler(svc *agentapp.Service, token string) *AgentHandler {
	return &AgentHandler{svc: svc, token: strings.TrimSpace(token)}
}

func (h *AgentHandler) checkToken(c *gin.Context) bool {
	if h.token == "" {
		return true
	}
	if c.GetHeader("X-Agent-Token") != h.token {
		response.Error(c, response.CodeUnauth, "Agent 令牌无效")
		return false
	}
	return true
}

// Register Agent 注册（公开接口，凭 X-Agent-Token）。
func (h *AgentHandler) Register(c *gin.Context) {
	if !h.checkToken(c) {
		return
	}
	var req agentapp.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	agent, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	plugins, _ := h.svc.ListPlugins(req.AgentID)
	response.OK(c, gin.H{"agent": agent, "plugins": plugins})
}

func (h *AgentHandler) Heartbeat(c *gin.Context) {
	if !h.checkToken(c) {
		return
	}
	var req agentapp.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Heartbeat(c.Request.Context(), req); err != nil {
		response.FromError(c, err)
		return
	}
	plugins, _ := h.svc.ListPlugins(req.AgentID)
	response.OK(c, gin.H{"ok": true, "plugins": plugins})
}

func (h *AgentHandler) List(c *gin.Context) {
	list, err := h.svc.ListAgents()
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *AgentHandler) Plugins(c *gin.Context) {
	list, err := h.svc.ListPlugins(c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *AgentHandler) SetPluginState(c *gin.Context) {
	action := c.Param("action")
	list, err := h.svc.SetPluginState(c.Request.Context(), c.Param("id"), c.Param("pluginId"), action)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

// DeletePlugin 删除内置/本地插件（须先删设备、再禁用）。
func (h *AgentHandler) DeletePlugin(c *gin.Context) {
	if err := h.svc.DeleteBuiltinPlugin(c.Request.Context(), c.Param("id"), c.Param("pluginId")); err != nil {
		response.Error(c, response.CodeBadReq, err.Error())
		return
	}
	response.OK(c, nil)
}
