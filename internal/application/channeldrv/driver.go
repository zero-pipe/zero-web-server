// Package channeldrv 封装各接入协议的通道操作，业务层只依赖 ChannelDriver 接口。
package channeldrv

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	deviceapp "zero-web-server/internal/application/device"
	ipcdevice "zero-web-server/internal/application/ipcdevice"
	onvifapp "zero-web-server/internal/application/onvif"
	playapp "zero-web-server/internal/application/play"
	domainasset "zero-web-server/internal/domain/asset"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/dto"
	"zero-web-server/internal/port"
)

// Session 记录停止/控制时恢复协议侧上下文（可为 nil）。
type Session struct {
	Protocol        string
	LegacyDeviceID  int
	LegacyChannelID int64
	ProfileToken    string
	ChannelKey      string
}

// PTZCommand 归一化的云台指令：速度统一到 [0,1]，方向沿用平台既有取值。
type PTZCommand struct {
	Direction string  // left/right/up/down/upleft/upright/downleft/downright/zoomin/zoomout/stop
	Speed     float64 // 0~1，水平/垂直
	Zoom      float64 // 0~1，变倍
}

// ChannelDriver 一种接入协议的通道操作。
type ChannelDriver interface {
	Protocol() string
	StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
		profile *domainasset.StreamProfile) (*dto.StreamContent, *Session, error)
	StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *Session) error
	Sync(ctx context.Context, binding *domainasset.AccessBinding) error
	PTZ(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd PTZCommand) error
}

// Deps 驱动所需的协议侧实现。
type Deps struct {
	Play    *playapp.Service
	ONVIF   *onvifapp.Service
	Plugin  *ipcdevice.Service
	Device  *deviceapp.Service
	Devices domaindevice.Repository
	PTZ     port.GBPTZNode
}

// Registry 协议 → 驱动。
type Registry map[string]ChannelDriver

// New 构建驱动表；新增协议在此注册。
func New(d Deps) Registry {
	return Registry{
		"gb28181": &gbDriver{play: d.Play, device: d.Device, devices: d.Devices, ptz: d.PTZ},
		"onvif":   &onvifDriver{onvif: d.ONVIF},
		"plugin":  &pluginDriver{ipc: d.Plugin},
	}
}

// Get 按协议取驱动。
func (r Registry) Get(protocol string) (ChannelDriver, bool) {
	d, ok := r[protocol]
	return d, ok
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ---- GB28181 ----

type gbDriver struct {
	play    *playapp.Service
	device  *deviceapp.Service
	devices domaindevice.Repository
	ptz     port.GBPTZNode
}

func (d *gbDriver) Protocol() string { return "gb28181" }

func (d *gbDriver) StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	profile *domainasset.StreamProfile) (*dto.StreamContent, *Session, error) {
	if d.play == nil {
		return nil, nil, fmt.Errorf("GB 播放服务未启用")
	}
	content, err := d.play.StartPlay(ctx, binding.ExternalID, res.ExternalKey)
	if err != nil {
		return nil, nil, err
	}
	return content, nil, nil
}

func (d *gbDriver) StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *Session) error {
	if d.play == nil {
		return fmt.Errorf("GB 播放服务未启用")
	}
	return d.play.StopPlay(binding.ExternalID, res.ExternalKey)
}

func (d *gbDriver) Sync(ctx context.Context, binding *domainasset.AccessBinding) error {
	if d.device == nil {
		return fmt.Errorf("设备服务未启用")
	}
	return d.device.SyncCatalog(binding.ExternalID)
}

func (d *gbDriver) PTZ(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd PTZCommand) error {
	if d.ptz == nil || d.devices == nil {
		return fmt.Errorf("GB 云台服务未启用")
	}
	device, err := d.devices.GetByDeviceID(binding.ExternalID)
	if err != nil || device == nil {
		return fmt.Errorf("设备不存在")
	}
	direction := strings.ToLower(strings.TrimSpace(cmd.Direction))
	speed := int(clamp01(cmd.Speed)*255 + 0.5)
	if speed <= 0 {
		speed = 100
	}
	zoom := int(clamp01(cmd.Zoom)*16 + 0.5)
	if zoom <= 0 {
		zoom = 16
	}
	return d.ptz.SendPTZ(device, res.ExternalKey, direction, speed, speed, zoom)
}

// ---- ONVIF ----

type onvifDriver struct {
	onvif *onvifapp.Service
}

func (d *onvifDriver) Protocol() string { return "onvif" }

func (d *onvifDriver) StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	profile *domainasset.StreamProfile) (*dto.StreamContent, *Session, error) {
	if d.onvif == nil {
		return nil, nil, fmt.Errorf("ONVIF 服务未启用")
	}
	token := ""
	if profile != nil {
		token = profileToken(profile.SelectorJSON)
	}
	content, err := d.onvif.StartPlay(ctx, int64(res.LegacyChannelID), token)
	if err != nil {
		return nil, nil, err
	}
	sess := &Session{Protocol: d.Protocol(), LegacyChannelID: int64(res.LegacyChannelID), ProfileToken: token}
	return &dto.StreamContent{
		App: content.App, Stream: content.Stream, MediaServerID: content.MediaServerID,
		Flv: content.URLs["flv"], WsFlv: content.URLs["ws"], Hls: content.URLs["hls"],
		Rtmp: content.URLs["rtmp"], Rtsp: content.URLs["rtsp"], Rtc: content.URLs["rtc"], Rtcs: content.URLs["rtcs"],
		VideoCodec: content.VideoCodec, AudioCodec: content.AudioCodec,
	}, sess, nil
}

func (d *onvifDriver) StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *Session) error {
	if d.onvif == nil {
		return fmt.Errorf("ONVIF 服务未启用")
	}
	token := ""
	if sess != nil {
		token = sess.ProfileToken
	}
	return d.onvif.StopPlay(ctx, int64(res.LegacyChannelID), token)
}

func (d *onvifDriver) Sync(ctx context.Context, binding *domainasset.AccessBinding) error {
	if d.onvif == nil {
		return fmt.Errorf("ONVIF 服务未启用")
	}
	_, err := d.onvif.SyncChannels(ctx, int64(binding.LegacyDeviceID))
	return err
}

func (d *onvifDriver) PTZ(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd PTZCommand) error {
	if d.onvif == nil {
		return fmt.Errorf("ONVIF 服务未启用")
	}
	direction := strings.ToLower(strings.TrimSpace(cmd.Direction))
	switch direction {
	case "left", "right", "up", "down", "stop":
	default:
		return fmt.Errorf("ONVIF 不支持该云台指令: %s", cmd.Direction)
	}
	return d.onvif.PTZControl(ctx, onvifapp.PTZRequest{
		ChannelID: int64(res.LegacyChannelID),
		Command:   direction,
		Speed:     clamp01(cmd.Speed),
	})
}

// ---- 插件（私有 SDK） ----

type pluginDriver struct {
	ipc *ipcdevice.Service
}

func (d *pluginDriver) Protocol() string { return "plugin" }

func (d *pluginDriver) StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	profile *domainasset.StreamProfile) (*dto.StreamContent, *Session, error) {
	if d.ipc == nil {
		return nil, nil, fmt.Errorf("插件服务未启用")
	}
	content, err := d.ipc.StartLive(ctx, binding.LegacyDeviceID, res.ExternalKey)
	if err != nil {
		return nil, nil, err
	}
	sess := &Session{Protocol: d.Protocol(), LegacyDeviceID: binding.LegacyDeviceID, ChannelKey: res.ExternalKey}
	return content, sess, nil
}

func (d *pluginDriver) StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *Session) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	devID := 0
	if binding != nil {
		devID = binding.LegacyDeviceID
	}
	if sess != nil {
		devID = sess.LegacyDeviceID
	}
	return d.ipc.StopLive(ctx, devID, res.ExternalKey)
}

func (d *pluginDriver) Sync(ctx context.Context, binding *domainasset.AccessBinding) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	_, err := d.ipc.SyncChannels(ctx, binding.LegacyDeviceID)
	return err
}

func (d *pluginDriver) PTZ(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd PTZCommand) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	speed := int(clamp01(cmd.Speed)*100 + 0.5)
	zoom := int(clamp01(cmd.Zoom)*16 + 0.5)
	return d.ipc.PTZ(ctx, binding.LegacyDeviceID, res.ExternalKey, cmd.Direction, speed, speed, zoom)
}

// profileToken 从 StreamProfile.SelectorJSON 解析 ONVIF profileToken。
func profileToken(selectorJSON string) string {
	selectorJSON = strings.TrimSpace(selectorJSON)
	if selectorJSON == "" {
		return ""
	}
	var sel struct {
		ProfileToken string `json:"profileToken"`
	}
	if err := json.Unmarshal([]byte(selectorJSON), &sel); err != nil {
		return ""
	}
	return sel.ProfileToken
}
