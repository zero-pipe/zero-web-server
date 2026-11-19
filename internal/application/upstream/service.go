package upstreamapp

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	domainplatform "zero-web-server/internal/domain/platform"
	"zero-web-server/internal/infrastructure/config"
	"zero-web-server/internal/port"
)

type Service struct {
	platforms domainplatform.Repository
	sip       port.PlatformClient
	cfg       config.SIPConfig
	serverID  string
	tasks     sync.Map // platformID -> cancel func
}

func NewService(platforms domainplatform.Repository, sipCfg config.SIPConfig, sipClient port.PlatformClient, serverID string) *Service {
	return &Service{
		platforms: platforms,
		sip:       sipClient,
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
		s.registerPlatform(p)
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
		s.registerPlatform(p)
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
		"deviceIp":     s.cfg.IP,
		"deviceId":     s.cfg.ID,
		"deviceDomain": s.cfg.Domain,
		"devicePort":   s.cfg.Port,
		"password":     s.cfg.Password,
	}
}

func (s *Service) ApplySIPConfig(cfg config.SIPConfig) {
	s.cfg = cfg
}

func (s *Service) registerPlatform(p *domainplatform.Platform) {
	go s.runPlatform(p)
}

// Registration lifecycle constants (01-§1): refresh before expiry, and retry
// a failed REGISTER no more often than every retryInterval.
const (
	registerRetryInterval = 60 * time.Second
	keepaliveDefaultSecs  = 60
	keepaliveFailLimit    = 3
)

func (s *Service) runPlatform(p *domainplatform.Platform) {
	s.stopKeepalive(p.ID)
	ctx, cancel := context.WithCancel(context.Background())
	s.tasks.Store(p.ID, cancel)
	defer s.tasks.Delete(p.ID)

	expires := platformExpiresSecs(p)
	for {
		if ctx.Err() != nil {
			return
		}
		if err := s.sip.Register(p); err != nil {
			_ = s.platforms.UpdateStatus(p.ID, false)
			if !sleepCtx(ctx, registerRetryInterval) {
				return
			}
			continue
		}
		_ = s.platforms.UpdateStatus(p.ID, true)
		if !s.keepaliveLoop(ctx, p, expires) {
			return // cancelled
		}
		// Keepalive exhausted: fall through and re-REGISTER.
	}
}

// keepaliveLoop runs heartbeats and refreshes the registration before it
// expires. Returns false when ctx is cancelled, true when the caller should
// re-register (keepalive exhausted or refresh failed).
func (s *Service) keepaliveLoop(ctx context.Context, p *domainplatform.Platform, expires int) bool {
	interval := keepaliveDefaultSecs
	if p.KeepTimeout != "" {
		if v, err := strconv.Atoi(p.KeepTimeout); err == nil && v > 0 {
			interval = v
		}
	}
	// Refresh at ~2/3 of the lifetime, but never faster than one keepalive tick.
	refreshAfter := time.Duration(expires) * time.Second * 2 / 3
	if refreshAfter < time.Duration(interval)*time.Second {
		refreshAfter = time.Duration(interval) * time.Second
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	failures := 0
	lastRegister := time.Now()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if time.Since(lastRegister) >= refreshAfter {
				if err := s.sip.Register(p); err != nil {
					_ = s.platforms.UpdateStatus(p.ID, false)
					return true
				}
				lastRegister = time.Now()
			}
			if err := s.sip.Keepalive(p); err != nil {
				failures++
				if failures >= keepaliveFailLimit {
					_ = s.platforms.UpdateStatus(p.ID, false)
					return true
				}
			} else {
				failures = 0
			}
		}
	}
}

func platformExpiresSecs(p *domainplatform.Platform) int {
	if p.Expires != "" {
		if v, err := strconv.Atoi(p.Expires); err == nil && v > 0 {
			return v
		}
	}
	return 3600
}

// sleepCtx waits for d or ctx cancellation. Returns false if cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
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
		s.registerPlatform(p)
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
