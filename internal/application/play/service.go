package playapp

import (
	"context"
	"fmt"
	"sync"

	publishauth "zero-web-server/internal/application/publishauth"
	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/infrastructure/media/mediakit"
	sipinfra "zero-web-server/internal/infrastructure/sip"
	"zero-web-server/internal/interfaces/http/dto"
	"zero-web-server/internal/port"
	applog "zero-web-server/pkg/log"
)

type Service struct {
	devices    domaindevice.Repository
	channels   domainchannel.Repository
	sip        *sipinfra.Server
	media      port.MediaCluster
	serverID   string
	serverPort int
	ssrcSeq    int
	sessions   sync.Map
	broadcast  *broadcastRegistry
}

func NewService(
	devices domaindevice.Repository,
	channels domainchannel.Repository,
	sipServer *sipinfra.Server,
	media port.MediaCluster,
	serverID string,
	serverPort int,
) *Service {
	return &Service{
		devices:    devices,
		channels:   channels,
		sip:        sipServer,
		media:      media,
		serverID:   serverID,
		serverPort: serverPort,
		broadcast:  newBroadcastRegistry(),
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

// PrepareCascadePlay cascade inbound INVITE -> leaf play, return answer SDP.
func (s *Service) PrepareCascadePlay(ctx context.Context, deviceID, channelDeviceID, preferSSRC string) (answerSDP, stream string, err error) {
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return "", "", fmt.Errorf("device not found")
	}
	channel, err := s.channels.GetOne(deviceID, channelDeviceID)
	if err != nil {
		return "", "", fmt.Errorf("channel not found")
	}
	app := publishauth.LiveApp
	stream = fmt.Sprintf("%s_%s", device.DeviceID, channel.GBDeviceID)

	node, err := s.media.ResolveForStream(ctx, app, stream, device.MediaServerID)
	if err != nil {
		return "", "", err
	}
	if _, hasInvite := s.sip.InviteManager().Get(stream); hasInvite {
		_ = s.sip.CloseInviteSession(stream)
		_ = node.CloseStreams(ctx, "__defaultVhost__", app, stream)
		s.media.UnbindStream(app, stream)
	}
	_ = node.CloseStreams(ctx, "__defaultVhost__", app, stream)

	sdpIP := device.SDPIP
	if sdpIP == "" {
		sdpIP = node.SDPIP()
	}
	streamMode := sipinfra.NormalizeStreamMode(device.StreamMode)
	tcpMode := streamModeToTCP(streamMode)
	rtpPort, err := node.OpenRtpServer(ctx, app, stream, 0, tcpMode)
	if err != nil {
		return "", "", fmt.Errorf("open RTP server failed: %w", err)
	}
	s.media.BindStream(node.ID(), app, stream)

	ssrc := preferSSRC
	if ssrc == "" {
		ssrc = sipinfra.PlaySSRC(s.sip.Domain(), s.nextSSRCSeq())
	}
	answerSDP = sipinfra.BuildPlaySDP(s.sip.Domain(), sdpIP, rtpPort, ssrc, streamMode, "")
	deviceSDP := sipinfra.BuildPlaySDP(device.DeviceID, sdpIP, rtpPort, ssrc, streamMode, "")

	go func() {
		tcpConnect := func(host string, port int) error {
			if streamMode != "TCP-ACTIVE" {
				return nil
			}
			return node.ConnectRtpServer(ctx, app, stream, host, port)
		}
		if err := s.sip.SendInvitePlay(device, channel, deviceSDP, ssrc, stream, streamMode, tcpConnect, nil); err != nil {
			applog.Warnf("[cascade play] SIP INVITE FAILED device=%s channel=%s err=%v", device.DeviceID, channel.GBDeviceID, err)
		}
	}()
	return answerSDP, stream, nil
}

func (s *Service) startPlay(ctx context.Context, device *domaindevice.Device, channel *domainchannel.Channel) (*dto.StreamContent, error) {
	app := publishauth.LiveApp
	stream := fmt.Sprintf("%s_%s", device.DeviceID, channel.GBDeviceID)

	node, err := s.media.ResolveForStream(ctx, app, stream, device.MediaServerID)
	if err != nil {
		return nil, err
	}

	// 流已在推：只返回拉流地址，禁止二次 INVITE。
	if info := node.LookupStream(ctx, app, stream); gbLiveStreamReady(info) {
		applog.Infof("[GB28181 play] skip re-INVITE, stream live app=%s stream=%s bytesSpeed=%d fps=%d readers=%d",
			app, stream, info.BytesSpeed, info.VideoFps, info.ReaderCount)
		s.media.BindStream(node.ID(), app, stream)
		content := s.buildStreamContent(app, stream, node)
		mediakit.LogPlayStreamPaths("[GB28181 play] reuse live URLs", app, stream, "", node.PlayURLs(app, stream))
		return content, nil
	}
	// INVITE 会话还在但媒体未就绪（常见于设备重启）：清掉僵死会话后重新 INVITE。
	if _, hasInvite := s.sip.InviteManager().Get(stream); hasInvite {
		applog.Warnf("[GB28181 play] stale invite without media, close and re-INVITE stream=%s", stream)
		_ = s.sip.CloseInviteSession(stream)
		_ = node.CloseStreams(ctx, "__defaultVhost__", app, stream)
	}

	sdpIP := device.SDPIP
	if sdpIP == "" {
		sdpIP = node.SDPIP()
	}

	streamMode := sipinfra.NormalizeStreamMode(device.StreamMode)
	applog.Debugf("[GB28181 play 1/6] start device=%s channel=%s target=%s:%d sipTransport=%s mediaStreamMode=%s sdpIP=%s mediaNode=%s zms=%s",
		device.DeviceID, channel.GBDeviceID, device.IP, device.Port, device.Transport, streamMode, sdpIP, node.ID(), node.BaseURL())

	tcpMode := streamModeToTCP(streamMode)
	rtpPort, err := node.OpenRtpServer(ctx, app, stream, 0, tcpMode)
	if err != nil {
		applog.Warnf("[GB28181 play 2/6] openRtpServer FAILED stream=%s tcp_mode=%d err=%v", stream, tcpMode, err)
		return nil, fmt.Errorf("open RTP server failed: %w", err)
	}
	applog.Debugf("[GB28181 play 2/6] openRtpServer OK stream=%s port=%d tcp_mode=%d node=%s", stream, rtpPort, tcpMode, node.ID())
	s.media.BindStream(node.ID(), app, stream)

	ssrc := sipinfra.PlaySSRC(s.sip.Domain(), s.nextSSRCSeq())
	sdp := sipinfra.BuildPlaySDP(device.DeviceID, sdpIP, rtpPort, ssrc, streamMode, "")
	applog.Debugf("[GB28181 play 3/6] SDP ready ssrc=%s streamMode=%s port=%d", ssrc, streamMode, rtpPort)

	go func() {
		applog.Debugf("[GB28181 play 4/6] SIP INVITE -> %s:%d channel=%s Subject SSRC=%s streamMode=%s",
			device.IP, device.Port, channel.GBDeviceID, ssrc, streamMode)
		tcpConnect := func(host string, port int) error {
			if streamMode != "TCP-ACTIVE" {
				return nil
			}
			return node.ConnectRtpServer(ctx, app, stream, host, port)
		}
		if err := s.sip.SendInvitePlay(device, channel, sdp, ssrc, stream, streamMode, tcpConnect, nil); err != nil {
			applog.Warnf("[GB28181 play 4/6] SIP INVITE FAILED device=%s channel=%s ssrc=%s rtpPort=%d err=%v",
				device.DeviceID, channel.GBDeviceID, ssrc, rtpPort, err)
		}
	}()

	pushURL := mediakit.BuildGB28181PushURL(sdpIP, rtpPort, tcpMode)
	content := s.buildStreamContent(app, stream, node)
	mediakit.LogPlayStreamPaths("[GB28181 play 6/6] URLs ready (invite async)", app, stream, pushURL, node.PlayURLs(app, stream))
	return content, nil
}

func gbLiveStreamReady(info *port.StreamProbe) bool {
	if info == nil || !info.Video {
		return false
	}
	const minBytes int64 = 2048
	if info.BytesSpeed >= minBytes {
		return true
	}
	if info.VideoFps > 0 && info.BytesSpeed > 0 {
		return true
	}
	if info.Width >= 320 && info.Height >= 240 && info.BytesSpeed > 0 {
		return true
	}
	return false
}

func (s *Service) StopPlay(deviceID, channelDeviceID string) error {
	app := publishauth.LiveApp
	stream := fmt.Sprintf("%s_%s", deviceID, channelDeviceID)
	_ = s.sip.CloseInviteSession(stream)
	err := s.closeStream(app, stream)
	s.media.UnbindStream(app, stream)
	return err
}

func (s *Service) OnStreamStarted(app, stream string) {
	key := streamKey(app, stream)
	applog.Debugf("[GB28181 play 5/6] hook on_stream_changed regist app=%s stream=%s key=%s", app, stream, key)
	if nodeID, ok := s.media.StreamNodeID(app, stream); ok {
		if node, err := s.media.Resolve(context.Background(), nodeID); err == nil {
			mediakit.LogPlayStreamPaths("[GB28181 play 5/6] pull URLs", app, stream, "", node.PlayURLs(app, stream))
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
	node, err := s.media.ResolveForStream(context.Background(), app, stream, "auto")
	if err != nil {
		return &dto.StreamContent{App: app, Stream: stream, ServerID: s.serverID}
	}
	return s.buildStreamContent(app, stream, node)
}

func (s *Service) buildStreamContent(app, stream string, node port.MediaEndpoint) *dto.StreamContent {
	urls := node.StreamPlayURLs(app, stream, false, s.serverPort)
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
	if info := node.LookupStream(context.Background(), app, stream); info != nil {
		content.VideoCodec = info.VideoCodec
		content.AudioCodec = info.AudioCodec
	}
	return content
}

func (s *Service) closeStream(app, stream string) error {
	node, err := s.media.ResolveForStream(context.Background(), app, stream, "auto")
	if err != nil {
		return err
	}
	return node.CloseStreams(context.Background(), "__defaultVhost__", app, stream)
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
