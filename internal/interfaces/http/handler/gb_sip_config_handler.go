package handler

import (
	gbsipconfig "zero-web-server/internal/application/gbsipconfig"
	gbdomain "zero-web-server/internal/domain/gbsipconfig"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type GbSipConfigHandler struct {
	svc *gbsipconfig.Service
}

func NewGbSipConfigHandler(svc *gbsipconfig.Service) *GbSipConfigHandler {
	return &GbSipConfigHandler{svc: svc}
}

func (h *GbSipConfigHandler) Get(c *gin.Context) {
	row, err := h.svc.GetOrEmpty()
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, row)
}

func (h *GbSipConfigHandler) Save(c *gin.Context) {
	var body gbdomain.Config
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	portChanged, err := h.svc.Save(&body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	msg := "保存成功"
	if portChanged {
		msg = "保存成功。SIP 端口需重启服务后生效"
	}
	response.OK(c, gin.H{
		"portChanged": portChanged,
		"message":     msg,
	})
}
