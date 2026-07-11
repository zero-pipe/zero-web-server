package streamproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mediaserverapp "zero-web-kit/internal/application/mediaserver"
	"zero-web-kit/internal/infrastructure/media/mediakit"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"
	"zero-web-kit/internal/interfaces/http/dto"
)

type Service struct {
	repo         *persistence.StreamProxyRepository
	mediaServers *mediaserverapp.Service
	serverID     string
	sessions     sync.Map
}

func NewService(repo *persistence.StreamProxyRepository, mediaServers *mediaserverapp.Service, serverID string) *Service {
	return &Service{repo: repo, mediaServers: mediaServers, serverID: serverID}
}

func (s *Service) List(page, count int, query string, pulling *bool, mediaServerID string) ([]model.StreamProxy, int64, error) {
	return s.repo.List(page, count, query, pulling, mediaServerID)
}

func (s *Service) Add(m *model.StreamProxy, gbDeviceID, gbName string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	m.CreateTime = now
	m.UpdateTime = now
	m.ServerID = s.serverID
	if m.Type == "" {
		m.Type = "default"
	}
	if m.MediaServerID == "" {
		m.MediaServerID = "auto"
	}
	if err := s.repo.Create(m); err != nil {
		return err
	}
	if gbDeviceID != "" {
		name := gbName
		if name == "" {
			name = m.Name
		}
		return s.repo.UpsertGBChannel(m.ID, gbDeviceID, name, m.App, m.Stream)
	}
	return nil
}

func (s *Service) Update(m *model.StreamProxy) error {
	m.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	return s.repo.Update(m)
}

func (s *Service) Delete(id int) error {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if proxy.StreamKey != "" {
		if node, err := s.resolveProxy(proxy); err == nil {
			_, _ = node.Client.DelStreamProxy(context.Background(), proxy.StreamKey)
		}
	}
	return s.repo.Delete(id)
}

func (s *Service) Start(ctx context.Context, id int) (*dto.StreamContent, error) {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("代理不存在")
	}
	node, err := s.resolveProxy(proxy)
	if err != nil {
		return nil, err
	}
	rtpType := proxy.RTSPType
	if rtpType == "" {
		rtpType = "0"
	}
	done := make(chan *dto.StreamContent, 1)
	key := streamKey(proxy.App, proxy.Stream)
	s.sessions.Store(key, done)
	defer s.sessions.Delete(key)

	resp, err := node.Client.AddStreamProxy(ctx, "__defaultVhost__", proxy.App, proxy.Stream, proxy.SrcURL, rtpType, false, proxy.EnableMP4, true)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("addStreamProxy: %s", resp.Msg)
	}
	var data struct {
		Key string `json:"key"`
	}
	_ = jsonUnmarshal(resp.Data, &data)
	if data.Key != "" {
		proxy.StreamKey = data.Key
		proxy.Pulling = true
		proxy.MediaServerID = node.ID()
		_ = s.repo.Update(proxy)
	}
	s.mediaServers.BindStream(proxy.App, proxy.Stream, node.ID())

	select {
	case content := <-done:
		return content, nil
	case <-time.After(15 * time.Second):
		return s.buildStreamContent(proxy.App, proxy.Stream, node), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Service) Stop(id int) error {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if node, err := s.resolveProxy(proxy); err == nil {
		if proxy.StreamKey != "" {
			_, _ = node.Client.DelStreamProxy(context.Background(), proxy.StreamKey)
		}
		_, _ = node.Client.CloseStreams(context.Background(), "__defaultVhost__", proxy.App, proxy.Stream)
	}
	s.mediaServers.UnbindStream(proxy.App, proxy.Stream)
	proxy.Pulling = false
	proxy.StreamKey = ""
	return s.repo.Update(proxy)
}

func (s *Service) FFmpegCmdList(mediaServerID string) map[string]string {
	return map[string]string{
		"ffmpeg.cmd": "-re -i ${url} -c copy -f flv ${dst_url}",
	}
}

func (s *Service) CloseOnNoneReader(app, stream string) *bool {
	proxy, err := s.repo.GetByAppStream(app, stream)
	if err != nil {
		return nil
	}
	v := proxy.EnableDisableNoneReader
	return &v
}

func (s *Service) OnStreamStarted(app, stream string) {
	key := streamKey(app, stream)
	if v, ok := s.sessions.Load(key); ok {
		if ch, ok := v.(chan *dto.StreamContent); ok {
			node, _ := s.mediaServers.ResolveForStream(app, stream, "auto")
			select {
			case ch <- s.buildStreamContent(app, stream, node):
			default:
			}
		}
	}
}

func (s *Service) resolveProxy(proxy *model.StreamProxy) (*mediaserverapp.Node, error) {
	prefer := proxy.MediaServerID
	if prefer == "" {
		prefer = "auto"
	}
	return s.mediaServers.ResolveForStream(proxy.App, proxy.Stream, prefer)
}

func (s *Service) buildStreamContent(app, stream string, node *mediaserverapp.Node) *dto.StreamContent {
	if node == nil {
		return &dto.StreamContent{App: app, Stream: stream, ServerID: s.serverID}
	}
	urls := mediakit.BuildPlayURLsFromConfig(node.MediaConfig(), app, stream)
	return &dto.StreamContent{
		App: app, Stream: stream, IP: node.StreamIP(),
		Flv: urls["flv"], WsFlv: urls["ws"], Hls: urls["hls"],
		Rtmp: urls["rtmp"], Rtsp: urls["rtsp"],
		MediaServerID: node.ID(), ServerID: s.serverID,
	}
}

func streamKey(app, stream string) string { return app + "/" + stream }

func jsonUnmarshal(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}
