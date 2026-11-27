// Package channeldrv 封装各接入协议的通道操作，业务层只依赖 ChannelDriver 接口。
package channeldrv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	deviceapp "zero-web-server/internal/application/device"
	ipcdevice "zero-web-server/internal/application/ipcdevice"
	onvifapp "zero-web-server/internal/application/onvif"
	playapp "zero-web-server/internal/application/play"
	domainasset "zero-web-server/internal/domain/asset"
	domaindevice "zero-web-server/internal/domain/device"
	domainptz "zero-web-server/internal/domain/ptz"
	"zero-web-server/internal/dto"
	"zero-web-server/internal/port"
)

// Session 记录停止/控制时恢复协议侧上下文（可为 nil）。
type Session struct {
	Protocol        string
	ProtocolDeviceID  int
	ProtocolChannelID int64
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
	// Capabilities 自报本协议**真实实现**的操作能力（唯一真相）。
	//
	// 前端「能力集」列据此点亮/置灰按钮：channelCaps 模板决定展示哪些图标，
	// 这里返回的集合决定哪些是亮的（实现了）。两者相交即最终效果。
	// 新增能力只在真正实现后往这里加一项，禁止在没实现的驱动里写。
	// 词表：play / ptz / snapshot / playback / record / event / subscribe / catalog / sync。
	Capabilities() []string
}

// 可选能力的标准错误，供不支持该能力的驱动返回，文案直接给前端看。
var (
	ErrPlaybackUnsupported = errors.New("该接入方式暂不支持回放")
	ErrSnapshotUnsupported = errors.New("该接入方式暂不支持抓图")
	ErrPresetUnsupported   = errors.New("该接入方式暂不支持预置位")
)

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

// PresetDriver 是可选的预置位能力（查询/调用/设置/删除）。
//
// 预置位编号在协议间语义不同：国标是通道内数字索引，ONVIF 是设备分配的
// 字符串 token。统一族对外一律用字符串 presetId，由各驱动转换。
// 插件（私有 SDK）当前未实现预置位，显式不实现本接口，服务层据类型断言给准确提示。
type PresetDriver interface {
	QueryPresets(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) ([]domainptz.Preset, error)
	CallPreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error
	AddPreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error
	DeletePreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error
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

// Capabilities 透传给内层驱动：能力真相在内层，壳子只负责加超时/并发保护。
func (l *limited) Capabilities() []string { return l.next.Capabilities() }

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

// ---- 可选接口的转发 ----
//
// 壳子必须转发 PlaybackDriver / SnapshotDriver，否则业务层对 Registry 取出的
// 驱动做 `drv.(PlaybackDriver)` 断言时永远命中不到内层实现（断言对象是壳子）。
// 内层不支持时返回标准错误，让调用方拿到准确文案而不是"不支持"的错误判定。

func (l *limited) QueryRecords(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	startTime, endTime string, streamType int) (json.RawMessage, error) {
	pd, ok := l.next.(PlaybackDriver)
	if !ok {
		return nil, ErrPlaybackUnsupported
	}
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	return pd.QueryRecords(cctx, res, binding, startTime, endTime, streamType)
}

func (l *limited) StartPlayback(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	startTime, endTime, streamKey string) (*dto.StreamContent, *Session, error) {
	pd, ok := l.next.(PlaybackDriver)
	if !ok {
		return nil, nil, ErrPlaybackUnsupported
	}
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer done()
	return pd.StartPlayback(cctx, res, binding, startTime, endTime, streamKey)
}

func (l *limited) StopPlayback(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) error {
	pd, ok := l.next.(PlaybackDriver)
	if !ok {
		return ErrPlaybackUnsupported
	}
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	return pd.StopPlayback(cctx, res, binding)
}

func (l *limited) Snapshot(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) ([]byte, error) {
	sd, ok := l.next.(SnapshotDriver)
	if !ok {
		return nil, ErrSnapshotUnsupported
	}
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	return sd.Snapshot(cctx, res, binding)
}

func (l *limited) QueryPresets(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) ([]domainptz.Preset, error) {
	pd, ok := l.next.(PresetDriver)
	if !ok {
		return nil, ErrPresetUnsupported
	}
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	return pd.QueryPresets(cctx, res, binding)
}

func (l *limited) CallPreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error {
	pd, ok := l.next.(PresetDriver)
	if !ok {
		return ErrPresetUnsupported
	}
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	return pd.CallPreset(cctx, res, binding, presetID)
}

func (l *limited) AddPreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error {
	pd, ok := l.next.(PresetDriver)
	if !ok {
		return ErrPresetUnsupported
	}
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	return pd.AddPreset(cctx, res, binding, presetID)
}

func (l *limited) DeletePreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error {
	pd, ok := l.next.(PresetDriver)
	if !ok {
		return ErrPresetUnsupported
	}
	cctx, done, err := l.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	return pd.DeletePreset(cctx, res, binding, presetID)
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

// clampByte 把整数钳到 [0,255]（SIP 前端控制参数字节）。
func clampByte(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
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

// Capabilities GB 统一驱动目前真实实现的通道能力：直播 + 云台。
// 回放仍走旧 SIP 播放服务、抓图未实现，故不声明——声明了就必须在这里实现，
// 否则前端会点亮一个点下去就报错的按钮。
func (d *gbDriver) Capabilities() []string { return []string{"play", "ptz", "preset"} }

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
	if d.ptz == nil || d.devices == nil {
		return nil, fmt.Errorf("GB 云台服务未启用")
	}
	device, err := d.devices.GetByDeviceID(binding.ExternalID)
	if err != nil || device == nil {
		return nil, fmt.Errorf("设备不存在")
	}
	ch := res.ExternalKey
	switch strings.ToLower(strings.TrimSpace(cmd)) {
	case "focus":
		var p struct {
			Command string `json:"command"`
			Speed   int    `json:"speed"`
		}
		_ = json.Unmarshal(params, &p)
		code, err := gbFocusCmd(p.Command)
		if err != nil {
			return nil, err
		}
		return nil, d.ptz.SendFrontEndCmd(device, ch, code, clampByte(p.Speed), 0, 0)
	case "iris":
		var p struct {
			Command string `json:"command"`
			Speed   int    `json:"speed"`
		}
		_ = json.Unmarshal(params, &p)
		code, err := gbIrisCmd(p.Command)
		if err != nil {
			return nil, err
		}
		return nil, d.ptz.SendFrontEndCmd(device, ch, code, 0, clampByte(p.Speed), 0)
	case "wiper":
		var p struct {
			Command string `json:"command"`
		}
		_ = json.Unmarshal(params, &p)
		code := 0x8c // aux on
		if strings.ToLower(strings.TrimSpace(p.Command)) == "off" {
			code = 0x8d
		}
		return nil, d.ptz.SendFrontEndCmd(device, ch, code, 1, 0, 0)
	case "drag_zoom_in", "drag_zoom_out":
		var p struct {
			Length, Width, MidX, MidY, LengthX, LengthY int
		}
		_ = json.Unmarshal(params, &p)
		zoomIn := strings.EqualFold(cmd, "drag_zoom_in")
		return nil, d.ptz.SendDragZoom(device, ch, zoomIn, p.Length, p.Width, p.MidX, p.MidY, p.LengthX, p.LengthY)
	default:
		return nil, fmt.Errorf("GB28181 接入不支持该命令: %s", cmd)
	}
}

// GB/T 28181 聚焦/光圈命令码。
const (
	gbFiStop      = 0x40
	gbFiFocusFar  = 0x41
	gbFiFocusNear = 0x42
	gbFiIrisOpen  = 0x44
	gbFiIrisClose = 0x48
)

func gbFocusCmd(command string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "near":
		return gbFiFocusNear, nil
	case "far":
		return gbFiFocusFar, nil
	case "stop":
		return gbFiStop, nil
	default:
		return 0, fmt.Errorf("无效聚焦命令: %s", command)
	}
}

func gbIrisCmd(command string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "in", "open":
		return gbFiIrisOpen, nil
	case "out", "close":
		return gbFiIrisClose, nil
	case "stop":
		return gbFiStop, nil
	default:
		return 0, fmt.Errorf("无效光圈命令: %s", command)
	}
}

// GB/T 28181 front-end preset 命令码（附录 A）。
const (
	gbPresetSet    = 0x81
	gbPresetCall   = 0x82
	gbPresetDelete = 0x83
)

func (d *gbDriver) QueryPresets(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) ([]domainptz.Preset, error) {
	if d.ptz == nil || d.devices == nil {
		return nil, fmt.Errorf("GB 云台服务未启用")
	}
	device, err := d.devices.GetByDeviceID(binding.ExternalID)
	if err != nil || device == nil {
		return nil, fmt.Errorf("设备不存在")
	}
	sn, ch := d.ptz.SendPresetQuery(device, res.ExternalKey)
	timeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remain := time.Until(deadline); remain > 0 && remain < timeout {
			timeout = remain
		}
	}
	select {
	case presets := <-ch:
		if presets == nil {
			return []domainptz.Preset{}, nil
		}
		return presets, nil
	case <-ctx.Done():
		d.ptz.CancelPresetQuery(sn)
		return nil, ctx.Err()
	case <-time.After(timeout):
		d.ptz.CancelPresetQuery(sn)
		return nil, fmt.Errorf("预置位查询超时")
	}
}

func (d *gbDriver) presetCmd(res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string, cmdCode int) error {
	if d.ptz == nil || d.devices == nil {
		return fmt.Errorf("GB 云台服务未启用")
	}
	idx, err := strconv.Atoi(strings.TrimSpace(presetID))
	if err != nil || idx < 1 || idx > 255 {
		return fmt.Errorf("预置位编号需在 1-255: %s", presetID)
	}
	device, err := d.devices.GetByDeviceID(binding.ExternalID)
	if err != nil || device == nil {
		return fmt.Errorf("设备不存在")
	}
	return d.ptz.SendFrontEndCmd(device, res.ExternalKey, cmdCode, 0, idx, 0)
}

func (d *gbDriver) CallPreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error {
	return d.presetCmd(res, binding, presetID, gbPresetCall)
}

func (d *gbDriver) AddPreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error {
	return d.presetCmd(res, binding, presetID, gbPresetSet)
}

func (d *gbDriver) DeletePreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error {
	return d.presetCmd(res, binding, presetID, gbPresetDelete)
}

// ---- ONVIF ----

type onvifDriver struct {
	onvif *onvifapp.Service
}

func (d *onvifDriver) Protocol() string { return "onvif" }

// Capabilities ONVIF 统一驱动实现了直播、云台、抓图；回放缺 Profile G 未实现。
func (d *onvifDriver) Capabilities() []string { return []string{"play", "ptz", "snapshot", "preset"} }

func (d *onvifDriver) StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	profile *domainasset.StreamProfile) (*dto.StreamContent, *Session, error) {
	if d.onvif == nil {
		return nil, nil, fmt.Errorf("ONVIF 服务未启用")
	}
	token := ""
	if profile != nil {
		token = profileToken(profile.SelectorJSON)
	}
	content, err := d.onvif.StartPlay(ctx, int64(res.ProtocolChannelID), token)
	if err != nil {
		return nil, nil, err
	}
	sess := &Session{Protocol: d.Protocol(), ProtocolChannelID: int64(res.ProtocolChannelID), ProfileToken: token}
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
	return d.onvif.StopPlay(ctx, int64(res.ProtocolChannelID), token)
}

func (d *onvifDriver) Sync(ctx context.Context, binding *domainasset.AccessBinding) error {
	if d.onvif == nil {
		return fmt.Errorf("ONVIF 服务未启用")
	}
	_, err := d.onvif.SyncChannels(ctx, int64(binding.ProtocolDeviceID))
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
		ChannelID: int64(res.ProtocolChannelID),
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
	return d.onvif.Snapshot(ctx, int64(res.ProtocolChannelID), "")
}

func (d *onvifDriver) QueryPresets(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) ([]domainptz.Preset, error) {
	if d.onvif == nil {
		return nil, fmt.Errorf("ONVIF 服务未启用")
	}
	return d.onvif.QueryPresets(ctx, int64(res.ProtocolChannelID))
}

func (d *onvifDriver) CallPreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error {
	if d.onvif == nil {
		return fmt.Errorf("ONVIF 服务未启用")
	}
	return d.onvif.GotoPreset(ctx, int64(res.ProtocolChannelID), presetID)
}

func (d *onvifDriver) AddPreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error {
	if d.onvif == nil {
		return fmt.Errorf("ONVIF 服务未启用")
	}
	// ONVIF 允许空 token 由设备分配；传 presetID 作 token 时若设备不复用也无妨。
	_, err := d.onvif.SetPreset(ctx, int64(res.ProtocolChannelID), presetID, "")
	return err
}

func (d *onvifDriver) DeletePreset(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, presetID string) error {
	if d.onvif == nil {
		return fmt.Errorf("ONVIF 服务未启用")
	}
	return d.onvif.RemovePreset(ctx, int64(res.ProtocolChannelID), presetID)
}

// ---- 插件（私有 SDK） ----

type pluginDriver struct {
	ipc *ipcdevice.Service
}

func (d *pluginDriver) Protocol() string { return "plugin" }

// Capabilities 插件统一驱动实现了直播、云台、抓图、回放/录像检索。
func (d *pluginDriver) Capabilities() []string {
	return []string{"play", "ptz", "snapshot", "playback", "record"}
}

func (d *pluginDriver) StartLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	profile *domainasset.StreamProfile) (*dto.StreamContent, *Session, error) {
	if d.ipc == nil {
		return nil, nil, fmt.Errorf("插件服务未启用")
	}
	content, err := d.ipc.StartLive(ctx, binding.ProtocolDeviceID, res.ExternalKey, streamKeyOf(profile))
	if err != nil {
		return nil, nil, err
	}
	sess := &Session{Protocol: d.Protocol(), ProtocolDeviceID: binding.ProtocolDeviceID, ChannelKey: res.ExternalKey}
	return content, sess, nil
}

func (d *pluginDriver) StopLive(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, sess *Session) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	devID := 0
	if binding != nil {
		devID = binding.ProtocolDeviceID
	}
	if sess != nil {
		devID = sess.ProtocolDeviceID
	}
	return d.ipc.StopLive(ctx, devID, res.ExternalKey)
}

func (d *pluginDriver) Sync(ctx context.Context, binding *domainasset.AccessBinding) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	_, err := d.ipc.SyncChannels(ctx, binding.ProtocolDeviceID)
	return err
}

func (d *pluginDriver) PTZ(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd PTZCommand) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	speed := int(clamp01(cmd.Speed)*100 + 0.5)
	zoom := int(clamp01(cmd.Zoom)*16 + 0.5)
	return d.ipc.PTZ(ctx, binding.ProtocolDeviceID, res.ExternalKey, cmd.Direction, speed, speed, zoom)
}

func (d *pluginDriver) Command(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding, cmd string, params json.RawMessage) (json.RawMessage, error) {
	if d.ipc == nil {
		return nil, fmt.Errorf("插件服务未启用")
	}
	return d.ipc.Command(ctx, binding.ProtocolDeviceID, res.ExternalKey, cmd, params)
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
	return d.ipc.QueryRecord(ctx, binding.ProtocolDeviceID, res.ExternalKey, startTime, endTime, streamType)
}

func (d *pluginDriver) StartPlayback(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding,
	startTime, endTime, streamKey string) (*dto.StreamContent, *Session, error) {
	if d.ipc == nil {
		return nil, nil, fmt.Errorf("插件服务未启用")
	}
	content, err := d.ipc.StartPlayback(ctx, binding.ProtocolDeviceID, res.ExternalKey, startTime, endTime, streamKey)
	if err != nil {
		return nil, nil, err
	}
	sess := &Session{Protocol: d.Protocol(), ProtocolDeviceID: binding.ProtocolDeviceID, ChannelKey: res.ExternalKey, Playback: true}
	return content, sess, nil
}

func (d *pluginDriver) StopPlayback(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) error {
	if d.ipc == nil {
		return fmt.Errorf("插件服务未启用")
	}
	devID := 0
	if binding != nil {
		devID = binding.ProtocolDeviceID
	}
	return d.ipc.StopPlayback(ctx, devID, res.ExternalKey)
}

func (d *pluginDriver) Snapshot(ctx context.Context, res *domainasset.Resource, binding *domainasset.AccessBinding) ([]byte, error) {
	if d.ipc == nil {
		return nil, fmt.Errorf("插件服务未启用")
	}
	return d.ipc.Snapshot(ctx, binding.ProtocolDeviceID, res.ExternalKey)
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

// streamKeyOf 从 StreamProfile.SelectorJSON 解析插件侧码流选择（main/sub）。
//
// 插件通道的 selector 是 {"channel":"main"} / {"channel":"sub"}。
// 解析不出来时返回空串，由插件按主码流兜底——不要在这里硬编码 "main"，
// 否则 profile 缺失时会把"没选"伪装成"选了主码流"，排查时看不出来。
func streamKeyOf(p *domainasset.StreamProfile) string {
	if p == nil || strings.TrimSpace(p.SelectorJSON) == "" {
		return ""
	}
	var sel struct {
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal([]byte(p.SelectorJSON), &sel); err != nil {
		return ""
	}
	return strings.TrimSpace(sel.Channel)
}
