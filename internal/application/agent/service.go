package agentapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	domainagent "zero-web-server/internal/domain/agent"
	domainplugin "zero-web-server/internal/domain/plugin"
	applog "zero-web-server/pkg/log"
)

type Service struct {
	repo    domainagent.Repository
	plugins domainplugin.Repository
	token   string
	http    *http.Client
	onStart func(ctx context.Context, pluginID string)
	devices DeviceCounter
}

// DeviceCounter 统计插件设备引用（由 ipcdevice 服务实现），用于禁用/删除前检查。
type DeviceCounter interface {
	CountByPlugin(pluginID string) (int, error)
}

func NewService(repo domainagent.Repository, plugins domainplugin.Repository, token string) *Service {
	return &Service{
		repo: repo, plugins: plugins, token: strings.TrimSpace(token),
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// SetDeviceCounter 注入设备计数器，用于「先删设备→再禁用→再删除」的门控。
func (s *Service) SetDeviceCounter(d DeviceCounter) { s.devices = d }

// SetPluginStartedHook 在插件由离线转为运行态时触发，用于补布防等恢复动作。
func (s *Service) SetPluginStartedHook(fn func(ctx context.Context, pluginID string)) {
	s.onStart = fn
}

// RegisterRequest 是 Agent 启动/重连时上报的完整状态。
type RegisterRequest struct {
	AgentID  string          `json:"agentId"`
	Hostname string          `json:"hostname"`
	IP       string          `json:"ip"`
	Version  string          `json:"version"`
	Endpoint string          `json:"endpoint"`
	Plugins  []PluginRequest `json:"plugins"`
}

// PluginRequest 是插件清单 + 当前状态。
type PluginRequest struct {
	InstanceID       string   `json:"instanceId"`
	PluginID         string   `json:"pluginId"`
	Name             string   `json:"name"`
	Vendor           string   `json:"vendor"`
	Version          string   `json:"version"`
	Protocols        []string `json:"protocols"`
	Capabilities     []string `json:"capabilities"`
	StreamTransports []string `json:"streamTransports"`
	EventTypes       []string `json:"eventTypes"`
	ConfigSchema     string   `json:"configSchema"`
	Enabled          bool     `json:"enabled"`
	DesiredState     string   `json:"desiredState"`
	ActualState      string   `json:"actualState"`
	Source           string   `json:"source"`
	LastError        string   `json:"lastError"`
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*domainagent.Agent, error) {
	req.AgentID = strings.TrimSpace(req.AgentID)
	req.Endpoint = strings.TrimRight(strings.TrimSpace(req.Endpoint), "/")
	if req.AgentID == "" {
		return nil, fmt.Errorf("agentId 不能为空")
	}
	if req.Endpoint == "" {
		return nil, fmt.Errorf("agent endpoint 不能为空")
	}
	now := time.Now().UnixMilli()
	agent := &domainagent.Agent{
		ID: req.AgentID, Hostname: req.Hostname, IP: req.IP, Version: req.Version,
		Endpoint: req.Endpoint, Status: "online", LastSeen: now,
	}
	if err := s.repo.UpsertAgent(agent); err != nil {
		return nil, err
	}
	for _, p := range req.Plugins {
		if err := s.upsertPlugin(req.AgentID, req.Endpoint, p, now); err != nil {
			return nil, err
		}
	}
	return s.repo.GetAgent(req.AgentID)
}

func (s *Service) Heartbeat(ctx context.Context, req RegisterRequest) error {
	req.AgentID = strings.TrimSpace(req.AgentID)
	if req.AgentID == "" {
		return fmt.Errorf("agentId 不能为空")
	}
	now := time.Now().UnixMilli()
	agent, err := s.repo.GetAgent(req.AgentID)
	if err != nil {
		return fmt.Errorf("agent 未注册")
	}
	if strings.TrimSpace(req.Endpoint) != "" {
		agent.Endpoint = strings.TrimRight(strings.TrimSpace(req.Endpoint), "/")
	}
	agent.Hostname = req.Hostname
	agent.IP = req.IP
	agent.Version = req.Version
	agent.Status = "online"
	agent.LastSeen = now
	if err := s.repo.UpsertAgent(agent); err != nil {
		return err
	}
	for _, p := range req.Plugins {
		if err := s.upsertPlugin(req.AgentID, agent.Endpoint, p, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) upsertPlugin(agentID, endpoint string, p PluginRequest, now int64) error {
	if strings.TrimSpace(p.PluginID) == "" {
		return fmt.Errorf("pluginId 不能为空")
	}
	instanceID := strings.TrimSpace(p.InstanceID)
	if instanceID == "" {
		instanceID = agentID + "-" + p.PluginID
	}
	desired := strings.TrimSpace(p.DesiredState)
	if desired == "" {
		if p.Enabled {
			desired = domainagent.StateEnabled
		} else {
			desired = domainagent.StateDisabled
		}
	}
	actual := strings.TrimSpace(p.ActualState)
	if actual == "" {
		actual = "discovered"
	}
	item := &domainagent.Plugin{
		ID: instanceID, AgentID: agentID, PluginID: p.PluginID, Name: p.Name,
		Vendor: p.Vendor, Version: p.Version,
		Protocols: p.Protocols, Capabilities: p.Capabilities,
		StreamTransports: p.StreamTransports, EventTypes: p.EventTypes,
		ConfigSchema: p.ConfigSchema, Enabled: desired == domainagent.StateEnabled,
		DesiredState: desired, ActualState: actual, Source: p.Source, LastError: p.LastError, LastSeen: now,
	}
	if err := s.repo.UpsertPlugin(item); err != nil {
		return err
	}
	// 以平台已保存的 desired state 为准，避免 Agent 本地状态回滚平台启停决定。
	if stored, err := s.repo.GetPlugin(agentID, p.PluginID); err == nil {
		item = stored
	}
	return s.syncRuntime(item, endpoint, now)
}

func (s *Service) syncRuntime(p *domainagent.Plugin, endpoint string, now int64) error {
	running := p.Enabled && p.ActualState == "running"
	existing, existingErr := s.plugins.GetPlugin(p.ID)
	if running {
		runtime := &domainplugin.Plugin{
			ID: p.ID, PluginID: p.PluginID, Vendor: p.Vendor, Version: p.Version,
			Protocols: p.Protocols, Capabilities: p.Capabilities,
			StreamTransports: p.StreamTransports, ConfigSchema: p.ConfigSchema,
			Endpoint: strings.TrimRight(endpoint, "/") + "/plugins/" + p.PluginID,
		}
		if err := s.plugins.UpsertPlugin(runtime); err != nil {
			return err
		}
		if err := s.plugins.UpdatePluginSeen(p.ID, now, true); err != nil {
			return err
		}
		if s.onStart != nil && (existingErr != nil || !existing.Online) {
			go s.onStart(context.Background(), p.ID)
		}
		return nil
	}
	return s.plugins.MarkPluginOffline(p.ID)
}

func (s *Service) ListAgents() ([]*domainagent.Agent, error) {
	return s.repo.ListAgents()
}

func (s *Service) ListPlugins(agentID string) ([]*domainagent.Plugin, error) {
	return s.repo.ListPlugins(strings.TrimSpace(agentID))
}

// SetPluginState 向 Agent 下发启停/重启命令，成功后更新平台状态。
func (s *Service) SetPluginState(ctx context.Context, agentID, pluginID, action string) ([]*domainagent.Plugin, error) {
	agentID = strings.TrimSpace(agentID)
	pluginID = strings.TrimSpace(pluginID)
	if agentID == "" || pluginID == "" {
		return nil, fmt.Errorf("agentId/pluginId 不能为空")
	}
	switch action {
	case "enable", "disable", "restart":
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
	agent, err := s.repo.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("Agent 不存在")
	}
	if agent.Status != "online" {
		return nil, fmt.Errorf("Agent 离线")
	}
	// 门控：禁用/重启插件前要求该插件下已无设备（先删设备，再停插件）。
	if action == "disable" && s.devices != nil {
		if n, err := s.devices.CountByPlugin(pluginID); err == nil && n > 0 {
			return nil, fmt.Errorf("该插件下仍有 %d 个设备，请先删除设备后再禁用", n)
		}
	}
	body, err := s.callAgent(ctx, agent, pluginID, action)
	if err != nil {
		return nil, err
	}
	desired := domainagent.StateDisabled
	if action == "enable" || action == "restart" {
		desired = domainagent.StateEnabled
	}
	if err := s.repo.UpdatePluginDesired(agentID, pluginID, desired); err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	for _, p := range body.Plugins {
		if p.PluginID != pluginID {
			continue
		}
		if err := s.repo.UpdatePluginReport(agentID, pluginID, p.ActualState, p.LastError, now); err != nil {
			return nil, err
		}
		if item, err := s.repo.GetPlugin(agentID, pluginID); err == nil {
			if err := s.syncRuntime(item, agent.Endpoint, now); err != nil {
				return nil, err
			}
		}
	}
	return s.repo.ListPlugins(agentID)
}

// DeleteBuiltinPlugin 删除内置/本地插件：要求已无设备且插件已禁用，随后让 Agent 删除插件目录。
func (s *Service) DeleteBuiltinPlugin(ctx context.Context, agentID, pluginID string) error {
	agentID = strings.TrimSpace(agentID)
	pluginID = strings.TrimSpace(pluginID)
	if agentID == "" || pluginID == "" {
		return fmt.Errorf("agentId/pluginId 不能为空")
	}
	p, err := s.repo.GetPlugin(agentID, pluginID)
	if err != nil || p == nil {
		return fmt.Errorf("插件不存在")
	}
	// 门控：先删设备。
	if s.devices != nil {
		if n, err := s.devices.CountByPlugin(pluginID); err == nil && n > 0 {
			return fmt.Errorf("该插件下仍有 %d 个设备，请先删除设备", n)
		}
	}
	// 门控：停用了插件才能删除插件包（插件须已禁用且未在运行）。
	if p.DesiredState == domainagent.StateEnabled || p.ActualState == "running" || p.ActualState == "starting" {
		return fmt.Errorf("请先禁用插件，再删除插件包")
	}
	agent, err := s.repo.GetAgent(agentID)
	if err != nil || agent == nil {
		return fmt.Errorf("Agent 不存在")
	}
	if agent.Status != "online" {
		return fmt.Errorf("Agent 离线，无法删除")
	}
	if err := s.callAgentRemove(ctx, agent, pluginID); err != nil {
		return err
	}
	// 清理平台侧记录：运行时实例 + Agent 插件清单。
	_ = s.plugins.DeletePlugin(p.ID)
	if err := s.repo.DeletePlugin(agentID, pluginID); err != nil {
		return err
	}
	applog.Infof("[agent] 已删除内置插件 %s（Agent %s）", pluginID, agentID)
	return nil
}

func (s *Service) callAgentRemove(ctx context.Context, agent *domainagent.Agent, pluginID string) error {
	url := strings.TrimRight(agent.Endpoint, "/") + "/api/v1/plugins/" + pluginID + "/remove"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("X-Agent-Token", s.token)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("调用 Agent 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("Agent 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

type agentStatus struct {
	Plugins []PluginRequest `json:"plugins"`
}

func (s *Service) callAgent(ctx context.Context, agent *domainagent.Agent, pluginID, action string) (*agentStatus, error) {
	url := strings.TrimRight(agent.Endpoint, "/") + "/api/v1/plugins/" + pluginID + "/" + action
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("X-Agent-Token", s.token)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 Agent 失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Agent 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out agentStatus
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("Agent 响应格式错误: %w", err)
		}
	}
	return &out, nil
}

// Start 巡检 Agent 心跳，超时则标记 Agent 与其插件离线。
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
				s.sweep(timeout)
			}
		}
	}()
}

func (s *Service) sweep(timeout time.Duration) {
	list, err := s.repo.ListAgents()
	if err != nil {
		return
	}
	now := time.Now().UnixMilli()
	for _, a := range list {
		if a.Status != "online" || now-a.LastSeen <= timeout.Milliseconds() {
			continue
		}
		if err := s.repo.UpdateAgentSeen(a.ID, a.LastSeen, false); err == nil {
			applog.Warnf("[agent] %s heartbeat timeout, marked offline", a.ID)
		}
		plugins, _ := s.repo.ListPlugins(a.ID)
		for _, p := range plugins {
			_ = s.repo.UpdatePluginReport(a.ID, p.PluginID, "offline", "agent offline", now)
			_ = s.plugins.MarkPluginOffline(p.ID)
		}
	}
}
