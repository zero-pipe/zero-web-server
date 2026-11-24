package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	ipcdevice "zero-web-server/internal/application/ipcdevice"
	pluginapp "zero-web-server/internal/application/plugin"
	domainplugin "zero-web-server/internal/domain/plugin"
	applog "zero-web-server/pkg/log"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type PluginHandler struct {
	plugins *pluginapp.Service
	ipc     *ipcdevice.Service
	assets  assetPruner
	project assetProjector
}

// assetProjector 把 legacy 插件设备/通道投影到统一资产模型（由 asset.Service 实现）。
type assetProjector interface {
	Backfill() error
}

func NewPluginHandler(plugins *pluginapp.Service, ipc *ipcdevice.Service, assets assetPruner) *PluginHandler {
	h := &PluginHandler{plugins: plugins, ipc: ipc, assets: assets}
	if p, ok := assets.(assetProjector); ok {
		h.project = p
	}
	return h
}

func (h *PluginHandler) List(c *gin.Context) {
	list, err := h.plugins.List()
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *PluginHandler) ListDevices(c *gin.Context) {
	list, err := h.ipc.ListDevices()
	if err != nil {
		response.FromError(c, err)
		return
	}
	out := make([]*pluginDeviceView, 0, len(list))
	for _, d := range list {
		out = append(out, newPluginDeviceView(d))
	}
	response.OK(c, gin.H{"list": out, "total": len(out)})
}

type pluginDeviceView struct {
	*domainplugin.Device
	DriverParams string `json:"driverParams"` // 覆盖为脱敏后的 JSON
}

func newPluginDeviceView(d *domainplugin.Device) *pluginDeviceView {
	if d == nil {
		return nil
	}
	cp := *d
	cp.DriverParams = maskSecrets(d.DriverParams)
	return &pluginDeviceView{Device: &cp, DriverParams: cp.DriverParams}
}

func (h *PluginHandler) Channels(c *gin.Context) {
	deviceID, _ := strconv.Atoi(c.Query("deviceId"))
	if deviceID <= 0 {
		response.Error(c, response.CodeBadReq, "deviceId 无效")
		return
	}
	list, err := h.ipc.ListChannels(deviceID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

type createIPCDeviceRequest struct {
	PluginID string         `json:"pluginId"`
	Protocol string         `json:"protocol"`
	Name     string         `json:"name"`
	Params   map[string]any `json:"params"`
}

func (h *PluginHandler) CreateDevice(c *gin.Context) {
	var req createIPCDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	dev, channels, err := h.ipc.AddDevice(c.Request.Context(), req.PluginID, req.Protocol, req.Name, req.Params)
	if err != nil {
		response.FromError(c, err)
		return
	}
	if err := h.projectAssets(); err != nil {
		applog.Warn("plugin asset projection failed after add device", "err", err)
	}
	response.OK(c, gin.H{"device": newPluginDeviceView(dev), "channels": channels})
}

func (h *PluginHandler) DeleteDevice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	if id <= 0 {
		response.Error(c, response.CodeBadReq, "id 无效")
		return
	}
	if err := h.ipc.DeleteDevice(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	if h.assets != nil {
		_ = h.assets.Prune()
	}
	response.OK(c, nil)
}

func (h *PluginHandler) SyncDevice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	if id <= 0 {
		response.Error(c, response.CodeBadReq, "id 无效")
		return
	}
	channels, err := h.ipc.SyncChannels(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	if err := h.projectAssets(); err != nil {
		applog.Warn("plugin asset projection failed after sync device", "err", err)
	}
	response.OK(c, gin.H{"channels": channels})
}

// projectAssets 把插件设备/通道同步到统一资产（设备资产、事件映射、播放都依赖它）。
func (h *PluginHandler) projectAssets() error {
	if h.project == nil {
		return nil
	}
	return h.project.Backfill()
}

func (h *PluginHandler) StartLive(c *gin.Context) {
	deviceID, _ := strconv.Atoi(c.Query("deviceId"))
	channelID := c.Query("channelId")
	if deviceID <= 0 || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId/channelId 无效")
		return
	}
	content, err := h.ipc.StartLive(c.Request.Context(), deviceID, channelID, c.Query("stream"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, content)
}

func (h *PluginHandler) StopLive(c *gin.Context) {
	deviceID, _ := strconv.Atoi(c.Query("deviceId"))
	channelID := c.Query("channelId")
	if deviceID <= 0 || channelID == "" {
		response.Error(c, response.CodeBadReq, "deviceId/channelId 无效")
		return
	}
	if err := h.ipc.StopLive(c.Request.Context(), deviceID, channelID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

type ptzRequest struct {
	DeviceID  int    `json:"deviceId"`
	ChannelID string `json:"channelId"`
	Direction string `json:"direction"`
	H         int    `json:"h"`
	V         int    `json:"v"`
	Z         int    `json:"z"`
}

func (h *PluginHandler) PTZ(c *gin.Context) {
	var req ptzRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.ipc.PTZ(c.Request.Context(), req.DeviceID, req.ChannelID, req.Direction, req.H, req.V, req.Z); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// maskSecrets 脱敏 driver_params 中的密码/密钥字段。
func maskSecrets(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return raw
	}
	masked := false
	for k := range m {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "pass") || strings.Contains(lk, "secret") || strings.Contains(lk, "token") {
			m[k] = "******"
			masked = true
		}
	}
	if !masked {
		return raw
	}
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return string(out)
}
