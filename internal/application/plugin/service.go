package pluginapp

import (
	"context"
	"time"

	domainplugin "zero-web-server/internal/domain/plugin"
	applog "zero-web-server/pkg/log"
)

// Service 管理运行时插件实例。插件注册/心跳由 zero-agent 统一代管。
type Service struct {
	repo domainplugin.Repository
}

func NewService(repo domainplugin.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List() ([]*domainplugin.Plugin, error) {
	return s.repo.ListPlugins()
}

func (s *Service) Resolve(protocol string) (*domainplugin.Plugin, error) {
	return s.repo.ResolvePlugin(protocol)
}

// MarkOffline 在 RPC 失败时标记插件离线。
func (s *Service) MarkOffline(id string) {
	_ = s.repo.UpdatePluginSeen(id, time.Now().UnixMilli(), false)
}

// Start 启动心跳超时巡检：超过 timeout 未上报心跳的插件标记离线，其设备置离线。
func (s *Service) Start(ctx context.Context, interval, timeout time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.SweepTimeout(timeout)
			}
		}
	}()
}

// SweepTimeout 将 last_seen 超时的在线插件置离线。
func (s *Service) SweepTimeout(timeout time.Duration) {
	list, err := s.repo.ListPlugins()
	if err != nil {
		return
	}
	now := time.Now().UnixMilli()
	for _, p := range list {
		if !p.Online {
			continue
		}
		if now-p.LastSeen > timeout.Milliseconds() {
			if err := s.repo.MarkPluginOffline(p.ID); err == nil {
				applog.Warnf("[plugin] %s heartbeat timeout, marked offline", p.ID)
			}
		}
	}
}
