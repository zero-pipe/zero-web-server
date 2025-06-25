package handler

import (
	mediaserverapp "zero-web-kit/internal/application/mediaserver"
	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/persistence/model"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type MediaServerHandler struct {
	svc        *mediaserverapp.Service
	sipCfg     config.SIPConfig
	mediaIP    string
	serverPort int
	serverID   string
	version    string
}

func NewMediaServerHandler(
	svc *mediaserverapp.Service,
	sipCfg config.SIPConfig,
	mediaIP string,
	serverPort int,
	serverID, version string,
) *MediaServerHandler {
	return &MediaServerHandler{
		svc:        svc,
		sipCfg:     sipCfg,
		mediaIP:    mediaIP,
		serverPort: serverPort,
		serverID:   serverID,
		version:    version,
	}
}

func (h *MediaServerHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *MediaServerHandler) OnlineList(c *gin.Context) {
	list, err := h.svc.ListOnline()
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *MediaServerHandler) One(c *gin.Context) {
	v, err := h.svc.GetOne(c.Param("id"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, v)
}

func (h *MediaServerHandler) Check(c *gin.Context) {
	ip := c.Query("ip")
	port := parseIntQuery(c, "port", 80)
	secret := c.Query("secret")
	v, err := h.svc.Check(ip, port, secret)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, v)
}

func (h *MediaServerHandler) Save(c *gin.Context) {
	var m model.MediaServer
	if err := c.ShouldBindJSON(&m); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Save(&m); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *MediaServerHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Query("id")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *MediaServerHandler) MediaInfo(c *gin.Context) {
	info, err := h.svc.MediaInfo(c.Query("app"), c.Query("stream"), c.Query("mediaServerId"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, info)
}

func (h *MediaServerHandler) Load(c *gin.Context) {
	list, err := h.svc.Load()
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *MediaServerHandler) RecordCheck(c *gin.Context) {
	response.Error(c, response.CodeError, "未配置 RecordAssist 服务")
}

func (h *MediaServerHandler) ResourceInfo(c *gin.Context) {
	response.OK(c, gin.H{})
}

func (h *MediaServerHandler) Info(c *gin.Context) {
	response.OK(c, gin.H{})
}

func (h *MediaServerHandler) MapConfig(c *gin.Context) {
	response.OK(c, []any{})
}

func (h *MediaServerHandler) MapModelIconList(c *gin.Context) {
	response.OK(c, []any{})
}

func (h *MediaServerHandler) SystemConfigInfo(c *gin.Context) {
	showIP := h.mediaIP
	if showIP == "" {
		showIP = "127.0.0.1"
	}
	response.OK(c, gin.H{
		"serverPort": h.serverPort,
		"sip": gin.H{
			"id":       h.sipCfg.ID,
			"domain":   h.sipCfg.Domain,
			"port":     h.sipCfg.Port,
			"password": h.sipCfg.Password,
			"showIp":   showIP,
		},
		"addOn": gin.H{
			"serverId": h.serverID,
		},
		"jt1078Config": gin.H{
			"port":     0,
			"password": "",
			"enable":   false,
		},
		"version": gin.H{
			"version": h.version,
		},
	})
}
