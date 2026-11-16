package pluginapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainplugin "zero-web-server/internal/domain/plugin"
	applog "zero-web-server/pkg/log"
)

type Service struct {
	repo domainplugin.Repository
}

func NewService(repo domainplugin.Repository) *Service {
	return &Service{repo: repo}
}

// HelloRequest 是插件启动时上报的注册信息。
type HelloRequest struct {
	ID               string   `json:"id"`
	PluginID         string   `json:"pluginId"`
	Vendor           string   `json:"vendor"`
	Version          string   `json:"version"`
	Protocols        []string `json:"protocols"`
	Capabilities     []string `json:"capabilities"`
	StreamTransports []string `json:"streamTransports"`
	ConfigSchema     string   `json:"configSchema"`
	Endpoint         string   `json:"endpoint"`
}

func (s *Service) Hello(_ context.Context, req HelloRequest) (*domainplugin.Plugin, error) {
	req.ID = strings.TrimSpace(req.ID)
	req.Endpoint = strings.TrimSpace(req.Endpoint)
	if req.ID == "" {
		return nil, fmt.Errorf("插件实例 ID 不能为空")
	}
	if req.Endpoint == "" {
		return nil, fmt.Errorf("插件 endpoint 不能为空")
	}
	if len(req.Protocols) == 0 {
		return nil, fmt.Errorf("插件必须上报至少一个协议")
	}
	p := &domainplugin.Plugin{
		ID: req.ID, PluginID: req.PluginID, Vendor: req.Vendor, Version: req.Version,
		Protocols: req.Protocols, Capabilities: req.Capabilities, StreamTransports: req.StreamTransports,
		ConfigSchema: req.ConfigSchema, Endpoint: req.Endpoint,
	}
	if err := s.repo.UpsertPlugin(p); err != nil {
		return nil, err
	}
	return s.repo.GetPlugin(req.ID)
}

func (s *Service) Heartbeat(_ context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("插件实例 ID 不能为空")
	}
	return s.repo.UpdatePluginSeen(id, time.Now().UnixMilli(), true)
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
