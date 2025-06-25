package streampush

import (
	"context"
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
	repo     *persistence.StreamPushRepository
	zlm      *mediakit.Client
	mediaCfg config.MediaConfig
	serverID string
	sessions sync.Map
}

func NewService(repo *persistence.StreamPushRepository, zlmClient *mediakit.Client, mediaCfg config.MediaConfig, serverID string) *Service {
	return &Service{repo: repo, zlm: zlmClient, mediaCfg: mediaCfg, serverID: serverID}
}

func (s *Service) List(page, count int, query string, pushing *bool, mediaServerID string) ([]persistence.StreamPushView, int64, error) {
	return s.repo.List(page, count, query, pushing, mediaServerID)
}

func (s *Service) Add(m *model.StreamPush, gbDeviceID, gbName string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	m.CreateTime = now
	m.UpdateTime = now
	m.ServerID = s.serverID
	m.MediaServerID = s.mediaCfg.ID
	if err := s.repo.Create(m); err != nil {
		return err
	}
	if gbDeviceID != "" {
		name := gbName
		if name == "" {
			name = m.Stream
		}
		return s.repo.UpsertGBChannel(m.ID, gbDeviceID, name)
	}
	return nil
}

func (s *Service) Update(m *model.StreamPush) error {
	m.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	return s.repo.Update(m)
}

func (s *Service) Remove(id int) error {
	_ = s.repo.RemoveGBChannel(id)
	return s.repo.Delete(id)
}

func (s *Service) BatchRemove(ids []int) error {
	for _, id := range ids {
		if err := s.Remove(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) SaveToGB(id int, gbDeviceID, gbName string) error {
	return s.repo.UpsertGBChannel(id, gbDeviceID, gbName)
}

func (s *Service) RemoveFromGB(id int) error {
	return s.repo.RemoveGBChannel(id)
}

func (s *Service) Start(ctx context.Context, id int) (*dto.StreamContent, error) {
	push, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("?????")
	}
	if !push.Pushing {
		return nil, fmt.Errorf("???????? RTMP/RTSP ??? %s/%s", push.App, push.Stream)
	}
	return s.buildStreamContent(push.App, push.Stream), nil
}

func (s *Service) OnPublish(app, stream, mediaServerID string) {
	if push, err := s.repo.GetByAppStream(app, stream); err == nil {
		_ = s.repo.UpdatePushing(push.ID, true, mediaServerID)
	}
}

func (s *Service) OnStreamDeparture(app, stream string) {
	if push, err := s.repo.GetByAppStream(app, stream); err == nil {
		_ = s.repo.UpdatePushing(push.ID, false, "")
	}
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
