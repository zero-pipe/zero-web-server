package playapp

import (
	"context"
	"fmt"
	"sync"
	"time"

	domainchannel "zero-web-kit/internal/domain/channel"
	domaindevice "zero-web-kit/internal/domain/device"
	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/media/mediakit"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
	"zero-web-kit/internal/interfaces/http/dto"
	applog "zero-web-kit/pkg/log"
)

type Service struct {
	devices    domaindevice.Repository
	channels   domainchannel.Repository
	sip        *sipinfra.Server
	zlm        *mediakit.Client
	mediaCfg   config.MediaConfig
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
	zlmClient *mediakit.Client,
	mediaCfg config.MediaConfig,
	serverID string,
	serverPort int,
) *Service {
	return &Service{
		devices:    devices,
		channels:   channels,
		sip:        sipServer,
		zlm:        zlmClient,
		mediaCfg:   mediaCfg,
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

func (s *Service) startPlay(ctx context.Context, device *domaindevice.Device, channel *domainchannel.Channel) (*dto.StreamContent, error) {
	app := "rtp"
	stream := fmt.Sprintf("%s_%s", device.DeviceID, channel.GBDeviceID)
	sdpIP := device.SDPIP
	if sdpIP == "" {
		sdpIP = s.mediaCfg.IP
	}

	streamMode := sipinfra.NormalizeStreamMode(device.StreamMode)
	applog.Debugf("[GB28181 play 1/6] start device=%s channel=%s target=%s:%d sipTransport=%s mediaStreamMode=%s sdpIP=%s zms=%s",
		device.DeviceID, channel.GBDeviceID, device.IP, device.Port, device.Transport, streamMode, sdpIP, s.mediaCfg.BaseURL())

	tcpMode := streamModeToTCP(streamMode)
	rtpResp, err := s.zlm.OpenRtpServer(ctx, app, stream, 0, tcpMode)
	if err != nil {
		applog.Warnf("[GB28181 play 2/6] openRtpServer FAILED stream=%s tcp_mode=%d err=%v", stream, tcpMode, err)
		return nil, fmt.Errorf("open RTP server failed: %w", err)
	}
	applog.Debugf("[GB28181 play 2/6] openRtpServer OK stream=%s port=%d tcp_mode=%d", stream, rtpResp.Port, tcpMode)

	ssrc := sipinfra.PlaySSRC(s.sip.Domain(), s.nextSSRCSeq())
	sdp := sipinfra.BuildPlaySDP(device.DeviceID, sdpIP, rtpResp.Port, ssrc, streamMode, "")
	applog.Debugf("[GB28181 play 3/6] SDP ready ssrc=%s streamMode=%s port=%d", ssrc, streamMode, rtpResp.Port)

	done := make(chan *dto.StreamContent, 1)
	s.sessions.Store(streamKey(app, stream), done)
	defer s.sessions.Delete(streamKey(app, stream))

	go func() {
		applog.Debugf("[GB28181 play 4/6] SIP INVITE -> %s:%d channel=%s Subject SSRC=%s streamMode=%s",
			device.IP, device.Port, channel.GBDeviceID, ssrc, streamMode)
		tcpConnect := func(host string, port int) error {
			if streamMode != "TCP-ACTIVE" {
				return nil
			}
			return s.zlm.ConnectRtpServer(ctx, app, stream, host, port)
		}
		if err := s.sip.SendInvitePlay(device, channel, sdp, ssrc, stream, streamMode, tcpConnect, nil); err != nil {
			applog.Warnf("[GB28181 play 4/6] SIP INVITE FAILED device=%s channel=%s ssrc=%s rtpPort=%d err=%v",
				device.DeviceID, channel.GBDeviceID, ssrc, rtpResp.Port, err)
		}
	}()

	pushURL := mediakit.BuildGB28181PushURL(sdpIP, rtpResp.Port, tcpMode)
	select {
	case content := <-done:
		mediakit.LogPlayStreamPaths("[GB28181 play 6/6] hook OK", app, stream, pushURL, mediakit.BuildPlayURLsFromConfig(s.mediaCfg, app, stream))
		return content, nil
	case <-time.After(15 * time.Second):
		applog.Warnf("[GB28181 play 6/6] hook TIMEOUT 15s (no on_stream_changed), check camera RTP push stream=%s", stream)
		content := s.buildStreamContent(app, stream)
		mediakit.LogPlayStreamPaths("[GB28181 play 6/6] timeout (URLs may be stale)", app, stream, pushURL, mediakit.BuildPlayURLsFromConfig(s.mediaCfg, app, stream))
		return content, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Service) StopPlay(deviceID, channelDeviceID string) error {
	app := "rtp"
	stream := fmt.Sprintf("%s_%s", deviceID, channelDeviceID)
	_ = s.sip.CloseInviteSession(stream)
	_, err := s.zlm.CloseStreams(context.Background(), "__defaultVhost__", app, stream)
	return err
}

func (s *Service) OnStreamStarted(app, stream string) {
	key := streamKey(app, stream)
	applog.Debugf("[GB28181 play 5/6] hook on_stream_changed regist app=%s stream=%s key=%s", app, stream, key)
	mediakit.LogPlayStreamPaths("[GB28181 play 5/6] pull URLs", app, stream, "", mediakit.BuildPlayURLsFromConfig(s.mediaCfg, app, stream))
	if v, ok := s.sessions.Load(key); ok {
		if ch, ok := v.(chan *dto.StreamContent); ok {
			select {
			case ch <- s.buildStreamContent(app, stream):
			default:
			}
		}
	} else {
		applog.Debugf("[GB28181 play 5/6] hook arrived but no waiting session key=%s", key)
	}
}

func (s *Service) buildStreamContent(app, stream string) *dto.StreamContent {
	urls := mediakit.BuildStreamPlayURLs(s.mediaCfg, app, stream, false, s.serverPort)
	content := &dto.StreamContent{
		App:           app,
		Stream:        stream,
		IP:            s.mediaCfg.IP,
		Flv:           urls.Flv,
		WsFlv:         urls.WsFlv,
		Hls:           urls.Hls,
		Rtmp:          urls.Rtmp,
		Rtsp:          urls.Rtsp,
		Rtc:           urls.Rtc,
		Rtcs:          urls.Rtcs,
		MediaServerID: s.mediaCfg.ID,
		ServerID:      s.serverID,
	}
	if info := s.zlm.LookupStreamMediaInfo(context.Background(), app, stream); info != nil {
		content.VideoCodec = info.VideoCodec
		content.AudioCodec = info.AudioCodec
	}
	return content
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