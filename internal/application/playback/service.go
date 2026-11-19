package playbackapp

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	publishauth "zero-web-server/internal/application/publishauth"
	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	domainrecord "zero-web-server/internal/domain/record"
	"zero-web-server/internal/dto"
	"zero-web-server/internal/gbsip"
	"zero-web-server/internal/mediaurl"
	"zero-web-server/internal/port"
)

type DownloadProgress struct {
	Progress      float64 `json:"progress"`
	DownloadSpeed int     `json:"downloadSpeed"`
	Stream        string  `json:"stream"`
}

type Service struct {
	devices       domaindevice.Repository
	channels      domainchannel.Repository
	sip           port.GBPlaybacker
	media         port.MediaCluster
	serverID      string
	ssrcSeq       atomic.Int64
	sessions      sync.Map
	recordTimeout time.Duration
}

func NewService(
	devices domaindevice.Repository,
	channels domainchannel.Repository,
	sipServer port.GBPlaybacker,
	media port.MediaCluster,
	serverID string,
	recordTimeoutSec int,
) *Service {
	if recordTimeoutSec <= 0 {
		recordTimeoutSec = 30
	}
	return &Service{
		devices: devices, channels: channels, sip: sipServer,
		media: media, serverID: serverID,
		recordTimeout: time.Duration(recordTimeoutSec) * time.Second,
	}
}

func (s *Service) QueryRecord(ctx context.Context, deviceID, channelDeviceID, startTime, endTime string) (*domainrecord.RecordInfo, error) {
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	channel, err := s.channels.GetOne(deviceID, channelDeviceID)
	if err != nil {
		return nil, fmt.Errorf("通道不存在")
	}
	sn, ch := s.sip.SendRecordInfoQuery(device, channel.GBDeviceID, startTime, endTime)
	select {
	case info := <-ch:
		return info, nil
	case <-ctx.Done():
		s.sip.CancelRecordQuery(sn)
		return nil, ctx.Err()
	case <-time.After(s.recordTimeout):
		s.sip.CancelRecordQuery(sn)
		return nil, gbsip.ErrRecordTimeout
	}
}

func (s *Service) StartPlayback(ctx context.Context, deviceID, channelDeviceID, startTime, endTime string) (*dto.StreamContent, error) {
	device, channel, err := s.loadDeviceChannel(deviceID, channelDeviceID)
	if err != nil {
		return nil, err
	}
	app := publishauth.LiveApp
	stream := mediaurl.PlaybackStream(device.InternalCode, channel.InternalCode)
	if stream == "" {
		return nil, fmt.Errorf("设备或通道内码缺失")
	}
	content, err := s.startMediaInvite(ctx, device, channel, app, stream, gbsip.SessionPlayback, startTime, endTime, 0,
		func(d *domaindevice.Device, ch *domainchannel.Channel, rtpPort int, ssrc string, node port.MediaEndpoint) string {
			sdpIP := d.SDPIP
			if sdpIP == "" {
				sdpIP = node.SDPIP()
			}
			return gbsip.BuildPlaybackSDP(d.DeviceID, ch.GBDeviceID, sdpIP, rtpPort, ssrc, gbsip.NormalizeStreamMode(d.StreamMode), startTime, endTime)
		})
	if err != nil {
		return nil, err
	}
	content.Stream = stream
	return content, nil
}

func (s *Service) StartDownload(ctx context.Context, deviceID, channelDeviceID, startTime, endTime string, downloadSpeed int) (*dto.StreamContent, error) {
	device, channel, err := s.loadDeviceChannel(deviceID, channelDeviceID)
	if err != nil {
		return nil, err
	}
	if downloadSpeed <= 0 {
		downloadSpeed = 4
	}
	app := publishauth.LiveApp
	stream := mediaurl.DownloadStream(device.InternalCode, channel.InternalCode)
	if stream == "" {
		return nil, fmt.Errorf("设备或通道内码缺失")
	}
	content, err := s.startMediaInvite(ctx, device, channel, app, stream, gbsip.SessionDownload, startTime, endTime, downloadSpeed,
		func(d *domaindevice.Device, ch *domainchannel.Channel, rtpPort int, ssrc string, node port.MediaEndpoint) string {
			sdpIP := d.SDPIP
			if sdpIP == "" {
				sdpIP = node.SDPIP()
			}
			return gbsip.BuildDownloadSDP(d.DeviceID, ch.GBDeviceID, sdpIP, rtpPort, ssrc, gbsip.NormalizeStreamMode(d.StreamMode), startTime, endTime, downloadSpeed)
		})
	if err != nil {
		return nil, err
	}
	content.Stream = stream
	return content, nil
}

// PrepareCascadePlayback serves a superior's Playback/Download INVITE: it opens
// the media server stream, INVITEs the leaf device and returns our answer SDP.
func (s *Service) PrepareCascadePlayback(ctx context.Context, deviceID, channelGBID, startTime, endTime string, download bool, preferSSRC string) ([]byte, string, error) {
	device, channel, err := s.loadDeviceChannel(deviceID, channelGBID)
	if err != nil {
		return nil, "", err
	}
	app := publishauth.LiveApp
	stream := mediaurl.PlaybackStream(device.InternalCode, channel.InternalCode)
	if download {
		stream = mediaurl.DownloadStream(device.InternalCode, channel.InternalCode)
	}
	if stream == "" {
		return nil, "", fmt.Errorf("设备或通道内码缺失")
	}
	node, err := s.media.ResolveForStream(ctx, app, stream, device.MediaServerID)
	if err != nil {
		return nil, "", err
	}
	if _, has := s.sip.InviteManager().Get(stream); has {
		_ = s.sip.CloseInviteSession(stream)
		_ = node.CloseStreams(ctx, "__defaultVhost__", app, stream)
		s.media.UnbindStream(app, stream)
	}
	sdpIP := node.SDPIP()
	streamMode := gbsip.NormalizeStreamMode(device.StreamMode)
	rtpPort, err := node.OpenRtpServer(ctx, app, stream, 0, streamModeToTCP(streamMode))
	if err != nil {
		return nil, "", fmt.Errorf("打开RTP端口失败: %w", err)
	}
	s.media.BindStream(node.ID(), app, stream)

	ssrc := preferSSRC
	if ssrc == "" {
		ssrc = gbsip.PlaybackSSRC(s.sip.Domain(), s.nextSSRCSeq())
	}
	var answer string
	if download {
		answer = gbsip.BuildDownloadSDP(device.DeviceID, channel.GBDeviceID, sdpIP, rtpPort, ssrc, streamMode, startTime, endTime, 4)
	} else {
		answer = gbsip.BuildPlaybackSDP(device.DeviceID, channel.GBDeviceID, sdpIP, rtpPort, ssrc, streamMode, startTime, endTime)
	}

	sessionType := gbsip.SessionPlayback
	if download {
		sessionType = gbsip.SessionDownload
	}
	deviceSDP := answer
	go func() {
		_ = s.sip.SendInviteSession(device, channel, deviceSDP, ssrc, stream, sessionType, startTime, endTime, 4)
	}()
	return []byte(answer), stream, nil
}

// StopCascade tears down a stream started for a superior's INVITE.
func (s *Service) StopCascade(stream string) error {
	if stream == "" {
		return nil
	}
	_ = s.sip.CloseInviteSession(stream)
	if node, err := s.media.ResolveForStream(context.Background(), publishauth.LiveApp, stream, "auto"); err == nil {
		_ = node.CloseStreams(context.Background(), "__defaultVhost__", publishauth.LiveApp, stream)
	}
	s.media.UnbindStream(publishauth.LiveApp, stream)
	return nil
}

func (s *Service) StopPlayback(deviceID, channelDeviceID, stream string) error {
	if stream == "" {
		device, channel, err := s.loadDeviceChannel(deviceID, channelDeviceID)
		if err != nil {
			return err
		}
		stream = mediaurl.PlaybackStream(device.InternalCode, channel.InternalCode)
		if stream == "" {
			return fmt.Errorf("设备或通道内码缺失")
		}
	}
	_ = s.sip.CloseInviteSession(stream)
	if node, err := s.media.ResolveForStream(context.Background(), publishauth.LiveApp, stream, "auto"); err == nil {
		_ = node.CloseStreams(context.Background(), "__defaultVhost__", publishauth.LiveApp, stream)
	}
	s.media.UnbindStream(publishauth.LiveApp, stream)
	return nil
}

func (s *Service) StopDownload(deviceID, channelDeviceID, stream string) error {
	if stream == "" {
		device, channel, err := s.loadDeviceChannel(deviceID, channelDeviceID)
		if err != nil {
			return err
		}
		stream = mediaurl.DownloadStream(device.InternalCode, channel.InternalCode)
		if stream == "" {
			return fmt.Errorf("设备或通道内码缺失")
		}
	}
	_ = s.sip.CloseInviteSession(stream)
	if node, err := s.media.ResolveForStream(context.Background(), publishauth.LiveApp, stream, "auto"); err == nil {
		_ = node.CloseStreams(context.Background(), "__defaultVhost__", publishauth.LiveApp, stream)
	}
	s.media.UnbindStream(publishauth.LiveApp, stream)
	return nil
}

// HandleMediaStatus reacts to a GB28181 MediaStatus notify. NotifyType 121
// ("File to End") means the device finished streaming a playback/download file,
// so we tear the session down instead of waiting for the client to stop it
// (2016 §7.8/§9.9, A.2.5.4).
func (s *Service) HandleMediaStatus(deviceID, channelGBID, notifyType string) error {
	if notifyType != "121" {
		return nil
	}
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil || device == nil {
		return nil
	}
	channel, err := s.channels.GetOne(deviceID, channelGBID)
	if err != nil || channel == nil {
		return nil
	}
	for _, stream := range []string{
		mediaurl.PlaybackStream(device.InternalCode, channel.InternalCode),
		mediaurl.DownloadStream(device.InternalCode, channel.InternalCode),
	} {
		if stream == "" {
			continue
		}
		if _, ok := s.sip.InviteManager().Get(stream); !ok {
			continue
		}
		_ = s.sip.CloseInviteSession(stream)
		if node, err := s.media.ResolveForStream(context.Background(), publishauth.LiveApp, stream, "auto"); err == nil {
			_ = node.CloseStreams(context.Background(), "__defaultVhost__", publishauth.LiveApp, stream)
		}
		s.media.UnbindStream(publishauth.LiveApp, stream)
		log.Printf("[GB28181] media status ended: stream=%s device=%s channel=%s", stream, deviceID, channelGBID)
	}
	return nil
}

func (s *Service) DownloadProgress(stream string) (*DownloadProgress, error) {	sess, ok := s.sip.InviteManager().Get(stream)
	if !ok {
		return &DownloadProgress{Progress: 100, Stream: stream}, nil
	}
	return &DownloadProgress{
		Progress:      sess.Progress(),
		DownloadSpeed: sess.DownloadSpeed(),
		Stream:        stream,
	}, nil
}

func (s *Service) PausePlayback(streamID string) error {
	return s.sip.SendPlaybackControl(streamID, gbsip.BuildPlaybackPause(s.sip.NextInfoCSeq()))
}

func (s *Service) ResumePlayback(streamID string) error {
	return s.sip.SendPlaybackControl(streamID, gbsip.BuildPlaybackResume(s.sip.NextInfoCSeq()))
}

func (s *Service) SpeedPlayback(streamID string, speed float64) error {
	return s.sip.SendPlaybackControl(streamID, gbsip.BuildPlaybackSpeed(s.sip.NextInfoCSeq(), speed))
}

func (s *Service) SeekPlayback(streamID string, seekTime int64) error {
	return s.sip.SendPlaybackControl(streamID, gbsip.BuildPlaybackSeek(s.sip.NextInfoCSeq(), seekTime))
}

func (s *Service) OnStreamStarted(app, stream string) {
	key := streamKey(app, stream)
	if v, ok := s.sessions.Load(key); ok {
		if ch, ok := v.(chan *dto.StreamContent); ok {
			select {
			case ch <- s.buildStreamContent(app, stream):
			default:
			}
		}
	}
}

func (s *Service) loadDeviceChannel(deviceID, channelDeviceID string) (*domaindevice.Device, *domainchannel.Channel, error) {
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return nil, nil, fmt.Errorf("设备不存在")
	}
	channel, err := s.channels.GetOne(deviceID, channelDeviceID)
	if err != nil {
		return nil, nil, fmt.Errorf("通道不存在")
	}
	return device, channel, nil
}

type sdpBuilder func(*domaindevice.Device, *domainchannel.Channel, int, string, port.MediaEndpoint) string

func (s *Service) startMediaInvite(
	ctx context.Context, device *domaindevice.Device, channel *domainchannel.Channel,
	app, stream string, sessionType gbsip.SessionType,
	startTime, endTime string, downloadSpeed int, buildSDP sdpBuilder,
) (*dto.StreamContent, error) {
	node, err := s.media.ResolveForStream(ctx, app, stream, device.MediaServerID)
	if err != nil {
		return nil, err
	}
	mediaMode := gbsip.NormalizeStreamMode(device.StreamMode)
	rtpPort, err := node.OpenRtpServer(ctx, app, stream, 0, streamModeToTCP(mediaMode))
	if err != nil {
		return nil, fmt.Errorf("打开RTP端口失败: %w", err)
	}
	s.media.BindStream(node.ID(), app, stream)

	ssrc := gbsip.PlaySSRC(s.sip.Domain(), s.nextSSRCSeq())
	if sessionType != gbsip.SessionPlay {
		// 历史回放/下载 SSRC 首位为 1（附录-编码与SDP）
		ssrc = gbsip.PlaybackSSRC(s.sip.Domain(), s.nextSSRCSeq())
	}
	sdp := buildSDP(device, channel, rtpPort, ssrc, node)

	done := make(chan *dto.StreamContent, 1)
	s.sessions.Store(streamKey(app, stream), done)
	defer s.sessions.Delete(streamKey(app, stream))

	go func() {
		_ = s.sip.SendInviteSession(device, channel, sdp, ssrc, stream, sessionType, startTime, endTime, downloadSpeed)
	}()

	select {
	case content := <-done:
		return content, nil
	case <-time.After(15 * time.Second):
		return s.buildStreamContentWithNode(app, stream, node), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Service) buildStreamContent(app, stream string) *dto.StreamContent {
	node, err := s.media.ResolveForStream(context.Background(), app, stream, "auto")
	if err != nil {
		return &dto.StreamContent{App: app, Stream: stream, ServerID: s.serverID}
	}
	return s.buildStreamContentWithNode(app, stream, node)
}

func (s *Service) buildStreamContentWithNode(app, stream string, node port.MediaEndpoint) *dto.StreamContent {
	urls := node.StreamPlayURLs(app, stream, false, 0)
	content := &dto.StreamContent{
		App: app, Stream: stream, IP: node.StreamIP(),
		Flv: urls.Flv, WsFlv: urls.WsFlv, Hls: urls.Hls,
		Rtmp: urls.Rtmp, Rtsp: urls.Rtsp,
		Rtc: urls.Rtc, Rtcs: urls.Rtcs,
		MediaServerID: node.ID(), ServerID: s.serverID,
	}
	if info := node.LookupStream(context.Background(), app, stream); info != nil {
		content.VideoCodec = info.VideoCodec
		content.AudioCodec = info.AudioCodec
	}
	return content
}

func (s *Service) nextSSRCSeq() int {
	return int(s.ssrcSeq.Add(1))
}

func streamKey(app, stream string) string { return app + "/" + stream }

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

func ParseDownloadSpeed(v string) int {
	speed, _ := strconv.Atoi(v)
	if speed <= 0 {
		return 4
	}
	return speed
}
