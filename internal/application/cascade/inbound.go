package cascadeapp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	playapp "zero-web-server/internal/application/play"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/port"
	applog "zero-web-server/pkg/log"

	"github.com/zero-pipe/gb28181-go/manscdp"
	gbserver "github.com/zero-pipe/gb28181-go/server"
	"github.com/zero-pipe/gb28181-go/sdp"
)

// CatalogPusher 向上级平台推送目录（订阅后主动通知）。
type CatalogPusher interface {
	PushCatalog(platformID int) error
}

// CascadePlayer 承载上级平台发起的回放/下载，返回应答 SDP 与叶子流名。
type CascadePlayer interface {
	PrepareCascadePlayback(ctx context.Context, deviceID, channelGBID, startTime, endTime string, download bool, preferSSRC string) (answerSDP []byte, stream string, err error)
	StopCascade(stream string) error
}

// PlaybackController 向下级设备转发回放控制（MANSRTSP）。
type PlaybackController interface {
	SendPlaybackControl(stream, content string) error
}

// InboundService forwards superior INVITE/PTZ to leaf devices.
type InboundService struct {
	resolver   *Resolver
	devices    domaindevice.Repository
	play       *playapp.Service
	playback   CascadePlayer
	playCtl    PlaybackController
	sip        port.GBDeviceControl
	platformID string
	serverName string
	pusher     CatalogPusher

	mu      sync.Mutex
	callMap map[string]cascadeCall // callID -> leaf stream key
}

type cascadeCall struct {
	DeviceID    string
	ChannelGBID string
	Stream      string
	Kind        string // live / playback / download
}

func NewInboundService(
	resolver *Resolver,
	devices domaindevice.Repository,
	play *playapp.Service,
	sipServer port.GBDeviceControl,
	playback CascadePlayer,
	playCtl PlaybackController,
	platformID, serverName string,
	pusher CatalogPusher,
) *InboundService {
	return &InboundService{
		resolver: resolver, devices: devices, play: play, sip: sipServer,
		playback: playback, playCtl: playCtl,
		platformID: platformID, serverName: serverName, pusher: pusher,
		callMap: make(map[string]cascadeCall),
	}
}

func (s *InboundService) UpstreamKnown(upstreamGBID string) bool {
	_, err := s.resolver.platforms.GetByServerGBID(strings.TrimSpace(upstreamGBID))
	return err == nil
}

func (s *InboundService) OnDeviceControl(ctx context.Context, ev gbserver.InboundControlEvent) error {
	_ = ctx
	resolved, err := s.resolver.Resolve(ev.UpstreamGBID, ev.ChannelGBID)
	if err != nil {
		applog.Warnf("[cascade] PTZ resolve failed upstream=%s channel=%s err=%v", ev.UpstreamGBID, ev.ChannelGBID, err)
		return err
	}
	if strings.TrimSpace(ev.PTZCmd) == "" {
		return fmt.Errorf("empty PTZCmd")
	}
	device, err := s.devices.GetByDeviceID(resolved.DeviceID)
	if err != nil {
		return err
	}
	sn := ev.SN
	if sn == "" {
		sn = "1"
	}
	body := manscdp.BuildDeviceControlPTZ(resolved.ChannelGBID, sn, ev.PTZCmd)
	applog.Infof("[cascade] PTZ forward upstream=%s catalog=%s -> device=%s channel=%s",
		ev.UpstreamGBID, ev.ChannelGBID, resolved.DeviceID, resolved.ChannelGBID)
	return s.sip.SendDeviceControl(device, resolved.ChannelGBID, body)
}

func (s *InboundService) OnInvite(ctx context.Context, ev gbserver.InboundInviteEvent) ([]byte, error) {
	resolved, err := s.resolver.Resolve(ev.UpstreamGBID, ev.ChannelGBID)
	if err != nil {
		return nil, err
	}
	offer := sdp.ParseOffer(ev.OfferSDP)
	ssrc := offer.SSRC
	if ssrc == "" {
		ssrc = extractSubjectSSRC(ev.Subject)
	}

	var answer []byte
	var stream string
	kind := "live"
	switch offer.Kind {
	case sdp.OfferPlayback, sdp.OfferDownload:
		if s.playback == nil {
			return nil, fmt.Errorf("回放服务未就绪")
		}
		download := offer.Kind == sdp.OfferDownload
		answer, stream, err = s.playback.PrepareCascadePlayback(
			ctx, resolved.DeviceID, resolved.ChannelGBID, offer.StartTime, offer.EndTime, download, ssrc)
		if download {
			kind = "download"
		} else {
			kind = "playback"
		}
	default:
		var liveAnswer string
		liveAnswer, stream, err = s.play.PrepareCascadePlay(ctx, resolved.DeviceID, resolved.ChannelGBID, ssrc)
		answer = []byte(liveAnswer)
	}
	if err != nil {
		return nil, err
	}
	if ev.CallID != "" {
		s.mu.Lock()
		s.callMap[ev.CallID] = cascadeCall{
			DeviceID: resolved.DeviceID, ChannelGBID: resolved.ChannelGBID, Stream: stream, Kind: kind,
		}
		s.mu.Unlock()
	}
	applog.Infof("[cascade] INVITE %s forward upstream=%s catalog=%s -> device=%s stream=%s",
		kind, ev.UpstreamGBID, ev.ChannelGBID, resolved.DeviceID, stream)
	return answer, nil
}

func (s *InboundService) OnInviteEnd(ctx context.Context, callID string) error {
	_ = ctx
	s.mu.Lock()
	call, ok := s.callMap[callID]
	if ok {
		delete(s.callMap, callID)
	}
	s.mu.Unlock()
	if !ok {
		return nil
	}
	if call.Kind == "playback" || call.Kind == "download" {
		if s.playback != nil {
			return s.playback.StopCascade(call.Stream)
		}
		return nil
	}
	return s.play.StopPlay(call.DeviceID, call.ChannelGBID)
}

// OnInfo 把上级的回放控制（PLAY/PAUSE/Scale/Seek）转发给对应叶子会话。
func (s *InboundService) OnInfo(ctx context.Context, ev gbserver.InboundInfoEvent) error {
	_ = ctx
	s.mu.Lock()
	call, ok := s.callMap[ev.CallID]
	s.mu.Unlock()
	if !ok {
		applog.Warnf("[cascade] INFO for unknown dialog callID=%s", ev.CallID)
		return nil
	}
	if s.playCtl == nil || call.Stream == "" {
		return fmt.Errorf("回放控制未就绪")
	}
	if err := s.playCtl.SendPlaybackControl(call.Stream, string(ev.Body)); err != nil {
		return err
	}
	applog.Infof("[cascade] INFO forwarded callID=%s stream=%s", ev.CallID, call.Stream)
	return nil
}

func extractSubjectSSRC(subject string) string {
	parts := strings.Split(subject, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if i := strings.LastIndex(p, ":"); i >= 0 && i+1 < len(p) {
			cand := p[i+1:]
			if len(cand) >= 4 {
				return cand
			}
		}
	}
	return ""
}

// OnSubscribe 接受上级平台的目录/报警/移动位置订阅，并在目录订阅后推送一次目录。
func (s *InboundService) OnSubscribe(ctx context.Context, ev gbserver.InboundSubscribeEvent) (int, bool, error) {
	platform, err := s.resolver.platforms.GetByServerGBID(strings.TrimSpace(ev.UpstreamGBID))
	if err != nil || platform == nil {
		return 0, false, nil
	}
	expires := 600
	switch strings.ToLower(strings.TrimSpace(ev.Event)) {
	case "alarm", "mobileposition":
		expires = 3600
	}
	if expires != 0 && ev.Expires != 0 && s.pusher != nil && strings.EqualFold(strings.TrimSpace(ev.Event), "Catalog") {
		go func(platformID int) {
			time.Sleep(200 * time.Millisecond)
			if err := s.pusher.PushCatalog(platformID); err != nil {
				applog.Warnf("[cascade] catalog push after subscribe failed platform=%d err=%v", platformID, err)
			}
		}(platform.ID)
	}
	applog.Infof("[cascade] subscribe accepted upstream=%s event=%s expires=%d", ev.UpstreamGBID, ev.Event, expires)
	return expires, true, nil
}

// OnQuery 应答上级平台的目录/设备信息/设备状态查询。
func (s *InboundService) OnQuery(ctx context.Context, ev gbserver.InboundQueryEvent) ([]byte, bool) {
	_ = ctx
	platform, err := s.resolver.platforms.GetByServerGBID(strings.TrimSpace(ev.UpstreamGBID))
	if err != nil || platform == nil {
		return nil, false
	}
	switch ev.CmdType {
	case "Catalog":
		items, err := s.resolver.ListSharedCatalogItems(platform.ID)
		if err != nil {
			applog.Warnf("[cascade] catalog query failed platform=%d err=%v", platform.ID, err)
			return nil, false
		}
		applog.Infof("[cascade] catalog query upstream=%s items=%d", ev.UpstreamGBID, len(items))
		return []byte(manscdp.BuildCatalogResponse(s.platformID, ev.SN, items)), true
	case "DeviceInfo":
		name := s.serverName
		if name == "" {
			name = s.platformID
		}
		return []byte(manscdp.BuildDeviceInfoResponse(s.platformID, ev.SN, name, "zero-pipe", "zws", "")), true
	case "DeviceStatus":
		return []byte(manscdp.BuildDeviceStatusResponse(s.platformID, ev.SN)), true
	}
	return nil, false
}

var _ gbserver.CascadeInboundHandler = (*InboundService)(nil)
