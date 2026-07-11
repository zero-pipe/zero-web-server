package playapp

import (
	"context"
	"fmt"
	"sync"

	mediaserverapp "zero-web-kit/internal/application/mediaserver"
	domainchannel "zero-web-kit/internal/domain/channel"
	domaindevice "zero-web-kit/internal/domain/device"
	"zero-web-kit/internal/infrastructure/media/mediakit"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
	"zero-web-kit/internal/interfaces/http/dto"
	applog "zero-web-kit/pkg/log"
)

type Service struct {
	devices      domaindevice.Repository
	channels     domainchannel.Repository
	sip          *sipinfra.Server
	mediaServers *mediaserverapp.Service
	serverID     string
	serverPort   int
	ssrcSeq      int
	sessions     sync.Map
	broadcast    *broadcastRegistry
}

func NewService(
	devices domaindevice.Repository,
	channels domainchannel.Repository,
	sipServer *sipinfra.Server,
	mediaServers *mediaserverapp.Service,
	serverID string,
	serverPort int,
) *Service {
	return &Service{
		devices:      devices,
		channels:     channels,
		sip:          sipServer,
		mediaServers: mediaServers,
		serverID:     serverID,
		serverPort:   serverPort,
		broadcast:    newBroadcastRegistry(),
	}
}

func (s *Service) StartPlay(ctx context.Context, deviceID, channelDeviceID string) (*dto.StreamContent, error) {
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	channel, err := s.channels.GetOne(deviceID, channelDeviceID)
	if err != nil {
		return nil, fmt.Errorf("通道不存在")
	}
	return s.startPlay(ctx, device, channel)
}

func (s *Service) startPlay(ctx context.Context, device *domaindevice.Device, channel *domainchannel.Channel) (*dto.StreamContent, error) {
	app := "rtp"
	stream := fmt.Sprintf("%s_%s", device.DeviceID, channel.GBDeviceID)

	node, err := s.mediaServers.ResolveForStream(app, stream, device.MediaServerID)
	if err != nil {
		return nil, err
	}

	sdpIP := device.SDPIP
	if sdpIP == "" {
		sdpIP = node.SDPIP()
	}

	streamMode := sipinfra.NormalizeStreamMode(device.StreamMode)
	applog.Debugf("[GB28181 play 1/6] start device=%s channel=%s target=%s:%d sipTransport=%s mediaStreamMode=%s sdpIP=%s mediaNode=%s zms=%s",
		device.DeviceID, channel.GBDeviceID, device.IP, device.Port, device.Transport, streamMode, sdpIP, node.ID(), node.MediaConfig().BaseURL())

	tcpMode := streamModeToTCP(streamMode)
	rtpResp, err := node.Client.OpenRtpServer(ctx, app, stream, 0, tcpMode)
	if err != nil {
		applog.Warnf("[GB28181 play 2/6] openRtpServer FAILED stream=%s tcp_mode=%d err=%v", stream, tcpMode, err)
		return nil, fmt.Errorf("open RTP server failed: %w", err)
	}
	applog.Debugf("[GB28181 play 2/6] openRtpServer OK stream=%s port=%d tcp_mode=%d node=%s", stream, rtpResp.Port, tcpMode, node.ID())
	s.mediaServers.BindStream(app, stream, node.ID())

	ssrc := sipinfra.PlaySSRC(s.sip.Domain(), s.nextSSRCSeq())
	sdp := sipinfra.BuildPlaySDP(device.DeviceID, sdpIP, rtpResp.Port, ssrc, streamMode, "")
	applog.Debugf("[GB28181 play 3/6] SDP ready ssrc=%s streamMode=%s port=%d", ssrc, streamMode, rtpResp.Port)

	zlm := node.Client
	go func() {
		applog.Debugf("[GB28181 play 4/6] SIP INVITE -> %s:%d channel=%s Subject SSRC=%s streamMode=%s",
			device.IP, device.Port, channel.GBDeviceID, ssrc, streamMode)
		tcpConnect := func(host string, port int) error {
			if streamMode != "TCP-ACTIVE" {
				return nil
			}
			return zlm.ConnectRtpServer(ctx, app, stream, host, port)
		}
		if err := s.sip.SendInvitePlay(device, channel, sdp, ssrc, stream, streamMode, tcpConnect, nil); err != nil {
			applog.Warnf("[GB28181 play 4/6] SIP INVITE FAILED device=%s channel=%s ssrc=%s rtpPort=%d err=%v",
				device.DeviceID, channel.GBDeviceID, ssrc, rtpResp.Port, err)
		}
	}()

	pushURL := mediakit.BuildGB28181PushURL(sdpIP, rtpResp.Port, tcpMode)
	content := s.buildStreamContent(app, stream, node)
	mediakit.LogPlayStreamPaths("[GB28181 play 6/6] URLs ready (invite async)", app, stream, pushURL,
		mediakit.BuildPlayURLsFromConfig(node.MediaConfig(), app, stream))
	return content, nil
}

func (s *Service) StopPlay(deviceID, channelDeviceID string) error {
	app := "rtp"
	stream := fmt.Sprintf("%s_%s", deviceID, channelDeviceID)
	_ = s.sip.CloseInviteSession(stream)
	client := s.clientForStream(app, stream)
	_, err := client.CloseStreams(context.Background(), "__defaultVhost__", app, stream)
	s.mediaServers.UnbindStream(app, stream)
	return err
}

func (s *Service) OnStreamStarted(app, stream string) {
	key := streamKey(app, stream)
	applog.Debugf("[GB28181 play 5/6] hook on_stream_changed regist app=%s stream=%s key=%s", app, stream, key)
	if nodeID, ok := s.mediaServers.StreamNodeID(app, stream); ok {
		if node, err := s.mediaServers.Resolve(nodeID); err == nil {
			mediakit.LogPlayStreamPaths("[GB28181 play 5/6] pull URLs", app, stream, "",
				mediakit.BuildPlayURLsFromConfig(node.MediaConfig(), app, stream))
		}
	}
	if v, ok := s.sessions.Load(key); ok {
		if ch, ok := v.(chan *dto.StreamContent); ok {
			select {
			case ch <- s.buildStreamContentForKey(app, stream):
			default:
			}
		}
	} else {
		applog.Debugf("[GB28181 play 5/6] hook arrived but no waiting session key=%s", key)
	}
}

func (s *Service) buildStreamContentForKey(app, stream string) *dto.StreamContent {
	node, err := s.mediaServers.ResolveForStream(app, stream, "auto")
	if err != nil {
		return &dto.StreamContent{App: app, Stream: stream, ServerID: s.serverID}
	}
	return s.buildStreamContent(app, stream, node)
}

func (s *Service) buildStreamContent(app, stream string, node *mediaserverapp.Node) *dto.StreamContent {
	cfg := node.MediaConfig()
	urls := mediakit.BuildStreamPlayURLs(cfg, app, stream, false, s.serverPort)
	content := &dto.StreamContent{
		App:           app,
		Stream:        stream,
		IP:            node.StreamIP(),
		Flv:           urls.Flv,
		WsFlv:         urls.WsFlv,
		Hls:           urls.Hls,
		Rtmp:          urls.Rtmp,
		Rtsp:          urls.Rtsp,
		Rtc:           urls.Rtc,
		Rtcs:          urls.Rtcs,
		MediaServerID: node.ID(),
		ServerID:      s.serverID,
	}
	if info := node.Client.LookupStreamMediaInfo(context.Background(), app, stream); info != nil {
		content.VideoCodec = info.VideoCodec
		content.AudioCodec = info.AudioCodec
	}
	return content
}

func (s *Service) clientForStream(app, stream string) *mediakit.Client {
	if node, err := s.mediaServers.ResolveForStream(app, stream, "auto"); err == nil {
		return node.Client
	}
	// 无节点时返回空壳，调用方会失败
	return mediakit.NewClientAddr("127.0.0.1", 1, "")
}

func (s *Service) nextSSRCSeq() int {
	s.ssrcSeq++
	return s.ssrcSeq
}

func streamKey(app, stream string) string {
	return app + "/" + stream
}

func streamModeToTCP(mode string) int {
	switch mode {
	case "TCP-ACTIVE":
		return 2
	case "TCP-PASSIVE":
		return 1
	default:
		return 0
	}
}
