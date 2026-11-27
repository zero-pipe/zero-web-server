package ipcdevice

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	pluginapp "zero-web-server/internal/application/plugin"
	domainplugin "zero-web-server/internal/domain/plugin"
	"zero-web-server/internal/dto"
	"zero-web-server/internal/mediaurl"
	"zero-web-server/internal/port"
	applog "zero-web-server/pkg/log"
)

var streamApp = mediaurl.LiveApp

type Service struct {
	plugins  *pluginapp.Service
	repo     domainplugin.Repository
	media    port.MediaCluster
	factory  port.IPCClientFactory
	serverID string
	keepalive KeepalivePusher

	mu     sync.Mutex
	active map[string]*activeStream
}

// KeepalivePusher 记录设备保活时间戳（Redis 实现；未注入则为空操作）。
// 插件设备没有协议层心跳，由周期探活（ProbeAll）探测到可达时写入。
type KeepalivePusher interface {
	PushKeepalive(ctx context.Context, deviceID string, ts int64) error
}

type activeStream struct {
	App       string
	Stream    string
	NodeID    string
	PluginID  string
	DeviceUID string
	ChannelID string
}

func NewService(plugins *pluginapp.Service, repo domainplugin.Repository, media port.MediaCluster, factory port.IPCClientFactory, serverID string) *Service {
	return &Service{
		plugins: plugins, repo: repo, media: media, factory: factory,
		serverID: serverID, active: map[string]*activeStream{},
	}
}

// AddDevice 登录设备、同步通道并入库。
func (s *Service) AddDevice(ctx context.Context, pluginID, protocol, name string, params map[string]any) (*domainplugin.Device, []*domainplugin.Channel, error) {
	p, err := s.resolvePlugin(pluginID, protocol)
	if err != nil {
		return nil, nil, err
	}
	deviceUID := strings.TrimSpace(stringParam(params, "device_uid"))
	if deviceUID == "" {
		deviceUID = deriveUID(p.PluginID, params)
	}
	if existing, err := s.repo.GetDeviceByUID(p.ID, deviceUID); err == nil && existing != nil {
		return nil, nil, fmt.Errorf("设备已存在: %s", deviceUID)
	}
	client := s.factory.For(p.Endpoint)
	info, err := client.Login(ctx, deviceUID, params)
	if err != nil {
		s.plugins.MarkOffline(p.ID)
		return nil, nil, fmt.Errorf("插件登录设备失败: %w", err)
	}
	channels, err := client.ListChannels(ctx, deviceUID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取通道失败: %w", err)
	}
	raw, _ := json.Marshal(params)
	dev := &domainplugin.Device{
		PluginID: p.ID, Protocol: firstNonEmpty(protocol, firstString(p.Protocols)),
		DeviceUID: deviceUID, Name: strings.TrimSpace(name), Enabled: true, Online: true,
		DriverParams: string(raw),
	}
	if dev.Name == "" {
		dev.Name = deviceUID
	}
	if info != nil {
		dev.Manufacturer, dev.Model, dev.Firmware, dev.SerialNumber = info.Manufacturer, info.Model, info.Firmware, info.SerialNumber
	}
	if err := s.repo.CreateDevice(dev); err != nil {
		return nil, nil, err
	}
	if err := s.repo.ReplaceChannels(dev.ID, toDomainChannels(channels)); err != nil {
		return nil, nil, err
	}
	saved, _ := s.repo.GetDevice(dev.ID)
	list, _ := s.repo.ListChannels(dev.ID)
	return saved, list, nil
}

func (s *Service) ListDevices() ([]*domainplugin.Device, error) { return s.repo.ListDevices() }

// CountByPlugin 统计插件设备数，用于插件卸载前的引用检查。
func (s *Service) CountByPlugin(pluginID string) (int, error) {
	return s.repo.CountDevicesByPlugin(pluginID)
}

func (s *Service) ListChannels(deviceID int) ([]*domainplugin.Channel, error) {
	return s.repo.ListChannels(deviceID)
}

func (s *Service) DeleteDevice(ctx context.Context, deviceID int) error {
	dev, err := s.repo.GetDevice(deviceID)
	if err == nil && dev != nil {
		if p, perr := s.repo.GetPlugin(dev.PluginID); perr == nil {
			_ = s.factory.For(p.Endpoint).Logout(ctx, dev.DeviceUID)
		}
	}
	return s.repo.DeleteDevice(deviceID)
}

func (s *Service) SyncChannels(ctx context.Context, deviceID int) ([]*domainplugin.Channel, error) {
	dev, p, client, err := s.deviceClient(deviceID)
	if err != nil {
		return nil, err
	}
	channels, err := client.ListChannels(ctx, dev.DeviceUID)
	if err != nil {
		if isNotLoggedIn(err) {
			// 插件重启后会话丢失：补登录后重试一次。
			if e := s.EnsureLogin(ctx, deviceID); e != nil {
				return nil, err
			}
			channels, err = client.ListChannels(ctx, dev.DeviceUID)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := s.repo.ReplaceChannels(deviceID, toDomainChannels(channels)); err != nil {
		return nil, err
	}
	_ = p
	return s.repo.ListChannels(deviceID)
}

// StartLive 让插件给出流源，再由平台交给 ZMS 拉流并生成播放地址。
// StartLive 起一路插件直播流。
//
// streamKey 是码流选择（main/sub），空则走主码流。它是"选哪条流"，
// 与 ZMS 侧流名（<设备内码>_<通道内码>）是两回事，不要混用。
func (s *Service) StartLive(ctx context.Context, deviceID int, channelID, streamKey string) (*dto.StreamContent, error) {
	dev, p, client, err := s.deviceClient(deviceID)
	if err != nil {
		return nil, err
	}
	if !dev.Enabled {
		return nil, fmt.Errorf("设备已禁用")
	}
	stream, err := s.streamName(dev, channelID)
	if err != nil {
		return nil, err
	}
	// active 只说明「平台认为这路流活着」。ZMS 侧可能已经没了（拉流被对端断开、
	// 媒体节点重启）或卡死（连着但不来帧）。直接返回旧地址会让播放器一直停在
	// “加载中”，所以先探活，不可用就清干净重建。
	if act := s.getActive(stream); act != nil {
		if s.activeStreamHealthy(ctx, act.NodeID, act.App, act.Stream) {
			return s.buildContent(ctx, stream)
		}
		s.dropStaleStream(ctx, act.NodeID, act.App, act.Stream)
	}
	node, err := s.media.ResolveForStream(ctx, streamApp, stream, "")
	if err != nil {
		return nil, err
	}
	transport := pickTransport(p.StreamTransports)
	var push *port.IPCPushParams
	if transport == "rtp_push" {
		rtpPort, perr := node.OpenRtpServer(ctx, streamApp, stream, 0, 0)
		if perr != nil {
			return nil, fmt.Errorf("打开 RTP 端口失败: %w", perr)
		}
		push = &port.IPCPushParams{SDPIP: node.SDPIP(), RTPPort: rtpPort, Mode: "UDP"}
	}
	src, err := client.StartLive(ctx, dev.DeviceUID, channelID, streamKey, transport, push)
	if err != nil {
		s.plugins.MarkOffline(p.ID)
		return nil, fmt.Errorf("插件开始取流失败: %w", err)
	}
	actual := transport
	if src != nil && strings.TrimSpace(src.Transport) != "" {
		actual = src.Transport
	}
	switch actual {
	case "rtp_push":
		// ZMS 已在 OpenRtpServer 时开始收流，无需拉流代理。
	case "rtsp_url", "":
		if src == nil || strings.TrimSpace(src.RTSPURL) == "" {
			return nil, fmt.Errorf("插件未返回可用的 RTSP 流源")
		}
		// ZMS 里可能已有这路流（平台重启后内存 active 丢失、或多客户端同时点播）。
		// 此时再 AddStreamProxy 会让 ZMS 踢掉现有发布者并清空 GOP 缓存，正在观看的
		// 客户端会被断流。已有且可用就直接接管；卡死的旧流先清掉再重建。
		if probe := node.LookupStream(ctx, streamApp, stream); probe != nil && !streamStalled(probe) {
			applog.Infof("[ipc] 复用 ZMS 已有直播流: app=%s stream=%s", streamApp, stream)
		} else {
			if probe != nil {
				_ = node.CloseStreams(ctx, "__defaultVhost__", streamApp, stream)
			}
			if _, err := node.AddStreamProxy(ctx, "__defaultVhost__", streamApp, stream, src.RTSPURL, "1", false, false, true); err != nil {
				return nil, fmt.Errorf("创建拉流代理失败: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("不支持的流传输方式: %s", actual)
	}
	s.media.BindStream(node.ID(), streamApp, stream)
	s.setActive(stream, &activeStream{
		App: streamApp, Stream: stream, NodeID: node.ID(),
		PluginID: dev.PluginID, DeviceUID: dev.DeviceUID, ChannelID: channelID,
	})
	// 等 ZMS 真正收到首帧再返回播放地址：插件拉流有启动延迟（RTSP 握手、SPS 解析），
	// 直接返回会让播放器“有时黑屏、有时马上有画面”。与国标侧 waitGBStreamReady 一致。
	if info := s.waitLiveReady(ctx, node, streamApp, stream, pluginFirstFrameWait); info != nil {
		applog.Infof("[ipc] 直播就绪 stream=%s bytesSpeed=%d fps=%d", stream, info.BytesSpeed, info.VideoFps)
	} else {
		applog.Warnf("[ipc] 直播未在等待窗口内就绪，仍返回地址 stream=%s", stream)
	}
	return s.buildContent(ctx, stream)
}

// pluginFirstFrameWait 是插件起流到首帧到达的就绪等待上限（拉流握手+首关键帧）。
const pluginFirstFrameWait = 15 * time.Second

// waitLiveReady 轮询探测直到该插件直播流在 ZMS 可播（有码率或帧率）或到达 maxWait；
// ctx 取消则提前返回 nil。探测节点不可达时按“未就绪”处理（到点返回 nil 即可）。
func (s *Service) waitLiveReady(ctx context.Context, node port.MediaEndpoint, app, stream string, maxWait time.Duration) *port.StreamProbe {
	deadline := time.Now().Add(maxWait)
	for {
		if info := node.LookupStream(ctx, app, stream); info != nil && (info.VideoFps > 0 || info.BytesSpeed >= minPlayBytesSpeed) {
			return info
		}
		if time.Now().After(deadline) {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// Command 通用命令透传（自定义能力）：平台不解析 params/result 语义。
func (s *Service) Command(ctx context.Context, deviceID int, channelID, cmd string, params json.RawMessage) (json.RawMessage, error) {
	dev, p, client, err := s.deviceClient(deviceID)
	if err != nil {
		return nil, err
	}
	if !dev.Enabled {
		return nil, fmt.Errorf("设备已禁用")
	}
	out, err := client.Command(ctx, dev.DeviceUID, channelID, cmd, params)
	if err != nil {
		s.plugins.MarkOffline(p.ID)
		return nil, fmt.Errorf("插件执行命令失败: %w", err)
	}
	return out, nil
}

// pickTransport 优先 rtsp_url（平台侧更简单），其次 rtp_push，默认 rtsp_url。
func pickTransport(transports []string) string {
	for _, t := range transports {
		if t == "rtsp_url" {
			return "rtsp_url"
		}
	}
	for _, t := range transports {
		if t == "rtp_push" {
			return "rtp_push"
		}
	}
	return "rtsp_url"
}

func (s *Service) StopLive(ctx context.Context, deviceID int, channelID string) error {
	dev, err := s.repo.GetDevice(deviceID)
	if err != nil {
		return err
	}
	stream, err := s.streamName(dev, channelID)
	if err != nil {
		return err
	}
	act := s.getActive(stream)
	if act == nil {
		return nil
	}
	if p, perr := s.repo.GetPlugin(dev.PluginID); perr == nil {
		_ = s.factory.For(p.Endpoint).StopLive(ctx, dev.DeviceUID, channelID, stream)
	}
	if node, nerr := s.media.Resolve(ctx, act.NodeID); nerr == nil {
		// 必须连"拉流代理"一起删：只 CloseStreams 的话 ZMS 会保留 auto_close 代理并自动重连，
		// 变成僵尸代理继续去拉相机 RTSP，与 onvif 等其他会话抢占相机(海康主码流并发会话有限)，
		// 互相踢断导致 err=-3 / 无帧。与 onvif resetONVIFStream 对齐：先删代理再关流。
		_ = node.DelStreamProxy(ctx, fmt.Sprintf("__defaultVhost__/%s/%s", act.App, act.Stream))
		_ = node.CloseStreams(ctx, "__defaultVhost__", act.App, act.Stream)
	}
	s.media.UnbindStream(act.App, act.Stream)
	s.setActive(stream, nil)
	return nil
}

// minPlayBytesSpeed 与 onvif 侧口径一致：低于此码率且 fps=0 视为“没在有帧走”。
const minPlayBytesSpeed int64 = 2048

// streamStalled 判断探针结果是否表示“流在、但没在有帧走”（卡死僵尸流）。
// 起播阶段本来就没有帧，用 AliveSecond 做宽限，避免误判正常慢启动的流。
//   - 前 10 秒：宽限，不判卡死
//   - fps>0：真有帧在走，健康
//   - 码率足够：容忍 fps 采样瞬时为 0，健康
//   - 其余(fps=0 且码率很低)：卡死 —— 旧实现只在 BytesSpeed==0 才算卡死，
//     把“码率 1~2KB/s、fps=0”的僵尸误判为可用复用到播放器 → 有字节但黑屏。
func streamStalled(p *port.StreamProbe) bool {
	if p == nil {
		return false
	}
	if p.AliveSecond <= 10 {
		return false
	}
	if p.VideoFps > 0 {
		return false
	}
	if p.BytesSpeed >= minPlayBytesSpeed {
		return false
	}
	return true
}

// activeStreamHealthy 校验内存 active 标记对应的流在 ZMS 是否仍然可用。
//   - 流已不在 ZMS：拉流被对端断开、代理被自动关闭、媒体节点重启等
//   - 流在但长时间零帧零码率：拉流卡死（连接还在，数据不来）
//
// 探测本身失败（节点不可达、接口异常）时按“可用”处理，避免误杀正在播的流。
func (s *Service) activeStreamHealthy(ctx context.Context, nodeID, app, stream string) bool {
	node, err := s.media.Resolve(ctx, nodeID)
	if err != nil || node == nil {
		return true
	}
	probe := node.LookupStream(ctx, app, stream)
	if probe == nil {
		return false
	}
	return !streamStalled(probe)
}

// dropStaleStream 清掉失效的 active 标记与 ZMS 残留流，为重建拉流做准备。
func (s *Service) dropStaleStream(ctx context.Context, nodeID, app, stream string) {
	applog.Warnf("[ipc] 直播流已失效，重建拉流: app=%s stream=%s node=%s", app, stream, nodeID)
	if node, err := s.media.Resolve(ctx, nodeID); err == nil && node != nil {
		_ = node.CloseStreams(ctx, "__defaultVhost__", app, stream)
	}
	s.media.UnbindStream(app, stream)
	s.setActive(stream, nil)
}

func (s *Service) GetDevice(deviceID int) (*domainplugin.Device, error) {
	return s.repo.GetDevice(deviceID)
}

// SetKeepalivePusher 注入保活时间戳写入器（main 装配时调用）。
func (s *Service) SetKeepalivePusher(p KeepalivePusher) {
	s.keepalive = p
}

// ProbeAll 周期探活：对每台插件设备发一次轻量请求（ListChannels），可达则置在线并写心跳。
// 插件设备没有协议层心跳，不周期探活会导致"设备已下线但平台仍显示在线"。
func (s *Service) ProbeAll(ctx context.Context) error {
	list, err := s.repo.ListDevices()
	if err != nil {
		return err
	}
	for _, dev := range list {
		if dev == nil {
			continue
		}
		p, err := s.repo.GetPlugin(dev.PluginID)
		if err != nil || p == nil || !p.Online {
			// 插件离线：设备必然不可达，置离线。
			if dev.Online {
				dev.Online = false
				_ = s.repo.UpdateDevice(dev)
			}
			continue
		}
		probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		_, lerr := s.factory.For(p.Endpoint).ListChannels(probeCtx, dev.DeviceUID)
		cancel()
		online := lerr == nil
		if !online && isNotLoggedIn(lerr) {
			// 插件进程重启会丢失设备登录会话（DEVICE_UNREACHABLE: device not logged in）。
			// 仅在"未登录"时补一次登录（避免真正离线时反复登录触发相机账号锁定），
			// 登录成功即视为在线。
			if e := s.EnsureLogin(ctx, dev.ID); e == nil {
				online = true
			}
		}
		if online != dev.Online {
			dev.Online = online
			_ = s.repo.UpdateDevice(dev)
		}
		if online && s.keepalive != nil {
			_ = s.keepalive.PushKeepalive(ctx, strconv.Itoa(dev.ID), time.Now().UnixMilli())
		}
	}
	return nil
}

// isNotLoggedIn 判断插件错误是否为"设备未登录"（插件进程重启后登录会话丢失的典型错误）。
func isNotLoggedIn(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not logged in")
}

// UpdateDevice 保存插件设备（名称等可编辑字段）。统一接入的编辑入口复用。
func (s *Service) UpdateDevice(d *domainplugin.Device) error {
	return s.repo.UpdateDevice(d)
}

// EnsureLogin 确保插件侧已登录该设备（插件重启后补登录）。
func (s *Service) EnsureLogin(ctx context.Context, deviceID int) error {
	dev, p, client, err := s.deviceClient(deviceID)
	if err != nil {
		return err
	}
	var params map[string]any
	if strings.TrimSpace(dev.DriverParams) != "" {
		_ = json.Unmarshal([]byte(dev.DriverParams), &params)
	}
	if _, err := client.Login(ctx, dev.DeviceUID, params); err != nil {
		s.plugins.MarkOffline(p.ID)
		return fmt.Errorf("插件重新登录设备失败: %w", err)
	}
	dev.Online = true
	_ = s.repo.UpdateDevice(dev)
	return nil
}

func (s *Service) Arm(ctx context.Context, deviceID int, cfg port.IPCArmConfig) (*port.IPCArmResult, error) {
	dev, _, client, err := s.deviceClient(deviceID)
	if err != nil {
		return nil, err
	}
	cfg.DeviceUID = dev.DeviceUID
	return client.Arm(ctx, cfg)
}

func (s *Service) Disarm(ctx context.Context, deviceID int, subscriptionID string) error {
	dev, _, client, err := s.deviceClient(deviceID)
	if err != nil {
		return err
	}
	return client.Disarm(ctx, dev.DeviceUID, subscriptionID)
}

func (s *Service) PTZ(ctx context.Context, deviceID int, channelID, direction string, h, v, z int) error {
	dev, _, client, err := s.deviceClient(deviceID)
	if err != nil {
		return err
	}
	return client.PTZ(ctx, dev.DeviceUID, channelID, direction, h, v, z)
}

func (s *Service) deviceClient(deviceID int) (*domainplugin.Device, *domainplugin.Plugin, port.IPCClient, error) {
	dev, err := s.repo.GetDevice(deviceID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("设备不存在")
	}
	p, err := s.repo.GetPlugin(dev.PluginID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("插件不存在")
	}
	if !p.Online {
		return nil, nil, nil, fmt.Errorf("插件离线: %s", p.ID)
	}
	return dev, p, s.factory.For(p.Endpoint), nil
}

func (s *Service) resolvePlugin(pluginID, protocol string) (*domainplugin.Plugin, error) {
	if strings.TrimSpace(pluginID) != "" {
		return s.repo.GetPlugin(strings.TrimSpace(pluginID))
	}
	if strings.TrimSpace(protocol) == "" {
		return nil, fmt.Errorf("必须指定插件或协议")
	}
	return s.plugins.Resolve(protocol)
}

func (s *Service) buildContent(ctx context.Context, stream string) (*dto.StreamContent, error) {
	node, err := s.media.ResolveForStream(ctx, streamApp, stream, "")
	if err != nil {
		return nil, err
	}
	urls := node.StreamPlayURLs(streamApp, stream, false, 0)
	return &dto.StreamContent{
		App: streamApp, Stream: stream, IP: node.StreamIP(),
		Flv: urls.Flv, WsFlv: urls.WsFlv, Hls: urls.Hls,
		Rtmp: urls.Rtmp, Rtsp: urls.Rtsp, Rtc: urls.Rtc, Rtcs: urls.Rtcs,
		MediaServerID: node.ID(), ServerID: s.serverID,
	}, nil
}

func (s *Service) getActive(key string) *activeStream {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active[key]
}

func (s *Service) setActive(key string, v *activeStream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v == nil {
		delete(s.active, key)
		return
	}
	s.active[key] = v
}

func toDomainChannels(in []port.IPCChannel) []*domainplugin.Channel {
	out := make([]*domainplugin.Channel, 0, len(in))
	for _, ch := range in {
		status := ch.Status
		if status == "" {
			status = "ON"
		}
		out = append(out, &domainplugin.Channel{
			ChannelID: ch.ChannelID, Name: ch.Name, Status: status,
			PTZType: ch.PTZType, HasAudio: ch.HasAudio,
		})
	}
	return out
}

// streamName 统一流名：<设备内码>_<通道内码>；任一缺失返回错误，禁止回退协议标识。
func (s *Service) streamName(dev *domainplugin.Device, channelID string) (string, error) {
	if dev == nil {
		return "", fmt.Errorf("设备不存在")
	}
	name := mediaurl.LiveStream(dev.InternalCode, s.channelInternalCode(dev, channelID))
	if name == "" {
		return "", fmt.Errorf("设备或通道内码缺失")
	}
	return name, nil
}

// channelInternalCode 用插件侧通道号查平台通道内码；流命名不允许回退协议标识。
func (s *Service) channelInternalCode(dev *domainplugin.Device, channelID string) string {
	if dev == nil {
		return ""
	}
	if list, err := s.repo.ListChannels(dev.ID); err == nil {
		for _, c := range list {
			if c.ChannelID == channelID {
				return c.InternalCode
			}
		}
	}
	return ""
}

func deriveUID(pluginID string, params map[string]any) string {
	ip := stringParam(params, "ip")
	port := stringParam(params, "port")
	uid := strings.TrimSpace(ip + "-" + port)
	if uid == "-" || uid == "" {
		return pluginID
	}
	return uid
}

func stringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	v, ok := params[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func firstString(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
