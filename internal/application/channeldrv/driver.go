// Package channeldrv 封装各接入协议的通道操作，业务层只依赖 ChannelDriver 接口。
package channeldrv

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

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
	// Playback 标记这是回放会话，避免 StopLive 误关同名直播流。
	Playback bool
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
	// Command 通用命令透传（自定义能力）；不支持的协议返回错误。平台不解析 params/result。
	Command(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd string, params json.RawMessage) (json.RawMessage, error)
}

// PlaybackDriver 是可选的「按时间回放」能力。
//
// 单独拆成一个接口而不是塞进 Driver：三类协议的回放实现路径差异太大
// （国标走 INVITE、插件走 RTSP 拉流、ONVIF 需要 Profile G），
// 让不支持回放的驱动显式不实现，服务层好据此给出准确提示，
// 而不是给一个空实现让调用方以为支持。
type PlaybackDriver interface {
	// QueryRecords 检索设备侧录像，返回插件/协议原始结构，平台不解析。
	QueryRecords(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
		startTime, endTime string, streamType int) (json.RawMessage, error)
	// StartPlayback 让协议侧给出回放流源，由 ZMS 拉成 <设备内码>_<通道内码>_playback。
	StartPlayback(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
		startTime, endTime, streamKey string) (*dto.StreamContent, *Session, error)
	StopPlayback(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) error
}

// SnapshotDriver 是可选的抓图能力，返回 JPEG 原始字节。
//
// 能力位里 `snapshot` 是所有视频通道都会声明的，但声明不等于能抓：
// 抓图必须走协议自己的通道（ONVIF 抓图地址 / SDK 抓图接口），
// 从直播流里截帧在没人在看的时候是拿不到的。
type SnapshotDriver interface {
	Snapshot(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) ([]byte, error)
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

// 外部设备调用统一保护：单次调用超时 + 全局并发上限。
const (
	defaultCallTimeout   = 30 * time.Second
	defaultMaxConcurrent = 32
)

// New 构建驱动表；新增协议在此注册。
func New(d Deps) Registry {
	return Registry{
		"gb28181": newLimited(&gbDriver{play: d.Play, device: d.Device, devices: d.Devices, ptz: d.PTZ}),
		"onvif":   newLimited(&onvifDriver{onvif: d.ONVIF}),
		"plugin":  newLimited(&pluginDriver{ipc: d.Plugin}),
	}
}

// limited 给驱动加超时与并发信号量，避免烂固件拖垮调用方。
type limited struct {
	next    ChannelDriver
	timeout time.Duration
	sem     chan struct{}
}

func newLimited(next ChannelDriver) *limited {
	return &limited{next: next, timeout: defaultCallTimeout, sem: make(chan struct{}, defaultMaxConcurrent)}
}

func (l *limited) begin(ctx context.Context) (context.Context, func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, l.timeout)
	select {
	case l.sem <- struct{}{}:
		return cctx, func() { <-l.sem; cancel() }, nil
	case <-cctx.Done():
		cancel()
		return nil, nil, fmt.Errorf("设备调用繁忙或超时: %w", cctx.Err())
	}
}

func (l *limited) Protocol() string { return l.next.Protocol() }

func (l *limited) StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	profile *domainasset.StreamProfile) (*dto.StreamContent, *Session, error) {
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer done()
	return l.next.StartLive(cctx, res, binding, profile)
}

func (l *limited) StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *Session) error {
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	return l.next.StopLive(cctx, res, binding, sess)
}

func (l *limited) Sync(ctx context.Context, binding *domainasset.AccessBinding) error {
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	return l.next.Sync(cctx, binding)
}

func (l *limited) PTZ(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd PTZCommand) error {
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	return l.next.PTZ(cctx, res, binding, cmd)
}

func (l *limited) Command(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd string, params json.RawMessage) (json.RawMessage, error) {
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	return l.next.Command(cctx, res, binding, cmd, params)
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

func (d *gbDriver) Command(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd string, params json.RawMessage) (json.RawMessage, error) {
	return nil, fmt.Errorf("GB28181 接入不支持自定义命令: %s", cmd)
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
	case "left", "right", "up", "down",
		"upleft", "upright", "downleft", "downright",
		"zoomin", "zoomout", "stop":
	default:
		return fmt.Errorf("ONVIF 不支持该云台指令: %s", cmd.Direction)
	}
	speed := clamp01(cmd.Speed)
	// 变倍用的是独立的速度分量；只按水平速度算会让变倍要么不动要么满速。
	if direction == "zoomin" || direction == "zoomout" {
		if z := clamp01(cmd.Zoom); z > 0 {
			speed = z
		}
	}
	return d.onvif.PTZControl(ctx, onvifapp.PTZRequest{
		ChannelID: int64(res.LegacyChannelID),
		Command:   direction,
		Speed:     speed,
	})
}

func (d *onvifDriver) Command(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd string, params json.RawMessage) (json.RawMessage, error) {
	return nil, fmt.Errorf("ONVIF 接入不支持自定义命令: %s", cmd)
}

func (d *onvifDriver) Snapshot(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) ([]byte, error) {
	if d.onvif == nil {
		return nil, fmt.Errorf("ONVIF 服务未启用")
	}
	return d.onvif.Snapshot(ctx, int64(res.LegacyChannelID), "")
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

func (d *pluginDriver) Command(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd string, params json.RawMessage) (json.RawMessage, error) {
	if d.ipc == nil {
		return nil, fmt.Errorf("插件服务未启用")
	}
	return d.ipc.Command(ctx, binding.LegacyDeviceID, res.ExternalKey, cmd, params)
}

// ---- 插件回放（PlaybackDriver） ----

// pluginPlaybackStreamType 是检索录像时的码流筛选默认值：0 = 同段只返一种（主>子>三）。
const pluginPlaybackStreamType = 0

func (d *pluginDriver) QueryRecords(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	startTime, endTime string, streamType int) (json.RawMessage, error) {
	if d.ipc == nil {
		return nil, fmt.Errorf("插件服务未启用")
	}
	if streamType < 0 {
		streamType = pluginPlaybackStreamType
	}
	return d.ipc.QueryRecord(ctx, binding.LegacyDeviceID, res.ExternalKey, startTime, endTime, streamType)
}

func (d *pluginDriver) StartPlayback(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	startTime, endTime, streamKey string) (*dto.StreamContent, *Session, error) {
	if d.ipc == nil {
		return nil, nil, fmt.Errorf("插件服务未启用")
	}
	content, err := d.ipc.StartPlayback(ctx, binding.LegacyDeviceID, res.ExternalKey, startTime, endTime, streamKey)
	if err != nil {
		return nil, nil, err
	}
	sess := &Session{Protocol: d.Protocol(), LegacyDeviceID: binding.LegacyDeviceID, ChannelKey: res.ExternalKey, Playback: true}
	return content, sess, nil
}

func (d *pluginDriver) StopPlayback(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	devID := 0
	if binding != nil {
		devID = binding.LegacyDeviceID
	}
	return d.ipc.StopPlayback(ctx, devID, res.ExternalKey)
}

func (d *pluginDriver) Snapshot(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) ([]byte, error) {
	if d.ipc == nil {
		return nil, fmt.Errorf("插件服务未启用")
	}
	return d.ipc.Snapshot(ctx, binding.LegacyDeviceID, res.ExternalKey)
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
