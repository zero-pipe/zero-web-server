package handler

import (
	"strconv"

	deviceapp "zero-web-server/internal/application/device"
	domaindevice "zero-web-server/internal/domain/device"
	sipinfra "zero-web-server/internal/infrastructure/sip"
	"zero-web-server/internal/interfaces/http/dto"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	svc *deviceapp.Service
}

func NewDeviceHandler(svc *deviceapp.Service) *DeviceHandler {
	return &DeviceHandler{svc: svc}
}

func (h *DeviceHandler) GetDevice(c *gin.Context) {
	device, err := h.svc.GetByDeviceID(c.Param("deviceId"))
	if err != nil {
		response.Error(c, response.CodeError, "设备不存在")
		return
	}
	response.OK(c, device)
}

func (h *DeviceHandler) ListDevices(c *gin.Context) {
	page, count := parsePageCount(c)
	query := c.Query("query")
	var online *bool
	if status := c.Query("status"); status != "" {
		v := status == "true"
		online = &v
	}
	devices, total, err := h.svc.List(page, count, query, online)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(devices, total, page, count))
}

func (h *DeviceHandler) ListChannels(c *gin.Context) {
	deviceID := c.Param("deviceId")
	page, count := parsePageCount(c)
	query := c.Query("query")
	var online *bool
	if v := c.Query("online"); v != "" {
		b := v == "true"
		online = &b
	}
	channels, total, err := h.svc.ListChannels(deviceID, page, count, query, online)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(dto.NewChannelViews(channels), total, page, count))
}

func (h *DeviceHandler) SyncDevice(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if err := h.svc.SyncCatalog(deviceID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) SyncStatus(c *gin.Context) {
	deviceID := c.Query("deviceId")
	st := h.svc.GetSyncStatus(deviceID)
	response.OK(c, dto.SyncStatus{
		Total:    st.Total,
		Current:  st.Current,
		SyncIng:  st.SyncIng,
		ErrorMsg: st.ErrorMsg,
	})
}

func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	if err := h.svc.Delete(c.Param("deviceId")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) AddDevice(c *gin.Context) {
	var device domaindevice.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.AddDevice(&device); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	var device domaindevice.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.UpdateDevice(&device); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) GetChannel(c *gin.Context) {
	deviceID := c.Query("deviceId")
	channelDeviceID := c.Query("channelDeviceId")
	ch, err := h.svc.GetChannel(deviceID, channelDeviceID)
	if err != nil {
		response.Error(c, response.CodeError, "通道不存在")
		return
	}
	response.OK(c, dto.NewChannelView(ch))
}

func (h *DeviceHandler) SetTransport(c *gin.Context) {
	device, err := h.svc.GetByDeviceID(c.Param("deviceId"))
	if err != nil {
		response.Error(c, response.CodeError, "设备不存在")
		return
	}
	device.StreamMode = sipinfra.NormalizeStreamMode(c.Param("streamMode"))
	if err := h.svc.UpdateDevice(device); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) SubscribeCatalog(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	cycle, _ := strconv.Atoi(c.Query("cycle"))
	if err := h.svc.SubscribeCatalog(id, cycle); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) SubscribeMobilePosition(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	cycle, _ := strconv.Atoi(c.Query("cycle"))
	interval, _ := strconv.Atoi(c.DefaultQuery("interval", "5"))
	if err := h.svc.SubscribeMobilePosition(id, cycle, interval); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) SubscribeAlarm(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	cycle, _ := strconv.Atoi(c.Query("cycle"))
	if err := h.svc.SubscribeAlarm(id, cycle); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) KeepaliveStatistics(c *gin.Context) {
	deviceID := c.Query("deviceId")
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	if deviceID == "" {
		response.OK(c, []domaindevice.TimeStatistics{})
		return
	}
	response.OK(c, h.svc.GetKeepaliveStatistics(deviceID, count))
}

func (h *DeviceHandler) RegisterStatistics(c *gin.Context) {
	deviceID := c.Query("deviceId")
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	if deviceID == "" {
		response.OK(c, []domaindevice.TimeStatistics{})
		return
	}
	response.OK(c, h.svc.GetRegisterStatistics(deviceID, count))
}

func (h *DeviceHandler) ChangeChannelAudio(c *gin.Context) {
	channelID, _ := strconv.Atoi(c.PostForm("channelId"))
	if channelID == 0 {
		channelID, _ = strconv.Atoi(c.Query("channelId"))
	}
	audio := c.PostForm("audio") == "true" || c.Query("audio") == "true"
	if err := h.svc.ChangeChannelAudio(channelID, audio); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func parseIntQuery(c *gin.Context, key string, def int) int {
	v, err := strconv.Atoi(c.DefaultQuery(key, strconv.Itoa(def)))
	if err != nil {
		return def
	}
	return v
}
