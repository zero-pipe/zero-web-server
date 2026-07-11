package handler

import (
	"strconv"

	onvifapp "zero-web-kit/internal/application/onvif"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type ONVIFHandler struct {
	svc *onvifapp.Service
}

func NewONVIFHandler(svc *onvifapp.Service) *ONVIFHandler {
	return &ONVIFHandler{svc: svc}
}

func (h *ONVIFHandler) Discover(c *gin.Context) {
	timeout, _ := strconv.Atoi(c.DefaultQuery("timeout", "5"))
	devices, err := h.svc.Discover(c.Request.Context(), timeout)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, devices)
}

func (h *ONVIFHandler) QueryDevices(c *gin.Context) {
	page, count := parsePageCount(c)
	keyword := c.Query("query")
	devices, total, err := h.svc.ListDevices(c.Request.Context(), page, count, keyword)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, gin.H{
		"list":  devices,
		"total": total,
		"page":  page,
		"count": count,
	})
}

func (h *ONVIFHandler) AddDevice(c *gin.Context) {
	var req onvifapp.AddDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// fallback to form params
		req.Name = c.PostForm("name")
		req.IP = c.PostForm("ip")
		req.Port, _ = strconv.Atoi(c.DefaultPostForm("port", "80"))
		req.Username = c.PostForm("username")
		req.Password = c.PostForm("password")
		req.Endpoint = c.PostForm("endpoint")
	}

	device, err := h.svc.AddDevice(c.Request.Context(), req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, device)
}

func (h *ONVIFHandler) DeleteDevice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeBadReq, "无效的设备ID")
		return
	}
	if err := h.svc.DeleteDevice(c.Request.Context(), id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *ONVIFHandler) SyncDevice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeBadReq, "无效的设备ID")
		return
	}
	channels, err := h.svc.SyncChannels(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, channels)
}

func (h *ONVIFHandler) Probe(c *gin.Context) {
	if err := h.svc.ProbeAll(c.Request.Context()); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *ONVIFHandler) QueryChannels(c *gin.Context) {
	page, count := parsePageCount(c)
	deviceID, _ := strconv.ParseInt(c.Query("deviceId"), 10, 64)
	channels, total, err := h.svc.ListChannels(c.Request.Context(), page, count, deviceID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, gin.H{
		"list":  channels,
		"total": total,
		"page":  page,
		"count": count,
	})
}

func (h *ONVIFHandler) StartPlay(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Query("channelId"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeBadReq, "无效的通道ID")
		return
	}
	result, err := h.svc.StartPlay(c.Request.Context(), channelID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *ONVIFHandler) StopPlay(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Query("channelId"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeBadReq, "无效的通道ID")
		return
	}
	if err := h.svc.StopPlay(c.Request.Context(), channelID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *ONVIFHandler) PTZControl(c *gin.Context) {
	var req onvifapp.PTZRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.ChannelID, _ = strconv.ParseInt(c.PostForm("channelId"), 10, 64)
		req.Command = c.PostForm("command")
		speed, _ := strconv.ParseFloat(c.PostForm("speed"), 64)
		req.Speed = speed
	}
	if err := h.svc.PTZControl(c.Request.Context(), req); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *ONVIFHandler) QueryPresets(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Query("channelId"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeBadReq, "无效的通道ID")
		return
	}
	list, err := h.svc.QueryPresets(c.Request.Context(), channelID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *ONVIFHandler) GotoPreset(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Query("channelId"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeBadReq, "无效的通道ID")
		return
	}
	presetID := c.Query("presetId")
	if presetID == "" {
		presetID = c.Query("presetToken")
	}
	if err := h.svc.GotoPreset(c.Request.Context(), channelID, presetID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *ONVIFHandler) SetPreset(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Query("channelId"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeBadReq, "无效的通道ID")
		return
	}
	token, err := h.svc.SetPreset(c.Request.Context(), channelID, c.Query("presetId"), c.Query("presetName"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, gin.H{"presetId": token})
}

func (h *ONVIFHandler) RemovePreset(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Query("channelId"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeBadReq, "无效的通道ID")
		return
	}
	presetID := c.Query("presetId")
	if presetID == "" {
		presetID = c.Query("presetToken")
	}
	if err := h.svc.RemovePreset(c.Request.Context(), channelID, presetID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}
