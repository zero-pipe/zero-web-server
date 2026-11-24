package ipcdevice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

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

	mu     sync.Mutex
	active map[string]*activeStream
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
		return nil, err
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
	return s.buildContent(ctx, stream)
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
		_ = node.CloseStreams(ctx, "__defaultVhost__", act.App, act.Stream)
	}
	s.media.UnbindStream(act.App, act.Stream)
	s.setActive(stream, nil)
	return nil
}

// streamStalled 判断探针结果是否表示“流在、但长时间不出数据”。
// 起播阶段本来就没有帧，用 AliveSecond 做宽限，避免误判正常慢启动的流。
func streamStalled(p *port.StreamProbe) bool {
	return p != nil && p.AliveSecond > 10 && p.VideoFps == 0 && p.BytesSpeed == 0
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
