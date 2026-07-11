package mediaserver

import (
	"context"
	"fmt"
	"time"

	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/media/mediakit"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"
)

type View struct {
	ID            string `json:"id"`
	IP            string `json:"ip"`
	HookIP        string `json:"hookIp"`
	StreamIP      string `json:"streamIp"`
	HTTPPort      int    `json:"httpPort"`
	RTMPPort      int    `json:"rtmpPort"`
	RTSPPort      int    `json:"rtspPort"`
	RTPProxyPort  int    `json:"rtpProxyPort"`
	Secret        string `json:"secret"`
	Type          string `json:"type"`
	DefaultServer bool   `json:"defaultServer"`
	Online        bool   `json:"status"`
	CreateTime    string `json:"createTime"`
}

type Service struct {
	repo     *persistence.MediaServerRepository
	zlm      *mediakit.Client
	mediaCfg config.MediaConfig
}

func NewService(repo *persistence.MediaServerRepository, zlmClient *mediakit.Client, mediaCfg config.MediaConfig) *Service {
	return &Service{repo: repo, zlm: zlmClient, mediaCfg: mediaCfg}
}

func (s *Service) EnsureDefault() {
	now := time.Now().Format("2006-01-02 15:04:05")
	row := &model.MediaServer{
		ID: s.mediaCfg.ID, IP: s.mediaCfg.IP, HookIP: s.mediaCfg.IP, SDPIP: s.mediaCfg.IP,
		StreamIP: s.mediaCfg.IP, HTTPPort: s.mediaCfg.HTTPPort,
		RTMPPort: 1935, RTSPPort: 8554, Secret: s.mediaCfg.Secret,
		Type: s.mediaCfg.BackendType(), DefaultServer: true,
		UpdateTime: now,
	}
	if existing, err := s.repo.GetByID(s.mediaCfg.ID); err == nil {
		row.CreateTime = existing.CreateTime
	} else {
		row.CreateTime = now
	}
	_ = s.repo.Save(row)
	_ = s.repo.ClearDefaultExcept(s.mediaCfg.ID)
}

func (s *Service) List() ([]View, error) {
	rows, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []View{s.defaultView()}, nil
	}
	out := make([]View, len(rows))
	for i, row := range rows {
		out[i] = s.toView(row, s.isOnline(row))
	}
	return out, nil
}

func (s *Service) ListOnline() ([]View, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	online := make([]View, 0, len(all))
	for _, v := range all {
		if v.Online {
			online = append(online, v)
		}
	}
	return online, nil
}

func (s *Service) GetOne(id string) (*View, error) {
	m, err := s.repo.GetByID(id)
	if err != nil {
		v := s.defaultView()
		if id == v.ID || id == "" {
			return &v, nil
		}
		return nil, fmt.Errorf("流媒体节点不存在")
	}
	v := s.toView(*m, s.isOnline(*m))
	return &v, nil
}

func (s *Service) Check(ip string, port int, secret string) (*View, error) {
	client := mediakit.NewClient(config.MediaConfig{IP: ip, HTTPPort: port, Secret: secret})
	if err := client.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	v := View{
		ID: fmt.Sprintf("%s:%d", ip, port), IP: ip, HTTPPort: port,
		Secret: secret, Type: s.mediaCfg.BackendType(), Online: true,
	}
	return &v, nil
}

func (s *Service) Save(m *model.MediaServer) error {
	return s.repo.Save(m)
}

func (s *Service) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *Service) MediaInfo(app, stream, mediaServerID string) (map[string]any, error) {
	_, err := s.GetOne(mediaServerID)
	if err != nil {
		return nil, err
	}
	info := s.zlm.LookupStreamMediaInfo(context.Background(), app, stream)
	if info == nil {
		return nil, fmt.Errorf("流不存在或已离线")
	}
	fps := any(info.VideoFps)
	if info.Fps > 0 {
		fps = info.Fps
	}
	var audioSampleRate any
	if info.SampleRate > 0 {
		audioSampleRate = info.SampleRate
	}
	var channels any
	if info.Channels > 0 {
		channels = info.Channels
	}
	return map[string]any{
		"app":             info.App,
		"stream":          info.Stream,
		"schema":          info.Schema,
		"videoCodec":      info.VideoCodec,
		"audioCodec":      info.AudioCodec,
		"readerCount":     info.ReaderCount,
		"aliveSecond":     info.AliveSecond,
		"bytesSpeed":      info.BytesSpeed,
		"fps":             fps,
		"width":           info.Width,
		"height":          info.Height,
		"audioSampleRate": audioSampleRate,
		"channels":        channels,
		"originType":      info.OriginType,
		"originTypeStr":   info.OriginTypeStr,
	}, nil
}

func (s *Service) Load() ([]View, error) {
	return s.ListOnline()
}

func (s *Service) isOnline(m model.MediaServer) bool {
	client := s.zlm
	if m.IP != s.mediaCfg.IP || m.HTTPPort != s.mediaCfg.HTTPPort {
		client = mediakit.NewClient(config.MediaConfig{IP: m.IP, HTTPPort: m.HTTPPort, Secret: m.Secret})
	}
	return client.Ping(context.Background()) == nil
}

func (s *Service) defaultView() View {
	return s.toView(model.MediaServer{
		ID: s.mediaCfg.ID, IP: s.mediaCfg.IP, HookIP: s.mediaCfg.IP,
		StreamIP: s.mediaCfg.IP, HTTPPort: s.mediaCfg.HTTPPort,
		RTMPPort: 1935, RTSPPort: 8554, Secret: s.mediaCfg.Secret,
		Type: s.mediaCfg.BackendType(), DefaultServer: true,
	}, s.isOnline(model.MediaServer{IP: s.mediaCfg.IP, HTTPPort: s.mediaCfg.HTTPPort, Secret: s.mediaCfg.Secret}))
}

func (s *Service) toView(m model.MediaServer, online bool) View {
	return View{
		ID: m.ID, IP: m.IP, HookIP: m.HookIP, StreamIP: m.StreamIP,
		HTTPPort: m.HTTPPort, RTMPPort: m.RTMPPort, RTSPPort: m.RTSPPort,
		RTPProxyPort: m.RTPProxyPort, Secret: m.Secret, Type: m.Type,
		DefaultServer: m.DefaultServer, Online: online, CreateTime: m.CreateTime,
	}
}
