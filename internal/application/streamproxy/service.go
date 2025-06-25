package streamproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/media/mediakit"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"
	"zero-web-kit/internal/interfaces/http/dto"
)

type Service struct {
	repo     *persistence.StreamProxyRepository
	zlm      *mediakit.Client
	mediaCfg config.MediaConfig
	serverID string
	sessions sync.Map
}

func NewService(repo *persistence.StreamProxyRepository, zlmClient *mediakit.Client, mediaCfg config.MediaConfig, serverID string) *Service {
	return &Service{repo: repo, zlm: zlmClient, mediaCfg: mediaCfg, serverID: serverID}
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
		m.MediaServerID = s.mediaCfg.ID
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
		_, _ = s.zlm.DelStreamProxy(context.Background(), proxy.StreamKey)
	}
	return s.repo.Delete(id)
}

func (s *Service) Start(ctx context.Context, id int) (*dto.StreamContent, error) {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("代理不存在")
	}
	rtpType := proxy.RTSPType
	if rtpType == "" {
		rtpType = "0"
	}
	done := make(chan *dto.StreamContent, 1)
	key := streamKey(proxy.App, proxy.Stream)
	s.sessions.Store(key, done)
	defer s.sessions.Delete(key)

	resp, err := s.zlm.AddStreamProxy(ctx, "__defaultVhost__", proxy.App, proxy.Stream, proxy.SrcURL, rtpType, false, proxy.EnableMP4, true)
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
		_ = s.repo.Update(proxy)
	}

	select {
	case content := <-done:
		return content, nil
	case <-time.After(15 * time.Second):
		return s.buildStreamContent(proxy.App, proxy.Stream), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Service) Stop(id int) error {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if proxy.StreamKey != "" {
		_, _ = s.zlm.DelStreamProxy(context.Background(), proxy.StreamKey)
	}
	_, _ = s.zlm.CloseStreams(context.Background(), "__defaultVhost__", proxy.App, proxy.Stream)
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
			select {
			case ch <- s.buildStreamContent(app, stream):
			default:
			}
		}
	}
}

func (s *Service) buildStreamContent(app, stream string) *dto.StreamContent {
	urls := mediakit.BuildPlayURLsFromConfig(s.mediaCfg, app, stream)
	return &dto.StreamContent{
		App: app, Stream: stream, IP: s.mediaCfg.IP,
		Flv: urls["flv"], WsFlv: urls["ws"], Hls: urls["hls"],
		Rtmp: urls["rtmp"], Rtsp: urls["rtsp"],
		MediaServerID: s.mediaCfg.ID, ServerID: s.serverID,
	}
}

func streamKey(app, stream string) string { return app + "/" + stream }

func jsonUnmarshal(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}
