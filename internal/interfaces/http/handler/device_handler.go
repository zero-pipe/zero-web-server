package handler

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	deviceapp "zero-web-server/internal/application/device"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/dto"
	sipinfra "zero-web-server/internal/infrastructure/sip"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	svc    *deviceapp.Service
	assets assetPruner
}

func NewDeviceHandler(svc *deviceapp.Service) *DeviceHandler {
	return &DeviceHandler{svc: svc}
}

// WithAssetPruner 注入统一资产清理器，删除设备后回收残留资产。
func (h *DeviceHandler) WithAssetPruner(p assetPruner) *DeviceHandler {
	h.assets = p
	return h
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
		response.FromError(c, err)
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
		response.FromError(c, err)
		return
	}
	response.OK(c, dto.NewPageInfo(dto.NewChannelViews(channels), total, page, count))
}

func (h *DeviceHandler) SyncDevice(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if err := h.svc.SyncCatalog(deviceID); err != nil {
		response.FromError(c, err)
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
		response.FromError(c, err)
		return
	}
	if h.assets != nil {
		_ = h.assets.Prune() // 清理该设备在统一资产中的残留
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
		response.FromError(c, err)
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
		response.FromError(c, err)
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
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) SubscribeCatalog(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	cycle, _ := strconv.Atoi(c.Query("cycle"))
	if err := h.svc.SubscribeCatalog(id, cycle); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) SubscribeMobilePosition(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	cycle, _ := strconv.Atoi(c.Query("cycle"))
	interval, _ := strconv.Atoi(c.DefaultQuery("interval", "5"))
	if err := h.svc.SubscribeMobilePosition(id, cycle, interval); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) SubscribeAlarm(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	cycle, _ := strconv.Atoi(c.Query("cycle"))
	if err := h.svc.SubscribeAlarm(id, cycle); err != nil {
		response.FromError(c, err)
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
		response.FromError(c, err)
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
		response.FromError(c, err)
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
		response.FromError(c, err)
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
		response.FromError(c, err)
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
		response.FromError(c, err)
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
		response.FromError(c, err)
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
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// ChangeChannelStreamIdentification 设置/重置通道码流类型（主/子码流）。
func (h *DeviceHandler) ChangeChannelStreamIdentification(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	if id == 0 {
		id, _ = strconv.Atoi(c.PostForm("id"))
	}
	value := c.Query("streamIdentification")
	if value == "" {
		value = c.PostForm("streamIdentification")
	}
	deviceKey := c.Query("deviceDbId")
	if deviceKey == "" {
		deviceKey = c.PostForm("deviceDbId")
	}
	if err := h.svc.ChangeChannelStreamIdentification(id, deviceKey, value); err != nil {
		response.FromError(c, err)
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

// AlarmReset GET /api/device/control/alarm_reset?deviceId=&channelId=
func (h *DeviceHandler) AlarmReset(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 不能为空")
		return
	}
	if err := h.svc.AlarmReset(deviceID, channelID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// TeleBoot GET /api/device/control/teleboot?deviceId=&channelId=
func (h *DeviceHandler) TeleBoot(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 不能为空")
		return
	}
	if err := h.svc.TeleBoot(deviceID, channelID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// IFrame GET /api/device/control/iframe?deviceId=&channelId=
func (h *DeviceHandler) IFrame(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 不能为空")
		return
	}
	if err := h.svc.IFrame(deviceID, channelID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// DeviceStatus GET /api/device/query/devices/:deviceId/status
func (h *DeviceHandler) DeviceStatus(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Param("deviceId"))
	if deviceID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 不能为空")
		return
	}
	st, err := h.svc.QueryDeviceStatus(deviceID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, st)
}

// HomePositionQuery GET /api/device/query/home_position?deviceId=&channelId=
func (h *DeviceHandler) HomePositionQuery(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 channelId 不能为空")
		return
	}
	v, err := h.svc.QueryHomePosition(deviceID, channelID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, v)
}

// CruiseTrackListQuery GET /api/device/query/cruise_track_list?deviceId=&channelId=
func (h *DeviceHandler) CruiseTrackListQuery(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 channelId 不能为空")
		return
	}
	v, err := h.svc.QueryCruiseTrackList(deviceID, channelID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, v)
}

// CruiseTrackQuery GET /api/device/query/cruise_track?deviceId=&channelId=&number=
func (h *DeviceHandler) CruiseTrackQuery(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	number, _ := strconv.Atoi(strings.TrimSpace(c.Query("number")))
	if deviceID == "" || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 channelId 不能为空")
		return
	}
	v, err := h.svc.QueryCruiseTrack(deviceID, channelID, number)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, v)
}

// PTZPositionQuery GET /api/device/query/ptz_position?deviceId=&channelId=
func (h *DeviceHandler) PTZPositionQuery(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 channelId 不能为空")
		return
	}
	v, err := h.svc.QueryPTZPosition(deviceID, channelID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, v)
}

// SDCardStatusQuery GET /api/device/query/sdcard_status?deviceId=&channelId=
func (h *DeviceHandler) SDCardStatusQuery(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 channelId 不能为空")
		return
	}
	v, err := h.svc.QuerySDCardStatus(deviceID, channelID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, v)
}

// PTZPrecise GET /api/device/control/ptz_precise?deviceId=&channelId=&pan=&tilt=&zoom=
func (h *DeviceHandler) PTZPrecise(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	pan, _ := strconv.ParseFloat(c.Query("pan"), 64)
	tilt, _ := strconv.ParseFloat(c.Query("tilt"), 64)
	zoom, _ := strconv.ParseFloat(c.Query("zoom"), 64)
	if deviceID == "" || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 channelId 不能为空")
		return
	}
	if err := h.svc.PTZPrecise(deviceID, channelID, pan, tilt, zoom); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// FormatSDCard GET /api/device/control/sdcard_format?deviceId=&channelId=
func (h *DeviceHandler) FormatSDCard(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	if deviceID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 不能为空")
		return
	}
	if err := h.svc.FormatSDCard(deviceID, channelID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// TargetTrack GET /api/device/control/target_track?deviceId=&channelId=&trackType=
func (h *DeviceHandler) TargetTrack(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	trackType := strings.TrimSpace(c.Query("trackType"))
	if deviceID == "" || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 channelId 不能为空")
		return
	}
	if err := h.svc.TargetTrack(deviceID, channelID, trackType); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// SnapShotConfig POST /api/device/query/snap/config?deviceId=&channelId=&snapNum=&interval=&uploadUrl=&sessionId=
// 下发图像抓拍配置；未给 uploadUrl 时使用当前服务地址生成。
func (h *DeviceHandler) SnapShotConfig(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	channelID := strings.TrimSpace(c.Query("channelId"))
	snapNum, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("snapNum", "1")))
	interval, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("interval", "1")))
	if deviceID == "" || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId 与 channelId 不能为空")
		return
	}
	sessionID := strings.TrimSpace(c.Query("sessionId"))
	if sessionID == "" {
		sessionID = fmt.Sprintf("snap-%d", time.Now().UnixNano())
	}
	uploadURL := strings.TrimSpace(c.Query("uploadUrl"))
	if uploadURL == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		uploadURL = fmt.Sprintf("%s://%s/api/v1/gb/snapshot/upload?deviceId=%s&sessionId=%s",
			scheme, c.Request.Host, url.QueryEscape(deviceID), url.QueryEscape(sessionID))
	}
	id, err := h.svc.SendSnapShot(deviceID, channelID, snapNum, interval, uploadURL, sessionID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"sessionId": id, "uploadUrl": uploadURL})
}
