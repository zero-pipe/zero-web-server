package handler

import (
	"strconv"
	"strings"

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
	if code := strings.TrimSpace(c.Query("internalCode")); code != "" {
		device, err := h.svc.GetByInternalCode(code)
		if err != nil {
			response.Error(c, response.CodeError, "设备不存在")
			return
		}
		response.OK(c, device)
		return
	}
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

// Guard GET /api/device/control/guard?deviceId=&guardCmd=SetGuard|ResetGuard
func (h *DeviceHandler) Guard(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	guardCmd := strings.TrimSpace(c.Query("guardCmd"))
	if deviceID == "" || guardCmd == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 guardCmd 不能为空")
		return
	}
	if err := h.svc.Guard(deviceID, guardCmd); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

// Record GET /api/device/control/record?deviceId=&channelId=&recordCmdStr=Record|StopRecord
func (h *DeviceHandler) Record(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	recordCmd := strings.TrimSpace(c.Query("recordCmdStr"))
	if deviceID == "" || recordCmd == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 recordCmdStr 不能为空")
		return
	}
	if err := h.svc.Record(deviceID, channelID, recordCmd); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

// DragZoomIn GET /api/device/control/drag_zoom/zoom_in
func (h *DeviceHandler) DragZoomIn(c *gin.Context) {
	h.dragZoom(c, true)
}

// DragZoomOut GET /api/device/control/drag_zoom/zoom_out
func (h *DeviceHandler) DragZoomOut(c *gin.Context) {
	h.dragZoom(c, false)
}

func (h *DeviceHandler) dragZoom(c *gin.Context, zoomIn bool) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 channelId 不能为空")
		return
	}
	length, _ := strconv.Atoi(c.Query("length"))
	width, _ := strconv.Atoi(c.Query("width"))
	midX, _ := strconv.Atoi(c.Query("midPointX"))
	midY, _ := strconv.Atoi(c.Query("midPointY"))
	lengthX, _ := strconv.Atoi(c.Query("lengthX"))
	lengthY, _ := strconv.Atoi(c.Query("lengthY"))
	if err := h.svc.DragZoom(deviceID, channelID, zoomIn, length, width, midX, midY, lengthX, lengthY); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

// HomePosition GET /api/device/control/home_position
func (h *DeviceHandler) HomePosition(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 不能为空")
		return
	}
	enabled, _ := strconv.Atoi(c.DefaultQuery("enabled", "1"))
	resetTime, _ := strconv.Atoi(c.DefaultQuery("resetTime", "0"))
	presetIndex, _ := strconv.Atoi(c.DefaultQuery("presetIndex", "1"))
	if err := h.svc.HomePosition(deviceID, channelID, enabled, resetTime, presetIndex); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

// QueryBasicParam GET /api/device/config/query/basicParam
func (h *DeviceHandler) QueryBasicParam(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	if deviceID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 不能为空")
		return
	}
	data, err := h.svc.QueryBasicParam(deviceID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, data)
}

// SetBasicParam GET /api/device/config/set/basicParam
func (h *DeviceHandler) SetBasicParam(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	if deviceID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 不能为空")
		return
	}
	name := c.Query("name")
	expiration, _ := strconv.Atoi(c.DefaultQuery("expiration", "3600"))
	heartBeatInterval, _ := strconv.Atoi(c.DefaultQuery("heartBeatInterval", "60"))
	heartBeatCount, _ := strconv.Atoi(c.DefaultQuery("heartBeatCount", "3"))
	positionCapability, _ := strconv.Atoi(c.DefaultQuery("positionCapability", "0"))
	if err := h.svc.SetBasicParam(deviceID, name, expiration, heartBeatInterval, heartBeatCount, positionCapability); err != nil {
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
