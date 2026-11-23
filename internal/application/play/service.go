package playapp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	// streamModeFn 返回本级「国标配置」的媒体流传输模式（全局默认）。
	// 为空时回退到设备自身 stream_mode 字段。
	streamModeFn func() string
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

// SetStreamModeProvider 注入本级「国标配置」的媒体流传输模式来源（全局默认）。
// 未注入时回退到设备自身 stream_mode。
func (s *Service) SetStreamModeProvider(fn func() string) {
	s.streamModeFn = fn
}

// resolveStreamMode 以本级「国标配置」的全局默认为准；未配置时回退设备自身 stream_mode，
// 两者皆空回退 UDP。标准接入页已不再提供逐设备设置，故全局优先，避免存量设备
// 落库的默认 UDP 覆盖全局选择。
func (s *Service) resolveStreamMode(device *domaindevice.Device) string {
	mode := ""
	if s.streamModeFn != nil {
		mode = strings.TrimSpace(s.streamModeFn())
	}
	if mode == "" && device != nil {
		mode = strings.TrimSpace(device.StreamMode)
	}
	return gbsip.NormalizeStreamMode(mode)
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
	if stream == "" {
		return "", "", fmt.Errorf("设备或通道内码缺失")
	}

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
	streamMode := s.resolveStreamMode(device)
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
	if stream == "" {
		return nil, fmt.Errorf("设备或通道内码缺失")
	}

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
	// INVITE 会话还在但媒体未就绪：可能是设备重启后的僵死会话，也可能只是正在起流
	// （GB 从 INVITE 到首帧 SPS 到达有数秒窗口）。先给一小段宽限期等它自己就绪，
	// 避免把正在起流的会话误杀、反复 openRtpServer 导致“第一次播放失败”。
	if _, hasInvite := s.sip.InviteManager().Get(stream); hasInvite {
		if info := s.waitGBStreamReady(ctx, node, app, stream, gbInviteGraceWait); info != nil {
			applog.Infof("[GB28181 play] pending invite became ready, reuse stream=%s bytesSpeed=%d fps=%d %dx%d",
				stream, info.BytesSpeed, info.VideoFps, info.Width, info.Height)
			s.media.BindStream(node.ID(), app, stream)
			content := s.buildStreamContent(app, stream, node)
			mediaurl.LogPlayStreamPaths("[GB28181 play] reuse live URLs", app, stream, "", node.PlayURLs(app, stream))
			return content, nil
		}
		applog.Warnf("[GB28181 play] stale invite without media after grace, close and re-INVITE stream=%s", stream)
		_ = s.sip.CloseInviteSession(stream)
		_ = node.CloseStreams(ctx, "__defaultVhost__", app, stream)
	}

	sdpIP := device.SDPIP
	if sdpIP == "" {
		sdpIP = node.SDPIP()
	}

	streamMode := s.resolveStreamMode(device)
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
	// INVITE 之后 ZMS 要等摄像头发 RTP/PS 并解析出 SPS 才有可播的 SDP。这个窗口内
	// RTSP DESCRIBE 会返回 404（active_sources=1 但 Video=false），客户端立即连接就会
	//“第一次播放失败、第二次才成功”。这里等媒体真正就绪再返回播放地址。
	if info := s.waitGBStreamReady(ctx, node, app, stream, gbFirstFrameWait); info != nil {
		applog.Infof("[GB28181 play 5/6] stream ready stream=%s bytesSpeed=%d fps=%d %dx%d",
			stream, info.BytesSpeed, info.VideoFps, info.Width, info.Height)
	} else {
		applog.Warnf("[GB28181 play 5/6] stream not ready within wait, return URLs anyway stream=%s", stream)
	}
	content := s.buildStreamContent(app, stream, node)
	mediaurl.LogPlayStreamPaths("[GB28181 play 6/6] URLs ready (invite async)", app, stream, pushURL, node.PlayURLs(app, stream))
	return content, nil
}

const (
	// gbFirstFrameWait: INVITE 到首帧 SPS 到达的就绪等待上限（实测约 10s，留裕量）。
	// 客户端 HTTP 超时 30s，此处留出连接余量。
	gbFirstFrameWait = 20 * time.Second
	// gbInviteGraceWait: 已有 invite 会话但媒体未就绪时的宽限期，避免误杀正在起流的会话。
	gbInviteGraceWait = 3 * time.Second
	gbReadyPollTick   = 200 * time.Millisecond
)

// waitGBStreamReady 轮询探测直到该流可播或到达 maxWait；ctx 取消则提前返回 nil。
func (s *Service) waitGBStreamReady(ctx context.Context, node port.MediaEndpoint, app, stream string, maxWait time.Duration) *port.StreamProbe {
	deadline := time.Now().Add(maxWait)
	for {
		if info := node.LookupStream(ctx, app, stream); gbLiveStreamReady(info) {
			return info
		}
		if time.Now().After(deadline) {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(gbReadyPollTick):
		}
	}
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
	if stream == "" {
		return fmt.Errorf("设备或通道内码缺失")
	}
	_ = s.sip.CloseInviteSession(stream)
	err := s.closeStream(app, stream)
	s.media.UnbindStream(app, stream)
	return err
}

// liveStreamName 统一流名：<设备内码>_<通道内码>；缺失返回空串，由调用方拒绝。
func (s *Service) liveStreamName(deviceID, channelDeviceID string) string {
	dev, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return ""
	}
	ch, err := s.channels.GetOne(deviceID, channelDeviceID)
	if err != nil {
		return ""
	}
	return mediaurl.LiveStream(dev.InternalCode, ch.InternalCode)
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
