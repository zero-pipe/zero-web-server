package platformapp

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	domainplatform "zero-web-kit/internal/domain/platform"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
	"zero-web-kit/internal/infrastructure/config"
)

type Service struct {
	platforms domainplatform.Repository
	sip       *sipinfra.PlatformClient
	cfg       config.SIPConfig
	serverID  string
	tasks     sync.Map // platformID -> cancel func
}

func NewService(platforms domainplatform.Repository, sipCfg config.SIPConfig, serverID string) *Service {
	return &Service{
		platforms: platforms,
		sip:       sipinfra.NewPlatformClient(sipCfg),
		cfg:       sipCfg,
		serverID:  serverID,
	}
}

func (s *Service) List(page, count int, query string) ([]*domainplatform.Platform, int64, error) {
	return s.platforms.List(page, count, query)
}

func (s *Service) Add(p *domainplatform.Platform) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	p.CreateTime = now
	p.UpdateTime = now
	p.ServerID = s.serverID
	if err := s.platforms.Create(p); err != nil {
		return err
	}
	if p.Enable {
		go s.registerPlatform(p)
	}
	return nil
}

func (s *Service) Update(p *domainplatform.Platform) error {
	p.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	if err := s.platforms.Update(p); err != nil {
		return err
	}
	s.stopKeepalive(p.ID)
	if p.Enable {
		go s.registerPlatform(p)
	} else {
		_ = s.platforms.UpdateStatus(p.ID, false)
	}
	return nil
}

func (s *Service) Delete(id int) error {
	s.stopKeepalive(id)
	return s.platforms.Delete(id)
}

func (s *Service) ServerConfig() map[string]any {
	return map[string]any{
		"deviceIp":   s.cfg.Domain,
		"deviceId":   s.cfg.ID,
		"devicePort": s.cfg.Port,
	}
}

func (s *Service) registerPlatform(p *domainplatform.Platform) {
	if err := s.sip.Register(p); err != nil {
		_ = s.platforms.UpdateStatus(p.ID, false)
		return
	}
	_ = s.platforms.UpdateStatus(p.ID, true)
	s.startKeepalive(p)
}

func (s *Service) startKeepalive(p *domainplatform.Platform) {
	ctx, cancel := context.WithCancel(context.Background())
	s.tasks.Store(p.ID, cancel)
	interval := 60
	if p.KeepTimeout != "" {
		if v, err := strconv.Atoi(p.KeepTimeout); err == nil && v > 0 {
			interval = v
		}
	}
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()
		failures := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.sip.Keepalive(p); err != nil {
					failures++
					if failures >= 3 {
						_ = s.platforms.UpdateStatus(p.ID, false)
						return
					}
				} else {
					failures = 0
				}
			}
		}
	}()
}

func (s *Service) stopKeepalive(platformID int) {
	if v, ok := s.tasks.LoadAndDelete(platformID); ok {
		if cancel, ok := v.(context.CancelFunc); ok {
			cancel()
		}
	}
}

func (s *Service) StartEnabledPlatforms() {
	list, err := s.platforms.ListEnabled()
	if err != nil {
		return
	}
	for _, p := range list {
		go s.registerPlatform(p)
	}
}

func (s *Service) Exit(deviceGBID string) error {
	p, err := s.platforms.GetByServerGBID(deviceGBID)
	if err != nil {
		return ErrPlatformNotFound
	}
	s.stopKeepalive(p.ID)
	_ = s.sip.Unregister(p)
	_ = s.platforms.UpdateStatus(p.ID, false)
	return nil
}

var ErrPlatformNotFound = errors.New("平台不存在")