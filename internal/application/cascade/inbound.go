package cascadeapp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	playapp "zero-web-kit/internal/application/play"
	domaindevice "zero-web-kit/internal/domain/device"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
	applog "zero-web-kit/pkg/log"

	"github.com/zero-pipe/gb28181-go/manscdp"
	gbserver "github.com/zero-pipe/gb28181-go/server"
)

// InboundService forwards superior INVITE/PTZ to leaf devices.
type InboundService struct {
	resolver *Resolver
	devices  domaindevice.Repository
	play     *playapp.Service
	sip      *sipinfra.Server

	mu      sync.Mutex
	callMap map[string]cascadeCall // callID -> leaf stream key
}

type cascadeCall struct {
	DeviceID    string
	ChannelGBID string
}

func NewInboundService(
	resolver *Resolver,
	devices domaindevice.Repository,
	play *playapp.Service,
	sipServer *sipinfra.Server,
) *InboundService {
	return &InboundService{
		resolver: resolver, devices: devices, play: play, sip: sipServer,
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
	ssrc := extractSubjectSSRC(ev.Subject)
	answer, stream, err := s.play.PrepareCascadePlay(ctx, resolved.DeviceID, resolved.ChannelGBID, ssrc)
	if err != nil {
		return nil, err
	}
	if ev.CallID != "" {
		s.mu.Lock()
		s.callMap[ev.CallID] = cascadeCall{DeviceID: resolved.DeviceID, ChannelGBID: resolved.ChannelGBID}
		s.mu.Unlock()
	}
	applog.Infof("[cascade] INVITE forward upstream=%s catalog=%s -> device=%s stream=%s",
		ev.UpstreamGBID, ev.ChannelGBID, resolved.DeviceID, stream)
	return []byte(answer), nil
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
	return s.play.StopPlay(call.DeviceID, call.ChannelGBID)
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

var _ gbserver.CascadeInboundHandler = (*InboundService)(nil)
