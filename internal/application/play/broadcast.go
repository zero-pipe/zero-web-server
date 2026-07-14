package playapp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	domainchannel "zero-web-server/internal/domain/channel"
	"zero-web-server/internal/infrastructure/media/mediakit"
	"zero-web-server/internal/interfaces/http/dto"
)

const (
	AppBroadcast = "broadcast"
	AppTalk      = "talk"
)

type broadcastState struct {
	DeviceID  string
	ChannelID string
	ChannelDB int
	App       string
	Stream    string
}

type broadcastRegistry struct {
	mu    sync.Mutex
	items map[int]*broadcastState
}

func newBroadcastRegistry() *broadcastRegistry {
	return &broadcastRegistry{items: make(map[int]*broadcastState)}
}

func (r *broadcastRegistry) set(ch *domainchannel.Channel, app, stream string) {
	r.mu.Lock()
	r.items[ch.ID] = &broadcastState{
		DeviceID: ch.DeviceID, ChannelID: ch.GBDeviceID,
		ChannelDB: ch.ID, App: app, Stream: stream,
	}
	r.mu.Unlock()
}

func (r *broadcastRegistry) getByStream(app, stream string) *broadcastState {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, v := range r.items {
		if v.App == app && v.Stream == stream {
			return v
		}
	}
	return nil
}

func (r *broadcastRegistry) removeByDevice(deviceID, channelGBID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, v := range r.items {
		if v.DeviceID == deviceID && v.ChannelID == channelGBID {
			delete(r.items, id)
		}
	}
}

func (s *Service) AudioBroadcast(deviceID, channelGBID string, broadcastMode bool) (*dto.AudioBroadcastResult, error) {
	if _, err := s.devices.GetByDeviceID(deviceID); err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	channel, err := s.channels.GetOne(deviceID, channelGBID)
	if err != nil {
		return nil, fmt.Errorf("通道不存在")
	}
	if broadcastMode {
		app := AppBroadcast
		stream := deviceID + "_" + channelGBID
		s.broadcast.set(channel, app, stream)
		info := s.buildStreamContentWithWebRTC(app, stream, true)
		return &dto.AudioBroadcastResult{
			StreamInfo: info, Codec: "G.711", App: app, Stream: stream,
		}, nil
	}
	app := AppTalk
	stream := deviceID + "_" + channelGBID
	playStream := stream + "_talk"
	s.broadcast.set(channel, app, stream)
	return &dto.AudioBroadcastResult{
		StreamInfo:     s.buildStreamContentWithWebRTC(app, stream, true),
		PlayStreamInfo: s.buildStreamContentWithWebRTC(app, playStream, false),
		Codec:          "G.711", App: app, Stream: stream,
	}, nil
}

func (s *Service) StopAudioBroadcast(deviceID, channelGBID string) error {
	s.broadcast.removeByDevice(deviceID, channelGBID)
	for _, spec := range []struct{ app, stream string }{
		{AppBroadcast, deviceID + "_" + channelGBID},
		{AppTalk, deviceID + "_" + channelGBID},
		{AppTalk, deviceID + "_" + channelGBID + "_talk"},
	} {
		_ = s.closeStream(spec.app, spec.stream)
		s.media.UnbindStream(spec.app, spec.stream)
	}
	return nil
}

func (s *Service) OnBroadcastStreamArrival(app, stream string) {
	if app != AppBroadcast && app != AppTalk {
		return
	}
	if strings.Count(stream, "_") < 1 {
		return
	}
	state := s.broadcast.getByStream(app, stream)
	if state == nil {
		return
	}
	device, err := s.devices.GetByDeviceID(state.DeviceID)
	if err != nil {
		return
	}
	if app == AppBroadcast {
		_ = s.sip.SendAudioBroadcast(device, state.ChannelID)
	}
}

func (s *Service) buildStreamContentWithWebRTC(app, stream string, push bool) *dto.StreamContent {
	node, err := s.media.ResolveForStream(context.Background(), app, stream, "auto")
	var c *dto.StreamContent
	if err != nil {
		c = &dto.StreamContent{App: app, Stream: stream, ServerID: s.serverID}
	} else {
		c = s.buildStreamContent(app, stream, node)
		s.media.BindStream(node.ID(), app, stream)
	}
	base := fmt.Sprintf("http://127.0.0.1:%d", s.serverPort)
	if node != nil {
		base = node.SignalingBaseURL(s.serverPort)
	}
	rtc, rtcs := mediakit.BuildWebRTCURLs(base, app, stream, push)
	c.Rtc, c.Rtcs = rtc, rtcs
	return c
}
