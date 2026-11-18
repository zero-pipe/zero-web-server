package playapp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/dto"
	"zero-web-server/internal/gbsip"
	"zero-web-server/internal/mediaurl"
	"zero-web-server/internal/port"
	applog "zero-web-server/pkg/log"
)

type Service struct {
	devices    domaindevice.Repository
	channels   domainchannel.Repository
	sip        port.GBInviter
	media      port.MediaCluster
	serverID   string
	serverPort int
	ssrcSeq    atomic.Int64
	sessions   sync.Map
	broadcast  *broadcastRegistry
}

func NewService(
	devices domaindevice.Repository,
	channels domainchannel.Repository,
	sipServer port.GBInviter,
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
	app := mediaurl.LiveApp
	stream = mediaurl.LiveStream(device.InternalCode, channel.InternalCode)

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
	streamMode := gbsip.NormalizeStreamMode(device.StreamMode)
	tcpMode := streamModeToTCP(streamMode)
	rtpPort, err := node.OpenRtpServer(ctx, app, stream, 0, tcpMode)
	if err != nil {
		return "", "", fmt.Errorf("open RTP server failed: %w", err)
	}
	s.media.BindStream(node.ID(), app, stream)

	ssrc := preferSSRC
	if ssrc == "" {
		ssrc = gbsip.PlaySSRC(s.sip.Domain(), s.nextSSRCSeq())
	}
	answerSDP = gbsip.BuildPlaySDP(s.sip.Domain(), sdpIP, rtpPort, ssrc, streamMode, "")
	deviceSDP := gbsip.BuildPlaySDP(device.DeviceID, sdpIP, rtpPort, ssrc, streamMode, "")

	go func() {
		tcpConnect := func(host string, port int) error {
			if streamMode != "TCP-ACTIVE" {
				return nil
			}
			return node.ConnectRtpServer(ctx, app, stream, host, port)
		}
		if err := s.sip.SendInvitePlay(device, channel, deviceSDP, ssrc, stream, streamMode, tcpConnect); err != nil {
			applog.Warnf("[cascade play] SIP INVITE FAILED device=%s channel=%s err=%v", device.DeviceID, channel.GBDeviceID, err)
		}
	}()
	return answerSDP, stream, nil
}

func (s *Service) startPlay(ctx context.Context, device *domaindevice.Device, channel *domainchannel.Channel) (*dto.StreamContent, error) {
	app := mediaurl.LiveApp
	stream := mediaurl.LiveStream(device.InternalCode, channel.InternalCode)

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
		mediaurl.LogPlayStreamPaths("[GB28181 play] reuse live URLs", app, stream, "", node.PlayURLs(app, stream))
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

	streamMode := gbsip.NormalizeStreamMode(device.StreamMode)
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

	ssrc := gbsip.PlaySSRC(s.sip.Domain(), s.nextSSRCSeq())
	sdp := gbsip.BuildPlaySDP(device.DeviceID, sdpIP, rtpPort, ssrc, streamMode, streamIdentificationForSDP(channel.StreamIdentification))
	applog.Debugf("[GB28181 play 3/6] SDP ready ssrc=%s streamMode=%s port=%d streamIdent=%s", ssrc, streamMode, rtpPort, channel.StreamIdentification)

	go func() {
		applog.Debugf("[GB28181 play 4/6] SIP INVITE -> %s:%d channel=%s Subject SSRC=%s streamMode=%s",
			device.IP, device.Port, channel.GBDeviceID, ssrc, streamMode)
		tcpConnect := func(host string, port int) error {
			if streamMode != "TCP-ACTIVE" {
				return nil
			}
			return node.ConnectRtpServer(ctx, app, stream, host, port)
		}
		if err := s.sip.SendInvitePlay(device, channel, sdp, ssrc, stream, streamMode, tcpConnect); err != nil {
			applog.Warnf("[GB28181 play 4/6] SIP INVITE FAILED device=%s channel=%s ssrc=%s rtpPort=%d err=%v",
				device.DeviceID, channel.GBDeviceID, ssrc, rtpPort, err)
		}
	}()

	pushURL := mediaurl.BuildGB28181PushURL(sdpIP, rtpPort, tcpMode)
	content := s.buildStreamContent(app, stream, node)
	mediaurl.LogPlayStreamPaths("[GB28181 play 6/6] URLs ready (invite async)", app, stream, pushURL, node.PlayURLs(app, stream))
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
	app := mediaurl.LiveApp
	stream := s.liveStreamName(deviceID, channelDeviceID)
	_ = s.sip.CloseInviteSession(stream)
	err := s.closeStream(app, stream)
	s.media.UnbindStream(app, stream)
	// 兼容旧命名（协议标识拼接）：内码切换后仍能清理历史流。
	if legacy := fmt.Sprintf("%s_%s", deviceID, channelDeviceID); legacy != stream {
		_ = s.sip.CloseInviteSession(legacy)
		_ = s.closeStream(app, legacy)
		s.media.UnbindStream(app, legacy)
	}
	return err
}

// liveStreamName 统一流名：优先内码，查不到时回退协议标识拼接（兼容期）。
func (s *Service) liveStreamName(deviceID, channelDeviceID string) string {
	if dev, err := s.devices.GetByDeviceID(deviceID); err == nil {
		if ch, cerr := s.channels.GetOne(deviceID, channelDeviceID); cerr == nil {
			if name := mediaurl.LiveStream(dev.InternalCode, ch.InternalCode); name != "" {
				return name
			}
		}
	}
	return fmt.Sprintf("%s_%s", deviceID, channelDeviceID)
}

func (s *Service) OnStreamStarted(app, stream string) {
	key := streamKey(app, stream)
	applog.Debugf("[GB28181 play 5/6] hook on_stream_changed regist app=%s stream=%s key=%s", app, stream, key)
	if nodeID, ok := s.media.StreamNodeID(app, stream); ok {
		if node, err := s.media.Resolve(context.Background(), nodeID); err == nil {
			mediaurl.LogPlayStreamPaths("[GB28181 play 5/6] pull URLs", app, stream, "", node.PlayURLs(app, stream))
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
	return int(s.ssrcSeq.Add(1))
}

func streamKey(app, stream string) string {
	return app + "/" + stream
}

// streamIdentificationForSDP 把通道码流类型映射为点播 SDP 的 a= 属性值。
// 主码流(stream:0/streamnumber:0)是设备默认，不下发属性以保持原有行为；
// 仅当选择子码流等非主码流时才下发 a=<ident>。
func streamIdentificationForSDP(v string) string {
	switch strings.TrimSpace(v) {
	case "", "stream:0", "streamnumber:0":
		return ""
	}
	return strings.TrimSpace(v)
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
