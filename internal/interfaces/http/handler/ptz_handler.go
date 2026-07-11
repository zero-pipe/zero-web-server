package handler

import (
	"strconv"

	ptzapp "zero-web-kit/internal/application/ptz"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type PTZHandler struct {
	svc *ptzapp.Service
}

func NewPTZHandler(svc *ptzapp.Service) *PTZHandler {
	return &PTZHandler{svc: svc}
}

func (h *PTZHandler) PTZ(c *gin.Context) {
	deviceID := c.Param("deviceId")
	channelID := c.Param("channelId")
	command := c.Query("command")
	hSpeed, _ := strconv.Atoi(c.DefaultQuery("horizonSpeed", "100"))
	vSpeed, _ := strconv.Atoi(c.DefaultQuery("verticalSpeed", "100"))
	zSpeed, _ := strconv.Atoi(c.DefaultQuery("zoomSpeed", "16"))
	if err := h.svc.Control(deviceID, channelID, command, hSpeed, vSpeed, zSpeed); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) QueryPreset(c *gin.Context) {
	list, err := h.svc.QueryPreset(c.Request.Context(), c.Param("deviceId"), c.Param("channelId"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *PTZHandler) AddPreset(c *gin.Context) {
	presetID, err := strconv.Atoi(c.Query("presetId"))
	if err != nil {
		response.Error(c, response.CodeBadReq, "presetId 无效")
		return
	}
	if err := h.svc.AddPreset(c.Param("deviceId"), c.Param("channelId"), presetID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) CallPreset(c *gin.Context) {
	presetID, err := strconv.Atoi(c.Query("presetId"))
	if err != nil {
		response.Error(c, response.CodeBadReq, "presetId 无效")
		return
	}
	if err := h.svc.CallPreset(c.Param("deviceId"), c.Param("channelId"), presetID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) DeletePreset(c *gin.Context) {
	presetID, err := strconv.Atoi(c.Query("presetId"))
	if err != nil {
		response.Error(c, response.CodeBadReq, "presetId 无效")
		return
	}
	if err := h.svc.DeletePreset(c.Param("deviceId"), c.Param("channelId"), presetID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}
