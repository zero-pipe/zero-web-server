package onvifapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	domainonvif "zero-web-kit/internal/domain/onvif"
	domainptz "zero-web-kit/internal/domain/ptz"
	mediaserverapp "zero-web-kit/internal/application/mediaserver"
	onvifinfra "zero-web-kit/internal/infrastructure/onvif"
	"zero-web-kit/internal/infrastructure/media/mediakit"

	onviflib "github.com/0x524a/onvif-go"
)

type Service struct {
	devices      domainonvif.DeviceRepository
	channels     domainonvif.ChannelRepository
	factory      *onvifinfra.ClientFactory
	mediaServers *mediaserverapp.Service
	serverID     string
	proxyKeys    sync.Map // stream -> ZMS addStreamProxy key
}

func NewService(
	devices domainonvif.DeviceRepository,
	channels domainonvif.ChannelRepository,
	factory *onvifinfra.ClientFactory,
	mediaServers *mediaserverapp.Service,
	serverID string,
) *Service {
	return &Service{
		devices:      devices,
		channels:     channels,
		factory:      factory,
		mediaServers: mediaServers,
		serverID:     serverID,
	}
}

func (s *Service) Discover(ctx context.Context, timeoutSec int) ([]*domainonvif.DiscoveredDevice, error) {
	found, err := onvifinfra.Discover(ctx, timeoutSec)
	if err != nil {
		return nil, err
	}

	result := make([]*domainonvif.DiscoveredDevice, 0, len(found))
	for _, d := range found {
		endpoint := d.GetDeviceEndpoint()
		host, port, _, _ := onvifinfra.ParseEndpoint(endpoint)
		result = append(result, &domainonvif.DiscoveredDevice{
			Name:     d.GetName(),
			IP:       host,
			Port:     port,
			Endpoint: endpoint,
			Location: d.GetLocation(),
		})
	}
	return result, nil
}

type AddDeviceRequest struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Endpoint string `json:"endpoint"`
}

func (s *Service) AddDevice(ctx context.Context, req AddDeviceRequest) (*domainonvif.Device, error) {
	endpoint := req.Endpoint
	host := req.IP
	port := req.Port

	if endpoint == "" {
		if port <= 0 {
			port = 80
		}
		endpoint = fmt.Sprintf("http://%s:%d/onvif/device_service", host, port)
	} else {
		var err error
		host, port, endpoint, err = onvifinfra.ParseEndpoint(endpoint)
		if err != nil {
			return nil, err
		}
	}

	exists, err := s.devices.ExistsByIPPort(ctx, host, port)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("设备 %s:%d 已存在", host, port)
	}

	client, err := s.factory.Initialize(ctx, endpoint, req.Username, req.Password)
	if err != nil {
		return nil, fmt.Errorf("连接设备失败: %w", err)
	}

	info, err := client.GetDeviceInformation(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取设备信息失败: %w", err)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	name := req.Name
	if name == "" {
		name = info.Manufacturer + " " + info.Model
	}

	device := &domainonvif.Device{
		Name:          name,
		IP:            host,
		Port:          port,
		Username:      req.Username,
		Password:      req.Password,
		Manufacturer:  info.Manufacturer,
		Model:         info.Model,
		Firmware:      info.FirmwareVersion,
		SerialNumber:  info.SerialNumber,
		HardwareID:    info.HardwareID,
		DeviceURI:     endpoint,
		MediaURI:      endpoint,
		PTZURI:        endpoint,
		OnLine:        true,
		DiscoveryMode: 0,
		MediaServerID: "auto",
		ServerID:      s.serverID,
		CreateTime:    now,
		UpdateTime:    now,
	}

	if err := s.devices.Create(ctx, device); err != nil {
		return nil, err
	}

	if _, err := s.SyncChannels(ctx, device.ID); err != nil {
		return device, fmt.Errorf("设备已添加，但同步通道失败: %w", err)
	}
	return device, nil
}

func (s *Service) ListDevices(ctx context.Context, page, count int, keyword string) ([]*domainonvif.Device, int64, error) {
	return s.devices.List(ctx, page, count, keyword)
}

func (s *Service) DeleteDevice(ctx context.Context, id int64) error {
	if err := s.channels.DeleteByDeviceID(ctx, id); err != nil {
		return err
	}
	return s.devices.Delete(ctx, id)
}

func (s *Service) SyncChannels(ctx context.Context, deviceID int64) ([]*domainonvif.Channel, error) {
	device, err := s.devices.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	client, err := s.factory.Initialize(ctx, device.DeviceURI, device.Username, device.Password)
	if err != nil {
		s.devices.UpdateOnlineStatus(ctx, deviceID, false)
		return nil, err
	}

	profiles, err := client.GetProfiles(ctx)
	if err != nil {
		s.devices.UpdateOnlineStatus(ctx, deviceID, false)
		return nil, err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	channels := make([]*domainonvif.Channel, 0, len(profiles))
	for _, p := range profiles {
		ch := &domainonvif.Channel{
			DeviceID:     deviceID,
			ProfileToken: p.Token,
			Name:         p.Name,
			HasAudio:     p.AudioSourceConfiguration != nil,
			HasPTZ:       p.PTZConfiguration != nil,
			Status:       "ON",
			CreateTime:   now,
			UpdateTime:   now,
		}
		if p.VideoEncoderConfiguration != nil {
			ch.EncoderToken = p.VideoEncoderConfiguration.Token
			if res := p.VideoEncoderConfiguration.Resolution; res != nil && res.Width > 0 && res.Height > 0 {
				ch.Resolution = fmt.Sprintf("%dx%d", res.Width, res.Height)
			}
			if p.VideoEncoderConfiguration.Encoding != "" {
				ch.Codec = p.VideoEncoderConfiguration.Encoding
			}
		}
		if p.VideoSourceConfiguration != nil {
			ch.VideoSource = p.VideoSourceConfiguration.SourceToken
		}

		streamURI, err := client.GetStreamURI(ctx, p.Token)
		if err == nil && streamURI != nil {
			ch.StreamURI = streamURI.URI
		}
		enrichChannelMeta(ch, p.Name)
		channels = append(channels, ch)
	}

	if err := s.channels.DeleteByDeviceID(ctx, deviceID); err != nil {
		return nil, err
	}
	if err := s.channels.BatchCreate(ctx, channels); err != nil {
		return nil, err
	}

	device.OnLine = true
	device.UpdateTime = now
	_ = s.devices.Update(ctx, device)

	return channels, nil
}

func (s *Service) ListChannels(ctx context.Context, page, count int, deviceID int64) ([]*domainonvif.Channel, int64, error) {
	list, total, err := s.channels.List(ctx, page, count, deviceID)
	for _, ch := range list {
		enrichChannelDisplay(ch)
	}
	return list, total, err
}

func (s *Service) ProbeAll(ctx context.Context) error {
	devices, _, err := s.devices.List(ctx, 1, 1000, "")
	if err != nil {
		return err
	}
	for _, d := range devices {
		client, err := s.factory.NewClient(d.DeviceURI, d.Username, d.Password)
		if err != nil {
			_ = s.devices.UpdateOnlineStatus(ctx, d.ID, false)
			continue
		}
		if err := client.Initialize(ctx); err != nil {
			_ = s.devices.UpdateOnlineStatus(ctx, d.ID, false)
			continue
		}
		if _, err := client.GetDeviceInformation(ctx); err != nil {
			_ = s.devices.UpdateOnlineStatus(ctx, d.ID, false)
			continue
		}
		_ = s.devices.UpdateOnlineStatus(ctx, d.ID, true)
	}
	return nil
}

type PlayResult struct {
	App              string            `json:"app"`
	Stream           string            `json:"stream"`
	MediaServerID    string            `json:"mediaServerId"`
	ConfigCodec      string            `json:"configCodec"`
	VideoCodec       string            `json:"videoCodec"`
	AudioCodec       string            `json:"audioCodec"`
	HasAudio         bool              `json:"hasAudio"`
	PreferredPlayer  string            `json:"preferredPlayer"`
	StreamChannel    string            `json:"streamChannel"`
	StreamType       string            `json:"streamType"`
	MediaResolution  string            `json:"mediaResolution"`
	URLs             map[string]string `json:"urls"`
}

func (s *Service) StartPlay(ctx context.Context, channelID int64) (*PlayResult, error) {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	device, err := s.devices.GetByID(ctx, ch.DeviceID)
	if err != nil {
		return nil, err
	}
	enrichChannelDisplay(ch)

	app := "onvif"
	vhost := "__defaultVhost__"
	stream := fmt.Sprintf("%d_%s", device.ID, sanitizeStream(ch.ProfileToken))

	prefer := device.MediaServerID
	if prefer == "" {
		prefer = "auto"
	}
	node, err := s.mediaServers.ResolveForStream(app, stream, prefer)
	if err != nil {
		return nil, err
	}

	if info := node.Client.LookupStreamMediaInfo(ctx, app, stream); isStreamReadyForPlay(info, ch.StreamChannel) {
		return s.buildPlayResult(app, stream, ch, info, node), nil
	}
	s.resetONVIFStream(ctx, vhost, app, stream, node)
	time.Sleep(150 * time.Millisecond)

	rtspURL, err := s.resolveRTSPURL(ctx, device, ch, ch.StreamURI == "")
	if err != nil {
		return nil, err
	}

	// rtp_type=1: RTSP over TCP（海康等仅支持 TCP 时必需）；auto_close=false 避免切换播放器时被 ZMS 拆掉代理
	resp, err := node.Client.AddStreamProxy(ctx, vhost, app, stream, rtspURL, "1", true, false, false)
	if err != nil {
		return nil, fmt.Errorf("媒体节点拉流代理失败: %w", err)
	}
	if resp != nil && resp.Code != 0 {
		return nil, fmt.Errorf("媒体节点拉流代理失败: %s", resp.Msg)
	}
	if resp != nil && len(resp.Data) > 0 {
		var proxyData struct {
			Key string `json:"key"`
		}
		if json.Unmarshal(resp.Data, &proxyData) == nil && proxyData.Key != "" {
			s.proxyKeys.Store(stream, proxyData.Key)
		}
	}
	s.mediaServers.BindStream(app, stream, node.ID())

	var mediaInfo *mediakit.StreamMediaInfo
	if info := s.waitStreamReady(ctx, app, stream, ch.StreamChannel, node); info != nil {
		mediaInfo = info
	} else {
		s.resetONVIFStream(ctx, vhost, app, stream, node)
		s.mediaServers.UnbindStream(app, stream)
		return nil, fmt.Errorf("等待%s就绪超时，请点「停止」后重试，或检查摄像机编码与带宽", streamChannelLabel(ch.StreamChannel))
	}

	return s.buildPlayResult(app, stream, ch, mediaInfo, node), nil
}

func (s *Service) resetONVIFStream(ctx context.Context, vhost, app, stream string, node *mediaserverapp.Node) {
	client := node.Client
	if v, ok := s.proxyKeys.Load(stream); ok {
		if key, ok := v.(string); ok && key != "" {
			_, _ = client.DelStreamProxy(ctx, key)
		}
		s.proxyKeys.Delete(stream)
	}
	_, _ = client.DelStreamProxy(ctx, fmt.Sprintf("%s/%s/%s", vhost, app, stream))
	_, _ = client.CloseStreams(ctx, vhost, app, stream)
}

func (s *Service) resolveRTSPURL(ctx context.Context, device *domainonvif.Device, ch *domainonvif.Channel, forceRefresh bool) (string, error) {
	rtspURL := ch.StreamURI
	if forceRefresh || rtspURL == "" {
		client, err := s.factory.Initialize(ctx, device.DeviceURI, device.Username, device.Password)
		if err != nil {
			return "", err
		}
		streamURI, err := client.GetStreamURI(ctx, ch.ProfileToken)
		if err != nil {
			return "", err
		}
		rtspURL = streamURI.URI
	}
	if device.Username != "" && !strings.Contains(rtspURL, "@") {
		rtspURL = injectRTSPAuth(rtspURL, device.Username, device.Password)
	}
	return rtspURL, nil
}

func (s *Service) waitStreamReady(ctx context.Context, app, stream, streamChannel string, node *mediaserverapp.Node) *mediakit.StreamMediaInfo {
	attempts := waitStreamReadyAttempts(streamChannel)
	for i := 0; i < attempts; i++ {
		if info := node.Client.LookupStreamMediaInfo(ctx, app, stream); isStreamReadyForPlay(info, streamChannel) {
			return info
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(200 * time.Millisecond):
		}
	}
	return nil
}

func (s *Service) buildPlayResult(app, stream string, ch *domainonvif.Channel, info *mediakit.StreamMediaInfo, node *mediaserverapp.Node) *PlayResult {
	configCodec := ch.ConfigCodec
	if configCodec == "" {
		configCodec = normalizeVideoCodec(ch.Codec)
	}
	mediaCodec := ""
	mediaAudio := ""
	hasAudio := false
	mediaRes := ""
	if info != nil {
		mediaCodec = normalizeVideoCodec(info.VideoCodec)
		mediaAudio = normalizeAudioCodec(info.AudioCodec)
		hasAudio = info.Audio
		if info.Width > 0 && info.Height > 0 {
			mediaRes = fmt.Sprintf("%dx%d", info.Width, info.Height)
		}
	}
	videoCodec := mediaCodec
	if videoCodec == "" {
		videoCodec = configCodec
	}
	return &PlayResult{
		App:             app,
		Stream:          stream,
		MediaServerID:   node.ID(),
		ConfigCodec:     configCodec,
		VideoCodec:      videoCodec,
		AudioCodec:      mediaAudio,
		HasAudio:        hasAudio,
		PreferredPlayer: resolvePreferredPlayer(videoCodec, mediaAudio, hasAudio),
		StreamChannel:   ch.StreamChannel,
		StreamType:      ch.StreamType,
		MediaResolution: mediaRes,
		URLs:            mediakit.BuildPlayURLsFromConfig(node.MediaConfig(), app, stream),
	}
}

func normalizeVideoCodec(raw string) string {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	switch {
	case raw == "", raw == "-":
		return ""
	case strings.Contains(raw, "265"), raw == "HEVC":
		return "H265"
	case strings.Contains(raw, "264"), raw == "AVC":
		return "H264"
	default:
		return raw
	}
}

func (s *Service) StopPlay(ctx context.Context, channelID int64) error {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	app := "onvif"
	vhost := "__defaultVhost__"
	stream := fmt.Sprintf("%d_%s", ch.DeviceID, sanitizeStream(ch.ProfileToken))
	if node, err := s.mediaServers.ResolveForStream(app, stream, "auto"); err == nil {
		s.resetONVIFStream(ctx, vhost, app, stream, node)
	}
	s.mediaServers.UnbindStream(app, stream)
	return nil
}

type PTZRequest struct {
	ChannelID int64   `json:"channelId"`
	Command   string  `json:"command"`
	Speed     float64 `json:"speed"`
}

func (s *Service) PTZControl(ctx context.Context, req PTZRequest) error {
	ch, err := s.channels.GetByID(ctx, req.ChannelID)
	if err != nil {
		return err
	}
	if !ch.HasPTZ {
		return fmt.Errorf("该通道不支持PTZ")
	}

	device, err := s.devices.GetByID(ctx, ch.DeviceID)
	if err != nil {
		return err
	}

	client, err := s.factory.Initialize(ctx, device.DeviceURI, device.Username, device.Password)
	if err != nil {
		return err
	}

	speed := req.Speed
	if speed <= 0 {
		speed = 0.5
	}

	var velocity *onviflib.PTZSpeed
	switch strings.ToUpper(req.Command) {
	case "LEFT":
		velocity = &onviflib.PTZSpeed{PanTilt: &onviflib.Vector2D{X: -speed, Y: 0}}
	case "RIGHT":
		velocity = &onviflib.PTZSpeed{PanTilt: &onviflib.Vector2D{X: speed, Y: 0}}
	case "UP":
		velocity = &onviflib.PTZSpeed{PanTilt: &onviflib.Vector2D{X: 0, Y: speed}}
	case "DOWN":
		velocity = &onviflib.PTZSpeed{PanTilt: &onviflib.Vector2D{X: 0, Y: -speed}}
	case "STOP":
		return client.Stop(ctx, ch.ProfileToken, true, true)
	default:
		return fmt.Errorf("不支持的PTZ命令: %s", req.Command)
	}

	timeout := "PT1S"
	return client.ContinuousMove(ctx, ch.ProfileToken, velocity, &timeout)
}

// QueryPresets returns presets reported by the ONVIF camera.
func (s *Service) QueryPresets(ctx context.Context, channelID int64) ([]domainptz.Preset, error) {
	ch, _, client, err := s.ptzClient(ctx, channelID)
	if err != nil {
		return nil, err
	}
	list, err := client.GetPresets(ctx, ch.ProfileToken)
	if err != nil {
		return nil, err
	}
	out := make([]domainptz.Preset, 0, len(list))
	for _, p := range list {
		if p == nil {
			continue
		}
		out = append(out, domainptz.Preset{
			PresetID:   p.Token,
			PresetName: p.Name,
		})
	}
	return out, nil
}

func (s *Service) GotoPreset(ctx context.Context, channelID int64, presetToken string) error {
	ch, _, client, err := s.ptzClient(ctx, channelID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(presetToken) == "" {
		return fmt.Errorf("预置位编号不能为空")
	}
	return client.GotoPreset(ctx, ch.ProfileToken, presetToken, nil)
}

func (s *Service) SetPreset(ctx context.Context, channelID int64, presetToken, presetName string) (string, error) {
	ch, _, client, err := s.ptzClient(ctx, channelID)
	if err != nil {
		return "", err
	}
	token, err := client.SetPreset(ctx, ch.ProfileToken, presetName, presetToken)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) RemovePreset(ctx context.Context, channelID int64, presetToken string) error {
	ch, _, client, err := s.ptzClient(ctx, channelID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(presetToken) == "" {
		return fmt.Errorf("预置位编号不能为空")
	}
	return client.RemovePreset(ctx, ch.ProfileToken, presetToken)
}

func (s *Service) ptzClient(ctx context.Context, channelID int64) (*domainonvif.Channel, *domainonvif.Device, *onviflib.Client, error) {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, nil, nil, err
	}
	if !ch.HasPTZ {
		return nil, nil, nil, fmt.Errorf("该通道不支持PTZ")
	}
	device, err := s.devices.GetByID(ctx, ch.DeviceID)
	if err != nil {
		return nil, nil, nil, err
	}
	client, err := s.factory.Initialize(ctx, device.DeviceURI, device.Username, device.Password)
	if err != nil {
		return nil, nil, nil, err
	}
	return ch, device, client, nil
}

func injectRTSPAuth(rawURL, username, password string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.User = url.UserPassword(username, password)
	return u.String()
}

func sanitizeStream(token string) string {
	token = strings.ReplaceAll(token, " ", "_")
	token = strings.ReplaceAll(token, "/", "_")
	if len(token) > 32 {
		return token[:32]
	}
	return token
}
