package handler

import (
	mediasessionapp "zero-web-server/internal/application/mediasession"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// MediaSessionHandler 媒体会话（实时流）运维视图。
type MediaSessionHandler struct {
	svc *mediasessionapp.Service
}

func NewMediaSessionHandler(svc *mediasessionapp.Service) *MediaSessionHandler {
	return &MediaSessionHandler{svc: svc}
}

func (h *MediaSessionHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *MediaSessionHandler) Stop(c *gin.Context) {
	var body struct {
		NodeID string `json:"nodeId"`
		App    string `json:"app"`
		Stream string `json:"stream"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Stop(c.Request.Context(), body.NodeID, body.App, body.Stream); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *MediaSessionHandler) AddProxy(c *gin.Context) {
	var body struct {
		NodeID string `json:"nodeId"`
		App    string `json:"app"`
		Stream string `json:"stream"`
		URL    string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.AddProxy(c.Request.Context(), body.NodeID, body.App, body.Stream, body.URL); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
