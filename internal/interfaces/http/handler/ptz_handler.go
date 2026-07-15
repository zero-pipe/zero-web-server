package handler

import (
	"strconv"

	ptzapp "zero-web-server/internal/application/ptz"
	"zero-web-server/pkg/response"

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

func (h *PTZHandler) Focus(c *gin.Context) {
	speed, _ := strconv.Atoi(c.DefaultQuery("speed", "50"))
	if err := h.svc.Focus(c.Param("deviceId"), c.Param("channelId"), c.Query("command"), speed); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) Iris(c *gin.Context) {
	speed, _ := strconv.Atoi(c.DefaultQuery("speed", "50"))
	if err := h.svc.Iris(c.Param("deviceId"), c.Param("channelId"), c.Query("command"), speed); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) Wiper(c *gin.Context) {
	if err := h.svc.Wiper(c.Param("deviceId"), c.Param("channelId"), c.Query("command")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) Auxiliary(c *gin.Context) {
	switchID, _ := strconv.Atoi(c.DefaultQuery("switchId", "1"))
	if err := h.svc.Auxiliary(c.Param("deviceId"), c.Param("channelId"), c.Query("command"), switchID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) AddCruisePoint(c *gin.Context) {
	cruiseID, _ := strconv.Atoi(c.Query("cruiseId"))
	presetID, _ := strconv.Atoi(c.Query("presetId"))
	if err := h.svc.AddCruisePoint(c.Param("deviceId"), c.Param("channelId"), cruiseID, presetID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) DeleteCruisePoint(c *gin.Context) {
	cruiseID, _ := strconv.Atoi(c.Query("cruiseId"))
	presetID, _ := strconv.Atoi(c.DefaultQuery("presetId", "0"))
	if err := h.svc.DeleteCruisePoint(c.Param("deviceId"), c.Param("channelId"), cruiseID, presetID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) SetCruiseSpeed(c *gin.Context) {
	cruiseID, _ := strconv.Atoi(c.Query("cruiseId"))
	speed, _ := strconv.Atoi(c.Query("speed"))
	if err := h.svc.SetCruiseSpeed(c.Param("deviceId"), c.Param("channelId"), cruiseID, speed); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) SetCruiseTime(c *gin.Context) {
	cruiseID, _ := strconv.Atoi(c.Query("cruiseId"))
	dwell, _ := strconv.Atoi(c.Query("time"))
	if err := h.svc.SetCruiseTime(c.Param("deviceId"), c.Param("channelId"), cruiseID, dwell); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) StartCruise(c *gin.Context) {
	cruiseID, _ := strconv.Atoi(c.Query("cruiseId"))
	if err := h.svc.StartCruise(c.Param("deviceId"), c.Param("channelId"), cruiseID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) StopCruise(c *gin.Context) {
	cruiseID, _ := strconv.Atoi(c.Query("cruiseId"))
	if err := h.svc.StopCruise(c.Param("deviceId"), c.Param("channelId"), cruiseID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) StartScan(c *gin.Context) {
	scanID, _ := strconv.Atoi(c.Query("scanId"))
	if err := h.svc.StartScan(c.Param("deviceId"), c.Param("channelId"), scanID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) StopScan(c *gin.Context) {
	scanID, _ := strconv.Atoi(c.Query("scanId"))
	if err := h.svc.StopScan(c.Param("deviceId"), c.Param("channelId"), scanID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) SetScanLeft(c *gin.Context) {
	scanID, _ := strconv.Atoi(c.Query("scanId"))
	if err := h.svc.SetScanLeft(c.Param("deviceId"), c.Param("channelId"), scanID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) SetScanRight(c *gin.Context) {
	scanID, _ := strconv.Atoi(c.Query("scanId"))
	if err := h.svc.SetScanRight(c.Param("deviceId"), c.Param("channelId"), scanID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PTZHandler) SetScanSpeed(c *gin.Context) {
	scanID, _ := strconv.Atoi(c.Query("scanId"))
	speed, _ := strconv.Atoi(c.Query("speed"))
	if err := h.svc.SetScanSpeed(c.Param("deviceId"), c.Param("channelId"), scanID, speed); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}
