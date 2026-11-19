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
func (s *Service) StartLive(ctx context.Context, deviceID int, channelID string) (*dto.StreamContent, error) {
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
	if s.getActive(stream) != nil {
		return s.buildContent(ctx, stream)
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
	src, err := client.StartLive(ctx, dev.DeviceUID, channelID, stream, transport, push)
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
		if _, err := node.AddStreamProxy(ctx, "__defaultVhost__", streamApp, stream, src.RTSPURL, "1", false, false, true); err != nil {
			return nil, fmt.Errorf("创建拉流代理失败: %w", err)
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
	chCode := ""
	if list, err := s.repo.ListChannels(dev.ID); err == nil {
		for _, c := range list {
			if c.ChannelID == channelID {
				chCode = c.InternalCode
				break
			}
		}
	}
	name := mediaurl.LiveStream(dev.InternalCode, chCode)
	if name == "" {
		return "", fmt.Errorf("设备或通道内码缺失")
	}
	return name, nil
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
